package securestore

import (
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"new-pay/internal/keymanager"
	"new-pay/internal/vault"
)

// SecureRecord represents an encrypted and signed record
type SecureRecord struct {
	ID                 int64     `json:"id"`
	ProcessID          string    `json:"process_id"`
	UserID             int64     `json:"user_id"`
	CreatedAt          time.Time `json:"created_at"`
	EncryptedData      []byte    `json:"-"`
	EncryptionNonce    []byte    `json:"-"`
	EncryptionTag      []byte    `json:"-"`
	KeyVersion         int       `json:"key_version"`
	SystemKeyID        string    `json:"system_key_id"`
	ProcessKeyHash     string    `json:"process_key_hash"`
	DataSignature      string    `json:"data_signature"`
	SignaturePublicKey string    `json:"signature_public_key"`
	RecordType         string    `json:"record_type"`
	Status             string    `json:"status,omitempty"`
	PrevRecordHash     string    `json:"prev_record_hash"`
	ChainHash          string    `json:"chain_hash"`
}

// PlainData represents unencrypted data structure
type PlainData struct {
	Fields   map[string]interface{} `json:"fields"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

// SecureStore manages encrypted records with hash chain audit trail
type SecureStore struct {
	db         *sql.DB
	keyManager *keymanager.KeyManager
}

// NewSecureStore creates a new SecureStore instance
func NewSecureStore(db *sql.DB, keyManager *keymanager.KeyManager) *SecureStore {
	return &SecureStore{
		db:         db,
		keyManager: keyManager,
	}
}

// CreateRecord encrypts, signs, and stores data
func (ss *SecureStore) CreateRecord(
	processID string,
	userID int64,
	recordType string,
	data *PlainData,
	status string,
) (*SecureRecord, error) {
	// Verify key access
	if err := ss.keyManager.VerifyKeyAccess(userID, processID); err != nil {
		return nil, fmt.Errorf("key access verification failed: %w", err)
	}

	// Serialize data
	plainBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	// Get data encryption key (derived from all three keys)
	dek, err := ss.keyManager.DeriveDataEncryptionKey(processID, userID)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	// Get signing key
	signingKey, err := ss.keyManager.GetUserSigningKey(userID)
	if err != nil {
		return nil, fmt.Errorf("signing key retrieval failed: %w", err)
	}

	// Encrypt with AES-256-GCM
	additionalData := []byte(fmt.Sprintf("process:%s:user:%d:type:%s", processID, userID, recordType))
	ciphertext, nonce, err := vault.EncryptLocal(plainBytes, dek, additionalData)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Extract GCM tag (last 16 bytes)
	tagSize := 16
	encryptedData := ciphertext[:len(ciphertext)-tagSize]
	tag := ciphertext[len(ciphertext)-tagSize:]

	// Sign: signature covers encrypted data + nonce + tag
	signatureInput := append(encryptedData, nonce...)
	signatureInput = append(signatureInput, tag...)
	signature := ed25519.Sign(signingKey, signatureInput)

	// Get previous hash for chain
	prevHash, err := ss.getLatestHash(processID)
	if err != nil {
		return nil, fmt.Errorf("prev hash retrieval failed: %w", err)
	}

	// Calculate chain hash
	now := time.Now().UTC()
	chainInput := fmt.Sprintf("%s:%s:%d:%s:%d",
		prevHash,
		hex.EncodeToString(signature),
		userID,
		processID,
		now.Unix(),
	)
	chainHashBytes := sha256.Sum256([]byte(chainInput))
	chainHash := hex.EncodeToString(chainHashBytes[:])

	// Get process key hash for verification
	processKeyHash, err := ss.keyManager.GetProcessKeyHash(processID)
	if err != nil {
		return nil, fmt.Errorf("process key hash retrieval failed: %w", err)
	}

	// Create record
	publicKey := ed25519.PublicKey(signingKey[32:])
	record := &SecureRecord{
		ProcessID:          processID,
		UserID:             userID,
		CreatedAt:          now,
		EncryptedData:      encryptedData,
		EncryptionNonce:    nonce,
		EncryptionTag:      tag,
		KeyVersion:         1,
		SystemKeyID:        ss.keyManager.GetActiveSystemKeyID(),
		ProcessKeyHash:     processKeyHash,
		DataSignature:      hex.EncodeToString(signature),
		SignaturePublicKey: hex.EncodeToString(publicKey),
		RecordType:         recordType,
		Status:             status,
		PrevRecordHash:     prevHash,
		ChainHash:          chainHash,
	}

	// Store in database
	if err := ss.insertRecord(record); err != nil {
		return nil, fmt.Errorf("database insert failed: %w", err)
	}

	return record, nil
}

// DecryptRecord decrypts a record automatically without user interaction
func (ss *SecureStore) DecryptRecord(recordID int64) (*PlainData, error) {
	// Load record from database
	record, err := ss.loadRecord(recordID)
	if err != nil {
		return nil, fmt.Errorf("record load failed: %w", err)
	}

	return ss.DecryptRecordData(record)
}

// DecryptRecordData decrypts the data from a SecureRecord
func (ss *SecureStore) DecryptRecordData(record *SecureRecord) (*PlainData, error) {
	// Verify signature first
	publicKey, err := hex.DecodeString(record.SignaturePublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	signature, err := hex.DecodeString(record.DataSignature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	signatureInput := append(record.EncryptedData, record.EncryptionNonce...)
	signatureInput = append(signatureInput, record.EncryptionTag...)

	if !ed25519.Verify(ed25519.PublicKey(publicKey), signatureInput, signature) {
		return nil, fmt.Errorf("signature verification failed - data may be tampered")
	}

	// Get data encryption key
	dek, err := ss.keyManager.DeriveDataEncryptionKey(record.ProcessID, record.UserID)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	// Decrypt
	additionalData := []byte(fmt.Sprintf("process:%s:user:%d:type:%s", record.ProcessID, record.UserID, record.RecordType))
	ciphertext := append(record.EncryptedData, record.EncryptionTag...)
	plainBytes, err := vault.DecryptLocal(ciphertext, dek, record.EncryptionNonce, additionalData)
	if err != nil {
		return nil, fmt.Errorf("decryption failed - data may be corrupted: %w", err)
	}

	// Deserialize
	var data PlainData
	if err := json.Unmarshal(plainBytes, &data); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return &data, nil
}

// VerifyChain verifies the integrity of the entire hash chain for a process
func (ss *SecureStore) VerifyChain(processID string) (bool, []string, error) {
	query := `
		SELECT id, process_id, user_id, created_at, encrypted_data,
		       encryption_nonce, encryption_tag, key_version,
		       system_key_id, process_key_hash, data_signature,
		       signature_public_key, record_type, status,
		       prev_record_hash, chain_hash
		FROM encrypted_records
		WHERE process_id = $1
		ORDER BY id ASC
	`

	rows, err := ss.db.Query(query, processID)
	if err != nil {
		return false, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var prevHash string = "0000000000000000000000000000000000000000000000000000000000000000"
	recordCount := 0
	var errors []string

	for rows.Next() {
		var record SecureRecord
		var status sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.ProcessID,
			&record.UserID,
			&record.CreatedAt,
			&record.EncryptedData,
			&record.EncryptionNonce,
			&record.EncryptionTag,
			&record.KeyVersion,
			&record.SystemKeyID,
			&record.ProcessKeyHash,
			&record.DataSignature,
			&record.SignaturePublicKey,
			&record.RecordType,
			&status,
			&record.PrevRecordHash,
			&record.ChainHash,
		)
		if err != nil {
			return false, nil, fmt.Errorf("scan failed: %w", err)
		}

		if status.Valid {
			record.Status = status.String
		}

		// Check chain integrity
		if record.PrevRecordHash != prevHash {
			errors = append(errors, fmt.Sprintf("chain broken at record %d: expected prev_hash=%s, got=%s",
				record.ID, prevHash, record.PrevRecordHash))
		}

		// Verify signature
		publicKey, err := hex.DecodeString(record.SignaturePublicKey)
		if err != nil {
			errors = append(errors, fmt.Sprintf("record %d: invalid public key", record.ID))
			continue
		}

		signature, err := hex.DecodeString(record.DataSignature)
		if err != nil {
			errors = append(errors, fmt.Sprintf("record %d: invalid signature", record.ID))
			continue
		}

		signatureInput := append(record.EncryptedData, record.EncryptionNonce...)
		signatureInput = append(signatureInput, record.EncryptionTag...)

		if !ed25519.Verify(ed25519.PublicKey(publicKey), signatureInput, signature) {
			errors = append(errors, fmt.Sprintf("record %d: signature verification failed", record.ID))
		}

		prevHash = record.ChainHash
		recordCount++
	}

	if len(errors) > 0 {
		return false, errors, nil
	}

	return true, []string{fmt.Sprintf("✓ Chain verified: %d records intact", recordCount)}, nil
}

// GetRecordsByProcess retrieves all records for a process (metadata only, not decrypted)
func (ss *SecureStore) GetRecordsByProcess(processID string) ([]*SecureRecord, error) {
	query := `
		SELECT id, process_id, user_id, created_at, key_version,
		       system_key_id, process_key_hash, record_type, status, chain_hash
		FROM encrypted_records
		WHERE process_id = $1
		ORDER BY created_at ASC
	`

	rows, err := ss.db.Query(query, processID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var records []*SecureRecord
	for rows.Next() {
		var record SecureRecord
		var status sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.ProcessID,
			&record.UserID,
			&record.CreatedAt,
			&record.KeyVersion,
			&record.SystemKeyID,
			&record.ProcessKeyHash,
			&record.RecordType,
			&status,
			&record.ChainHash,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if status.Valid {
			record.Status = status.String
		}

		records = append(records, &record)
	}

	return records, nil
}

// insertRecord stores a record in the database
func (ss *SecureStore) insertRecord(record *SecureRecord) error {
	query := `
		INSERT INTO encrypted_records (
			process_id, user_id, created_at, encrypted_data,
			encryption_nonce, encryption_tag, key_version,
			system_key_id, process_key_hash, data_signature,
			signature_public_key, record_type, status,
			prev_record_hash, chain_hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	var status interface{}
	if record.Status != "" {
		status = record.Status
	}

	return ss.db.QueryRow(
		query,
		record.ProcessID,
		record.UserID,
		record.CreatedAt,
		record.EncryptedData,
		record.EncryptionNonce,
		record.EncryptionTag,
		record.KeyVersion,
		record.SystemKeyID,
		record.ProcessKeyHash,
		record.DataSignature,
		record.SignaturePublicKey,
		record.RecordType,
		status,
		record.PrevRecordHash,
		record.ChainHash,
	).Scan(&record.ID)
}

// loadRecord retrieves a complete record from the database
func (ss *SecureStore) loadRecord(recordID int64) (*SecureRecord, error) {
	record := &SecureRecord{}
	var status sql.NullString

	query := `
		SELECT id, process_id, user_id, created_at, encrypted_data,
		       encryption_nonce, encryption_tag, key_version,
		       system_key_id, process_key_hash, data_signature,
		       signature_public_key, record_type, status,
		       prev_record_hash, chain_hash
		FROM encrypted_records
		WHERE id = $1
	`

	err := ss.db.QueryRow(query, recordID).Scan(
		&record.ID,
		&record.ProcessID,
		&record.UserID,
		&record.CreatedAt,
		&record.EncryptedData,
		&record.EncryptionNonce,
		&record.EncryptionTag,
		&record.KeyVersion,
		&record.SystemKeyID,
		&record.ProcessKeyHash,
		&record.DataSignature,
		&record.SignaturePublicKey,
		&record.RecordType,
		&status,
		&record.PrevRecordHash,
		&record.ChainHash,
	)
	if err != nil {
		return nil, err
	}

	if status.Valid {
		record.Status = status.String
	}

	return record, nil
}

// getLatestHash retrieves the latest chain hash for a process
func (ss *SecureStore) getLatestHash(processID string) (string, error) {
	var hash string
	err := ss.db.QueryRow(`
		SELECT chain_hash 
		FROM encrypted_records 
		WHERE process_id = $1
		ORDER BY id DESC 
		LIMIT 1
	`, processID).Scan(&hash)

	if err == sql.ErrNoRows {
		// Genesis block: no previous hash
		return "0000000000000000000000000000000000000000000000000000000000000000", nil
	}

	return hash, err
}
