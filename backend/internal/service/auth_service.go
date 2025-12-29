package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"new-pay/internal/auth"
	"new-pay/internal/email"
	"new-pay/internal/models"
	"new-pay/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	roleRepo      *repository.RoleRepository
	sessionRepo   *repository.SessionRepository
	oauthConnRepo *repository.OAuthConnectionRepository
	authSvc       *auth.Service
	emailSvc      *email.Service
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	oauthConnRepo *repository.OAuthConnectionRepository,
	authSvc *auth.Service,
	emailSvc *email.Service,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		roleRepo:      roleRepo,
		sessionRepo:   sessionRepo,
		oauthConnRepo: oauthConnRepo,
		authSvc:       authSvc,
		emailSvc:      emailSvc,
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

	// Assign initial role (admin for first user, user for others)
	s.assignInitialRole(user.ID, email)

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
		slog.Error("Failed to send verification email", "error", err, "email", email)
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

// UpdateLastLogin updates the last login timestamp for a user
func (s *AuthService) UpdateLastLogin(userID uint) error {
	return s.userRepo.UpdateLastLogin(userID)
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(userID uint) (*models.User, error) {
	return s.userRepo.GetByID(userID)
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
		slog.Error("Failed to send password reset email", "error", err, "email", email)
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
func (s *AuthService) RefreshToken(refreshToken, ipAddress, userAgent string) (accessToken, newRefreshToken string, user *models.User, err error) {
	// Validate refresh token
	claims, err := s.authSvc.ValidateToken(refreshToken)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if JTI exists in session (validates token hasn't been revoked)
	if claims.ID == "" {
		return "", "", nil, errors.New("token missing JTI")
	}

	session, err := s.sessionRepo.GetByJTI(claims.ID)
	if err != nil {
		return "", "", nil, fmt.Errorf("session not found or expired: %w", err)
	}

	// Verify session belongs to the user from the token
	if session.UserID != claims.UserID {
		return "", "", nil, errors.New("session user mismatch")
	}

	// Verify it's a refresh token session
	if session.TokenType != "refresh" {
		return "", "", nil, errors.New("invalid token type")
	}

	// Get user data
	user, err = s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return "", "", nil, fmt.Errorf("user not found: %w", err)
	}

	// Delete old session (all tokens from this session - access + refresh)
	_ = s.sessionRepo.DeleteBySessionID(session.SessionID)

	// Generate new session ID for the new token pair
	newSessionID, err := auth.GenerateRandomToken(16)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate new access token
	accessToken, accessJTI, err := s.authSvc.GenerateToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token (token rotation for security)
	var refreshJTI string
	newRefreshToken, refreshJTI, err = s.authSvc.GenerateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create new refresh session with new session ID
	if err := s.CreateSession(claims.UserID, newSessionID, refreshJTI, "refresh", ipAddress, userAgent, time.Now().Add(7*24*time.Hour)); err != nil {
		return "", "", nil, fmt.Errorf("failed to create refresh session: %w", err)
	}

	// Create access token session for tracking (same session ID)
	if err := s.CreateSession(claims.UserID, newSessionID, accessJTI, "access", ipAddress, userAgent, time.Now().Add(24*time.Hour)); err != nil {
		// Log but don't fail - access tokens can still work without session tracking
		slog.Warn("Failed to create access token session", "error", err, "session_id", newSessionID, "user_id", claims.UserID)
	}

	return accessToken, newRefreshToken, user, nil
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
		slog.Error("Failed to extract JTI", "error", err)
		return err
	}
	if jti == "" {
		slog.Info("Token missing JTI")
		return errors.New("token missing JTI")
	}
	slog.Debug("Deleting session with JTI", "jti", jti)
	err = s.sessionRepo.DeleteByJTI(jti)
	if err != nil {
		slog.Error("Failed to delete session", "error", err)
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
	slog.Debug("Deleting session for user ID", "session_id", session.SessionID, "session_id", session.UserID)
	return s.sessionRepo.DeleteBySessionID(session.SessionID)
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (s *AuthService) InvalidateAllUserSessions(userID uint) error {
	return s.sessionRepo.DeleteAllUserSessions(userID)
}

// GetUserRoles retrieves all roles for a user
func (s *AuthService) GetUserRoles(userID uint) ([]models.Role, error) {
	return s.userRepo.GetUserRoles(userID)
}

// GenerateTokensForUser generates access and refresh tokens for a user
func (s *AuthService) GenerateTokensForUser(user *models.User) (accessToken, refreshToken, accessJTI, refreshJTI string, err error) {
	// Generate JWT tokens
	accessToken, accessJTI, err = s.authSvc.GenerateToken(user.ID, user.Email)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshJTI, err = s.authSvc.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, accessJTI, refreshJTI, nil
}

// ExtractJTI extracts the JTI from a token without validation
func (s *AuthService) ExtractJTI(tokenString string) (string, error) {
	return s.authSvc.ExtractJTI(tokenString)
}

// FindOrCreateOAuthUser finds an existing user by OAuth provider or email and manages OAuth connections
// Returns the user and a boolean indicating if the user was newly created
func (s *AuthService) FindOrCreateOAuthUser(email, firstName, lastName, oauthProvider, oauthProviderID string) (*models.User, bool, error) {
	var user *models.User
	var isNewUser bool

	// First, try to find user by OAuth provider and provider ID
	if oauthProviderID != "" {
		conn, err := s.oauthConnRepo.GetByProviderAndID(oauthProvider, oauthProviderID)
		if err == nil {
			// Found existing OAuth connection, get the user
			user, err = s.userRepo.GetByID(conn.UserID)
			if err != nil {
				return nil, false, fmt.Errorf("failed to get user from OAuth connection: %w", err)
			}

			// Update last login
			user.LastLoginAt = timePtr(time.Now())
			if err := s.userRepo.Update(user); err != nil {
				slog.Error("Failed to update last login for user", "user_id", user.ID, "error", err)
			}

			// Update connection timestamp
			if err := s.oauthConnRepo.Update(conn); err != nil {
				slog.Error("Failed to update OAuth connection for user", "user_id", user.ID, "error", err)
			}

			slog.Info("Existing user logged in via OAuth: id=%d, email=%s, provider=%s", "user_id", user.ID, "email", user.Email, "provider", oauthProvider)
			return user, false, nil
		}
	}

	// OAuth connection not found, try to find user by email
	user, err := s.userRepo.GetByEmail(email)
	if err == nil {
		// User exists but no OAuth connection for this provider
		// Create new OAuth connection for this user
		if oauthProviderID != "" {
			conn := &models.OAuthConnection{
				UserID:     user.ID,
				Provider:   oauthProvider,
				ProviderID: oauthProviderID,
			}

			if err := s.oauthConnRepo.Create(conn); err != nil {
				slog.Error("Failed to create OAuth connection for user", "user_id", user.ID, "error", err)
			} else {
				slog.Info("Created OAuth connection for existing user: id=%d, provider=%s", "user_id", user.ID, "provider", oauthProvider)
			}
		}

		// Update last login
		user.LastLoginAt = timePtr(time.Now())
		if err := s.userRepo.Update(user); err != nil {
			slog.Error("Failed to update last login for user", "email", email, "error", err)
		}

		slog.Info("Existing user logged in via OAuth (new provider link): id=%d, email=%s, provider=%s", "user_id", user.ID, "email", user.Email, "provider", oauthProvider)
		return user, false, nil
	}

	// User doesn't exist, create new one
	slog.Debug("Creating new user from OAuth: email=%s, firstName=%s, lastName=%s, provider=%s", "email", email, "first_name", firstName, "last_name", lastName, "provider", oauthProvider)

	// Set default names if not provided
	if firstName == "" {
		firstName = email[:len(email)-len("@...")]
	}
	if lastName == "" {
		lastName = "User"
	}

	user = &models.User{
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		EmailVerified: true, // OAuth users are considered verified
		IsActive:      true,
		// No password for OAuth-only users
	}

	// Create user
	if err := s.userRepo.Create(user); err != nil {
		return nil, false, fmt.Errorf("failed to create OAuth user: %w", err)
	}
	isNewUser = true

	// Create OAuth connection
	if oauthProviderID != "" {
		conn := &models.OAuthConnection{
			UserID:     user.ID,
			Provider:   oauthProvider,
			ProviderID: oauthProviderID,
		}

		if err := s.oauthConnRepo.Create(conn); err != nil {
			slog.Error("Failed to create OAuth connection for new user", "user_id", user.ID, "error", err)
		}
	}

	// Assign initial role (admin for first user, user for others)
	s.assignInitialRole(user.ID, email)

	slog.Info("Successfully created OAuth user: id=%d, email=%s, provider=%s", "user_id", user.ID, "email", user.Email, "provider", oauthProvider)
	return user, isNewUser, nil
}

// GetUserOAuthConnections gets all OAuth connections for a user
func (s *AuthService) GetUserOAuthConnections(userID uint) ([]models.OAuthConnection, error) {
	return s.oauthConnRepo.GetByUserID(userID)
}

// CountAllUsers returns the total number of users in the database
func (s *AuthService) CountAllUsers() (int, error) {
	return s.userRepo.CountAll()
}

// UserExistsByEmail checks if a user with the given email exists
func (s *AuthService) UserExistsByEmail(email string) (bool, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// If error is "not found", return false without error
		return false, nil
	}
	return user != nil, nil
}

// ResendVerificationEmail creates a new verification token and sends the email
func (s *AuthService) ResendVerificationEmail(userID uint) error {
	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return errors.New("email already verified")
	}

	return s.sendVerificationEmail(userID, user.Email)
}

// SendVerificationEmailToUser sends a verification email to any user (admin only)
func (s *AuthService) SendVerificationEmailToUser(userID uint) error {
	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	return s.sendVerificationEmail(userID, user.Email)
}

// sendVerificationEmail is a helper that creates and sends a verification email
func (s *AuthService) sendVerificationEmail(userID uint, email string) error {
	// Delete any existing pending verification tokens for this user
	if err := s.tokenRepo.DeletePendingEmailVerificationTokens(userID); err != nil {
		slog.Error("Failed to delete pending tokens", "error", err)
	}

	// Generate new token
	token, err := auth.GenerateRandomToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	verificationToken := &models.EmailVerificationToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.tokenRepo.CreateEmailVerificationToken(verificationToken); err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	// Send verification email
	if err := s.emailSvc.SendVerificationEmail(email, token); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// CancelEmailVerification cancels all pending email verification tokens for a user (admin only)
func (s *AuthService) CancelEmailVerification(userID uint) error {
	if err := s.tokenRepo.DeletePendingEmailVerificationTokens(userID); err != nil {
		return fmt.Errorf("failed to cancel verification: %w", err)
	}
	return nil
}

// RevokeEmailVerification marks a user's email as unverified (admin only)
func (s *AuthService) RevokeEmailVerification(userID uint) error {
	// Delete pending tokens first
	if err := s.tokenRepo.DeletePendingEmailVerificationTokens(userID); err != nil {
		slog.Error("Failed to delete pending tokens", "error", err)
	}

	// Mark email as unverified
	if err := s.userRepo.UnverifyEmail(userID); err != nil {
		return fmt.Errorf("failed to revoke verification: %w", err)
	}

	return nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// assignInitialRole assigns the appropriate role to a user based on whether they're the first user.
// the first user gets admin role, subsequent users get user role
func (s *AuthService) assignInitialRole(userID uint, email string) {
	// Check if this is the first user in the system
	userCount, err := s.userRepo.CountAll()
	if err != nil {
		slog.Error("Failed to count users", "error", err)
		return
	}

	// Assign role: first user gets admin, others get user role
	var roleName string
	if userCount == 1 {
		roleName = "admin"
		slog.Info("Assigning admin role to first user", "email", email)
	} else {
		roleName = "user"
	}

	role, err := s.roleRepo.GetByName(roleName)
	if err != nil {
		slog.Error("Failed to find role", "role", roleName, "error", err)
		return
	}

	if err := s.userRepo.AssignRole(userID, role.ID); err != nil {
		slog.Error("Failed to assign role to user", "role", roleName, "user_id", userID, "error", err)
	}
}

// SyncUserRolesFromGroups synchronizes user roles based on OAuth group membership
// Returns lists of added and removed roles for audit logging
func (s *AuthService) SyncUserRolesFromGroups(userID uint, groups []string, groupMapping map[string]string) (added []string, removed []string, err error) {
	if len(groupMapping) == 0 {
		// No mapping configured, skip sync
		return nil, nil, nil
	}

	// Determine target roles from OAuth groups
	targetRoleNames := make(map[string]bool)
	for _, group := range groups {
		if roleName, ok := groupMapping[group]; ok {
			targetRoleNames[roleName] = true
		}
	}

	// Get current user roles
	currentRoles, err := s.GetUserRoles(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current roles: %w", err)
	}

	currentRoleNames := make(map[string]bool)
	currentRoleIDs := make(map[string]uint)
	for _, role := range currentRoles {
		currentRoleNames[role.Name] = true
		currentRoleIDs[role.Name] = role.ID
	}

	// Find roles to add
	for roleName := range targetRoleNames {
		if !currentRoleNames[roleName] {
			added = append(added, roleName)
		}
	}

	// Find roles to remove (but check admin protection)
	for roleName := range currentRoleNames {
		// Only remove roles that are in the mapping
		inMapping := false
		for _, mappedRole := range groupMapping {
			if mappedRole == roleName {
				inMapping = true
				break
			}
		}

		// If role is in mapping but not in target, it should be removed
		if inMapping && !targetRoleNames[roleName] {
			removed = append(removed, roleName)
		}
	}

	// Check admin protection before removing admin role
	if contains(removed, "admin") {
		canRemove, err := s.CanRemoveAdminRole(userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check admin removal: %w", err)
		}
		if !canRemove {
			slog.Warn("Cannot remove admin role - would leave system without admin", "user_id", userID)
			removed = removeString(removed, "admin")
		}
	}

	// Apply role changes
	for _, roleName := range added {
		role, err := s.roleRepo.GetByName(roleName)
		if err != nil {
			slog.Error("Failed to find role for assignment", "role", roleName, "error", err)
			continue
		}
		if err := s.userRepo.AssignRole(userID, role.ID); err != nil {
			slog.Error("Failed to assign role", "role", roleName, "user_id", userID, "error", err)
		}
	}

	for _, roleName := range removed {
		if roleID, ok := currentRoleIDs[roleName]; ok {
			if err := s.userRepo.RemoveRole(userID, roleID); err != nil {
				slog.Error("Failed to remove role", "role", roleName, "user_id", userID, "error", err)
			}
		}
	}

	return added, removed, nil
}

// CanRemoveAdminRole checks if an admin role can be removed from a user
// Returns false if this is the last admin in the system
func (s *AuthService) CanRemoveAdminRole(userID uint) (bool, error) {
	// Get admin role
	adminRole, err := s.roleRepo.GetByName("admin")
	if err != nil {
		return false, fmt.Errorf("failed to get admin role: %w", err)
	}

	// Count users with admin role
	adminUsers, err := s.userRepo.GetUsersByRole(adminRole.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get admin users: %w", err)
	}

	// If there's only one admin, and it's this user, cannot remove
	if len(adminUsers) <= 1 {
		for _, admin := range adminUsers {
			if admin.ID == userID {
				return false, nil
			}
		}
	}

	return true, nil
}

// GetRoleByName retrieves a role by its name
func (s *AuthService) GetRoleByName(name string) (*models.Role, error) {
	return s.roleRepo.GetByName(name)
}

// AssignRoleToUser assigns a role to a user
func (s *AuthService) AssignRoleToUser(userID, roleID uint) error {
	return s.userRepo.AssignRole(userID, roleID)
}
