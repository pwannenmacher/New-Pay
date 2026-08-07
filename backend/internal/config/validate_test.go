package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

func validECKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}

func baseConfig(env, secret string) *Config {
	c := &Config{}
	c.App.Env = env
	c.JWT.Secret = secret
	c.Encryption.MasterKey = strings.Repeat("a", 64)
	c.Database.Password = "pw"
	return c
}

func TestValidate_ProductionRejectsNonPEMSecret(t *testing.T) {
	c := baseConfig("production", "not-a-pem-key")
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for invalid JWT_SECRET in production")
	}
}

func TestValidate_ProductionAcceptsValidECKey(t *testing.T) {
	c := baseConfig("production", validECKeyPEM(t))
	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}

func TestValidate_DevelopmentAllowsNonPEMSecret(t *testing.T) {
	c := baseConfig("development", "dev-secret")
	if err := c.Validate(); err != nil {
		t.Fatalf("development must not enforce PEM secret, got %v", err)
	}
}

func TestIsValidECPrivateKeyPEM_EscapedNewlines(t *testing.T) {
	escaped := strings.ReplaceAll(validECKeyPEM(t), "\n", "\\n")
	if !isValidECPrivateKeyPEM(escaped) {
		t.Fatal("expected escaped-newline PEM to be recognized as valid")
	}
}
