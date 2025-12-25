package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"new-pay/internal/config"
	"new-pay/internal/middleware"
	"new-pay/internal/service"
	"new-pay/pkg/validator"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *service.AuthService
	auditMw     *middleware.AuditMiddleware
	config      *config.Config
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, auditMw *middleware.AuditMiddleware, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		auditMw:     auditMw,
		config:      cfg,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// VerifyEmailRequest represents an email verification request
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a password reset confirmation
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account with email verification
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} map[string]interface{} "Registration successful"
// @Failure 400 {object} map[string]string "Invalid request"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Check if registration is enabled (allow if no users exist)
	if !h.config.App.EnableRegistration {
		// Check if any users exist - allow registration if database is empty
		userCount, err := h.authService.CountAllUsers()
		if err != nil || userCount > 0 {
			respondWithError(w, http.StatusForbidden, "Registration is disabled")
			return
		}
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Validate request
	if err := validator.ValidateStruct(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Register user
	user, err := h.authService.Register(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		slog.Error("Registration failed", "email", req.Email, "error", err)
		_ = h.auditMw.LogAction(nil, "user.register.error", "users", "Registration failed for "+req.Email+": "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	slog.Info("User registered successfully", "user_id", user.ID, "email", user.Email)

	// Log audit event
	_ = h.auditMw.LogAction(&user.ID, "user.register", "users", "User registered", getIP(r), r.UserAgent())
	// Log verification email send
	_ = h.auditMw.LogAction(&user.ID, "email.verification.sent", "emails", "Verification email sent to "+user.Email, getIP(r), r.UserAgent())

	// Auto-login after registration: generate JWT tokens
	accessToken, refreshToken, accessJTI, refreshJTI, err := h.authService.GenerateTokensForUser(user)
	if err != nil {
		_ = h.auditMw.LogAction(&user.ID, "user.register.token.error", "users", "Token generation failed after registration", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Update last login time for auto-login after registration
	_ = h.authService.UpdateLastLogin(user.ID)

	// Reload user to get updated last_login_at timestamp
	user, _ = h.authService.GetUserByID(user.ID)

	// Generate a session ID that links the access and refresh tokens
	sessionID, err := h.authService.GenerateSessionID()
	if err != nil {
		_ = h.auditMw.LogAction(&user.ID, "user.register.session.error", "users", "Session ID generation failed", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to generate session ID")
		return
	}

	// Create session for refresh token
	if err := h.authService.CreateSession(user.ID, sessionID, refreshJTI, "refresh", getIP(r), r.UserAgent(), time.Now().Add(7*24*time.Hour)); err != nil {
		_ = h.auditMw.LogAction(&user.ID, "user.register.session.error", "users", "Session creation failed", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Create session for access token (linked via same sessionID)
	_ = h.authService.CreateSession(user.ID, sessionID, accessJTI, "access", getIP(r), r.UserAgent(), time.Now().Add(24*time.Hour))

	// Set refresh token as HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     AuthAPIBasePath,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   r.TLS != nil, // Only send over HTTPS in production
		SameSite: http.SameSiteStrictMode,
	})

	// Get user roles
	roles, _ := h.authService.GetUserRoles(user.ID)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    86400, // 24 hours in seconds
		"user": map[string]interface{}{
			"id":                user.ID,
			"email":             user.Email,
			"first_name":        user.FirstName,
			"last_name":         user.LastName,
			"email_verified":    user.EmailVerified,
			"email_verified_at": user.EmailVerifiedAt,
			"is_active":         user.IsActive,
			"last_login_at":     user.LastLoginAt,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
			"roles":             roles,
		},
	})
}

// Login handles user login
// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful with tokens"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Validate request
	if err := validator.ValidateStruct(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Login user
	accessToken, refreshToken, accessJTI, refreshJTI, user, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		slog.Warn("Login failed", "email", req.Email, "error", err, "ip", getIP(r))
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		// Log failed login attempt
		_ = h.auditMw.LogAction(nil, "user.login.failed", "users", "Failed login attempt for "+req.Email, getIP(r), r.UserAgent())
		return
	}

	slog.Info("User logged in successfully", "user_id", user.ID, "email", user.Email, "ip", getIP(r))
	// Log successful login
	_ = h.auditMw.LogAction(&user.ID, "user.login", "users", "User logged in", getIP(r), r.UserAgent())

	// Generate a session ID that links the access and refresh tokens
	sessionID, err := h.authService.GenerateSessionID()
	if err != nil {
		_ = h.auditMw.LogAction(&user.ID, "user.login.session.error", "users", "Session ID generation failed during login", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to generate session ID")
		return
	}

	// Create session for refresh token
	if err := h.authService.CreateSession(user.ID, sessionID, refreshJTI, "refresh", getIP(r), r.UserAgent(), time.Now().Add(7*24*time.Hour)); err != nil {
		_ = h.auditMw.LogAction(&user.ID, "user.login.session.error", "users", "Session creation failed during login", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Create session for access token (linked via same sessionID)
	_ = h.authService.CreateSession(user.ID, sessionID, accessJTI, "access", getIP(r), r.UserAgent(), time.Now().Add(24*time.Hour))

	// Set refresh token as HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     AuthAPIBasePath,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   r.TLS != nil, // Only send over HTTPS in production
		SameSite: http.SameSiteStrictMode,
	})

	// Get user roles
	roles, _ := h.authService.GetUserRoles(user.ID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    86400, // 24 hours in seconds
		"user": map[string]interface{}{
			"id":                user.ID,
			"email":             user.Email,
			"first_name":        user.FirstName,
			"last_name":         user.LastName,
			"email_verified":    user.EmailVerified,
			"email_verified_at": user.EmailVerifiedAt,
			"is_active":         user.IsActive,
			"last_login_at":     user.LastLoginAt,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
			"roles":             roles,
		},
	})
}

// VerifyEmail handles email verification
// @Summary Verify email address
// @Description Verify user's email address using token from email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param token query string true "Verification token"
// @Success 200 {object} map[string]string "Email verified successfully"
// @Failure 400 {object} map[string]string "Invalid or expired token"
// @Router /auth/verify-email [get]
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		var req VerifyEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Token is required")
			return
		}
		token = req.Token
	}

	// Verify email
	if err := h.authService.VerifyEmail(token); err != nil {
		_ = h.auditMw.LogAction(nil, "user.email.verify.error", "users", "Email verification failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log email verification and welcome email send
	_ = h.auditMw.LogAction(nil, "user.email.verified", "users", "Email verified", getIP(r), r.UserAgent())
	_ = h.auditMw.LogAction(nil, "email.welcome.sent", "emails", "Welcome email sent", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

// RequestPasswordReset handles password reset requests
// @Summary Request password reset
// @Description Send password reset email to user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body PasswordResetRequest true "Email address"
// @Success 200 {object} map[string]string "Reset email sent if user exists"
// @Failure 400 {object} map[string]string "Invalid request"
// @Router /auth/password-reset/request [post]
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Validate request
	if err := validator.ValidateStruct(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Request password reset
	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		_ = h.auditMw.LogAction(nil, "user.password.reset.error", "users", "Password reset request failed for "+req.Email+": "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(nil, "user.password.reset.request", "users", "Password reset requested for "+req.Email, getIP(r), r.UserAgent())
	// Log password reset email send
	_ = h.auditMw.LogAction(nil, "email.password_reset.sent", "emails", "Password reset email sent to "+req.Email, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset confirmation
// @Summary Reset password
// @Description Reset user password using token from email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} map[string]string "Password reset successful"
// @Failure 400 {object} map[string]string "Invalid or expired token"
// @Router /auth/password-reset/confirm [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.ValidateStruct(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Reset password
	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		_ = h.auditMw.LogAction(nil, "user.password.reset.error", "users", "Password reset failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(nil, "user.password.reset", "users", "Password reset completed", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Password reset successfully",
	})
}

// RefreshToken handles token refresh requests
// @Summary Refresh access token
// @Description Get a new access token using refresh token from cookie
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "New access token"
// @Failure 401 {object} map[string]string "Invalid refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Refresh token not found")
		return
	}

	// Refresh token
	accessToken, newRefreshToken, user, err := h.authService.RefreshToken(cookie.Value, getIP(r), r.UserAgent())
	if err != nil {
		// Log refresh token failure
		_ = h.auditMw.LogAction(nil, "user.token.refresh.error", "users", "Token refresh failed: "+err.Error(), getIP(r), r.UserAgent())
		// Clear invalid cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/api/v1/auth/refresh",
			MaxAge:   -1,
			HttpOnly: true,
		})
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Set new refresh token as cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     AuthAPIBasePath,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Get user roles
	roles, _ := h.authService.GetUserRoles(user.ID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    86400, // 24 hours in seconds
		"user": map[string]interface{}{
			"id":                user.ID,
			"email":             user.Email,
			"first_name":        user.FirstName,
			"last_name":         user.LastName,
			"email_verified":    user.EmailVerified,
			"email_verified_at": user.EmailVerifiedAt,
			"is_active":         user.IsActive,
			"last_login_at":     user.LastLoginAt,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
			"roles":             roles,
		},
	})
}

// Logout handles user logout
// @Summary User logout
// @Description Clear refresh token cookie and invalidate session
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Logout successful"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context for audit logging
	userID, hasUserID := middleware.GetUserID(r)

	// Get refresh token from cookie
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		// Invalidate only the current session (access + refresh tokens from this login)
		if err := h.authService.InvalidateCurrentSession(cookie.Value); err != nil {
			slog.Error("Failed to invalidate session during logout", "error", err)
			// Log error in audit
			if hasUserID {
				_ = h.auditMw.LogAction(&userID, "user.logout.error", "users", "Failed to invalidate session: "+err.Error(), getIP(r), r.UserAgent())
			}
		}
	}

	// Log successful logout
	if hasUserID {
		slog.Info("User logged out", "user_id", userID, "ip", getIP(r))
		_ = h.auditMw.LogAction(&userID, "user.logout", "users", "User logged out", getIP(r), r.UserAgent())
	}

	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     AuthAPIBasePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// Helper functions

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.WriteHeader(code)
	if err := JSONResponse(w, payload); err != nil {
		// If marshaling fails, log the error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal server error"}`))
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

// OAuthLogin initiates the OAuth login flow
// @Summary Initiate OAuth login
// @Description Redirects to the OAuth provider for authentication
// @Tags Authentication
// @Param provider query string true "Provider name"
// @Success 302 {string} string "Redirect to OAuth provider"
// @Router /auth/oauth/login [get]
func (h *AuthHandler) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		http.Error(w, "Provider parameter is required", http.StatusBadRequest)
		return
	}

	// Find the provider configuration
	var providerConfig *config.OAuthProviderConfig
	for i := range h.config.OAuth.Providers {
		if h.config.OAuth.Providers[i].Name == providerName && h.config.OAuth.Providers[i].Enabled {
			providerConfig = &h.config.OAuth.Providers[i]
			break
		}
	}

	if providerConfig == nil {
		http.Error(w, "Provider not found or not enabled", http.StatusNotFound)
		return
	}

	// Generate state for CSRF protection
	state := generateRandomState()

	// Store state and provider in session/cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})

	// Build authorization URL
	authURL := h.buildAuthorizationURL(state, providerConfig)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallback handles the OAuth callback
// @Summary Handle OAuth callback
// @Description Processes the OAuth callback and creates/logs in user
// @Tags Authentication
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 302 {string} string "Redirect to frontend"
// @Router /auth/oauth/callback [get]
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Get provider from cookie
	providerCookie, err := r.Cookie("oauth_provider")
	if err != nil {
		slog.Error("OAuth callback failed: provider cookie not found", "error", err)
		redirectURL := fmt.Sprintf("%s/login?error=invalid_provider", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Find the provider configuration
	var providerConfig *config.OAuthProviderConfig
	for i := range h.config.OAuth.Providers {
		if h.config.OAuth.Providers[i].Name == providerCookie.Value && h.config.OAuth.Providers[i].Enabled {
			providerConfig = &h.config.OAuth.Providers[i]
			break
		}
	}

	if providerConfig == nil {
		slog.Error("OAuth callback failed: provider not found or not enabled",
			"provider", providerCookie.Value,
		)
		redirectURL := fmt.Sprintf("%s/login?error=invalid_provider", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		slog.Error("OAuth callback failed: state cookie not found", "error", err)
		redirectURL := fmt.Sprintf("%s/login?error=invalid_state", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		slog.Error("OAuth callback failed: state mismatch")
		redirectURL := fmt.Sprintf("%s/login?error=invalid_state", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Clear cookies
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_provider",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Error("OAuth callback failed: authorization code not provided")
		redirectURL := fmt.Sprintf("%s/login?error=no_code", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Exchange code for token
	token, err := h.exchangeCodeForToken(code, providerConfig)
	if err != nil {
		slog.Error("OAuth callback failed: code exchange failed", "error", err)
		redirectURL := fmt.Sprintf("%s/login?error=token_exchange_failed", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Get user info
	userInfo, err := h.getUserInfo(token, providerConfig)
	if err != nil {
		slog.Error("OAuth callback failed: failed to get user info", "error", err)
		redirectURL := fmt.Sprintf("%s/login?error=userinfo_failed", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Extract email from user info
	email, ok := userInfo["email"].(string)
	if !ok || email == "" {
		slog.Error("OAuth callback failed: email not found in user info")
		redirectURL := fmt.Sprintf("%s/login?error=no_email", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Extract name (optional, try different fields)
	var firstName, lastName string
	if name, ok := userInfo["name"].(string); ok {
		firstName = name
	}
	if preferredUsername, ok := userInfo["preferred_username"].(string); ok && firstName == "" {
		firstName = preferredUsername
	}
	if givenName, ok := userInfo["given_name"].(string); ok {
		firstName = givenName
	}
	if familyName, ok := userInfo["family_name"].(string); ok {
		lastName = familyName
	}

	// Extract OAuth provider user ID (sub claim is standard in OAuth 2.0/OIDC)
	var oauthProviderID string
	if sub, ok := userInfo["sub"].(string); ok {
		oauthProviderID = sub
	}
	// Fallback to other possible ID fields
	if oauthProviderID == "" {
		if id, ok := userInfo["id"].(string); ok {
			oauthProviderID = id
		}
	}

	// Extract groups from user info using configured claim name
	var groups []string
	groupsClaim := providerConfig.GroupsClaim
	if groupsClaim == "" {
		groupsClaim = "groups"
	}

	if groupsData, ok := userInfo[groupsClaim]; ok {
		switch v := groupsData.(type) {
		case []interface{}:
			for _, g := range v {
				if groupStr, ok := g.(string); ok {
					groups = append(groups, groupStr)
				}
			}
		case []string:
			groups = v
		case string:
			// Single group as string
			groups = append(groups, v)
		}
	}

	// If OAuth registration is disabled, check if this would be a new user registration
	if !h.config.App.EnableOAuthRegistration {
		// Check if user already exists
		userExists, err := h.authService.UserExistsByEmail(email)
		if err != nil {
			slog.Error("OAuth callback failed: failed to check if user exists", "error", err)
			redirectURL := fmt.Sprintf("%s/login?error=server_error", h.getBaseLoginURL())
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}

		// If user doesn't exist, this would be a new registration
		if !userExists {
			// Check if database is completely empty - allow first user
			userCount, err := h.authService.CountAllUsers()
			if err != nil {
				slog.Error("OAuth callback failed: failed to count users", "error", err)
				redirectURL := fmt.Sprintf("%s/login?error=server_error", h.getBaseLoginURL())
				http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				return
			}

			// Block registration if database already has users
			if userCount > 0 {
				slog.Warn("OAuth registration rejected: registration disabled",
					"email", email,
					"provider", providerConfig.Name,
					"user_count", userCount,
				)
				_ = h.auditMw.LogAction(nil, "user.oauth.registration.disabled", "users", fmt.Sprintf("OAuth registration blocked for %s via %s (registration disabled)", email, providerConfig.Name), getIP(r), r.UserAgent())
				redirectURL := fmt.Sprintf("%s/login?error=registration_disabled", h.getBaseLoginURL())
				http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				return
			}
			// If userCount == 0, allow first user registration
			slog.Info("Allowing first OAuth user registration despite ENABLE_OAUTH_REGISTRATION=false", "email", email)
		}
	}

	// Try to find or create user
	user, isNewUser, err := h.authService.FindOrCreateOAuthUser(email, firstName, lastName, providerConfig.Name, oauthProviderID)
	if err != nil {
		slog.Error("OAuth callback failed: user creation failed",
			"email", email,
			"provider", providerConfig.Name,
			"error", err,
		)
		_ = h.auditMw.LogAction(nil, AuditActionOAuthError, "users", fmt.Sprintf("OAuth user creation failed for %s via %s: %v", email, providerConfig.Name, err), getIP(r), r.UserAgent())
		redirectURL := fmt.Sprintf("%s/login?error=user_creation_failed", h.getBaseLoginURL())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	if isNewUser {
		slog.Info("New user registered via OAuth",
			"user_id", user.ID,
			"email", user.Email,
			"provider", providerConfig.Name,
		)
		_ = h.auditMw.LogAction(&user.ID, "user.oauth.register", "users", fmt.Sprintf("New user registered via OAuth (%s)", providerConfig.Name), getIP(r), r.UserAgent())
	} else {
		slog.Info("Existing user logged in via OAuth",
			"user_id", user.ID,
			"email", user.Email,
			"provider", providerConfig.Name,
		)
	}

	// Sync roles from OAuth groups if mapping is configured
	if len(providerConfig.GroupMapping) > 0 {
		addedRoles, removedRoles, err := h.authService.SyncUserRolesFromGroups(user.ID, groups, providerConfig.GroupMapping)
		if err != nil {
			slog.Error("Failed to sync user roles from OAuth groups",
				"user_id", user.ID,
				"error", err,
			)
			_ = h.auditMw.LogAction(&user.ID, "user.roles.sync.error", "users",
				fmt.Sprintf("Failed to sync roles from OAuth groups: %v", err), getIP(r), r.UserAgent())
		} else if len(addedRoles) > 0 || len(removedRoles) > 0 {
			// Log role changes to audit log
			if len(addedRoles) > 0 {
				details := fmt.Sprintf("Roles added from OAuth groups (%s): %v", providerConfig.Name, addedRoles)
				_ = h.auditMw.LogAction(&user.ID, "user.roles.added", "users", details, getIP(r), r.UserAgent())
				slog.Info("Roles added from OAuth groups",
					"user_id", user.ID,
					"roles", addedRoles,
				)
			}
			if len(removedRoles) > 0 {
				details := fmt.Sprintf("Roles removed based on OAuth groups (%s): %v", providerConfig.Name, removedRoles)
				_ = h.auditMw.LogAction(&user.ID, "user.roles.removed", "users", details, getIP(r), r.UserAgent())
				slog.Info("Roles removed based on OAuth groups",
					"user_id", user.ID,
					"roles", removedRoles,
				)
			}
		} else {
			// No role changes needed
		}
	}

	// Assign default role if configured and user has no roles
	if providerConfig.DefaultRole != "" {
		currentRoles, _ := h.authService.GetUserRoles(user.ID)
		if len(currentRoles) == 0 {
			role, err := h.authService.GetRoleByName(providerConfig.DefaultRole)
			if err == nil {
				if err := h.authService.AssignRoleToUser(user.ID, role.ID); err != nil {
					slog.Error("Failed to assign default role",
						"user_id", user.ID,
						"role", providerConfig.DefaultRole,
						"error", err,
					)
				} else {
					slog.Info("Assigned default role to OAuth user",
						"user_id", user.ID,
						"role", providerConfig.DefaultRole,
					)
					_ = h.auditMw.LogAction(&user.ID, "user.role.assigned", "users",
						fmt.Sprintf("Default role '%s' assigned via OAuth (%s)", providerConfig.DefaultRole, providerConfig.Name),
						getIP(r), r.UserAgent())
				}
			} else {
				slog.Warn("Default role not found",
					"role", providerConfig.DefaultRole,
					"error", err,
				)
			}
		}
	}

	// Update last login time for OAuth login
	_ = h.authService.UpdateLastLogin(user.ID)

	// Reload user to get updated last_login_at timestamp
	user, _ = h.authService.GetUserByID(user.ID)

	// Generate JWT tokens
	accessToken, refreshToken, accessJTI, refreshJTI, err := h.authService.GenerateTokensForUser(user)
	if err != nil {
		slog.Error("OAuth callback failed: token generation failed", "error", err, "user_id", user.ID)
		_ = h.auditMw.LogAction(&user.ID, AuditActionOAuthError, "users", "Token generation failed: "+err.Error(), getIP(r), r.UserAgent())
		http.Redirect(w, r, "http://localhost:5173/login?error=token_generation_failed", http.StatusTemporaryRedirect)
		return
	}

	// Generate session ID
	sessionID, err := h.authService.GenerateSessionID()
	if err != nil {
		slog.Error("OAuth callback failed: session ID generation failed", "error", err, "user_id", user.ID)
		_ = h.auditMw.LogAction(&user.ID, AuditActionOAuthError, "users", "Session ID generation failed: "+err.Error(), getIP(r), r.UserAgent())
		http.Redirect(w, r, "http://localhost:5173/login?error=session_failed", http.StatusTemporaryRedirect)
		return
	}

	// Create sessions
	if err := h.authService.CreateSession(user.ID, sessionID, refreshJTI, "refresh", getIP(r), r.UserAgent(), time.Now().Add(7*24*time.Hour)); err != nil {
		slog.Error("OAuth callback failed: refresh session creation failed", "error", err, "user_id", user.ID)
		http.Redirect(w, r, "http://localhost:5173/login?error=session_failed", http.StatusTemporaryRedirect)
		return
	}

	_ = h.authService.CreateSession(user.ID, sessionID, accessJTI, "access", getIP(r), r.UserAgent(), time.Now().Add(24*time.Hour))

	// Set refresh token as HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     AuthAPIBasePath,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Log successful OAuth login
	_ = h.auditMw.LogAction(&user.ID, "user.oauth.login", "users", fmt.Sprintf("OAuth login successful via %s", providerConfig.Name), getIP(r), r.UserAgent())

	slog.Info("OAuth callback successful",
		"user_id", user.ID,
		"email", email,
		"provider", providerConfig.Name,
	)

	// Redirect to frontend with access token in URL (will be stored in localStorage by frontend)
	redirectURL := fmt.Sprintf("%s?access_token=%s", h.config.OAuth.FrontendCallbackURL, accessToken)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) buildAuthorizationURL(state string, providerConfig *config.OAuthProviderConfig) string {
	params := url.Values{}
	params.Set("client_id", providerConfig.ClientID)
	params.Set("redirect_uri", h.config.OAuth.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", providerConfig.AuthURL, params.Encode())
}

func (h *AuthHandler) exchangeCodeForToken(code string, providerConfig *config.OAuthProviderConfig) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", h.config.OAuth.RedirectURL)
	data.Set("client_id", providerConfig.ClientID)
	data.Set("client_secret", providerConfig.ClientSecret)

	resp, err := http.PostForm(providerConfig.TokenURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access_token not found in response")
	}

	return accessToken, nil
}

func (h *AuthHandler) getUserInfo(accessToken string, providerConfig *config.OAuthProviderConfig) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", providerConfig.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// getBaseLoginURL returns the base frontend URL for login/error redirects
// Uses the configured frontend callback URL and extracts the base URL
func (h *AuthHandler) getBaseLoginURL() string {
	// The frontend callback URL is something like "http://localhost:3001/oauth/callback"
	// We need to extract just "http://localhost:3001"
	callbackURL := h.config.OAuth.FrontendCallbackURL

	// Parse the URL to extract scheme and host
	parsedURL, err := url.Parse(callbackURL)
	if err != nil {
		// Fallback to a default if parsing fails
		slog.Warn("Failed to parse frontend callback URL, using default",
			"url", callbackURL,
			"error", err,
		)
		return "http://localhost:3001"
	}

	return fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
}
