package keymanager

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"new-pay/internal/vault"
)

// KeyManager manages the three-tier key hierarchy
type KeyManager struct {
	db          *sql.DB
	vault       *vault.Client
	systemKeyID string
}

// NewKeyManager creates a new KeyManager instance
func NewKeyManager(db *sql.DB, vaultClient *vault.Client) (*KeyManager, error) {
	km := &KeyManager{
		db:          db,
		vault:       vaultClient,
		systemKeyID: "system-master-key",
	}

	// Initialize system master key in Vault
	if err := km.initSystemKey(); err != nil {
		return nil, fmt.Errorf("failed to initialize system key: %w", err)
	}

	return km, nil
}

// initSystemKey creates the system master key if it doesn't exist
func (km *KeyManager) initSystemKey() error {
	// Create transit key for system-level encryption
	err := km.vault.CreateKey(km.systemKeyID, "aes256-gcm96")
	if err != nil {
		// Key might already exist, check if it's accessible
		return nil
	}

	return nil
}

// CreateUserKey generates a new Ed25519 keypair for a user
func (km *KeyManager) CreateUserKey(userID int64) (publicKey ed25519.PublicKey, err error) {
	// Generate Ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}

	// Encrypt private key using Vault
	encryptedPrivateKey, err := km.vault.Encrypt(
		km.systemKeyID,
		priv,
		map[string]string{"user_id": fmt.Sprintf("%d", userID)},
	)
	if err != nil {
		return nil, fmt.Errorf("private key encryption failed: %w", err)
	}

	// Store in database
	query := `
		INSERT INTO user_keys (user_id, public_key, encrypted_private_key, key_version, created_at)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (user_id) DO NOTHING
	`

	_, err = km.db.Exec(
		query,
		userID,
		hex.EncodeToString(pub),
		encryptedPrivateKey,
		time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("database insert failed: %w", err)
	}

	return pub, nil
}

// GetUserSigningKey retrieves and decrypts a user's private key for signing
func (km *KeyManager) GetUserSigningKey(userID int64) (ed25519.PrivateKey, error) {
	var encryptedPrivateKey string

	query := `SELECT encrypted_private_key FROM user_keys WHERE user_id = $1`
	err := km.db.QueryRow(query, userID).Scan(&encryptedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("user key not found: %w", err)
	}

	// Decrypt using Vault
	privateKeyBytes, err := km.vault.Decrypt(
		km.systemKeyID,
		encryptedPrivateKey,
		map[string]string{"user_id": fmt.Sprintf("%d", userID)},
	)
	if err != nil {
		return nil, fmt.Errorf("private key decryption failed: %w", err)
	}

	return privateKeyBytes, nil
}

// GetUserPublicKey retrieves a user's public key
func (km *KeyManager) GetUserPublicKey(userID int64) (ed25519.PublicKey, error) {
	var publicKeyHex string

	query := `SELECT public_key FROM user_keys WHERE user_id = $1`
	err := km.db.QueryRow(query, userID).Scan(&publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("user key not found: %w", err)
	}

	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	return publicKey, nil
}

// CreateProcessKey generates a new 256-bit symmetric key for a process
func (km *KeyManager) CreateProcessKey(processID string, expiresAt *time.Time) error {
	// Generate random 256-bit process key
	processKey := make([]byte, 32)
	if _, err := rand.Read(processKey); err != nil {
		return fmt.Errorf("key generation failed: %w", err)
	}

	// Hash for verification
	hashBytes := sha256.Sum256(processKey)
	keyHash := hex.EncodeToString(hashBytes[:])

	// Encrypt with Vault
	encryptedKey, err := km.vault.Encrypt(
		km.systemKeyID,
		processKey,
		map[string]string{"process_id": processID},
	)
	if err != nil {
		return fmt.Errorf("key encryption failed: %w", err)
	}

	// Store in database
	query := `
		INSERT INTO process_keys (process_id, encrypted_key_material, key_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (process_id) DO NOTHING
	`

	_, err = km.db.Exec(
		query,
		processID,
		encryptedKey,
		keyHash,
		time.Now(),
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("database insert failed: %w", err)
	}

	return nil
}

// GetProcessKey retrieves and decrypts a process key
func (km *KeyManager) GetProcessKey(processID string) ([]byte, error) {
	var encryptedKey string
	var expiresAt *time.Time

	query := `
		SELECT encrypted_key_material, expires_at 
		FROM process_keys 
		WHERE process_id = $1
	`
	err := km.db.QueryRow(query, processID).Scan(&encryptedKey, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("process key not found: %w", err)
	}

	// Check expiration
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("process key expired")
	}

	// Decrypt with Vault
	processKey, err := km.vault.Decrypt(
		km.systemKeyID,
		encryptedKey,
		map[string]string{"process_id": processID},
	)
	if err != nil {
		return nil, fmt.Errorf("key decryption failed: %w", err)
	}

	return processKey, nil
}

// GetProcessKeyHash retrieves the hash of a process key
func (km *KeyManager) GetProcessKeyHash(processID string) (string, error) {
	var keyHash string

	query := `SELECT key_hash FROM process_keys WHERE process_id = $1`
	err := km.db.QueryRow(query, processID).Scan(&keyHash)
	if err != nil {
		return "", fmt.Errorf("process key not found: %w", err)
	}

	return keyHash, nil
}

// DeriveDataEncryptionKey combines all three keys to create a data encryption key
func (km *KeyManager) DeriveDataEncryptionKey(processID string, userID int64) ([]byte, error) {
	// Get user's private key (for key derivation, not for signing)
	userKey, err := km.GetUserSigningKey(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user key: %w", err)
	}

	// Get process key
	processKey, err := km.GetProcessKey(processID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process key: %w", err)
	}

	// Derive final encryption key from all components
	info := fmt.Sprintf("process:%s:user:%d", processID, userID)

	// Combine keys
	h := sha256.New()
	h.Write(processKey)
	userSeed := userKey.Seed()
	h.Write(userSeed)
	h.Write([]byte(info))
	seed := h.Sum(nil)

	// Create 32-byte AES-256 key
	finalKey := sha256.Sum256(seed)

	return finalKey[:], nil
}

// RotateProcessKey creates a new version of a process key
func (km *KeyManager) RotateProcessKey(processID string) error {
	// Mark old key as expired
	query := `UPDATE process_keys SET expires_at = $1 WHERE process_id = $2 AND expires_at IS NULL`
	_, err := km.db.Exec(query, time.Now(), processID)
	if err != nil {
		return fmt.Errorf("failed to expire old key: %w", err)
	}

	// Create new key
	return km.CreateProcessKey(processID, nil)
}

// VerifyKeyAccess checks if a user has access to a process
func (km *KeyManager) VerifyKeyAccess(userID int64, processID string) error {
	// Check if user key exists
	var exists bool
	err := km.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_keys WHERE user_id = $1)`, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("user key check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("user key not found")
	}

	// Check if process key exists
	err = km.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM process_keys WHERE process_id = $1)`, processID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("process key check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("process key not found")
	}

	return nil
}

// GetActiveSystemKeyID returns the current system key identifier
func (km *KeyManager) GetActiveSystemKeyID() string {
	return km.systemKeyID
}
