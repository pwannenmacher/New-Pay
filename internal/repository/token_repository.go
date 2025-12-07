package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pwannenmacher/New-Pay/internal/models"
)

// TokenRepository handles token database operations
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// CreateEmailVerificationToken creates a new email verification token
func (r *TokenRepository) CreateEmailVerificationToken(token *models.EmailVerificationToken) error {
	query := `
		INSERT INTO email_verification_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		token.UserID,
		token.Token,
		token.ExpiresAt,
		time.Now(),
	).Scan(&token.ID)

	if err != nil {
		return fmt.Errorf("failed to create email verification token: %w", err)
	}

	return nil
}

// GetEmailVerificationToken retrieves an email verification token
func (r *TokenRepository) GetEmailVerificationToken(token string) (*models.EmailVerificationToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token = $1
	`

	t := &models.EmailVerificationToken{}
	err := r.db.QueryRow(query, token).Scan(
		&t.ID,
		&t.UserID,
		&t.Token,
		&t.ExpiresAt,
		&t.UsedAt,
		&t.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email verification token: %w", err)
	}

	return t, nil
}

// MarkEmailVerificationTokenUsed marks a token as used
func (r *TokenRepository) MarkEmailVerificationTokenUsed(tokenID uint) error {
	query := `
		UPDATE email_verification_tokens
		SET used_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, time.Now(), tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	return nil
}

// CreatePasswordResetToken creates a new password reset token
func (r *TokenRepository) CreatePasswordResetToken(token *models.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		token.UserID,
		token.Token,
		token.ExpiresAt,
		time.Now(),
	).Scan(&token.ID)

	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	return nil
}

// GetPasswordResetToken retrieves a password reset token
func (r *TokenRepository) GetPasswordResetToken(token string) (*models.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token = $1
	`

	t := &models.PasswordResetToken{}
	err := r.db.QueryRow(query, token).Scan(
		&t.ID,
		&t.UserID,
		&t.Token,
		&t.ExpiresAt,
		&t.UsedAt,
		&t.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}

	return t, nil
}

// MarkPasswordResetTokenUsed marks a token as used
func (r *TokenRepository) MarkPasswordResetTokenUsed(tokenID uint) error {
	query := `
		UPDATE password_reset_tokens
		SET used_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, time.Now(), tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	return nil
}

// DeleteExpiredTokens deletes all expired tokens
func (r *TokenRepository) DeleteExpiredTokens() error {
	now := time.Now()

	// Delete expired email verification tokens
	query1 := `DELETE FROM email_verification_tokens WHERE expires_at < $1`
	if _, err := r.db.Exec(query1, now); err != nil {
		return fmt.Errorf("failed to delete expired email tokens: %w", err)
	}

	// Delete expired password reset tokens
	query2 := `DELETE FROM password_reset_tokens WHERE expires_at < $1`
	if _, err := r.db.Exec(query2, now); err != nil {
		return fmt.Errorf("failed to delete expired password tokens: %w", err)
	}

	return nil
}
