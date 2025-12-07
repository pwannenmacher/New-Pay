package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/pwannenmacher/New-Pay/internal/config"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	enabled  bool
	requests int
	duration time.Duration
	visitors map[string]*visitor
	mu       sync.RWMutex
}

type visitor struct {
	lastSeen time.Time
	tokens   int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		enabled:  cfg.Enabled,
		requests: cfg.Requests,
		duration: cfg.Duration,
		visitors: make(map[string]*visitor),
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

		ip := getIP(r)

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

// getIP gets the client IP address from the request
func getIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
