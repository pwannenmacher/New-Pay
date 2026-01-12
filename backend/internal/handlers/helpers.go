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
