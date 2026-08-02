package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"new-pay/internal/config"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	enabled           bool
	requests          int
	duration          time.Duration
	trustProxyHeaders bool
	visitors          map[string]*visitor
	mu                sync.RWMutex
}

type visitor struct {
	lastSeen time.Time
	tokens   int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		enabled:           cfg.Enabled,
		requests:          cfg.Requests,
		duration:          cfg.Duration,
		trustProxyHeaders: cfg.TrustProxyHeaders,
		visitors:          make(map[string]*visitor),
	}

	// Clean up old visitors every minute
	go rl.cleanupVisitors()

	return rl
}

// Limit rate limits requests based on IP address
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.enabled {
			next.ServeHTTP(w, r)
			return
		}

		ip := rl.clientIP(r)

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &visitor{
				lastSeen: time.Now(),
				tokens:   rl.requests - 1,
			}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		// Refill tokens based on time passed
		now := time.Now()
		elapsed := now.Sub(v.lastSeen)
		if elapsed >= rl.duration {
			v.tokens = rl.requests - 1
			v.lastSeen = now
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if v.tokens > 0 {
			v.tokens--
			v.lastSeen = now
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		rl.mu.Unlock()

		// Rate limit exceeded
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"Rate limit exceeded. Please try again later."}`))
	})
}

// cleanupVisitors removes old visitors from the map
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(1 * time.Minute)

		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// clientIP determines the rate-limiting key for a request.
//
// By default it uses the connection's RemoteAddr, which a client cannot forge.
// Proxy headers (X-Forwarded-For / X-Real-IP) are only honored when
// TrustProxyHeaders is enabled, i.e. when a trusted reverse proxy sits in front
// and overwrites them. In that case the rightmost X-Forwarded-For entry is used
// because it is the address the trusted proxy actually observed; entries a
// client prepends to spoof the header end up to the left and are ignored.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remoteIP := hostOnly(r.RemoteAddr)

	if !rl.trustProxyHeaders {
		return remoteIP
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		candidate := strings.TrimSpace(parts[len(parts)-1])
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(realIP) != nil {
		return realIP
	}

	return remoteIP
}

// auditIP returns the raw client IP information for audit logging, preserving
// the full X-Forwarded-For proxy chain for forensic purposes. Unlike the
// rate-limiting key, this value is client-influenced and must not be used for
// security decisions.
func auditIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

// hostOnly strips the port from a "host:port" address, returning the host as-is
// if it has no port (e.g. when RemoteAddr was already sanitized).
func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
