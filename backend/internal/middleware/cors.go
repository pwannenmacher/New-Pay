package middleware

import (
	"net/http"

	"new-pay/internal/config"
)

// CORSMiddleware handles CORS
type CORSMiddleware struct {
	config *config.CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(cfg *config.CORSConfig) *CORSMiddleware {
	return &CORSMiddleware{
		config: cfg,
	}
}

// Handler handles CORS headers
func (m *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := ""
		for _, allowedOrig := range m.config.AllowedOrigins {
			if allowedOrig == "*" || allowedOrig == origin {
				allowedOrigin = allowedOrig
				break
			}
		}

		if allowedOrigin != "" {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

			if m.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Convert allowed methods to string
			methods := ""
			for i, method := range m.config.AllowedMethods {
				if i > 0 {
					methods += ", "
				}
				methods += method
			}
			w.Header().Set("Access-Control-Allow-Methods", methods)

			// Convert allowed headers to string
			headers := ""
			for i, header := range m.config.AllowedHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += header
			}
			w.Header().Set("Access-Control-Allow-Headers", headers)

			// Convert exposed headers to string
			exposedHeaders := ""
			for i, header := range m.config.ExposedHeaders {
				if i > 0 {
					exposedHeaders += ", "
				}
				exposedHeaders += header
			}
			if exposedHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
