package repository

import (
	"database/sql"
	"fmt"
	"new-pay/internal/models"
)

// AssessmentResponseRepository handles database operations for assessment responses
type AssessmentResponseRepository struct {
	db *sql.DB
}

// NewAssessmentResponseRepository creates a new assessment response repository
func NewAssessmentResponseRepository(db *sql.DB) *AssessmentResponseRepository {
	return &AssessmentResponseRepository{db: db}
}

// Create creates a new assessment response
func (r *AssessmentResponseRepository) Create(response *models.AssessmentResponse) error {
	query := `
		INSERT INTO assessment_responses (assessment_id, category_id, path_id, level_id, justification)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(
		query,
		response.AssessmentID,
		response.CategoryID,
		response.PathID,
		response.LevelID,
		response.Justification,
	).Scan(&response.ID, &response.CreatedAt, &response.UpdatedAt)

	return err
}

// Update updates an existing assessment response
func (r *AssessmentResponseRepository) Update(response *models.AssessmentResponse) error {
	query := `
		UPDATE assessment_responses
		SET path_id = $1, level_id = $2, justification = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING updated_at
	`
	err := r.db.QueryRow(
		query,
		response.PathID,
		response.LevelID,
		response.Justification,
		response.ID,
	).Scan(&response.UpdatedAt)

	return err
}

// Delete deletes an assessment response
func (r *AssessmentResponseRepository) Delete(responseID uint) error {
	query := `DELETE FROM assessment_responses WHERE id = $1`
	result, err := r.db.Exec(query, responseID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("response not found")
	}

	return nil
}

// GetByID retrieves an assessment response by ID
func (r *AssessmentResponseRepository) GetByID(responseID uint) (*models.AssessmentResponse, error) {
	var response models.AssessmentResponse
	query := `
		SELECT id, assessment_id, category_id, path_id, level_id, justification, 
		       created_at, updated_at
		FROM assessment_responses
		WHERE id = $1
	`
	err := r.db.QueryRow(query, responseID).Scan(
		&response.ID,
		&response.AssessmentID,
		&response.CategoryID,
		&response.PathID,
		&response.LevelID,
		&response.Justification,
		&response.CreatedAt,
		&response.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetByAssessmentAndCategory retrieves a response by assessment and category
func (r *AssessmentResponseRepository) GetByAssessmentAndCategory(assessmentID, categoryID uint) (*models.AssessmentResponse, error) {
	var response models.AssessmentResponse
	query := `
		SELECT id, assessment_id, category_id, path_id, level_id, justification,
		       created_at, updated_at
		FROM assessment_responses
		WHERE assessment_id = $1 AND category_id = $2
	`
	err := r.db.QueryRow(query, assessmentID, categoryID).Scan(
		&response.ID,
		&response.AssessmentID,
		&response.CategoryID,
		&response.PathID,
		&response.LevelID,
		&response.Justification,
		&response.CreatedAt,
		&response.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetAllByAssessment retrieves all responses for an assessment with details
func (r *AssessmentResponseRepository) GetAllByAssessment(assessmentID uint) ([]models.AssessmentResponseWithDetails, error) {
	query := `
		SELECT 
			ar.id, ar.assessment_id, ar.category_id, ar.path_id, ar.level_id, 
			ar.justification, ar.created_at, ar.updated_at,
			c.name as category_name, c.sort_order as category_sort_order,
			p.name as path_name, p.description as path_description,
			l.name as level_name, l.level_number, l.description as level_description,
			pld.description as path_level_description
		FROM assessment_responses ar
		JOIN categories c ON ar.category_id = c.id
		JOIN paths p ON ar.path_id = p.id
		JOIN levels l ON ar.level_id = l.id
		LEFT JOIN path_level_descriptions pld ON pld.path_id = ar.path_id AND pld.level_id = ar.level_id
		WHERE ar.assessment_id = $1
		ORDER BY c.sort_order, c.name
	`

	rows, err := r.db.Query(query, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []models.AssessmentResponseWithDetails
	for rows.Next() {
		var response models.AssessmentResponseWithDetails
		err := rows.Scan(
			&response.ID,
			&response.AssessmentID,
			&response.CategoryID,
			&response.PathID,
			&response.LevelID,
			&response.Justification,
			&response.CreatedAt,
			&response.UpdatedAt,
			&response.CategoryName,
			&response.CategorySortOrder,
			&response.PathName,
			&response.PathDescription,
			&response.LevelName,
			&response.LevelNumber,
			&response.LevelDescription,
			&response.PathLevelDescription,
		)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return responses, nil
}

// CountByAssessment counts responses for an assessment
func (r *AssessmentResponseRepository) CountByAssessment(assessmentID uint) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM assessment_responses WHERE assessment_id = $1`
	err := r.db.QueryRow(query, assessmentID).Scan(&count)
	return count, err
}

// GetCompleteness calculates the completeness status of an assessment
func (r *AssessmentResponseRepository) GetCompleteness(assessmentID uint) (*models.AssessmentCompleteness, error) {
	// First, get the catalog ID from the assessment
	var catalogID uint
	err := r.db.QueryRow(`
		SELECT catalog_id FROM self_assessments WHERE id = $1
	`, assessmentID).Scan(&catalogID)
	if err != nil {
		return nil, err
	}

	// Count total categories in the catalog
	var totalCategories int
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM categories WHERE catalog_id = $1
	`, catalogID).Scan(&totalCategories)
	if err != nil {
		return nil, err
	}

	// Count completed categories (categories with responses)
	completedCategories, err := r.CountByAssessment(assessmentID)
	if err != nil {
		return nil, err
	}

	// Calculate percentage
	var percentComplete float64
	if totalCategories > 0 {
		percentComplete = float64(completedCategories) / float64(totalCategories) * 100
	}

	isComplete := completedCategories == totalCategories && totalCategories > 0

	// Get missing category IDs
	var missingCategories []uint
	if !isComplete {
		query := `
			SELECT c.id 
			FROM categories c
			WHERE c.catalog_id = $1
			AND NOT EXISTS (
				SELECT 1 FROM assessment_responses ar 
				WHERE ar.assessment_id = $2 AND ar.category_id = c.id
			)
			ORDER BY c.sort_order, c.name
		`
		rows, err := r.db.Query(query, catalogID, assessmentID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var categoryID uint
			if err := rows.Scan(&categoryID); err != nil {
				return nil, err
			}
			missingCategories = append(missingCategories, categoryID)
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}
	}

	return &models.AssessmentCompleteness{
		TotalCategories:     totalCategories,
		CompletedCategories: completedCategories,
		PercentComplete:     percentComplete,
		IsComplete:          isComplete,
		MissingCategories:   missingCategories,
	}, nil
}

// ValidatePathBelongsToCategory validates that a path belongs to the given category
func (r *AssessmentResponseRepository) ValidatePathBelongsToCategory(pathID, categoryID uint) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM paths WHERE id = $1 AND category_id = $2`
	err := r.db.QueryRow(query, pathID, categoryID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ValidateLevelBelongsToCatalog validates that a level belongs to the catalog
func (r *AssessmentResponseRepository) ValidateLevelBelongsToCatalog(levelID, catalogID uint) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM levels WHERE id = $1 AND catalog_id = $2`
	err := r.db.QueryRow(query, levelID, catalogID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
