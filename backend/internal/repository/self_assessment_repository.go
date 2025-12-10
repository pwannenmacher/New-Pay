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

// Create creates a new self-assessment
func (r *SelfAssessmentRepository) Create(assessment *models.SelfAssessment) error {
	query := `
		INSERT INTO self_assessments (catalog_id, user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(
		query,
		assessment.CatalogID,
		assessment.UserID,
		assessment.Status,
	).Scan(&assessment.ID, &assessment.CreatedAt, &assessment.UpdatedAt)

	return err
}

// GetByID retrieves a self-assessment by ID
func (r *SelfAssessmentRepository) GetByID(assessmentID uint) (*models.SelfAssessment, error) {
	var assessment models.SelfAssessment
	query := `
		SELECT id, catalog_id, user_id, status, created_at, updated_at, 
		       submitted_at, in_review_at, reviewed_at, discussion_started_at, 
		       archived_at, closed_at, previous_status
		FROM self_assessments
		WHERE id = $1
	`
	err := r.db.QueryRow(query, assessmentID).Scan(
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

// Update updates a self-assessment
func (r *SelfAssessmentRepository) Update(assessment *models.SelfAssessment) error {
	query := `
		UPDATE self_assessments
		SET status = $1, submitted_at = $2, in_review_at = $3, reviewed_at = $4,
		    discussion_started_at = $5, archived_at = $6, closed_at = $7, previous_status = $8,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
	`
	_, err := r.db.Exec(
		query,
		assessment.Status,
		assessment.SubmittedAt,
		assessment.InReviewAt,
		assessment.ReviewedAt,
		assessment.DiscussionStartedAt,
		assessment.ArchivedAt,
		assessment.ClosedAt,
		assessment.PreviousStatus,
		assessment.ID,
	)
	return err
}

// GetByUserID retrieves all self-assessments for a user
func (r *SelfAssessmentRepository) GetByUserID(userID uint) ([]models.SelfAssessment, error) {
	query := `
		SELECT id, catalog_id, user_id, status, created_at, updated_at, 
		       submitted_at, in_review_at, reviewed_at, discussion_started_at, 
		       archived_at, closed_at, previous_status
		FROM self_assessments
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessment
	for rows.Next() {
		var assessment models.SelfAssessment
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}

// GetAllMetadata retrieves metadata for all self-assessments (for admins)
func (r *SelfAssessmentRepository) GetAllMetadata() ([]models.SelfAssessment, error) {
	query := `
		SELECT id, catalog_id, user_id, status, created_at, updated_at, 
		       submitted_at, in_review_at, reviewed_at, discussion_started_at, 
		       archived_at, closed_at, previous_status
		FROM self_assessments
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessment
	for rows.Next() {
		var assessment models.SelfAssessment
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}

// GetVisibleToReviewers retrieves self-assessments visible to reviewers (submitted or later)
func (r *SelfAssessmentRepository) GetVisibleToReviewers() ([]models.SelfAssessment, error) {
	query := `
		SELECT id, catalog_id, user_id, status, created_at, updated_at, 
		       submitted_at, in_review_at, reviewed_at, discussion_started_at, 
		       archived_at, closed_at, previous_status
		FROM self_assessments
		WHERE status IN ('submitted', 'in_review', 'reviewed', 'discussion', 'archived')
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessment
	for rows.Next() {
		var assessment models.SelfAssessment
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}
