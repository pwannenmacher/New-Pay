package service

import (
	"errors"
	"fmt"
	"log"
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
	userRepo    *repository.UserRepository
	tokenRepo   *repository.TokenRepository
	roleRepo    *repository.RoleRepository
	sessionRepo *repository.SessionRepository
	authSvc     *auth.Service
	emailSvc    *email.Service
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	authSvc *auth.Service,
	emailSvc *email.Service,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		roleRepo:    roleRepo,
		sessionRepo: sessionRepo,
		authSvc:     authSvc,
		emailSvc:    emailSvc,
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

// Login authenticates a user and returns JWT tokens with their JTIs
func (s *AuthService) Login(email, password string) (accessToken, refreshToken, accessJTI, refreshJTI string, user *models.User, err error) {
	// Get user by email
	user, err = s.userRepo.GetByEmail(email)
	if err != nil {
		return "", "", "", "", nil, ErrInvalidCredentials
	}

	// Verify password
	if err := s.authSvc.VerifyPassword(user.PasswordHash, password); err != nil {
		return "", "", "", "", nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return "", "", "", "", nil, ErrUserInactive
	}

	// Note: Email verification is not enforced by default for better user experience.
	// To enforce email verification, set REQUIRE_EMAIL_VERIFICATION=true in config
	// and uncomment the check below.
	// if cfg.RequireEmailVerification && !user.EmailVerified {
	// 	return "", "", "", "", nil, ErrEmailNotVerified
	// }

	// Generate JWT tokens
	accessToken, accessJTI, err = s.authSvc.GenerateToken(user.ID, user.Email)
	if err != nil {
		return "", "", "", "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshJTI, err = s.authSvc.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return "", "", "", "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(user.ID)

	return accessToken, refreshToken, accessJTI, refreshJTI, user, nil
}

// CreateSession creates a session for a token JTI
func (s *AuthService) CreateSession(userID uint, sessionID, jti, tokenType, ipAddress, userAgent string, expiresAt time.Time) error {
	// Generate unique ID for this specific token session entry
	id, err := auth.GenerateRandomToken(16)
	if err != nil {
		return fmt.Errorf("failed to generate session entry ID: %w", err)
	}

	session := &models.Session{
		ID:             id,
		UserID:         userID,
		SessionID:      sessionID, // Links access and refresh tokens from same login
		JTI:            jti,
		TokenType:      tokenType,
		ExpiresAt:      expiresAt,
		LastActivityAt: time.Now(),
		CreatedAt:      time.Now(),
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}

	return s.sessionRepo.Create(session)
}

// GenerateSessionID generates a unique session identifier
func (s *AuthService) GenerateSessionID() (string, error) {
	return auth.GenerateRandomToken(16)
}

// GetAccessAndRefreshJTI returns both access and refresh token JTIs from login
func (s *AuthService) GetAccessAndRefreshJTI(email, password string) (accessJTI, refreshJTI string, user *models.User, err error) {
	// This is a helper to get JTIs after token generation in Login
	// We need to refactor Login to return JTIs
	return "", "", nil, errors.New("not implemented - use Login instead")
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
func (s *AuthService) RefreshToken(refreshToken, ipAddress, userAgent string) (string, string, error) {
	// Validate refresh token
	claims, err := s.authSvc.ValidateToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if JTI exists in session (validates token hasn't been revoked)
	if claims.ID == "" {
		return "", "", errors.New("token missing JTI")
	}

	session, err := s.sessionRepo.GetByJTI(claims.ID)
	if err != nil {
		return "", "", fmt.Errorf("session not found or expired: %w", err)
	}

	// Verify session belongs to the user from the token
	if session.UserID != claims.UserID {
		return "", "", errors.New("session user mismatch")
	}

	// Verify it's a refresh token session
	if session.TokenType != "refresh" {
		return "", "", errors.New("invalid token type")
	}

	// Delete old session (all tokens from this session - access + refresh)
	_ = s.sessionRepo.DeleteBySessionID(session.SessionID)

	// Generate new session ID for the new token pair
	newSessionID, err := auth.GenerateRandomToken(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate new access token
	accessToken, accessJTI, err := s.authSvc.GenerateToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token (token rotation for security)
	newRefreshToken, refreshJTI, err := s.authSvc.GenerateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create new refresh session with new session ID
	if err := s.CreateSession(claims.UserID, newSessionID, refreshJTI, "refresh", ipAddress, userAgent, time.Now().Add(7*24*time.Hour)); err != nil {
		return "", "", fmt.Errorf("failed to create refresh session: %w", err)
	}

	// Create access token session for tracking (same session ID)
	if err := s.CreateSession(claims.UserID, newSessionID, accessJTI, "access", ipAddress, userAgent, time.Now().Add(24*time.Hour)); err != nil {
		// Log but don't fail - access tokens can still work without session tracking
		fmt.Printf("Warning: failed to create access token session: %v\n", err)
	}

	return accessToken, newRefreshToken, nil
}

// InvalidateSession invalidates a session by JTI
func (s *AuthService) InvalidateSession(jti string) error {
	return s.sessionRepo.DeleteByJTI(jti)
}

// InvalidateSessionByToken invalidates a session by extracting JTI from token
// Note: We extract JTI without validation to allow logout even with expired tokens
func (s *AuthService) InvalidateSessionByToken(token string) error {
	// Parse token without validation to extract JTI
	jti, err := s.authSvc.ExtractJTI(token)
	if err != nil {
		log.Printf("Failed to extract JTI: %v", err)
		return err
	}
	if jti == "" {
		log.Printf("Token missing JTI")
		return errors.New("token missing JTI")
	}
	log.Printf("Deleting session with JTI: %s", jti)
	err = s.sessionRepo.DeleteByJTI(jti)
	if err != nil {
		log.Printf("Failed to delete session: %v", err)
	}
	return err
}

// InvalidateCurrentSession invalidates only the current login session
// This deletes both the access and refresh tokens from the same login
func (s *AuthService) InvalidateCurrentSession(token string) error {
	// Extract JTI without validation (works with expired tokens)
	jti, err := s.authSvc.ExtractJTI(token)
	if err != nil {
		return fmt.Errorf("failed to extract JTI: %w", err)
	}

	// Get session to find session_id
	session, err := s.sessionRepo.GetByJTI(jti)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Delete all tokens with the same session_id (access + refresh from this login)
	log.Printf("Deleting session %s for user ID: %d", session.SessionID, session.UserID)
	return s.sessionRepo.DeleteBySessionID(session.SessionID)
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (s *AuthService) InvalidateAllUserSessions(userID uint) error {
	return s.sessionRepo.DeleteAllUserSessions(userID)
}
