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
	// If user_id is provided, fetch the email
	var userEmail *string
	if log.UserID != nil {
		var email string
		err := r.db.QueryRow("SELECT email FROM users WHERE id = $1", *log.UserID).Scan(&email)
		if err == nil {
			userEmail = &email
		}
		// If error (user deleted), userEmail stays nil
	}

	query := `
		INSERT INTO audit_logs (user_id, user_email, action, resource, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		log.UserID,
		userEmail,
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

// AuditFilters holds filter parameters for audit log queries
type AuditFilters struct {
	UserID    *uint
	Action    string
	Resource  string
	SortBy    string
	SortOrder string
}

// GetAll retrieves all audit logs with pagination
func (r *AuditRepository) GetAll(limit, offset int) ([]models.AuditLog, error) {
	query := `
		SELECT id, user_id, user_email, action, resource, details, ip_address, user_agent, created_at
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
			&log.UserEmail,
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

// GetAllWithFilters retrieves audit logs with filtering, sorting, and pagination
func (r *AuditRepository) GetAllWithFilters(filters AuditFilters, limit, offset int) ([]models.AuditLog, error) {
	query := `
		SELECT id, user_id, user_email, action, resource, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// User ID filter
	if filters.UserID != nil {
		query += fmt.Sprintf(` AND user_id = $%d`, argPos)
		args = append(args, *filters.UserID)
		argPos++
	}

	// Action filter
	if filters.Action != "" {
		query += fmt.Sprintf(` AND action ILIKE $%d`, argPos)
		args = append(args, "%"+filters.Action+"%")
		argPos++
	}

	// Resource filter
	if filters.Resource != "" {
		query += fmt.Sprintf(` AND resource ILIKE $%d`, argPos)
		args = append(args, "%"+filters.Resource+"%")
		argPos++
	}

	// Sorting
	sortColumn := "created_at"
	sortOrder := "DESC"

	if filters.SortBy != "" {
		switch filters.SortBy {
		case "id":
			sortColumn = "id"
		case "user_id":
			sortColumn = "user_id"
		case "action":
			sortColumn = "action"
		case "resource":
			sortColumn = "resource"
		case "created_at":
			sortColumn = "created_at"
		}
	}

	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(` ORDER BY %s %s LIMIT $%d OFFSET $%d`, sortColumn, sortOrder, argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
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
			&log.UserEmail,
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

// CountWithFilters returns the total count of audit logs matching the filters
func (r *AuditRepository) CountWithFilters(filters AuditFilters) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if filters.UserID != nil {
		query += fmt.Sprintf(` AND user_id = $%d`, argPos)
		args = append(args, *filters.UserID)
		argPos++
	}

	if filters.Action != "" {
		query += fmt.Sprintf(` AND action ILIKE $%d`, argPos)
		args = append(args, "%"+filters.Action+"%")
		argPos++
	}

	if filters.Resource != "" {
		query += fmt.Sprintf(` AND resource ILIKE $%d`, argPos)
		args = append(args, "%"+filters.Resource+"%")
		argPos++
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}
