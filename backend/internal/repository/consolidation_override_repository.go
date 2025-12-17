package repository

import (
	"database/sql"
	"new-pay/internal/models"
)

// ConsolidationOverrideRepository handles database operations for consolidation overrides
type ConsolidationOverrideRepository struct {
	db *sql.DB
}

// NewConsolidationOverrideRepository creates a new consolidation override repository
func NewConsolidationOverrideRepository(db *sql.DB) *ConsolidationOverrideRepository {
	return &ConsolidationOverrideRepository{db: db}
}

// CreateOrUpdate creates or updates a consolidation override
func (r *ConsolidationOverrideRepository) CreateOrUpdate(override *models.ConsolidationOverride) error {
	query := `
		INSERT INTO consolidation_overrides (
			assessment_id, category_id, path_id, level_id, encrypted_justification_id, 
			created_by_user_id, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (assessment_id, category_id)
		DO UPDATE SET
			path_id = EXCLUDED.path_id,
			level_id = EXCLUDED.level_id,
			encrypted_justification_id = EXCLUDED.encrypted_justification_id,
			created_by_user_id = EXCLUDED.created_by_user_id,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		override.AssessmentID,
		override.CategoryID,
		override.PathID,
		override.LevelID,
		override.EncryptedJustificationID,
		override.CreatedByUserID,
	).Scan(&override.ID, &override.CreatedAt, &override.UpdatedAt)
}

// GetByAssessment retrieves all consolidation overrides for an assessment
func (r *ConsolidationOverrideRepository) GetByAssessment(assessmentID uint) ([]models.ConsolidationOverride, error) {
	query := `
		SELECT id, assessment_id, category_id, path_id, level_id, 
		       encrypted_justification_id, created_by_user_id, created_at, updated_at
		FROM consolidation_overrides
		WHERE assessment_id = $1
		ORDER BY category_id
	`

	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overrides := []models.ConsolidationOverride{}
	for rows.Next() {
		var override models.ConsolidationOverride
		err := rows.Scan(
			&override.ID,
			&override.AssessmentID,
			&override.CategoryID,
			&override.PathID,
			&override.LevelID,
			&override.EncryptedJustificationID,
			&override.CreatedByUserID,
			&override.CreatedAt,
			&override.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}

	return overrides, rows.Err()
}

// GetByAssessmentAndCategory retrieves a single override by assessment and category
func (r *ConsolidationOverrideRepository) GetByAssessmentAndCategory(assessmentID, categoryID uint) (*models.ConsolidationOverride, error) {
	query := `
		SELECT id, assessment_id, category_id, path_id, level_id, 
		       encrypted_justification_id, created_by_user_id, created_at, updated_at
		FROM consolidation_overrides
		WHERE assessment_id = $1 AND category_id = $2
	`

	var override models.ConsolidationOverride
	err := r.db.QueryRow(query, assessmentID, categoryID).Scan(
		&override.ID,
		&override.AssessmentID,
		&override.CategoryID,
		&override.PathID,
		&override.LevelID,
		&override.EncryptedJustificationID,
		&override.CreatedByUserID,
		&override.CreatedAt,
		&override.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &override, nil
}

// Delete deletes a consolidation override by ID
func (r *ConsolidationOverrideRepository) Delete(id uint) error {
	query := `DELETE FROM consolidation_overrides WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteByAssessmentAndCategory deletes a consolidation override by assessment and category
func (r *ConsolidationOverrideRepository) DeleteByAssessmentAndCategory(assessmentID, categoryID uint) error {
	query := `
		DELETE FROM consolidation_overrides 
		WHERE assessment_id = $1 AND category_id = $2
	`

	result, err := r.db.Exec(query, assessmentID, categoryID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
