package middleware

import (
	"net/http"
)

// SecurityHeaders adds security headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable HSTS (HTTP Strict Transport Security)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}
