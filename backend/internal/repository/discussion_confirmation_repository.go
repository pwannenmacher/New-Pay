package repository

import (
	"database/sql"
	"new-pay/internal/models"
)

type DiscussionConfirmationRepository struct {
	db *sql.DB
}

func NewDiscussionConfirmationRepository(db *sql.DB) *DiscussionConfirmationRepository {
	return &DiscussionConfirmationRepository{db: db}
}

// Create creates a new discussion confirmation
func (r *DiscussionConfirmationRepository) Create(confirmation *models.DiscussionConfirmation) error {
	query := `
		INSERT INTO discussion_confirmations (assessment_id, user_id, user_type, confirmed_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, confirmed_at, created_at
	`
	return r.db.QueryRow(
		query,
		confirmation.AssessmentID,
		confirmation.UserID,
		confirmation.UserType,
	).Scan(&confirmation.ID, &confirmation.ConfirmedAt, &confirmation.CreatedAt)
}

// GetByAssessment retrieves all confirmations for an assessment with user details
func (r *DiscussionConfirmationRepository) GetByAssessment(assessmentID uint) ([]models.DiscussionConfirmation, error) {
	query := `
		SELECT 
			dc.id, dc.assessment_id, dc.user_id, dc.user_type, dc.confirmed_at, dc.created_at,
			u.first_name || ' ' || u.last_name as user_name, u.email as user_email
		FROM discussion_confirmations dc
		JOIN users u ON dc.user_id = u.id
		WHERE dc.assessment_id = $1
		ORDER BY dc.confirmed_at ASC
	`
	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var confirmations []models.DiscussionConfirmation
	for rows.Next() {
		var c models.DiscussionConfirmation
		err := rows.Scan(
			&c.ID,
			&c.AssessmentID,
			&c.UserID,
			&c.UserType,
			&c.ConfirmedAt,
			&c.CreatedAt,
			&c.UserName,
			&c.UserEmail,
		)
		if err != nil {
			return nil, err
		}
		confirmations = append(confirmations, c)
	}

	return confirmations, rows.Err()
}

// GetByAssessmentAndUser retrieves a confirmation for a specific user and assessment
func (r *DiscussionConfirmationRepository) GetByAssessmentAndUser(assessmentID, userID uint) (*models.DiscussionConfirmation, error) {
	query := `
		SELECT 
			dc.id, dc.assessment_id, dc.user_id, dc.user_type, dc.confirmed_at, dc.created_at,
			u.first_name || ' ' || u.last_name as user_name, u.email as user_email
		FROM discussion_confirmations dc
		JOIN users u ON dc.user_id = u.id
		WHERE dc.assessment_id = $1 AND dc.user_id = $2
	`
	var c models.DiscussionConfirmation
	err := r.db.QueryRow(query, assessmentID, userID).Scan(
		&c.ID,
		&c.AssessmentID,
		&c.UserID,
		&c.UserType,
		&c.ConfirmedAt,
		&c.CreatedAt,
		&c.UserName,
		&c.UserEmail,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// HasReviewerConfirmation checks if at least one reviewer has confirmed
func (r *DiscussionConfirmationRepository) HasReviewerConfirmation(assessmentID uint) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM discussion_confirmations 
			WHERE assessment_id = $1 AND user_type = 'reviewer'
		)
	`
	var exists bool
	err := r.db.QueryRow(query, assessmentID).Scan(&exists)
	return exists, err
}

// HasOwnerConfirmation checks if the owner has confirmed
func (r *DiscussionConfirmationRepository) HasOwnerConfirmation(assessmentID uint) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM discussion_confirmations 
			WHERE assessment_id = $1 AND user_type = 'owner'
		)
	`
	var exists bool
	err := r.db.QueryRow(query, assessmentID).Scan(&exists)
	return exists, err
}

// Delete removes a confirmation (for testing/admin purposes)
func (r *DiscussionConfirmationRepository) Delete(id uint) error {
	query := `DELETE FROM discussion_confirmations WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
