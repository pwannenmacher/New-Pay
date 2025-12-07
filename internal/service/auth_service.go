package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/pwannenmacher/New-Pay/internal/auth"
	"github.com/pwannenmacher/New-Pay/internal/email"
	"github.com/pwannenmacher/New-Pay/internal/models"
	"github.com/pwannenmacher/New-Pay/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrUserInactive       = errors.New("user account is inactive")
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
	roleRepo  *repository.RoleRepository
	authSvc   *auth.Service
	emailSvc  *email.Service
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	roleRepo *repository.RoleRepository,
	authSvc *auth.Service,
	emailSvc *email.Service,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		roleRepo:  roleRepo,
		authSvc:   authSvc,
		emailSvc:  emailSvc,
	}
}

// Register registers a new user
func (s *AuthService) Register(email, password, firstName, lastName string) (*models.User, error) {
	// Check if user already exists
	existing, _ := s.userRepo.GetByEmail(email)
	if existing != nil {
		return nil, repository.ErrUserExists
	}

	// Hash the password
	passwordHash, err := s.authSvc.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default "user" role
	defaultRole, err := s.roleRepo.GetByName("user")
	if err == nil {
		_ = s.userRepo.AssignRole(user.ID, defaultRole.ID)
	}

	// Generate email verification token
	token, err := auth.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	verificationToken := &models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.tokenRepo.CreateEmailVerificationToken(verificationToken); err != nil {
		return nil, fmt.Errorf("failed to create verification token: %w", err)
	}

	// Send verification email
	if err := s.emailSvc.SendVerificationEmail(email, token); err != nil {
		// Log error but don't fail registration
		fmt.Printf("Failed to send verification email: %v\n", err)
	}

	return user, nil
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(email, password string) (accessToken, refreshToken string, user *models.User, err error) {
	// Get user by email
	user, err = s.userRepo.GetByEmail(email)
	if err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	// Verify password
	if err := s.authSvc.VerifyPassword(user.PasswordHash, password); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return "", "", nil, ErrUserInactive
	}

	// Note: Email verification is not enforced by default for better user experience.
	// To enforce email verification, set REQUIRE_EMAIL_VERIFICATION=true in config
	// and uncomment the check below.
	// if cfg.RequireEmailVerification && !user.EmailVerified {
	// 	return "", "", nil, ErrEmailNotVerified
	// }

	// Generate JWT tokens
	accessToken, err = s.authSvc.GenerateToken(user.ID, user.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err = s.authSvc.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(user.ID)

	return accessToken, refreshToken, user, nil
}

// VerifyEmail verifies a user's email address
func (s *AuthService) VerifyEmail(tokenString string) error {
	// Get the token
	token, err := s.tokenRepo.GetEmailVerificationToken(tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is already used
	if token.UsedAt != nil {
		return errors.New("token already used")
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return errors.New("token expired")
	}

	// Verify the user's email
	if err := s.userRepo.VerifyEmail(token.UserID); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	// Mark token as used
	if err := s.tokenRepo.MarkEmailVerificationTokenUsed(token.ID); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Get user to send welcome email
	user, err := s.userRepo.GetByID(token.UserID)
	if err == nil {
		name := user.FirstName
		if name == "" {
			name = user.Email
		}
		_ = s.emailSvc.SendWelcomeEmail(user.Email, name)
	}

	return nil
}

// RequestPasswordReset initiates a password reset
func (s *AuthService) RequestPasswordReset(email string) error {
	// Get user by email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if user exists or not
		return nil
	}

	// Generate password reset token
	token, err := auth.GenerateRandomToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	resetToken := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.tokenRepo.CreatePasswordResetToken(resetToken); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// Send password reset email
	if err := s.emailSvc.SendPasswordResetEmail(email, token); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to send password reset email: %v\n", err)
	}

	return nil
}

// ResetPassword resets a user's password
func (s *AuthService) ResetPassword(tokenString, newPassword string) error {
	// Get the token
	token, err := s.tokenRepo.GetPasswordResetToken(tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is already used
	if token.UsedAt != nil {
		return errors.New("token already used")
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return errors.New("token expired")
	}

	// Hash the new password
	passwordHash, err := s.authSvc.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update the password
	if err := s.userRepo.UpdatePassword(token.UserID, passwordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.tokenRepo.MarkPasswordResetTokenUsed(token.ID); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	return nil
}

// RefreshToken refreshes an access token using a refresh token and returns a new refresh token
func (s *AuthService) RefreshToken(refreshToken string) (string, string, error) {
	// Validate refresh token
	claims, err := s.authSvc.ValidateToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token
	accessToken, err := s.authSvc.GenerateToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token (token rotation for security)
	newRefreshToken, err := s.authSvc.GenerateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}
