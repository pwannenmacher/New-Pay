package middleware

import (
	"context"
	"net/http"
	"strings"

	"new-pay/internal/auth"
	"new-pay/internal/repository"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

// AuthMiddleware validates JWT tokens
type AuthMiddleware struct {
	authService *auth.Service
	sessionRepo *repository.SessionRepository
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *auth.Service, sessionRepo *repository.SessionRepository) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		sessionRepo: sessionRepo,
	}
}

// Authenticate validates the JWT token and adds user info to context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		// Extract the token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		token := parts[1]

		// Validate the token
		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Check if session exists (validates that token hasn't been invalidated)
		if claims.ID != "" {
			_, err := m.sessionRepo.GetByJTI(claims.ID)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, "Token has been invalidated")
				return
			}
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth validates JWT token if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := m.authService.ValidateToken(parts[1])
				if err == nil {
					ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
					ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
					r = r.WithContext(ctx)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserID retrieves the user ID from the request context
func GetUserID(r *http.Request) (uint, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uint)
	return userID, ok
}

// GetUserEmail retrieves the user email from the request context
func GetUserEmail(r *http.Request) (string, bool) {
	email, ok := r.Context().Value(UserEmailKey).(string)
	return email, ok
}

// Helper function to respond with JSON error
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
