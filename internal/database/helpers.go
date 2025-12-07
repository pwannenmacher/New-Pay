package database

import (
	"context"
	"time"
)

// getContext creates a context with timeout
func getContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
