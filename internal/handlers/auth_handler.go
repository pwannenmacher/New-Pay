package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pwannenmacher/New-Pay/internal/middleware"
	"github.com/pwannenmacher/New-Pay/internal/service"
	"github.com/pwannenmacher/New-Pay/pkg/validator"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *service.AuthService
	auditMw     *middleware.AuditMiddleware
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, auditMw *middleware.AuditMiddleware) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		auditMw:     auditMw,
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
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
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
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(&user.ID, "user.register", "users", "User registered", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Registration successful. Please check your email to verify your account.",
		"user": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
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
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.ValidateStruct(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Login user
	accessToken, refreshToken, user, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		// Log failed login attempt
		_ = h.auditMw.LogAction(nil, "user.login.failed", "users", "Failed login attempt for "+req.Email, getIP(r), r.UserAgent())
		return
	}

	// Log successful login
	_ = h.auditMw.LogAction(&user.ID, "user.login", "users", "User logged in", getIP(r), r.UserAgent())

	// Create session for refresh token
	if err := h.authService.CreateSession(user.ID, refreshToken, getIP(r), r.UserAgent()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Set refresh token as HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth/refresh",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   r.TLS != nil, // Only send over HTTPS in production
		SameSite: http.SameSiteStrictMode,
	})

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"access_token": accessToken,
		"user": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
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
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

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
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.ValidateStruct(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Request password reset
	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(nil, "user.password.reset.request", "users", "Password reset requested for "+req.Email, getIP(r), r.UserAgent())

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
	accessToken, newRefreshToken, err := h.authService.RefreshToken(cookie.Value, getIP(r), r.UserAgent())
	if err != nil {
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
		Path:     "/api/v1/auth/refresh",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	respondWithJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
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
	// Get refresh token from cookie
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		// Invalidate session in database
		_ = h.authService.InvalidateSession(cookie.Value)
	}

	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth/refresh",
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
	response, err := json.Marshal(payload)
	if err != nil {
		// If marshaling fails, send a generic error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
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
