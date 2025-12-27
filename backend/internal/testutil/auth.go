package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"new-pay/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

// AuthHelper provides JWT token generation for tests
type AuthHelper struct {
	JWTSecret []byte
}

// NewAuthHelper creates a new auth helper
func NewAuthHelper() *AuthHelper {
	return &AuthHelper{
		JWTSecret: []byte("test-secret-key-for-testing-only"),
	}
}

// GenerateToken generates a JWT token for a user with specified roles
func (h *AuthHelper) GenerateToken(userID uint, email string, roles []string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"roles":   roles,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.JWTSecret)
}

// AddAuthHeader adds an authorization header to the request
func (h *AuthHelper) AddAuthHeader(t *testing.T, req *http.Request, user *models.User, roles []string) {
	t.Helper()

	token, err := h.GenerateToken(user.ID, user.Email, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
}

// CreateAuthenticatedRequest creates a request with auth header
func (h *AuthHelper) CreateAuthenticatedRequest(t *testing.T, method, url string, user *models.User, roles []string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, url, nil)
	h.AddAuthHeader(t, req, user, roles)
	return req
}

// TestResponse holds response data for assertions
type TestResponse struct {
	*httptest.ResponseRecorder
}

// NewTestResponse creates a new test response recorder
func NewTestResponse() *TestResponse {
	return &TestResponse{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

// AssertStatus asserts the HTTP status code
func (r *TestResponse) AssertStatus(t *testing.T, expected int) {
	t.Helper()

	if r.Code != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, r.Code, r.Body.String())
	}
}

// AssertStatusOK asserts 200 OK
func (r *TestResponse) AssertStatusOK(t *testing.T) {
	r.AssertStatus(t, http.StatusOK)
}

// AssertStatusCreated asserts 201 Created
func (r *TestResponse) AssertStatusCreated(t *testing.T) {
	r.AssertStatus(t, http.StatusCreated)
}

// AssertStatusUnauthorized asserts 401 Unauthorized
func (r *TestResponse) AssertStatusUnauthorized(t *testing.T) {
	r.AssertStatus(t, http.StatusUnauthorized)
}

// AssertStatusForbidden asserts 403 Forbidden
func (r *TestResponse) AssertStatusForbidden(t *testing.T) {
	r.AssertStatus(t, http.StatusForbidden)
}

// AssertStatusNotFound asserts 404 Not Found
func (r *TestResponse) AssertStatusNotFound(t *testing.T) {
	r.AssertStatus(t, http.StatusNotFound)
}

// AssertStatusBadRequest asserts 400 Bad Request
func (r *TestResponse) AssertStatusBadRequest(t *testing.T) {
	r.AssertStatus(t, http.StatusBadRequest)
}
