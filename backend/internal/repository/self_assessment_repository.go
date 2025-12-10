package repository

import (
	"database/sql"
	"new-pay/internal/models"
)

// SelfAssessmentRepository handles database operations for self-assessments
type SelfAssessmentRepository struct {
	db *sql.DB
}

// NewSelfAssessmentRepository creates a new self-assessment repository
func NewSelfAssessmentRepository(db *sql.DB) *SelfAssessmentRepository {
	return &SelfAssessmentRepository{db: db}
}

// CountByCatalogID counts self-assessments for a catalog
func (r *SelfAssessmentRepository) CountByCatalogID(catalogID uint) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM self_assessments WHERE catalog_id = $1`
	err := r.db.QueryRow(query, catalogID).Scan(&count)
	return count, err
}

// HasSelfAssessments checks if a catalog has any self-assessments
func (r *SelfAssessmentRepository) HasSelfAssessments(catalogID uint) (bool, error) {
	count, err := r.CountByCatalogID(catalogID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByCatalogAndUser retrieves a self-assessment by catalog and user
func (r *SelfAssessmentRepository) GetByCatalogAndUser(catalogID, userID uint) (*models.SelfAssessment, error) {
	var assessment models.SelfAssessment
	query := `
		SELECT id, catalog_id, user_id, status, created_at, updated_at, 
		       submitted_at, in_review_at, reviewed_at, discussion_started_at, 
		       archived_at, closed_at, previous_status
		FROM self_assessments
		WHERE catalog_id = $1 AND user_id = $2
	`
	err := r.db.QueryRow(query, catalogID, userID).Scan(
		&assessment.ID,
		&assessment.CatalogID,
		&assessment.UserID,
		&assessment.Status,
		&assessment.CreatedAt,
		&assessment.UpdatedAt,
		&assessment.SubmittedAt,
		&assessment.InReviewAt,
		&assessment.ReviewedAt,
		&assessment.DiscussionStartedAt,
		&assessment.ArchivedAt,
		&assessment.ClosedAt,
		&assessment.PreviousStatus,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &assessment, nil
}
