// Package keymanager manages the two-tier key hierarchy used by SecureStore.
// The master key is supplied via ENCRYPTION_MASTER_KEY (32 bytes, hex-encoded).
package keymanager

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	appCrypto "new-pay/internal/crypto"
)

// KeyManager manages user keys and process keys using a local master key.
type KeyManager struct {
	db        *sql.DB
	masterKey []byte
}

// NewKeyManager creates a new KeyManager. masterKey must be exactly 32 bytes.
func NewKeyManager(db *sql.DB, masterKey []byte) (*KeyManager, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("masterKey must be 32 bytes, got %d", len(masterKey))
	}
	mkCopy := make([]byte, 32)
	copy(mkCopy, masterKey)
	return &KeyManager{db: db, masterKey: mkCopy}, nil
}

// CreateUserKey generates a new Ed25519 keypair, encrypts the private key with
// the master key, and stores it in user_keys.
func (km *KeyManager) CreateUserKey(userID int64) (ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}

	encryptedPriv, err := appCrypto.EncryptWithMasterKey(priv, km.masterKey,
		fmt.Sprintf("user_key:user_id=%d", userID))
	if err != nil {
		return nil, fmt.Errorf("private key encryption failed: %w", err)
	}

	_, err = km.db.Exec(
		`INSERT INTO user_keys (user_id, public_key, encrypted_private_key, key_version, created_at)
		 VALUES ($1, $2, $3, 1, $4) ON CONFLICT (user_id) DO NOTHING`,
		userID, hex.EncodeToString(pub), encryptedPriv, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("database insert failed: %w", err)
	}
	return pub, nil
}

// GetUserSigningKey retrieves and decrypts a user's Ed25519 private key.
func (km *KeyManager) GetUserSigningKey(userID int64) (ed25519.PrivateKey, error) {
	var encryptedPriv string
	if err := km.db.QueryRow(
		`SELECT encrypted_private_key FROM user_keys WHERE user_id = $1`, userID,
	).Scan(&encryptedPriv); err != nil {
		return nil, fmt.Errorf("user key not found: %w", err)
	}

	privBytes, err := appCrypto.DecryptWithMasterKey(encryptedPriv, km.masterKey,
		fmt.Sprintf("user_key:user_id=%d", userID))
	if err != nil {
		return nil, fmt.Errorf("private key decryption failed: %w", err)
	}
	return privBytes, nil
}

// GetUserPublicKey retrieves a user's Ed25519 public key.
func (km *KeyManager) GetUserPublicKey(userID int64) (ed25519.PublicKey, error) {
	var pubHex string
	if err := km.db.QueryRow(
		`SELECT public_key FROM user_keys WHERE user_id = $1`, userID,
	).Scan(&pubHex); err != nil {
		return nil, fmt.Errorf("user key not found: %w", err)
	}
	pub, err := hex.DecodeString(pubHex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key encoding: %w", err)
	}
	return pub, nil
}

// CreateProcessKey generates a 256-bit process key, encrypts it with the
// master key, and stores it in process_keys.
func (km *KeyManager) CreateProcessKey(processID string, expiresAt *time.Time) error {
	processKey := make([]byte, 32)
	if _, err := rand.Read(processKey); err != nil {
		return fmt.Errorf("key generation failed: %w", err)
	}

	hashBytes := sha256.Sum256(processKey)
	keyHash := hex.EncodeToString(hashBytes[:])

	encryptedKey, err := appCrypto.EncryptWithMasterKey(processKey, km.masterKey,
		fmt.Sprintf("process_key:process_id=%s", processID))
	if err != nil {
		return fmt.Errorf("key encryption failed: %w", err)
	}

	_, err = km.db.Exec(
		`INSERT INTO process_keys (process_id, encrypted_key_material, key_hash, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT (process_id) DO NOTHING`,
		processID, encryptedKey, keyHash, time.Now(), expiresAt,
	)
	return err
}

// GetProcessKey retrieves and decrypts a process key.
func (km *KeyManager) GetProcessKey(processID string) ([]byte, error) {
	var encryptedKey string
	var expiresAt *time.Time
	if err := km.db.QueryRow(
		`SELECT encrypted_key_material, expires_at FROM process_keys WHERE process_id = $1`,
		processID,
	).Scan(&encryptedKey, &expiresAt); err != nil {
		return nil, fmt.Errorf("process key not found: %w", err)
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("process key expired")
	}

	processKey, err := appCrypto.DecryptWithMasterKey(encryptedKey, km.masterKey,
		fmt.Sprintf("process_key:process_id=%s", processID))
	if err != nil {
		return nil, fmt.Errorf("key decryption failed: %w", err)
	}
	return processKey, nil
}

// GetProcessKeyHash retrieves the SHA-256 hash of a process key.
func (km *KeyManager) GetProcessKeyHash(processID string) (string, error) {
	var keyHash string
	if err := km.db.QueryRow(
		`SELECT key_hash FROM process_keys WHERE process_id = $1`, processID,
	).Scan(&keyHash); err != nil {
		return "", fmt.Errorf("process key not found: %w", err)
	}
	return keyHash, nil
}

// DeriveDataEncryptionKey derives a 32-byte AES-256 DEK from the process key
// and the user's private key seed.
func (km *KeyManager) DeriveDataEncryptionKey(processID string, userID int64) ([]byte, error) {
	userKey, err := km.GetUserSigningKey(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user key: %w", err)
	}
	processKey, err := km.GetProcessKey(processID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process key: %w", err)
	}
	salt := append(processKey, userKey.Seed()...)
	dek, err := appCrypto.DeriveKey(km.masterKey, salt, fmt.Sprintf("process:%s:user:%d", processID, userID))
	if err != nil {
		return nil, fmt.Errorf("failed to derive data encryption key: %w", err)
	}
	return dek, nil
}

// RotateProcessKey expires the current process key and creates a new one.
func (km *KeyManager) RotateProcessKey(processID string) error {
	_, err := km.db.Exec(
		`UPDATE process_keys SET expires_at = $1 WHERE process_id = $2 AND expires_at IS NULL`,
		time.Now(), processID,
	)
	if err != nil {
		return fmt.Errorf("failed to expire old key: %w", err)
	}
	return km.CreateProcessKey(processID, nil)
}

// VerifyKeyAccess checks that both user key and process key exist.
func (km *KeyManager) VerifyKeyAccess(userID int64, processID string) error {
	var exists bool
	if err := km.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_keys WHERE user_id = $1)`, userID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("user key check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("user key not found")
	}

	if err := km.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM process_keys WHERE process_id = $1)`, processID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("process key check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("process key not found")
	}
	return nil
}

// GetActiveSystemKeyID returns a constant identifier stored as metadata in
// encrypted_records. Always returns "local".
func (km *KeyManager) GetActiveSystemKeyID() string {
	return "local"
}
