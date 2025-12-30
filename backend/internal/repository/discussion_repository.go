package repository

import (
	"database/sql"
	"fmt"

	"new-pay/internal/models"
)

type DiscussionRepository struct {
	db *sql.DB
}

func NewDiscussionRepository(db *sql.DB) *DiscussionRepository {
	return &DiscussionRepository{db: db}
}

// Create creates a new discussion result
func (r *DiscussionRepository) Create(result *models.DiscussionResult) error {
	query := `
		INSERT INTO discussion_results (
			assessment_id, weighted_overall_level_number, weighted_overall_level_id,
			encrypted_final_comment_id, encrypted_discussion_note_id, user_approved_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query,
		result.AssessmentID,
		result.WeightedOverallLevelNum,
		result.WeightedOverallLevelID,
		result.EncryptedFinalCommentID,
		result.EncryptedDiscussionNoteID,
		result.UserApprovedAt,
	).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt)
}

// CreateCategoryResult creates a category result
func (r *DiscussionRepository) CreateCategoryResult(categoryResult *models.DiscussionCategoryResult) error {
	query := `
		INSERT INTO discussion_category_results (
			discussion_result_id, category_id, user_level_id, reviewer_level_id,
			reviewer_level_number, encrypted_justification_id, is_override
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return r.db.QueryRow(query,
		categoryResult.DiscussionResultID,
		categoryResult.CategoryID,
		categoryResult.UserLevelID,
		categoryResult.ReviewerLevelID,
		categoryResult.ReviewerLevelNumber,
		categoryResult.EncryptedJustificationID,
		categoryResult.IsOverride,
	).Scan(&categoryResult.ID)
}

// CreateReviewer creates a reviewer record
func (r *DiscussionRepository) CreateReviewer(reviewer *models.DiscussionReviewer) error {
	query := `
		INSERT INTO discussion_reviewers (discussion_result_id, reviewer_user_id, reviewer_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	return r.db.QueryRow(query,
		reviewer.DiscussionResultID,
		reviewer.ReviewerUserID,
		reviewer.ReviewerName,
	).Scan(&reviewer.ID)
}

// GetByAssessmentID retrieves discussion result by assessment ID
func (r *DiscussionRepository) GetByAssessmentID(assessmentID uint) (*models.DiscussionResult, error) {
	var result models.DiscussionResult
	query := `
		SELECT id, assessment_id, weighted_overall_level_number, weighted_overall_level_id,
			encrypted_final_comment_id, encrypted_discussion_note_id, user_approved_at, created_at, updated_at
		FROM discussion_results
		WHERE assessment_id = $1
	`
	err := r.db.QueryRow(query, assessmentID).Scan(
		&result.ID,
		&result.AssessmentID,
		&result.WeightedOverallLevelNum,
		&result.WeightedOverallLevelID,
		&result.EncryptedFinalCommentID,
		&result.EncryptedDiscussionNoteID,
		&result.UserApprovedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get discussion result: %w", err)
	}
	return &result, nil
}

// GetCategoryResults retrieves all category results for a discussion
func (r *DiscussionRepository) GetCategoryResults(discussionResultID uint) ([]models.DiscussionCategoryResult, error) {
	results := []models.DiscussionCategoryResult{}
	query := `
		SELECT id, discussion_result_id, category_id, user_level_id, reviewer_level_id,
			reviewer_level_number, encrypted_justification_id, is_override
		FROM discussion_category_results
		WHERE discussion_result_id = $1
		ORDER BY category_id
	`
	rows, err := r.db.Query(query, discussionResultID)
	if err != nil {
		return nil, fmt.Errorf("failed to query category results: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result models.DiscussionCategoryResult
		err := rows.Scan(
			&result.ID,
			&result.DiscussionResultID,
			&result.CategoryID,
			&result.UserLevelID,
			&result.ReviewerLevelID,
			&result.ReviewerLevelNumber,
			&result.EncryptedJustificationID,
			&result.IsOverride,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category result: %w", err)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category results: %w", err)
	}

	return results, nil
}

// GetReviewers retrieves all reviewers for a discussion
func (r *DiscussionRepository) GetReviewers(discussionResultID uint) ([]models.DiscussionReviewer, error) {
	reviewers := []models.DiscussionReviewer{}
	query := `
		SELECT id, discussion_result_id, reviewer_user_id, reviewer_name
		FROM discussion_reviewers
		WHERE discussion_result_id = $1
		ORDER BY reviewer_name
	`
	rows, err := r.db.Query(query, discussionResultID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviewers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reviewer models.DiscussionReviewer
		err := rows.Scan(
			&reviewer.ID,
			&reviewer.DiscussionResultID,
			&reviewer.ReviewerUserID,
			&reviewer.ReviewerName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reviewer: %w", err)
		}
		reviewers = append(reviewers, reviewer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reviewers: %w", err)
	}

	return reviewers, nil
}

// UpdateDiscussionNote updates the discussion note and user approval
func (r *DiscussionRepository) UpdateDiscussionNote(discussionResultID uint, encryptedNoteID *int64, userApprovedAt *sql.NullTime) error {
	query := `
		UPDATE discussion_results
		SET encrypted_discussion_note_id = $1, user_approved_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := r.db.Exec(query, encryptedNoteID, userApprovedAt, discussionResultID)
	if err != nil {
		return fmt.Errorf("failed to update discussion note: %w", err)
	}
	return nil
}
