package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pwannenmacher/New-Pay/internal/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Service handles authentication operations
type Service struct {
	privateKey        *ecdsa.PrivateKey
	publicKey         *ecdsa.PublicKey
	jwtExpiration     time.Duration
	refreshExpiration time.Duration
}

// NewService creates a new authentication service
func NewService(cfg *config.JWTConfig) *Service {
	privateKey, publicKey := loadOrGenerateKeys(cfg.Secret)
	return &Service{
		privateKey:        privateKey,
		publicKey:         publicKey,
		jwtExpiration:     cfg.Expiration,
		refreshExpiration: cfg.RefreshExpiration,
	}
}

// HashPassword hashes a password using bcrypt
func (s *Service) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword verifies a password against a hash
func (s *Service) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// generateTokenWithExpiration generates a JWT token with the specified expiration and JTI
func (s *Service) generateTokenWithExpiration(userID uint, email, jti string, expiration time.Duration) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateToken generates a JWT access token for a user
func (s *Service) GenerateToken(userID uint, email string) (string, string, error) {
	// Generate JTI for access token
	jti, err := GenerateRandomToken(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate JTI: %w", err)
	}
	token, err := s.generateTokenWithExpiration(userID, email, jti, s.jwtExpiration)
	return token, jti, err
}

// GenerateRefreshToken generates a refresh token for a user with JTI
func (s *Service) GenerateRefreshToken(userID uint, email string) (string, string, error) {
	// Generate JTI for refresh token
	jti, err := GenerateRandomToken(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate JTI: %w", err)
	}
	token, err := s.generateTokenWithExpiration(userID, email, jti, s.refreshExpiration)
	return token, jti, err
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

// ExtractJTI extracts the JTI from a token without validating signature or expiration
// This is useful for logout where we want to invalidate even expired tokens
func (s *Service) ExtractJTI(tokenString string) (string, error) {
	// Parse token without validation
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	return claims.ID, nil
}

// GenerateRandomToken generates a random token for email verification or password reset
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// loadOrGenerateKeys loads ECDSA keys from secret or generates new ones
func loadOrGenerateKeys(secret string) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	// Try to parse secret as PEM-encoded private key
	if block, _ := pem.Decode([]byte(secret)); block != nil {
		if privateKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return privateKey, &privateKey.PublicKey
		}
	}

	// Generate new key pair for development
	// In production, you should load from a secure key management system
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("failed to generate ECDSA key: %v", err))
	}

	return privateKey, &privateKey.PublicKey
}
