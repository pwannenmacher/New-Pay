package repository

import (
	"database/sql"
	"new-pay/internal/models"
)

type ConsolidationOverrideApprovalRepository struct {
	db *sql.DB
}

func NewConsolidationOverrideApprovalRepository(db *sql.DB) *ConsolidationOverrideApprovalRepository {
	return &ConsolidationOverrideApprovalRepository{db: db}
}

// CreateApproval creates a new approval for an override
func (r *ConsolidationOverrideApprovalRepository) CreateApproval(overrideID, userID uint) error {
	query := `
		INSERT INTO consolidation_override_approvals (override_id, approved_by_user_id)
		VALUES ($1, $2)
		ON CONFLICT (override_id, approved_by_user_id) DO NOTHING
	`
	_, err := r.db.Exec(query, overrideID, userID)
	return err
}

// DeleteApproval removes an approval
func (r *ConsolidationOverrideApprovalRepository) DeleteApproval(overrideID, userID uint) error {
	query := `DELETE FROM consolidation_override_approvals WHERE override_id = $1 AND approved_by_user_id = $2`
	_, err := r.db.Exec(query, overrideID, userID)
	return err
}

// DeleteAllApprovalsForOverride removes all approvals for an override (used when override is edited)
func (r *ConsolidationOverrideApprovalRepository) DeleteAllApprovalsForOverride(overrideID uint) error {
	query := `DELETE FROM consolidation_override_approvals WHERE override_id = $1`
	_, err := r.db.Exec(query, overrideID)
	return err
}

// GetApprovalsByOverride gets all approvals for an override with user names
func (r *ConsolidationOverrideApprovalRepository) GetApprovalsByOverride(overrideID uint) ([]models.ConsolidationOverrideApproval, error) {
	query := `
		SELECT 
			a.id,
			a.override_id,
			a.approved_by_user_id,
			COALESCE(u.first_name || ' ' || u.last_name, u.email) as approved_by_name,
			a.approved_at
		FROM consolidation_override_approvals a
		LEFT JOIN users u ON a.approved_by_user_id = u.id
		WHERE a.override_id = $1
		ORDER BY a.approved_at ASC
	`

	rows, err := r.db.Query(query, overrideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []models.ConsolidationOverrideApproval
	for rows.Next() {
		var approval models.ConsolidationOverrideApproval
		err := rows.Scan(
			&approval.ID,
			&approval.OverrideID,
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

// HasUserApproved checks if a user has approved an override
func (r *ConsolidationOverrideApprovalRepository) HasUserApproved(overrideID, userID uint) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM consolidation_override_approvals WHERE override_id = $1 AND approved_by_user_id = $2`
	err := r.db.QueryRow(query, overrideID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
