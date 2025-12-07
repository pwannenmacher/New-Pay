package middleware

import (
	"database/sql"
	"net/http"

	"new-pay/internal/models"
	"new-pay/internal/repository"
)

// AuditMiddleware logs security-related actions
type AuditMiddleware struct {
	auditRepo *repository.AuditRepository
}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware(db *sql.DB) *AuditMiddleware {
	return &AuditMiddleware{
		auditRepo: repository.NewAuditRepository(db),
	}
}

// Log logs an action to the audit log
func (m *AuditMiddleware) Log(action, resource string, details string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Call the next handler first
			next.ServeHTTP(w, r)

			// Get user ID from context if available
			var userID *uint
			if id, ok := GetUserID(r); ok {
				userID = &id
			}

			// Create audit log entry
			log := &models.AuditLog{
				UserID:    userID,
				Action:    action,
				Resource:  resource,
				Details:   details,
				IPAddress: getIP(r),
				UserAgent: r.UserAgent(),
			}

			// Save to database (ignore errors to not block the request)
			_ = m.auditRepo.Create(log)
		})
	}
}

// LogAction logs a specific action
func (m *AuditMiddleware) LogAction(userID *uint, action, resource, details, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	return m.auditRepo.Create(log)
}
