package repository

import (
	"database/sql"
	"fmt"
	"time"

	"new-pay/internal/models"
)

// SessionRepository handles session database operations
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *models.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, session_id, jti, token_type, expires_at, last_activity_at, created_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(
		query,
		session.ID,
		session.UserID,
		session.SessionID,
		session.JTI,
		session.TokenType,
		session.ExpiresAt,
		session.LastActivityAt,
		session.CreatedAt,
		session.IPAddress,
		session.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetByJTI retrieves a session by JTI
func (r *SessionRepository) GetByJTI(jti string) (*models.Session, error) {
	query := `
		SELECT id, user_id, session_id, jti, token_type, expires_at, last_activity_at, created_at, ip_address, user_agent
		FROM sessions
		WHERE jti = $1 AND expires_at > $2
	`

	session := &models.Session{}
	err := r.db.QueryRow(query, jti, time.Now()).Scan(
		&session.ID,
		&session.UserID,
		&session.SessionID,
		&session.JTI,
		&session.TokenType,
		&session.ExpiresAt,
		&session.LastActivityAt,
		&session.CreatedAt,
		&session.IPAddress,
		&session.UserAgent,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetByUserID retrieves all active sessions for a user
func (r *SessionRepository) GetByUserID(userID uint) ([]models.Session, error) {
	query := `
		SELECT id, user_id, session_id, jti, token_type, expires_at, last_activity_at, created_at, ip_address, user_agent
		FROM sessions
		WHERE user_id = $1 AND expires_at > $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var session models.Session
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.SessionID,
			&session.JTI,
			&session.TokenType,
			&session.ExpiresAt,
			&session.LastActivityAt,
			&session.CreatedAt,
			&session.IPAddress,
			&session.UserAgent,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateLastActivity updates the last activity timestamp for a session
func (r *SessionRepository) UpdateLastActivity(sessionID string) error {
	query := `
		UPDATE sessions
		SET last_activity_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	return nil
}

// Delete deletes a specific session
func (r *SessionRepository) Delete(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteByJTI deletes a session by JTI
func (r *SessionRepository) DeleteByJTI(jti string) error {
	query := `DELETE FROM sessions WHERE jti = $1`
	_, err := r.db.Exec(query, jti)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteBySessionID deletes all tokens from a specific session (access + refresh)
func (r *SessionRepository) DeleteBySessionID(sessionID string) error {
	query := `DELETE FROM sessions WHERE session_id = $1`
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteAllUserSessions deletes all sessions for a user
func (r *SessionRepository) DeleteAllUserSessions(userID uint) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

// DeleteExpiredSessions deletes all expired sessions
func (r *SessionRepository) DeleteExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at < $1`
	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}
