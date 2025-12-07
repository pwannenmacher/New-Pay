package repository

import (
	"database/sql"
	"fmt"

	"new-pay/internal/models"
)

// OAuthConnectionRepository handles OAuth connection data access
type OAuthConnectionRepository struct {
	db *sql.DB
}

// NewOAuthConnectionRepository creates a new OAuth connection repository
func NewOAuthConnectionRepository(db *sql.DB) *OAuthConnectionRepository {
	return &OAuthConnectionRepository{db: db}
}

// Create creates a new OAuth connection
func (r *OAuthConnectionRepository) Create(conn *models.OAuthConnection) error {
	query := `
		INSERT INTO oauth_connections (user_id, provider, provider_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query, conn.UserID, conn.Provider, conn.ProviderID).Scan(
		&conn.ID, &conn.CreatedAt, &conn.UpdatedAt,
	)
}

// GetByUserID gets all OAuth connections for a user
func (r *OAuthConnectionRepository) GetByUserID(userID uint) ([]models.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_id, created_at, updated_at
		FROM oauth_connections
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []models.OAuthConnection
	for rows.Next() {
		var conn models.OAuthConnection
		if err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderID,
			&conn.CreatedAt, &conn.UpdatedAt,
		); err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}

	return connections, rows.Err()
}

// GetByProviderAndID gets an OAuth connection by provider and provider ID
func (r *OAuthConnectionRepository) GetByProviderAndID(provider, providerID string) (*models.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_id, created_at, updated_at
		FROM oauth_connections
		WHERE provider = $1 AND provider_id = $2
	`
	var conn models.OAuthConnection
	err := r.db.QueryRow(query, provider, providerID).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderID,
		&conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// GetByUserAndProvider gets a specific OAuth connection for a user and provider
func (r *OAuthConnectionRepository) GetByUserAndProvider(userID uint, provider string) (*models.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_id, created_at, updated_at
		FROM oauth_connections
		WHERE user_id = $1 AND provider = $2
	`
	var conn models.OAuthConnection
	err := r.db.QueryRow(query, userID, provider).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderID,
		&conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// Update updates an OAuth connection's updated_at timestamp
func (r *OAuthConnectionRepository) Update(conn *models.OAuthConnection) error {
	query := `
		UPDATE oauth_connections
		SET updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(query, conn.ID).Scan(&conn.UpdatedAt)
}

// Delete deletes an OAuth connection
func (r *OAuthConnectionRepository) Delete(id uint) error {
	query := `DELETE FROM oauth_connections WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("oauth connection not found")
	}
	return nil
}

// DeleteByUserAndProvider deletes an OAuth connection for a user and provider
func (r *OAuthConnectionRepository) DeleteByUserAndProvider(userID uint, provider string) error {
	query := `DELETE FROM oauth_connections WHERE user_id = $1 AND provider = $2`
	result, err := r.db.Exec(query, userID, provider)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("oauth connection not found")
	}
	return nil
}
