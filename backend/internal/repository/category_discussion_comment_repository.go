package repository

import (
	"database/sql"
	"fmt"
	"new-pay/internal/models"
)

type CategoryDiscussionCommentRepository struct {
	db *sql.DB
}

func NewCategoryDiscussionCommentRepository(db *sql.DB) *CategoryDiscussionCommentRepository {
	return &CategoryDiscussionCommentRepository{db: db}
}

// Create creates a new category discussion comment
func (r *CategoryDiscussionCommentRepository) Create(comment *models.CategoryDiscussionComment) error {
	query := `
		INSERT INTO category_discussion_comments 
		(assessment_id, category_id, encrypted_comment_id, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(
		query,
		comment.AssessmentID,
		comment.CategoryID,
		comment.EncryptedCommentID,
		comment.CreatedByUserID,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
}

// Update updates an existing category discussion comment
func (r *CategoryDiscussionCommentRepository) Update(comment *models.CategoryDiscussionComment) error {
	query := `
		UPDATE category_discussion_comments
		SET encrypted_comment_id = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`
	return r.db.QueryRow(
		query,
		comment.EncryptedCommentID,
		comment.ID,
	).Scan(&comment.UpdatedAt)
}

// GetByAssessmentAndCategory retrieves a comment by assessment and category ID
func (r *CategoryDiscussionCommentRepository) GetByAssessmentAndCategory(assessmentID, categoryID uint) (*models.CategoryDiscussionComment, error) {
	query := `
		SELECT id, assessment_id, category_id, encrypted_comment_id, created_by_user_id, created_at, updated_at
		FROM category_discussion_comments
		WHERE assessment_id = $1 AND category_id = $2
	`
	comment := &models.CategoryDiscussionComment{}
	err := r.db.QueryRow(query, assessmentID, categoryID).Scan(
		&comment.ID,
		&comment.AssessmentID,
		&comment.CategoryID,
		&comment.EncryptedCommentID,
		&comment.CreatedByUserID,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get category discussion comment: %w", err)
	}
	return comment, nil
}

// GetByAssessment retrieves all comments for an assessment
func (r *CategoryDiscussionCommentRepository) GetByAssessment(assessmentID uint) ([]models.CategoryDiscussionComment, error) {
	query := `
		SELECT id, assessment_id, category_id, encrypted_comment_id, created_by_user_id, created_at, updated_at
		FROM category_discussion_comments
		WHERE assessment_id = $1
		ORDER BY category_id
	`
	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query category discussion comments: %w", err)
	}
	defer rows.Close()

	var comments []models.CategoryDiscussionComment
	for rows.Next() {
		var comment models.CategoryDiscussionComment
		err := rows.Scan(
			&comment.ID,
			&comment.AssessmentID,
			&comment.CategoryID,
			&comment.EncryptedCommentID,
			&comment.CreatedByUserID,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category discussion comment: %w", err)
		}
		comments = append(comments, comment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category discussion comments: %w", err)
	}

	return comments, nil
}

// Delete deletes a category discussion comment
func (r *CategoryDiscussionCommentRepository) Delete(id uint) error {
	query := `DELETE FROM category_discussion_comments WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
