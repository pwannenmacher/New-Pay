package service

import (
	"fmt"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"time"
)

// SelfAssessmentService handles business logic for self-assessments
type SelfAssessmentService struct {
	selfAssessmentRepo *repository.SelfAssessmentRepository
	catalogRepo        *repository.CatalogRepository
	auditRepo          *repository.AuditRepository
	responseRepo       *repository.AssessmentResponseRepository
}

// NewSelfAssessmentService creates a new self-assessment service
func NewSelfAssessmentService(
	selfAssessmentRepo *repository.SelfAssessmentRepository,
	catalogRepo *repository.CatalogRepository,
	auditRepo *repository.AuditRepository,
	responseRepo *repository.AssessmentResponseRepository,
) *SelfAssessmentService {
	return &SelfAssessmentService{
		selfAssessmentRepo: selfAssessmentRepo,
		catalogRepo:        catalogRepo,
		auditRepo:          auditRepo,
		responseRepo:       responseRepo,
	}
}

// CreateSelfAssessment creates a new self-assessment in draft status
func (s *SelfAssessmentService) CreateSelfAssessment(catalogID uint, userID uint) (*models.SelfAssessment, error) {
	// Verify catalog exists and is in active phase
	catalog, err := s.catalogRepo.GetCatalogByID(catalogID)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}
	if catalog.Phase != "active" {
		return nil, fmt.Errorf("can only create self-assessments for active catalogs")
	}

	// Check if catalog is valid for current date
	now := time.Now()
	if now.Before(catalog.ValidFrom) || now.After(catalog.ValidUntil) {
		return nil, fmt.Errorf("catalog is not valid for current date")
	}

	// Check if user already has a non-closed self-assessment for this catalog
	// Multiple assessments for the same catalog are allowed, but only if all previous ones are closed
	hasOpen, err := s.selfAssessmentRepo.HasOpenAssessmentForCatalog(catalogID, userID)
	if err != nil {
		return nil, err
	}
	if hasOpen {
		return nil, fmt.Errorf("self-assessment already exists for this catalog")
	}

	// Check if user has any active self-assessment (not archived or closed)
	hasActive, err := s.selfAssessmentRepo.HasActiveAssessment(userID)
	if err != nil {
		return nil, err
	}
	if hasActive {
		return nil, fmt.Errorf("user already has an active self-assessment in progress")
	}

	// Create new assessment
	assessment := &models.SelfAssessment{
		CatalogID: catalogID,
		UserID:    userID,
		Status:    "draft",
	}

	if err := s.selfAssessmentRepo.Create(assessment); err != nil {
		return nil, err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "create",
		Resource: "self_assessment",
		Details:  fmt.Sprintf("Created self-assessment (ID: %d) for catalog %d", assessment.ID, catalogID),
	})

	return assessment, nil
}

// GetSelfAssessment retrieves a self-assessment by ID with permission checks
func (s *SelfAssessmentService) GetSelfAssessment(assessmentID uint, userID uint, userRoles []string) (*models.SelfAssessment, error) {
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, fmt.Errorf("self-assessment not found")
	}

	// Permission checks
	isOwner := assessment.UserID == userID
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	// Owners can always see their own assessments
	if isOwner {
		return assessment, nil
	}

	// Admins can only see metadata (status, dates) - content filtered in handler
	if isAdmin {
		return assessment, nil
	}

	// Reviewers can only see submitted or later assessments
	if isReviewer && assessment.Status != "draft" && assessment.Status != "closed" {
		return assessment, nil
	}

	return nil, fmt.Errorf("permission denied: cannot view this self-assessment")
}

// GetSelfAssessmentWithDetails retrieves a self-assessment with user and catalog details
func (s *SelfAssessmentService) GetSelfAssessmentWithDetails(assessmentID uint, userID uint, userRoles []string) (*models.SelfAssessmentWithDetails, error) {
	assessment, err := s.selfAssessmentRepo.GetByIDWithDetails(assessmentID)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, fmt.Errorf("self-assessment not found")
	}

	// Permission checks
	isOwner := assessment.UserID == userID
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	// Owners can always see their own assessments
	if isOwner {
		return assessment, nil
	}

	// Admins can see all assessments with details
	if isAdmin {
		return assessment, nil
	}

	// Reviewers can only see submitted or later assessments
	if isReviewer && assessment.Status != "draft" && assessment.Status != "closed" {
		return assessment, nil
	}

	return nil, fmt.Errorf("permission denied: cannot view this self-assessment")
}

// GetUserSelfAssessments retrieves all self-assessments for a user
func (s *SelfAssessmentService) GetUserSelfAssessments(userID uint) ([]models.SelfAssessmentWithDetails, error) {
	return s.selfAssessmentRepo.GetByUserIDWithDetails(userID)
}

// GetActiveCatalogs retrieves catalogs that are active and valid for current date
// Returns at most one catalog (only one catalog can be active at a time)
func (s *SelfAssessmentService) GetActiveCatalogs() ([]models.CriteriaCatalog, error) {
	catalogs, err := s.catalogRepo.GetCatalogsByPhase("active")
	if err != nil {
		return nil, err
	}

	// Filter by current date validity
	now := time.Now()
	var validCatalogs []models.CriteriaCatalog
	for _, catalog := range catalogs {
		if !now.Before(catalog.ValidFrom) && !now.After(catalog.ValidUntil) {
			validCatalogs = append(validCatalogs, catalog)
		}
	}

	return validCatalogs, nil
}

// GetVisibleSelfAssessments retrieves self-assessments visible to a user based on role
func (s *SelfAssessmentService) GetVisibleSelfAssessments(userID uint, userRoles []string) ([]models.SelfAssessment, error) {
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	if isAdmin {
		// Admins see all metadata
		return s.selfAssessmentRepo.GetAllMetadata()
	}

	if isReviewer {
		// Reviewers see submitted and later assessments
		return s.selfAssessmentRepo.GetVisibleToReviewers()
	}

	// Regular users see only their own
	return s.selfAssessmentRepo.GetByUserID(userID)
}

// GetAllSelfAssessmentsWithFilters retrieves all self-assessments with optional filters (admin only)
func (s *SelfAssessmentService) GetAllSelfAssessmentsWithFilters(status, username string, fromDate, toDate *time.Time) ([]models.SelfAssessment, error) {
	return s.selfAssessmentRepo.GetAllWithFilters(status, username, fromDate, toDate)
}

// GetAllSelfAssessmentsWithFiltersAndDetails retrieves all self-assessments with optional filters including details (admin only)
func (s *SelfAssessmentService) GetAllSelfAssessmentsWithFiltersAndDetails(status, username string, fromDate, toDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	return s.selfAssessmentRepo.GetAllWithFiltersAndDetails(status, username, fromDate, toDate)
}

// UpdateSelfAssessmentStatus transitions a self-assessment to a new status
func (s *SelfAssessmentService) UpdateSelfAssessmentStatus(assessmentID uint, newStatus string, userID uint, userRoles []string) error {
	// Get existing assessment
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return err
	}
	if assessment == nil {
		return fmt.Errorf("self-assessment not found")
	}

	oldStatus := assessment.Status
	isOwner := assessment.UserID == userID
	isAdmin := contains(userRoles, "admin")

	// Check ownership for draft/submitted transitions
	// Only owners can submit their assessments
	if newStatus == "submitted" && !isOwner {
		return fmt.Errorf("permission denied: only the owner can submit their self-assessment")
	}

	// Admins can only close assessments, not submit them
	if isAdmin && !isOwner && newStatus != "closed" {
		return fmt.Errorf("permission denied: admins can only close other users' self-assessments")
	}

	// Validate status transitions
	if err := s.validateStatusTransition(oldStatus, newStatus, userRoles, isOwner); err != nil {
		return err
	}

	// Handle closed status reversal within 24h
	if oldStatus == "closed" && assessment.ClosedAt != nil {
		if time.Since(*assessment.ClosedAt) > 24*time.Hour {
			return fmt.Errorf("cannot revert closed status after 24 hours")
		}
		if assessment.PreviousStatus != nil {
			newStatus = *assessment.PreviousStatus
		}
	}

	// Update timestamps based on new status
	now := time.Now()
	switch newStatus {
	case "submitted":
		assessment.SubmittedAt = &now
	case "in_review":
		assessment.InReviewAt = &now
	case "reviewed":
		assessment.ReviewedAt = &now
	case "discussion":
		assessment.DiscussionStartedAt = &now
	case "archived":
		assessment.ArchivedAt = &now
	case "closed":
		assessment.ClosedAt = &now
		assessment.PreviousStatus = &oldStatus
	}

	assessment.Status = newStatus

	// Update assessment
	if err := s.selfAssessmentRepo.Update(assessment); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "update_status",
		Resource: "self_assessment",
		Details:  fmt.Sprintf("Self-assessment %d status changed: %s -> %s", assessmentID, oldStatus, newStatus),
	})

	return nil
}

// validateStatusTransition validates if a status transition is allowed
func (s *SelfAssessmentService) validateStatusTransition(fromStatus, toStatus string, userRoles []string, isOwner bool) error {
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	// Define allowed transitions
	allowedTransitions := map[string][]string{
		"draft":      {"submitted", "closed"},
		"submitted":  {"in_review", "closed"},
		"in_review":  {"reviewed", "closed"},
		"reviewed":   {"discussion", "closed"},
		"discussion": {"archived", "closed"},
		"archived":   {},                                                            // Final state
		"closed":     {"draft", "submitted", "in_review", "reviewed", "discussion"}, // Can revert within 24h
	}

	// Check if user has permission for this transition
	if toStatus == "in_review" || toStatus == "reviewed" || toStatus == "discussion" || toStatus == "archived" {
		if !isReviewer && !isAdmin {
			return fmt.Errorf("permission denied: only reviewers can transition to %s status", toStatus)
		}
	}

	// Owners can submit or close their own assessments
	// Admins can only close (not submit) other users' assessments
	if toStatus == "submitted" && !isOwner {
		return fmt.Errorf("permission denied: only the owner can submit their self-assessment")
	}

	allowed, ok := allowedTransitions[fromStatus]
	if !ok {
		return fmt.Errorf("invalid current status: %s", fromStatus)
	}

	for _, validStatus := range allowed {
		if toStatus == validStatus {
			return nil
		}
	}

	return fmt.Errorf("cannot transition from %s to %s status", fromStatus, toStatus)
}

// DeleteSelfAssessment deletes a self-assessment (admin only, only if closed without submission)
func (s *SelfAssessmentService) DeleteSelfAssessment(assessmentID uint, userID uint, userRoles []string) error {
	isAdmin := contains(userRoles, "admin")
	if !isAdmin {
		return fmt.Errorf("permission denied: only admins can delete self-assessments")
	}

	// Get existing assessment
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return err
	}
	if assessment == nil {
		return fmt.Errorf("self-assessment not found")
	}

	// Can only delete if closed and never submitted
	if assessment.Status != "closed" {
		return fmt.Errorf("can only delete closed self-assessments")
	}
	if assessment.SubmittedAt != nil {
		return fmt.Errorf("cannot delete self-assessment that was submitted")
	}

	// Delete assessment
	if err := s.selfAssessmentRepo.Delete(assessmentID); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "delete",
		Resource: "self_assessment",
		Details:  fmt.Sprintf("Deleted self-assessment %d (catalog: %d, user: %d)", assessmentID, assessment.CatalogID, assessment.UserID),
	})

	return nil
}

// SaveResponse saves or updates an assessment response
func (s *SelfAssessmentService) SaveResponse(userID, assessmentID uint, response *models.AssessmentResponse) (*models.AssessmentResponse, error) {
	// Get assessment and verify ownership and status
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, fmt.Errorf("assessment not found")
	}
	if assessment.UserID != userID {
		return nil, fmt.Errorf("permission denied: not owner of assessment")
	}
	if assessment.Status != "draft" {
		return nil, fmt.Errorf("can only edit responses in draft status")
	}

	// Get catalog ID
	catalog, err := s.catalogRepo.GetCatalogByID(assessment.CatalogID)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}

	// Validate justification length
	if len(response.Justification) < 150 {
		return nil, fmt.Errorf("justification must be at least 150 characters")
	}

	// Validate path belongs to category
	valid, err := s.responseRepo.ValidatePathBelongsToCategory(response.PathID, response.CategoryID)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("path does not belong to the specified category")
	}

	// Validate level belongs to catalog
	valid, err = s.responseRepo.ValidateLevelBelongsToCatalog(response.LevelID, catalog.ID)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("level does not belong to the catalog")
	}

	// Check if response already exists for this category
	existing, err := s.responseRepo.GetByAssessmentAndCategory(assessmentID, response.CategoryID)
	if err != nil {
		return nil, err
	}

	response.AssessmentID = assessmentID

	if existing != nil {
		// Update existing response
		response.ID = existing.ID
		response.CreatedAt = existing.CreatedAt
		if err := s.responseRepo.Update(response); err != nil {
			return nil, err
		}

		// Audit log
		s.auditRepo.Create(&models.AuditLog{
			UserID:   &userID,
			Action:   "update",
			Resource: "assessment_response",
			Details:  fmt.Sprintf("Updated response for assessment %d, category %d", assessmentID, response.CategoryID),
		})
	} else {
		// Create new response
		if err := s.responseRepo.Create(response); err != nil {
			return nil, err
		}

		// Audit log
		s.auditRepo.Create(&models.AuditLog{
			UserID:   &userID,
			Action:   "create",
			Resource: "assessment_response",
			Details:  fmt.Sprintf("Created response for assessment %d, category %d", assessmentID, response.CategoryID),
		})
	}

	return response, nil
}

// DeleteResponse deletes an assessment response
func (s *SelfAssessmentService) DeleteResponse(userID, assessmentID, categoryID uint) error {
	// Get assessment and verify ownership and status
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return err
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}
	if assessment.UserID != userID {
		return fmt.Errorf("permission denied: not owner of assessment")
	}
	if assessment.Status != "draft" {
		return fmt.Errorf("can only delete responses in draft status")
	}

	// Get response
	response, err := s.responseRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("response not found")
	}

	// Delete response
	if err := s.responseRepo.Delete(response.ID); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "delete",
		Resource: "assessment_response",
		Details:  fmt.Sprintf("Deleted response for assessment %d, category %d", assessmentID, categoryID),
	})

	return nil
}

// GetResponses retrieves all responses for an assessment
func (s *SelfAssessmentService) GetResponses(userID uint, assessmentID uint, userRoles []string) ([]models.AssessmentResponseWithDetails, error) {
	// Get assessment
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, fmt.Errorf("assessment not found")
	}

	// Check permission: owner, admin, or reviewer (for submitted/later status)
	isOwner := assessment.UserID == userID
	isAdminOrReviewer := false
	for _, role := range userRoles {
		if role == "admin" || role == "reviewer" {
			isAdminOrReviewer = true
			break
		}
	}

	if !isOwner && !isAdminOrReviewer {
		return nil, fmt.Errorf("permission denied")
	}

	if !isOwner && assessment.Status == "draft" {
		return nil, fmt.Errorf("permission denied: cannot view draft responses of other users")
	}

	// Get all responses
	responses, err := s.responseRepo.GetAllByAssessment(assessmentID)
	if err != nil {
		return nil, err
	}

	return responses, nil
}

// GetCompleteness calculates the completeness of an assessment
func (s *SelfAssessmentService) GetCompleteness(userID uint, assessmentID uint) (*models.AssessmentCompleteness, error) {
	// Get assessment and verify ownership
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, fmt.Errorf("assessment not found")
	}
	if assessment.UserID != userID {
		return nil, fmt.Errorf("permission denied: not owner of assessment")
	}

	// Get completeness from repository
	completeness, err := s.responseRepo.GetCompleteness(assessmentID)
	if err != nil {
		return nil, err
	}

	return completeness, nil
}

// SubmitAssessment submits an assessment for review (changes status from draft to submitted)
func (s *SelfAssessmentService) SubmitAssessment(userID, assessmentID uint) error {
	// Get assessment and verify ownership and status
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return err
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}
	if assessment.UserID != userID {
		return fmt.Errorf("permission denied: not owner of assessment")
	}
	if assessment.Status != "draft" {
		return fmt.Errorf("can only submit assessments in draft status")
	}

	// Check completeness
	completeness, err := s.responseRepo.GetCompleteness(assessmentID)
	if err != nil {
		return err
	}
	if !completeness.IsComplete {
		return fmt.Errorf("assessment is incomplete: %d of %d categories completed",
			completeness.CompletedCategories, completeness.TotalCategories)
	}

	// Update status to submitted
	now := time.Now()
	assessment.Status = "submitted"
	assessment.SubmittedAt = &now

	if err := s.selfAssessmentRepo.Update(assessment); err != nil {
		return err
	}

	// Audit log
	s.auditRepo.Create(&models.AuditLog{
		UserID:   &userID,
		Action:   "submit",
		Resource: "self_assessment",
		Details:  fmt.Sprintf("Submitted self-assessment %d for review", assessmentID),
	})

	return nil
}
