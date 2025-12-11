package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

// Client wraps HashiCorp Vault API
type Client struct {
	client       *api.Client
	transitMount string
}

// Config holds Vault configuration
type Config struct {
	Address      string
	Token        string
	TransitMount string
}

// NewClient creates a new Vault client
func NewClient(cfg *Config) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = cfg.Address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)

	vaultClient := &Client{
		client:       client,
		transitMount: cfg.TransitMount,
	}

	// Initialize transit engine
	if err := vaultClient.initTransitEngine(); err != nil {
		return nil, fmt.Errorf("failed to initialize transit engine: %w", err)
	}

	return vaultClient, nil
}

// initTransitEngine enables the transit secrets engine if not already enabled
func (c *Client) initTransitEngine() error {
	ctx := context.Background()

	// Check if transit engine is already mounted
	mounts, err := c.client.Sys().ListMountsWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to list mounts: %w", err)
	}

	mountPath := c.transitMount + "/"
	if _, exists := mounts[mountPath]; exists {
		return nil // Already mounted
	}

	// Mount transit engine
	err = c.client.Sys().MountWithContext(ctx, c.transitMount, &api.MountInput{
		Type:        "transit",
		Description: "Transit encryption for New Pay",
		Config: api.MountConfigInput{
			DefaultLeaseTTL: "768h",
			MaxLeaseTTL:     "8760h",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to mount transit engine: %w", err)
	}

	return nil
}

// CreateKey creates or updates a transit encryption key
func (c *Client) CreateKey(keyName string, keyType string) error {
	ctx := context.Background()

	path := fmt.Sprintf("%s/keys/%s", c.transitMount, keyName)

	data := map[string]interface{}{
		"type":       keyType, // aes256-gcm96, ed25519, etc.
		"exportable": false,   // Keys cannot be exported for security
		"derived":    false,   // No key derivation by Vault
	}

	_, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to create key %s: %w", keyName, err)
	}

	return nil
}

// Encrypt encrypts data using Vault's transit engine
func (c *Client) Encrypt(keyName string, plaintext []byte, ctx map[string]string) (string, error) {
	context := context.Background()

	path := fmt.Sprintf("%s/encrypt/%s", c.transitMount, keyName)

	encodedPlaintext := base64.StdEncoding.EncodeToString(plaintext)

	data := map[string]interface{}{
		"plaintext": encodedPlaintext,
	}

	// Add context for additional authenticated data (AAD)
	if len(ctx) > 0 {
		contextStr := c.encodeContext(ctx)
		data["context"] = base64.StdEncoding.EncodeToString([]byte(contextStr))
	}

	secret, err := c.client.Logical().WriteWithContext(context, path, data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("invalid ciphertext response")
	}

	return ciphertext, nil
}

// Decrypt decrypts data using Vault's transit engine
func (c *Client) Decrypt(keyName string, ciphertext string, ctx map[string]string) ([]byte, error) {
	context := context.Background()

	path := fmt.Sprintf("%s/decrypt/%s", c.transitMount, keyName)

	data := map[string]interface{}{
		"ciphertext": ciphertext,
	}

	// Add context for AAD verification
	if len(ctx) > 0 {
		contextStr := c.encodeContext(ctx)
		data["context"] = base64.StdEncoding.EncodeToString([]byte(contextStr))
	}

	secret, err := c.client.Logical().WriteWithContext(context, path, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	encodedPlaintext, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid plaintext response")
	}

	plaintext, err := base64.StdEncoding.DecodeString(encodedPlaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode plaintext: %w", err)
	}

	return plaintext, nil
}

// GenerateDataKey generates a new data encryption key (DEK)
func (c *Client) GenerateDataKey(keyName string, bits int) (plaintext []byte, ciphertext string, err error) {
	ctx := context.Background()

	path := fmt.Sprintf("%s/datakey/plaintext/%s", c.transitMount, keyName)

	data := map[string]interface{}{
		"bits": bits,
	}

	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate data key: %w", err)
	}

	// Get plaintext DEK
	plaintextB64, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, "", fmt.Errorf("invalid plaintext in response")
	}

	plaintext, err = base64.StdEncoding.DecodeString(plaintextB64)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode plaintext: %w", err)
	}

	// Get encrypted DEK
	ciphertext, ok = secret.Data["ciphertext"].(string)
	if !ok {
		return nil, "", fmt.Errorf("invalid ciphertext in response")
	}

	return plaintext, ciphertext, nil
}

// StoreSecret stores a secret in Vault KV
func (c *Client) StoreSecret(path string, data map[string]interface{}) error {
	ctx := context.Background()

	secretPath := fmt.Sprintf("secret/data/%s", path)

	payload := map[string]interface{}{
		"data": data,
	}

	_, err := c.client.Logical().WriteWithContext(ctx, secretPath, payload)
	if err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	return nil
}

// GetSecret retrieves a secret from Vault KV
func (c *Client) GetSecret(path string) (map[string]interface{}, error) {
	ctx := context.Background()

	secretPath := fmt.Sprintf("secret/data/%s", path)

	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found")
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid secret data format")
	}

	return data, nil
}

// Health checks Vault health status
func (c *Client) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := c.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

// encodeContext converts context map to string
func (c *Client) encodeContext(ctx map[string]string) string {
	result := ""
	for k, v := range ctx {
		result += fmt.Sprintf("%s=%s;", k, v)
	}
	return result
}

// EncryptLocal performs local AES-256-GCM encryption (for fallback)
func EncryptLocal(plaintext, key []byte, additionalData []byte) (ciphertext []byte, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("GCM creation failed: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, additionalData)
	return ciphertext, nonce, nil
}

// DecryptLocal performs local AES-256-GCM decryption (for fallback)
func DecryptLocal(ciphertext, key, nonce []byte, additionalData []byte) ([]byte, error) {
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

// DeriveKey derives a key using HKDF
func DeriveKey(masterKey []byte, salt []byte, info string, length int) []byte {
	h := sha256.New()
	h.Write(masterKey)
	if salt != nil {
		h.Write(salt)
	}
	h.Write([]byte(info))
	hash := h.Sum(nil)

	// Simple key derivation (for production use crypto/hkdf)
	if length <= len(hash) {
		return hash[:length]
	}

	// Extend if needed
	result := make([]byte, length)
	copy(result, hash)
	for i := len(hash); i < length; i++ {
		result[i] = hash[i%len(hash)]
	}

	return result
}

// HashData creates a SHA-256 hash of data
func HashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
