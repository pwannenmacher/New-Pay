package handlers

import "net/http"

// getIP extracts the IP address from the request, checking X-Forwarded-For and X-Real-IP headers
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

// isSecureConnection checks if the request is over HTTPS, considering reverse proxy headers
func isSecureConnection(r *http.Request) bool {
	// Check if direct TLS connection
	if r.TLS != nil {
		return true
	}

	// Check X-Forwarded-Proto header (set by reverse proxy)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return true
	}

	// Check X-Forwarded-Ssl header (some proxies use this)
	if ssl := r.Header.Get("X-Forwarded-Ssl"); ssl == "on" {
		return true
	}

	return false
}
