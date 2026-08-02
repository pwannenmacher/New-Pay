package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"new-pay/internal/config"
)

func newCORS(origins []string, credentials bool) *CORSMiddleware {
	return NewCORSMiddleware(&config.CORSConfig{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: credentials,
	})
}

func aco(t *testing.T, m *CORSMiddleware, origin string) (string, string) {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	m.Handler(next).ServeHTTP(rec, req)
	return rec.Header().Get("Access-Control-Allow-Origin"), rec.Header().Get("Access-Control-Allow-Credentials")
}

func TestCORS_WildcardWithCredentials_NotReflectedAsStar(t *testing.T) {
	origin, creds := aco(t, newCORS([]string{"*"}, true), "https://evil.example")
	if origin == "*" {
		t.Fatalf("wildcard origin must not be sent together with credentials")
	}
	if origin != "" {
		t.Fatalf("unlisted origin must not be allowed when credentials are enabled, got %q", origin)
	}
	if creds == "true" {
		t.Fatalf("credentials header must not be set when no origin is allowed")
	}
}

func TestCORS_WildcardWithoutCredentials_AllowsStar(t *testing.T) {
	origin, _ := aco(t, newCORS([]string{"*"}, false), "https://any.example")
	if origin != "*" {
		t.Fatalf("expected wildcard origin, got %q", origin)
	}
}

func TestCORS_ExactOriginWithCredentials_Reflected(t *testing.T) {
	origin, creds := aco(t, newCORS([]string{"https://app.example.com"}, true), "https://app.example.com")
	if origin != "https://app.example.com" {
		t.Fatalf("expected reflected origin, got %q", origin)
	}
	if creds != "true" {
		t.Fatalf("expected credentials header to be set")
	}
}

func TestCORS_UnlistedOrigin_Denied(t *testing.T) {
	origin, _ := aco(t, newCORS([]string{"https://app.example.com"}, true), "https://evil.example")
	if origin != "" {
		t.Fatalf("unlisted origin must not be allowed, got %q", origin)
	}
}
