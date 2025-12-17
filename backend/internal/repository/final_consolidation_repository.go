package repository

import (
	"database/sql"
	"new-pay/internal/models"
)

type FinalConsolidationRepository struct {
	db *sql.DB
}

func NewFinalConsolidationRepository(db *sql.DB) *FinalConsolidationRepository {
	return &FinalConsolidationRepository{db: db}
}

// CreateOrUpdate creates or updates a final consolidation
func (r *FinalConsolidationRepository) CreateOrUpdate(fc *models.FinalConsolidation) error {
	query := `
		INSERT INTO final_consolidations (
			assessment_id, encrypted_comment_id, created_by_user_id, updated_at
		) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (assessment_id)
		DO UPDATE SET
			encrypted_comment_id = EXCLUDED.encrypted_comment_id,
			created_by_user_id = EXCLUDED.created_by_user_id,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		fc.AssessmentID,
		fc.EncryptedCommentID,
		fc.CreatedByUserID,
	).Scan(&fc.ID, &fc.CreatedAt, &fc.UpdatedAt)
}

// GetByAssessment retrieves final consolidation by assessment ID
func (r *FinalConsolidationRepository) GetByAssessment(assessmentID uint) (*models.FinalConsolidation, error) {
	query := `
		SELECT id, assessment_id, encrypted_comment_id, created_by_user_id, created_at, updated_at
		FROM final_consolidations
		WHERE assessment_id = $1
	`

	var fc models.FinalConsolidation
	err := r.db.QueryRow(query, assessmentID).Scan(
		&fc.ID,
		&fc.AssessmentID,
		&fc.EncryptedCommentID,
		&fc.CreatedByUserID,
		&fc.CreatedAt,
		&fc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &fc, nil
}

type FinalConsolidationApprovalRepository struct {
	db *sql.DB
}

func NewFinalConsolidationApprovalRepository(db *sql.DB) *FinalConsolidationApprovalRepository {
	return &FinalConsolidationApprovalRepository{db: db}
}

// CreateApproval creates a new approval for final consolidation
func (r *FinalConsolidationApprovalRepository) CreateApproval(assessmentID, userID uint) error {
	query := `
		INSERT INTO final_consolidation_approvals (assessment_id, approved_by_user_id)
		VALUES ($1, $2)
		ON CONFLICT (assessment_id, approved_by_user_id) DO NOTHING
	`
	_, err := r.db.Exec(query, assessmentID, userID)
	return err
}

// DeleteApproval removes a single approval from a user
func (r *FinalConsolidationApprovalRepository) DeleteApproval(assessmentID, userID uint) error {
	query := `DELETE FROM final_consolidation_approvals WHERE assessment_id = $1 AND approved_by_user_id = $2`
	_, err := r.db.Exec(query, assessmentID, userID)
	return err
}

// DeleteAllApprovalsForAssessment removes all approvals (used when final consolidation is edited)
func (r *FinalConsolidationApprovalRepository) DeleteAllApprovalsForAssessment(assessmentID uint) error {
	query := `DELETE FROM final_consolidation_approvals WHERE assessment_id = $1`
	_, err := r.db.Exec(query, assessmentID)
	return err
}

// GetApprovalsByAssessment gets all approvals for an assessment with user names
func (r *FinalConsolidationApprovalRepository) GetApprovalsByAssessment(assessmentID uint) ([]models.FinalConsolidationApproval, error) {
	query := `
		SELECT 
			a.id,
			a.assessment_id,
			a.approved_by_user_id,
			COALESCE(u.first_name || ' ' || u.last_name, u.email) as approved_by_name,
			a.approved_at
		FROM final_consolidation_approvals a
		LEFT JOIN users u ON a.approved_by_user_id = u.id
		WHERE a.assessment_id = $1
		ORDER BY a.approved_at ASC
	`

	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []models.FinalConsolidationApproval
	for rows.Next() {
		var approval models.FinalConsolidationApproval
		err := rows.Scan(
			&approval.ID,
			&approval.AssessmentID,
			&approval.ApprovedByUserID,
			&approval.ApprovedByName,
			&approval.ApprovedAt,
		)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}

	return approvals, nil
}

// HasUserApproved checks if a user has approved the final consolidation
func (r *FinalConsolidationApprovalRepository) HasUserApproved(assessmentID, userID uint) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM final_consolidation_approvals WHERE assessment_id = $1 AND approved_by_user_id = $2`
	err := r.db.QueryRow(query, assessmentID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetApprovalCount returns the number of approvals for an assessment
func (r *FinalConsolidationApprovalRepository) GetApprovalCount(assessmentID uint) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM final_consolidation_approvals WHERE assessment_id = $1`
	err := r.db.QueryRow(query, assessmentID).Scan(&count)
	return count, err
}
