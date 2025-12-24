package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code and response body
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	// Capture response body for DEBUG logging
	if rw.body != nil {
		rw.body.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware logs all HTTP requests with level-based detail
//
// Log levels:
// - INFO: Every request with Remote-IP, User-Agent, HTTP-Method, and Path
// - DEBUG: Additionally logs Request-Body, Response-Body, and all Query-Parameters
// - WARN: Only failed requests (status 4xx)
// - ERROR: Only errors (status 5xx)
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read and buffer request body for DEBUG logging
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create response writer with optional body capture
		var responseBodyBuffer *bytes.Buffer
		if slog.Default().Enabled(r.Context(), slog.LevelDebug) {
			responseBodyBuffer = &bytes.Buffer{}
		}

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        false,
			body:           responseBodyBuffer,
		}

		// Log incoming request - either INFO or DEBUG depending on log level
		if slog.Default().Enabled(r.Context(), slog.LevelDebug) {
			// DEBUG-Level: Log detailed request info
			attrs := []any{
				"remote_ip", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"method", r.Method,
				"path", r.URL.Path,
			}

			// Add query parameters
			if len(r.URL.Query()) > 0 {
				queryParams := make(map[string][]string)
				for key, values := range r.URL.Query() {
					queryParams[key] = values
				}
				attrs = append(attrs, "query_params", queryParams)
			}

			// Add request body (if present and non-empty)
			if len(requestBody) > 0 {
				attrs = append(attrs, "request_body", string(requestBody))
			}

			slog.Debug("Incoming request", attrs...)
		} else {
			// INFO-Level: Log basic request info
			slog.Info("Incoming request",
				"remote_ip", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"method", r.Method,
				"path", r.URL.Path,
			)
		}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on status code
		var logLevel slog.Level
		var logMessage string

		if wrapped.statusCode >= 500 {
			// ERROR-Level: Only errors (5xx)
			logLevel = slog.LevelError
			logMessage = "Request failed with error"
		} else if wrapped.statusCode >= 400 {
			// WARN-Level: Only failed requests (4xx)
			logLevel = slog.LevelWarn
			logMessage = "Request failed"
		} else {
			// INFO-Level: Successful requests
			logLevel = slog.LevelInfo
			logMessage = "Request completed"
		}

		// Build log attributes
		attrs := []any{
			"remote_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		}

		// DEBUG-Level: Add response body
		if slog.Default().Enabled(r.Context(), slog.LevelDebug) && responseBodyBuffer != nil {
			if responseBodyBuffer.Len() > 0 {
				attrs = append(attrs, "response_body", responseBodyBuffer.String())
			}
		}

		// Log with appropriate level
		slog.Log(r.Context(), logLevel, logMessage, attrs...)
	})
}
