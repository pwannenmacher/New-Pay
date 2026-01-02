package repository

import (
	"database/sql"
	"fmt"
	"new-pay/internal/models"
	"time"
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

// GetCatalogIDsByUserID retrieves all catalog IDs where the user has self-assessments
func (r *SelfAssessmentRepository) GetCatalogIDsByUserID(userID uint) ([]uint, error) {
	query := `SELECT DISTINCT catalog_id FROM self_assessments WHERE user_id = $1`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalogIDs []uint
	for rows.Next() {
		var catalogID uint
		if err := rows.Scan(&catalogID); err != nil {
			return nil, err
		}
		catalogIDs = append(catalogIDs, catalogID)
	}
	return catalogIDs, rows.Err()
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

// GetByStatus retrieves all self-assessments with a specific status
func (r *SelfAssessmentRepository) GetByStatus(status string) ([]models.SelfAssessmentWithDetails, error) {
	query := `
		SELECT 
			sa.id, sa.catalog_id, sa.user_id, sa.status, 
			sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
			sa.reviewed_at, sa.discussion_started_at, sa.archived_at, sa.closed_at, sa.previous_status,
			CONCAT(u.first_name, ' ', u.last_name) as user_name, u.email as user_email,
			c.name as catalog_name
		FROM self_assessments sa
		INNER JOIN users u ON sa.user_id = u.id
		INNER JOIN criteria_catalogs c ON sa.catalog_id = c.id
		WHERE sa.status = $1
		ORDER BY sa.created_at ASC
	`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessmentWithDetails
	for rows.Next() {
		var sa models.SelfAssessmentWithDetails
		err := rows.Scan(
			&sa.ID, &sa.CatalogID, &sa.UserID, &sa.Status,
			&sa.CreatedAt, &sa.UpdatedAt, &sa.SubmittedAt, &sa.InReviewAt,
			&sa.ReviewedAt, &sa.DiscussionStartedAt, &sa.ArchivedAt, &sa.ClosedAt, &sa.PreviousStatus,
			&sa.UserName, &sa.UserEmail, &sa.CatalogName,
		)
		if err != nil {
			return nil, err
		}
		assessments = append(assessments, sa)
	}

	return assessments, nil
}

// GetByCatalogID retrieves all self-assessments for a catalog with user details
func (r *SelfAssessmentRepository) GetByCatalogID(catalogID uint) ([]models.SelfAssessmentWithDetails, error) {
	query := `
		SELECT 
			sa.id, sa.catalog_id, sa.user_id, sa.status, 
			sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
			sa.reviewed_at, sa.discussion_started_at, sa.archived_at, sa.closed_at, sa.previous_status,
			CONCAT(u.first_name, ' ', u.last_name) as user_name, u.email as user_email,
			c.name as catalog_name
		FROM self_assessments sa
		JOIN users u ON sa.user_id = u.id
		JOIN criteria_catalogs c ON sa.catalog_id = c.id
		WHERE sa.catalog_id = $1
		ORDER BY sa.created_at DESC
	`

	rows, err := r.db.Query(query, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessmentWithDetails
	for rows.Next() {
		var assessment models.SelfAssessmentWithDetails
		err := rows.Scan(
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
			&assessment.UserName,
			&assessment.UserEmail,
			&assessment.CatalogName,
		)
		if err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}

	return assessments, rows.Err()
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

// GetByIDWithDetails retrieves a self-assessment with user and catalog details
func (r *SelfAssessmentRepository) GetByIDWithDetails(assessmentID uint) (*models.SelfAssessmentWithDetails, error) {
	var assessment models.SelfAssessmentWithDetails
	query := `
		SELECT 
			sa.id, sa.catalog_id, sa.user_id, sa.status, sa.created_at, sa.updated_at,
			sa.submitted_at, sa.in_review_at, sa.reviewed_at, sa.discussion_started_at,
			sa.archived_at, sa.closed_at, sa.previous_status,
			u.email, u.first_name || ' ' || u.last_name as user_name,
			c.name as catalog_name
		FROM self_assessments sa
		JOIN users u ON sa.user_id = u.id
		JOIN criteria_catalogs c ON sa.catalog_id = c.id
		WHERE sa.id = $1
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
		&assessment.UserEmail,
		&assessment.UserName,
		&assessment.CatalogName,
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

// Delete deletes a self-assessment by ID
func (r *SelfAssessmentRepository) Delete(assessmentID uint) error {
	query := `DELETE FROM self_assessments WHERE id = $1`
	_, err := r.db.Exec(query, assessmentID)
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

// GetByUserIDWithDetails retrieves all self-assessments for a user with catalog details
func (r *SelfAssessmentRepository) GetByUserIDWithDetails(userID uint) ([]models.SelfAssessmentWithDetails, error) {
	query := `
		SELECT sa.id, sa.catalog_id, sa.user_id, sa.status, 
		       sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
		       sa.reviewed_at, sa.discussion_started_at, sa.archived_at, 
		       sa.closed_at, sa.previous_status,
		       c.name as catalog_name
		FROM self_assessments sa
		JOIN criteria_catalogs c ON sa.catalog_id = c.id
		WHERE sa.user_id = $1
		ORDER BY sa.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessmentWithDetails
	for rows.Next() {
		var assessment models.SelfAssessmentWithDetails
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
			&assessment.CatalogName,
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
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}

// HasActiveAssessment checks if a user has any active self-assessment
// Active means status is NOT 'archived' or 'closed'
func (r *SelfAssessmentRepository) HasActiveAssessment(userID uint) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM self_assessments 
		WHERE user_id = $1 
		AND status NOT IN ('archived', 'closed')
	`
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasOpenAssessmentForCatalog checks if a user has any non-closed self-assessment for a specific catalog
// Returns true if there's any assessment with status other than 'closed'
func (r *SelfAssessmentRepository) HasOpenAssessmentForCatalog(catalogID, userID uint) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM self_assessments 
		WHERE catalog_id = $1 AND user_id = $2 
		AND status != 'closed'
	`
	var count int
	err := r.db.QueryRow(query, catalogID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAllWithFilters retrieves all self-assessments with optional filters
func (r *SelfAssessmentRepository) GetAllWithFilters(status, username string, fromDate, toDate *time.Time) ([]models.SelfAssessment, error) {
	query := `
		SELECT sa.id, sa.catalog_id, sa.user_id, sa.status, 
		       sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
		       sa.reviewed_at, sa.discussion_started_at, sa.archived_at, 
		       sa.closed_at, sa.previous_status
		FROM self_assessments sa
		JOIN users u ON sa.user_id = u.id
		WHERE 1=1
	`
	var args []interface{}
	argCount := 1

	if status != "" {
		query += ` AND sa.status = $` + fmt.Sprintf("%d", argCount)
		args = append(args, status)
		argCount++
	}

	if username != "" {
		query += ` AND (u.email ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.first_name ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.last_name ILIKE $` + fmt.Sprintf("%d", argCount) + `)`
		args = append(args, "%"+username+"%")
		argCount++
	}

	if fromDate != nil {
		query += ` AND sa.created_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *fromDate)
		argCount++
	}

	if toDate != nil {
		query += ` AND sa.created_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *toDate)
		argCount++
	}

	query += ` ORDER BY sa.created_at DESC`

	rows, err := r.db.Query(query, args...)
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

// GetAllWithFiltersAndDetails retrieves all self-assessments with filters including user and catalog details
func (r *SelfAssessmentRepository) GetAllWithFiltersAndDetails(status, username string, fromDate, toDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	query := `
		SELECT sa.id, sa.catalog_id, sa.user_id, sa.status, 
		       sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
		       sa.review_consolidation_at, sa.reviewed_at, sa.discussion_started_at, 
		       sa.archived_at, sa.closed_at, sa.previous_status,
		       u.email, u.first_name || ' ' || u.last_name as user_name,
		       c.name as catalog_name
		FROM self_assessments sa
		JOIN users u ON sa.user_id = u.id
		JOIN criteria_catalogs c ON sa.catalog_id = c.id
		WHERE 1=1
	`
	var args []interface{}
	argCount := 1

	if status != "" {
		query += ` AND sa.status = $` + fmt.Sprintf("%d", argCount)
		args = append(args, status)
		argCount++
	}

	if username != "" {
		query += ` AND (u.email ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.first_name ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.last_name ILIKE $` + fmt.Sprintf("%d", argCount) + `)`
		args = append(args, "%"+username+"%")
		argCount++
	}

	if fromDate != nil {
		query += ` AND sa.created_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *fromDate)
		argCount++
	}

	if toDate != nil {
		query += ` AND sa.created_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *toDate)
		argCount++
	}

	query += ` ORDER BY sa.created_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessmentWithDetails
	for rows.Next() {
		var assessment models.SelfAssessmentWithDetails
		if err := rows.Scan(
			&assessment.ID,
			&assessment.CatalogID,
			&assessment.UserID,
			&assessment.Status,
			&assessment.CreatedAt,
			&assessment.UpdatedAt,
			&assessment.SubmittedAt,
			&assessment.InReviewAt,
			&assessment.ReviewConsolidationAt,
			&assessment.ReviewedAt,
			&assessment.DiscussionStartedAt,
			&assessment.ArchivedAt,
			&assessment.ClosedAt,
			&assessment.PreviousStatus,
			&assessment.UserEmail,
			&assessment.UserName,
			&assessment.CatalogName,
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

// GetOpenAssessmentsForReview retrieves open assessments for reviewers with filters
func (r *SelfAssessmentRepository) GetOpenAssessmentsForReview(catalogID *int, username string, status string, fromDate, toDate, fromSubmittedDate, toSubmittedDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	query := `
SELECT sa.id, sa.catalog_id, sa.user_id, sa.status, 
       sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
       sa.review_consolidation_at, sa.reviewed_at, sa.discussion_started_at, sa.archived_at, 
       sa.closed_at, sa.previous_status,
       u.email, u.first_name || ' ' || u.last_name as user_name,
       c.name as catalog_name
FROM self_assessments sa
JOIN users u ON sa.user_id = u.id
JOIN criteria_catalogs c ON sa.catalog_id = c.id
WHERE sa.status IN ('submitted', 'in_review', 'review_consolidation', 'reviewed', 'discussion')
`
	var args []interface{}
	argCount := 1

	if catalogID != nil {
		query += ` AND sa.catalog_id = $` + fmt.Sprintf("%d", argCount)
		args = append(args, *catalogID)
		argCount++
	}

	if username != "" {
		query += ` AND (u.email ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.first_name ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.last_name ILIKE $` + fmt.Sprintf("%d", argCount) + `)`
		args = append(args, "%"+username+"%")
		argCount++
	}

	if status != "" {
		query += ` AND sa.status = $` + fmt.Sprintf("%d", argCount)
		args = append(args, status)
		argCount++
	}

	if fromDate != nil {
		query += ` AND sa.created_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *fromDate)
		argCount++
	}

	if toDate != nil {
		query += ` AND sa.created_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *toDate)
		argCount++
	}

	if fromSubmittedDate != nil {
		query += ` AND sa.submitted_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *fromSubmittedDate)
		argCount++
	}

	if toSubmittedDate != nil {
		query += ` AND sa.submitted_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *toSubmittedDate)
		argCount++
	}

	query += ` ORDER BY sa.submitted_at DESC NULLS LAST, sa.created_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessmentWithDetails
	for rows.Next() {
		var assessment models.SelfAssessmentWithDetails
		if err := rows.Scan(
			&assessment.ID,
			&assessment.CatalogID,
			&assessment.UserID,
			&assessment.Status,
			&assessment.CreatedAt,
			&assessment.UpdatedAt,
			&assessment.SubmittedAt,
			&assessment.InReviewAt,
			&assessment.ReviewConsolidationAt,
			&assessment.ReviewedAt,
			&assessment.DiscussionStartedAt,
			&assessment.ArchivedAt,
			&assessment.ClosedAt,
			&assessment.PreviousStatus,
			&assessment.UserEmail,
			&assessment.UserName,
			&assessment.CatalogName,
		); err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}

// GetCompletedAssessmentsForReview retrieves archived assessments for reviewers with filters
// Admins see all archived assessments, reviewers only see assessments they participated in
func (r *SelfAssessmentRepository) GetCompletedAssessmentsForReview(userID uint, isAdmin bool, catalogID *int, username string, fromDate, toDate, fromSubmittedDate, toSubmittedDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	query := `
SELECT DISTINCT sa.id, sa.catalog_id, sa.user_id, sa.status, 
       sa.created_at, sa.updated_at, sa.submitted_at, sa.in_review_at, 
       sa.review_consolidation_at, sa.reviewed_at, sa.discussion_started_at, sa.archived_at, 
       sa.closed_at, sa.previous_status,
       u.email, u.first_name || ' ' || u.last_name as user_name,
       c.name as catalog_name
FROM self_assessments sa
JOIN users u ON sa.user_id = u.id
JOIN criteria_catalogs c ON sa.catalog_id = c.id
`
	var args []interface{}
	argCount := 1

	// Base condition: only archived assessments
	query += ` WHERE sa.status = 'archived'`

	// If not admin, filter by reviewer participation
	if !isAdmin {
		query += ` AND EXISTS (
			SELECT 1 FROM reviewer_responses rr 
			WHERE rr.assessment_id = sa.id 
			AND rr.reviewer_user_id = $` + fmt.Sprintf("%d", argCount) + `
		)`
		args = append(args, userID)
		argCount++
	}

	if catalogID != nil {
		query += ` AND sa.catalog_id = $` + fmt.Sprintf("%d", argCount)
		args = append(args, *catalogID)
		argCount++
	}

	if username != "" {
		query += ` AND (u.email ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.first_name ILIKE $` + fmt.Sprintf("%d", argCount) +
			` OR u.last_name ILIKE $` + fmt.Sprintf("%d", argCount) + `)`
		args = append(args, "%"+username+"%")
		argCount++
	}

	if fromDate != nil {
		query += ` AND sa.created_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *fromDate)
		argCount++
	}

	if toDate != nil {
		query += ` AND sa.created_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *toDate)
		argCount++
	}

	if fromSubmittedDate != nil {
		query += ` AND sa.submitted_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *fromSubmittedDate)
		argCount++
	}

	if toSubmittedDate != nil {
		query += ` AND sa.submitted_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, *toSubmittedDate)
		argCount++
	}

	query += ` ORDER BY sa.archived_at DESC NULLS LAST, sa.created_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []models.SelfAssessmentWithDetails
	for rows.Next() {
		var assessment models.SelfAssessmentWithDetails
		if err := rows.Scan(
			&assessment.ID,
			&assessment.CatalogID,
			&assessment.UserID,
			&assessment.Status,
			&assessment.CreatedAt,
			&assessment.UpdatedAt,
			&assessment.SubmittedAt,
			&assessment.InReviewAt,
			&assessment.ReviewConsolidationAt,
			&assessment.ReviewedAt,
			&assessment.DiscussionStartedAt,
			&assessment.ArchivedAt,
			&assessment.ClosedAt,
			&assessment.PreviousStatus,
			&assessment.UserEmail,
			&assessment.UserName,
			&assessment.CatalogName,
		); err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}

// UpdateStatus updates the status of a self-assessment
func (r *SelfAssessmentRepository) UpdateStatus(assessmentID uint, newStatus string) error {
	// First, get the current status to store as previous_status
	var currentStatus string
	err := r.db.QueryRow(`SELECT status FROM self_assessments WHERE id = $1`, assessmentID).Scan(&currentStatus)
	if err != nil {
		return err
	}

	// Build the update query dynamically based on the new status
	query := `UPDATE self_assessments SET status = $1, previous_status = $2, updated_at = NOW()`
	args := []interface{}{newStatus, currentStatus, assessmentID}

	// Add status-specific timestamp updates
	switch newStatus {
	case "submitted":
		query += `, submitted_at = NOW()`
	case "in_review":
		query += `, in_review_at = NOW()`
	case "review_consolidation":
		query += `, review_consolidation_at = NOW()`
	case "reviewed":
		query += `, reviewed_at = NOW()`
	case "discussion":
		query += `, discussion_started_at = NOW()`
	case "archived":
		query += `, archived_at = NOW()`
	case "closed":
		query += `, closed_at = NOW()`
	}

	query += ` WHERE id = $3`
	_, err = r.db.Exec(query, args...)
	return err
}
