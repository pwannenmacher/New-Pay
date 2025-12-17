package repository

import (
	"database/sql"
	"fmt"
	"new-pay/internal/models"
)

// ReviewerResponseRepository handles database operations for reviewer responses
type ReviewerResponseRepository struct {
	db *sql.DB
}

// NewReviewerResponseRepository creates a new reviewer response repository
func NewReviewerResponseRepository(db *sql.DB) *ReviewerResponseRepository {
	return &ReviewerResponseRepository{db: db}
}

// CreateOrUpdate creates or updates a reviewer response
func (r *ReviewerResponseRepository) CreateOrUpdate(response *models.ReviewerResponse) error {
	query := `
		INSERT INTO reviewer_responses (
			assessment_id, category_id, reviewer_user_id, path_id, level_id, encrypted_justification_id, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (assessment_id, category_id, reviewer_user_id)
		DO UPDATE SET
			path_id = EXCLUDED.path_id,
			level_id = EXCLUDED.level_id,
			encrypted_justification_id = EXCLUDED.encrypted_justification_id,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		response.AssessmentID,
		response.CategoryID,
		response.ReviewerUserID,
		response.PathID,
		response.LevelID,
		response.EncryptedJustificationID,
	).Scan(&response.ID, &response.CreatedAt, &response.UpdatedAt)
}

// GetByAssessmentAndReviewer retrieves all responses by a specific reviewer for an assessment
func (r *ReviewerResponseRepository) GetByAssessmentAndReviewer(assessmentID, reviewerUserID uint) ([]models.ReviewerResponse, error) {
	query := `
		SELECT id, assessment_id, category_id, reviewer_user_id, path_id, level_id, 
		       encrypted_justification_id, created_at, updated_at
		FROM reviewer_responses
		WHERE assessment_id = $1 AND reviewer_user_id = $2
		ORDER BY category_id
	`

	rows, err := r.db.Query(query, assessmentID, reviewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil to avoid JSON null
	responses := []models.ReviewerResponse{}
	for rows.Next() {
		var resp models.ReviewerResponse
		err := rows.Scan(
			&resp.ID,
			&resp.AssessmentID,
			&resp.CategoryID,
			&resp.ReviewerUserID,
			&resp.PathID,
			&resp.LevelID,
			&resp.EncryptedJustificationID,
			&resp.CreatedAt,
			&resp.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, rows.Err()
}

// GetByCategoryAndReviewer retrieves a specific reviewer response for a category
func (r *ReviewerResponseRepository) GetByCategoryAndReviewer(assessmentID, categoryID, reviewerUserID uint) (*models.ReviewerResponse, error) {
	var resp models.ReviewerResponse
	query := `
		SELECT id, assessment_id, category_id, reviewer_user_id, path_id, level_id, 
		       encrypted_justification_id, created_at, updated_at
		FROM reviewer_responses
		WHERE assessment_id = $1 AND category_id = $2 AND reviewer_user_id = $3
	`

	err := r.db.QueryRow(query, assessmentID, categoryID, reviewerUserID).Scan(
		&resp.ID,
		&resp.AssessmentID,
		&resp.CategoryID,
		&resp.ReviewerUserID,
		&resp.PathID,
		&resp.LevelID,
		&resp.EncryptedJustificationID,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Delete deletes a reviewer response
func (r *ReviewerResponseRepository) Delete(assessmentID, categoryID, reviewerUserID uint) error {
	query := `
		DELETE FROM reviewer_responses 
		WHERE assessment_id = $1 AND category_id = $2 AND reviewer_user_id = $3
	`

	result, err := r.db.Exec(query, assessmentID, categoryID, reviewerUserID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no reviewer response found to delete")
	}

	return nil
}

// CountByAssessmentAndReviewer counts responses by a reviewer for an assessment
func (r *ReviewerResponseRepository) CountByAssessmentAndReviewer(assessmentID, reviewerUserID uint) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM reviewer_responses 
		WHERE assessment_id = $1 AND reviewer_user_id = $2
	`
	err := r.db.QueryRow(query, assessmentID, reviewerUserID).Scan(&count)
	return count, err
}

// GetCompleteReviewers returns reviewers who have completed all categories for an assessment
func (r *ReviewerResponseRepository) GetCompleteReviewers(assessmentID uint) ([]models.ReviewerCompletionInfo, error) {
	query := `
		WITH category_count AS (
			SELECT COUNT(DISTINCT ar.category_id) as total_categories
			FROM assessment_responses ar
			WHERE ar.assessment_id = $1
		),
		reviewer_progress AS (
			SELECT 
				rr.reviewer_user_id,
				COUNT(DISTINCT rr.category_id) as completed_categories,
				MAX(rr.updated_at) as last_updated
			FROM reviewer_responses rr
			WHERE rr.assessment_id = $1
			GROUP BY rr.reviewer_user_id
		)
		SELECT 
			rp.reviewer_user_id,
			CONCAT(u.first_name, ' ', u.last_name) as reviewer_name,
			rp.last_updated
		FROM reviewer_progress rp
		CROSS JOIN category_count cc
		JOIN users u ON u.id = rp.reviewer_user_id
		WHERE rp.completed_categories >= cc.total_categories
		ORDER BY rp.last_updated
	`

	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []models.ReviewerCompletionInfo
	for rows.Next() {
		var info models.ReviewerCompletionInfo
		err := rows.Scan(&info.ReviewerID, &info.ReviewerName, &info.CompletedAt)
		if err != nil {
			return nil, err
		}
		reviewers = append(reviewers, info)
	}

	return reviewers, rows.Err()
}

// CountCompleteReviews counts how many reviewers have completed all categories
func (r *ReviewerResponseRepository) CountCompleteReviews(assessmentID uint) (int, error) {
	reviewers, err := r.GetCompleteReviewers(assessmentID)
	if err != nil {
		return 0, err
	}
	return len(reviewers), nil
}

// CountTotalReviewers counts distinct reviewers who have at least one response
func (r *ReviewerResponseRepository) CountTotalReviewers(assessmentID uint) (int, error) {
	var count int
	query := `
		SELECT COUNT(DISTINCT reviewer_user_id) 
		FROM reviewer_responses 
		WHERE assessment_id = $1
	`
	err := r.db.QueryRow(query, assessmentID).Scan(&count)
	return count, err
}

// GetReviewStats returns statistics about reviews for an assessment
func (r *ReviewerResponseRepository) GetReviewStats(assessmentID uint) (startedCount, completedCount int, err error) {
	// Count reviewers who have completed all categories
	completedCount, err = r.CountCompleteReviews(assessmentID)
	if err != nil {
		return 0, 0, err
	}

	// Count total reviewers who have at least one response
	totalReviewers, err := r.CountTotalReviewers(assessmentID)
	if err != nil {
		return 0, 0, err
	}

	// Started = total reviewers minus completed reviewers
	startedCount = totalReviewers - completedCount

	return startedCount, completedCount, nil
}

// GetAllByAssessment retrieves all reviewer responses for an assessment (admin only)
func (r *ReviewerResponseRepository) GetAllByAssessment(assessmentID uint) ([]models.ReviewerResponse, error) {
	query := `
		SELECT id, assessment_id, category_id, reviewer_user_id, path_id, level_id, 
		       encrypted_justification_id, created_at, updated_at
		FROM reviewer_responses
		WHERE assessment_id = $1
		ORDER BY reviewer_user_id, category_id
	`

	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil to avoid JSON null
	responses := []models.ReviewerResponse{}
	for rows.Next() {
		var resp models.ReviewerResponse
		err := rows.Scan(
			&resp.ID,
			&resp.AssessmentID,
			&resp.CategoryID,
			&resp.ReviewerUserID,
			&resp.PathID,
			&resp.LevelID,
			&resp.EncryptedJustificationID,
			&resp.CreatedAt,
			&resp.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, rows.Err()
}
