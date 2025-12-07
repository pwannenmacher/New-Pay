package auth

import (
	"testing"
	"time"

	"github.com/pwannenmacher/New-Pay/internal/config"
)

func TestHashPassword(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:            "test-secret",
		Expiration:        24 * time.Hour,
		RefreshExpiration: 168 * time.Hour,
	}
	svc := NewService(cfg)

	password := "testpassword123"
	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal the original password")
	}
}

func TestVerifyPassword(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:            "test-secret",
		Expiration:        24 * time.Hour,
		RefreshExpiration: 168 * time.Hour,
	}
	svc := NewService(cfg)

	password := "testpassword123"
	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	err = svc.VerifyPassword(hash, password)
	if err != nil {
		t.Errorf("Should verify correct password, got error: %v", err)
	}

	// Test incorrect password
	err = svc.VerifyPassword(hash, "wrongpassword")
	if err == nil {
		t.Error("Should not verify incorrect password")
	}
}

func TestGenerateToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:            "test-secret",
		Expiration:        24 * time.Hour,
		RefreshExpiration: 168 * time.Hour,
	}
	svc := NewService(cfg)

	userID := uint(1)
	email := "test@example.com"

	token, _, err := svc.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}
}

func TestValidateToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:            "test-secret",
		Expiration:        24 * time.Hour,
		RefreshExpiration: 168 * time.Hour,
	}
	svc := NewService(cfg)

	userID := uint(1)
	email := "test@example.com"

	// Generate a token
	token, _, err := svc.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the token
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %d, got %d", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected email %s, got %s", email, claims.Email)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:            "test-secret",
		Expiration:        -1 * time.Hour, // Already expired
		RefreshExpiration: 168 * time.Hour,
	}
	svc := NewService(cfg)

	userID := uint(1)
	email := "test@example.com"

	// Generate an expired token
	token, _, err := svc.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate the expired token
	_, err = svc.ValidateToken(token)
	if err == nil {
		t.Error("Should reject expired token")
	}
}

func TestGenerateRandomToken(t *testing.T) {
	token1, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("Failed to generate random token: %v", err)
	}

	if token1 == "" {
		t.Error("Token should not be empty")
	}

	// Generate another token and ensure they're different
	token2, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("Failed to generate second random token: %v", err)
	}

	if token1 == token2 {
		t.Error("Random tokens should be different")
	}
}
