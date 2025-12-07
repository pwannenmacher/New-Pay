package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pwannenmacher/New-Pay/internal/models"
)

// AuditRepository handles audit log database operations
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditRepository) Create(log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, action, resource, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		log.UserID,
		log.Action,
		log.Resource,
		log.Details,
		log.IPAddress,
		log.UserAgent,
		time.Now(),
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetByUserID retrieves audit logs for a specific user
func (r *AuditRepository) GetByUserID(userID uint, limit, offset int) ([]models.AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		if err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.Resource,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetAll retrieves all audit logs with pagination
func (r *AuditRepository) GetAll(limit, offset int) ([]models.AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource, details, ip_address, user_agent, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		if err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.Resource,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
