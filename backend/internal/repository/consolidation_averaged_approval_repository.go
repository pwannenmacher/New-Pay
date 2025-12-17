package repository

import (
	"database/sql"
	"new-pay/internal/models"
)

type ConsolidationAveragedApprovalRepository struct {
	db *sql.DB
}

func NewConsolidationAveragedApprovalRepository(db *sql.DB) *ConsolidationAveragedApprovalRepository {
	return &ConsolidationAveragedApprovalRepository{db: db}
}

// CreateApproval creates a new approval for an averaged response
func (r *ConsolidationAveragedApprovalRepository) CreateApproval(assessmentID, categoryID, userID uint) error {
	query := `
		INSERT INTO consolidation_averaged_approvals (assessment_id, category_id, approved_by_user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (assessment_id, category_id, approved_by_user_id) DO NOTHING
	`
	_, err := r.db.Exec(query, assessmentID, categoryID, userID)
	return err
}

// DeleteApproval removes an approval
func (r *ConsolidationAveragedApprovalRepository) DeleteApproval(assessmentID, categoryID, userID uint) error {
	query := `DELETE FROM consolidation_averaged_approvals WHERE assessment_id = $1 AND category_id = $2 AND approved_by_user_id = $3`
	_, err := r.db.Exec(query, assessmentID, categoryID, userID)
	return err
}

// DeleteAllApprovalsForCategory removes all approvals for a category (used when override is created)
func (r *ConsolidationAveragedApprovalRepository) DeleteAllApprovalsForCategory(assessmentID, categoryID uint) error {
	query := `DELETE FROM consolidation_averaged_approvals WHERE assessment_id = $1 AND category_id = $2`
	_, err := r.db.Exec(query, assessmentID, categoryID)
	return err
}

// GetApprovalsByAssessment gets all approvals for an assessment with user names
func (r *ConsolidationAveragedApprovalRepository) GetApprovalsByAssessment(assessmentID uint) ([]models.ConsolidationAveragedApproval, error) {
	query := `
		SELECT 
			a.id,
			a.assessment_id,
			a.category_id,
			a.approved_by_user_id,
			COALESCE(u.first_name || ' ' || u.last_name, u.email) as approved_by_name,
			a.approved_at
		FROM consolidation_averaged_approvals a
		LEFT JOIN users u ON a.approved_by_user_id = u.id
		WHERE a.assessment_id = $1
		ORDER BY a.category_id, a.approved_at ASC
	`

	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []models.ConsolidationAveragedApproval
	for rows.Next() {
		var approval models.ConsolidationAveragedApproval
		err := rows.Scan(
			&approval.ID,
			&approval.AssessmentID,
			&approval.CategoryID,
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

// GetApprovalsByCategory gets all approvals for a specific category
func (r *ConsolidationAveragedApprovalRepository) GetApprovalsByCategory(assessmentID, categoryID uint) ([]models.ConsolidationAveragedApproval, error) {
	query := `
		SELECT 
			a.id,
			a.assessment_id,
			a.category_id,
			a.approved_by_user_id,
			COALESCE(u.first_name || ' ' || u.last_name, u.email) as approved_by_name,
			a.approved_at
		FROM consolidation_averaged_approvals a
		LEFT JOIN users u ON a.approved_by_user_id = u.id
		WHERE a.assessment_id = $1 AND a.category_id = $2
		ORDER BY a.approved_at ASC
	`

	rows, err := r.db.Query(query, assessmentID, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []models.ConsolidationAveragedApproval
	for rows.Next() {
		var approval models.ConsolidationAveragedApproval
		err := rows.Scan(
			&approval.ID,
			&approval.AssessmentID,
			&approval.CategoryID,
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

// HasUserApproved checks if a user has approved an averaged response
func (r *ConsolidationAveragedApprovalRepository) HasUserApproved(assessmentID, categoryID, userID uint) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM consolidation_averaged_approvals WHERE assessment_id = $1 AND category_id = $2 AND approved_by_user_id = $3`
	err := r.db.QueryRow(query, assessmentID, categoryID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
