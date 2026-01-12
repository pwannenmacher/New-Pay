package handlers

import (
	"net/http"
	"strings"
)

// getIP extracts the raw IP address(es) from request headers for database storage
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

// formatIPForAdmin formats IP address with proxy information for admin views
// Input: "192.168.1.1, 10.0.0.1" -> Output: "192.168.1.1 (proxied by 10.0.0.1)"
func formatIPForAdmin(ipAddress string) string {
	if ipAddress == "" {
		return ""
	}

	ips := strings.Split(ipAddress, ",")
	if len(ips) <= 1 {
		return strings.TrimSpace(ipAddress)
	}

	clientIP := strings.TrimSpace(ips[0])
	proxyIPs := make([]string, 0, len(ips)-1)
	for i := 1; i < len(ips); i++ {
		proxyIPs = append(proxyIPs, strings.TrimSpace(ips[i]))
	}

	return clientIP + " (proxied by " + strings.Join(proxyIPs, ", ") + ")"
}

// getClientIP extracts only the client IP (first IP) for user-facing views
// Input: "192.168.1.1, 10.0.0.1" -> Output: "192.168.1.1"
func getClientIP(ipAddress string) string {
	if ipAddress == "" {
		return ""
	}

	ips := strings.Split(ipAddress, ",")
	return strings.TrimSpace(ips[0])
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
