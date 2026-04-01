// Package crypto provides AES-256-GCM encryption/decryption primitives and
// HKDF-SHA256-based key derivation for the local master-key encryption architecture.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Encrypt performs AES-256-GCM encryption and returns (ciphertext+tag, nonce, error).
// additionalData is used as AAD and must be supplied identically on decryption.
func Encrypt(plaintext, key, additionalData []byte) (ciphertext []byte, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("GCM creation failed: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, additionalData)
	return ciphertext, nonce, nil
}

// Decrypt performs AES-256-GCM decryption.
// ciphertext must include the GCM authentication tag (appended by Encrypt).
func Decrypt(ciphertext, key, nonce, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM creation failed: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// DeriveKey derives a 32-byte AES-256 key from masterKey using HKDF-SHA256,
// with salt and info providing domain separation and context binding.
func DeriveKey(masterKey, salt []byte, info string) ([]byte, error) {
	derived := make([]byte, 32)
	reader := hkdf.New(sha256.New, masterKey, salt, []byte(info))
	if _, err := io.ReadFull(reader, derived); err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}
	return derived, nil
}

// EncryptWithMasterKey encrypts small secrets (e.g. private keys, process keys)
// under the application master key. Returns a hex-encoded string of
// nonce‖ciphertext‖tag suitable for database storage.
func EncryptWithMasterKey(plaintext, masterKey []byte, contextInfo string) (string, error) {
	aad := []byte(contextInfo)
	ct, nonce, err := Encrypt(plaintext, masterKey, aad)
	if err != nil {
		return "", err
	}
	// Layout: nonce (12 bytes) ‖ ciphertext+tag
	blob := append(nonce, ct...)
	return hex.EncodeToString(blob), nil
}

// DecryptWithMasterKey is the inverse of EncryptWithMasterKey.
func DecryptWithMasterKey(hexBlob string, masterKey []byte, contextInfo string) ([]byte, error) {
	blob, err := hex.DecodeString(hexBlob)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}

	// Need at least nonce (12) + 1 byte of ciphertext + tag (16)
	if len(blob) < 29 {
		return nil, fmt.Errorf("blob too short")
	}

	nonceSize := 12
	nonce := blob[:nonceSize]
	ct := blob[nonceSize:]
	aad := []byte(contextInfo)

	return Decrypt(ct, masterKey, nonce, aad)
}
