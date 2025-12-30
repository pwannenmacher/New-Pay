package service

import (
	"fmt"
	"log/slog"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"time"
)

// SelfAssessmentService handles business logic for self-assessments
type SelfAssessmentService struct {
	selfAssessmentRepo   *repository.SelfAssessmentRepository
	catalogRepo          *repository.CatalogRepository
	auditSvc             *AuditService
	responseRepo         *repository.AssessmentResponseRepository
	encryptedResponseSvc *EncryptedResponseService
	reviewerRepo         *repository.ReviewerResponseRepository
}

// NewSelfAssessmentService creates a new self-assessment service
func NewSelfAssessmentService(
	selfAssessmentRepo *repository.SelfAssessmentRepository,
	catalogRepo *repository.CatalogRepository,
	auditSvc *AuditService,
	responseRepo *repository.AssessmentResponseRepository,
	encryptedResponseSvc *EncryptedResponseService,
	reviewerRepo *repository.ReviewerResponseRepository,
) *SelfAssessmentService {
	return &SelfAssessmentService{
		selfAssessmentRepo:   selfAssessmentRepo,
		catalogRepo:          catalogRepo,
		auditSvc:             auditSvc,
		responseRepo:         responseRepo,
		encryptedResponseSvc: encryptedResponseSvc,
		reviewerRepo:         reviewerRepo,
	}
}

// Helper functions

// checkPermissionForAssessment checks if user can view an assessment
func (s *SelfAssessmentService) checkPermissionForAssessment(assessment *models.SelfAssessment, userID uint, userRoles []string) error {
	isOwner := assessment.UserID == userID
	isReviewer := contains(userRoles, "reviewer")
	isAdmin := contains(userRoles, "admin")

	// Owners can always see their own assessments
	if isOwner {
		return nil
	}

	// CRITICAL: Admins without reviewer role cannot view assessments
	if isAdmin && !isReviewer {
		return fmt.Errorf("permission denied: admins cannot view assessment details")
	}

	// Reviewers can see submitted/in_review/review_consolidation/reviewed/discussion assessments
	if isReviewer && assessment.Status != "draft" && assessment.Status != "closed" && assessment.Status != "archived" {
		return nil
	}

	return fmt.Errorf("permission denied: cannot view this self-assessment")
}

// getAssessmentAndCheckOwnership loads an assessment and verifies ownership
func (s *SelfAssessmentService) getAssessmentAndCheckOwnership(assessmentID, userID uint) (*models.SelfAssessment, error) {
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
	return assessment, nil
}

// addReviewStatsToAssessments adds review statistics to each assessment in the list
func (s *SelfAssessmentService) addReviewStatsToAssessments(assessments []models.SelfAssessmentWithDetails) {
	if s.reviewerRepo == nil {
		return
	}
	for i := range assessments {
		started, completed, err := s.reviewerRepo.GetReviewStats(assessments[i].ID)
		if err != nil {
			slog.Error("Failed to get review stats", "error", err, "assessment_id", assessments[i].ID)
			continue
		}
		assessments[i].ReviewsStarted = started
		assessments[i].ReviewsCompleted = completed
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
	if now.Before(catalog.ValidFrom) || !now.Before(catalog.ValidUntil.Add(24*time.Hour)) {
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
	s.auditSvc.Log(userID, "create", "self_assessment",
		fmt.Sprintf("Created self-assessment (ID: %d) for catalog %d", assessment.ID, catalogID))

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
	if err := s.checkPermissionForAssessment(assessment, userID, userRoles); err != nil {
		return nil, err
	}

	return assessment, nil
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

	// Permission checks - convert to SelfAssessment for permission check
	assessmentBase := &models.SelfAssessment{
		ID:     assessment.ID,
		UserID: assessment.UserID,
		Status: assessment.Status,
	}
	if err := s.checkPermissionForAssessment(assessmentBase, userID, userRoles); err != nil {
		return nil, err
	}

	return assessment, nil
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
		if !now.Before(catalog.ValidFrom) && now.Before(catalog.ValidUntil.Add(24*time.Hour)) {
			validCatalogs = append(validCatalogs, catalog)
		}
	}

	return validCatalogs, nil
}

// GetAllSelfAssessmentsWithFilters retrieves all self-assessments with optional filters (admin only)
func (s *SelfAssessmentService) GetAllSelfAssessmentsWithFilters(status, username string, fromDate, toDate *time.Time) ([]models.SelfAssessment, error) {
	return s.selfAssessmentRepo.GetAllWithFilters(status, username, fromDate, toDate)
}

// GetAllSelfAssessmentsWithFiltersAndDetails retrieves all self-assessments with optional filters including details (admin only)
func (s *SelfAssessmentService) GetAllSelfAssessmentsWithFiltersAndDetails(status, username string, fromDate, toDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	return s.selfAssessmentRepo.GetAllWithFiltersAndDetails(status, username, fromDate, toDate)
}

// GetOpenAssessmentsForReview retrieves open assessments for reviewers with filters
// Filters out assessments in review_consolidation or later phases if reviewer has not completed their review
func (s *SelfAssessmentService) GetOpenAssessmentsForReview(userID uint, isAdmin bool, catalogID *int, username string, status string, fromDate, toDate, fromSubmittedDate, toSubmittedDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	assessments, err := s.selfAssessmentRepo.GetOpenAssessmentsForReview(catalogID, username, status, fromDate, toDate, fromSubmittedDate, toSubmittedDate)
	if err != nil {
		return nil, err
	}

	// Filter assessments based on review completion (only for non-admin reviewers)
	if !isAdmin {
		assessments, err = s.filterAssessmentsByReviewCompletion(assessments, userID)
		if err != nil {
			return nil, err
		}
	}

	// Add review statistics to each assessment
	s.addReviewStatsToAssessments(assessments)

	return assessments, nil
}

// GetCompletedAssessmentsForReview retrieves archived assessments for reviewers with filters
// Filters out assessments in review_consolidation or later phases if reviewer has not completed their review
func (s *SelfAssessmentService) GetCompletedAssessmentsForReview(userID uint, isAdmin bool, catalogID *int, username string, fromDate, toDate, fromSubmittedDate, toSubmittedDate *time.Time) ([]models.SelfAssessmentWithDetails, error) {
	assessments, err := s.selfAssessmentRepo.GetCompletedAssessmentsForReview(userID, isAdmin, catalogID, username, fromDate, toDate, fromSubmittedDate, toSubmittedDate)
	if err != nil {
		return nil, err
	}

	// Filter assessments based on review completion (only for non-admin reviewers)
	if !isAdmin {
		assessments, err = s.filterAssessmentsByReviewCompletion(assessments, userID)
		if err != nil {
			return nil, err
		}
	}

	// Add review statistics to each assessment
	s.addReviewStatsToAssessments(assessments)

	return assessments, nil
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
	isReviewer := contains(userRoles, "reviewer")

	// Check ownership for draft/submitted transitions
	// Only owners can submit their assessments
	if newStatus == "submitted" && !isOwner {
		return fmt.Errorf("permission denied: only the owner can submit their self-assessment")
	}

	// Users can close their own draft self-assessments
	// Users cannot close self-assessments that have been submitted
	if newStatus == "closed" && isOwner && !isAdmin {
		if oldStatus != "draft" {
			return fmt.Errorf("permission denied: users can only close their own self-assessments in draft status")
		}
	}

	// Admins (without reviewer role) can close assessments or reopen closed assessments (within 24h)
	// Admins cannot submit assessments for other users or perform review-related status changes
	if isAdmin && !isOwner && !isReviewer {
		if newStatus == "submitted" {
			return fmt.Errorf("permission denied: admins cannot submit self-assessments for other users")
		}
		// Allow reopening closed assessments
		if oldStatus == "closed" && newStatus != "closed" {
			// This is allowed - admin is reopening a closed assessment
		} else if oldStatus != "closed" && newStatus != "closed" {
			// Admin trying to change status from non-closed to non-closed (not allowed, except closing)
			return fmt.Errorf("permission denied: admins can only close or reopen self-assessments")
		}
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
	case "review_consolidation":
		assessment.ReviewConsolidationAt = &now
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
	s.auditSvc.Log(userID, "update_status", "self_assessment",
		fmt.Sprintf("Self-assessment %d status changed: %s -> %s", assessmentID, oldStatus, newStatus))

	return nil
}

// validateStatusTransition validates if a status transition is allowed
func (s *SelfAssessmentService) validateStatusTransition(fromStatus, toStatus string, userRoles []string, isOwner bool) error {
	isReviewer := contains(userRoles, "reviewer")

	// Define allowed transitions
	allowedTransitions := map[string][]string{
		"draft":                {"submitted", "closed"},
		"submitted":            {"in_review", "closed"},
		"in_review":            {"review_consolidation", "reviewed", "closed"},
		"review_consolidation": {"in_review", "reviewed", "closed"},
		"reviewed":             {"discussion", "closed"},
		"discussion":           {"archived", "closed"},
		"archived":             {},                                                                                    // Final state
		"closed":               {"draft", "submitted", "in_review", "review_consolidation", "reviewed", "discussion"}, // Can revert within 24h
	}

	// Check if user has permission for this transition
	if toStatus == "in_review" || toStatus == "review_consolidation" || toStatus == "reviewed" || toStatus == "discussion" || toStatus == "archived" {
		if !isReviewer {
			return fmt.Errorf("permission denied: only reviewers can transition to %s status", toStatus)
		}
	}

	// Owners can submit their own assessments or close them if still in draft
	// Owners cannot close assessments that have been submitted
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
	s.auditSvc.Log(userID, "delete", "self_assessment",
		fmt.Sprintf("Deleted self-assessment %d (catalog: %d, user: %d)", assessmentID, assessment.CatalogID, assessment.UserID))

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

	// Check if encryption service is available
	if s.encryptedResponseSvc == nil {
		return nil, fmt.Errorf("encryption service not available - Vault must be enabled")
	}

	if existing != nil {
		// Update existing response using encrypted service
		response.ID = existing.ID
		response.CreatedAt = existing.CreatedAt
		if err := s.encryptedResponseSvc.UpdateResponse(response, userID); err != nil {
			return nil, err
		}

		// Audit log
		s.auditSvc.Log(userID, "update", "assessment_response",
			fmt.Sprintf("Updated response for assessment %d, category %d", assessmentID, response.CategoryID))
	} else {
		// Create new response using encrypted service
		if err := s.encryptedResponseSvc.CreateResponse(response, userID); err != nil {
			return nil, err
		}

		// Audit log
		s.auditSvc.Log(userID, "create", "assessment_response",
			fmt.Sprintf("Created response for assessment %d, category %d", assessmentID, response.CategoryID))
	}

	return response, nil
}

// DeleteResponse deletes an assessment response
func (s *SelfAssessmentService) DeleteResponse(userID, assessmentID, categoryID uint) error {
	// Get assessment and verify ownership and status
	assessment, err := s.getAssessmentAndCheckOwnership(assessmentID, userID)
	if err != nil {
		return err
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
	s.auditSvc.Log(userID, "delete", "assessment_response",
		fmt.Sprintf("Deleted response for assessment %d, category %d", assessmentID, categoryID))

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

	// Check permission: owner or reviewer (for submitted/later status)
	isOwner := assessment.UserID == userID
	isAdmin := contains(userRoles, "admin")
	isReviewer := contains(userRoles, "reviewer")

	// CRITICAL: Strict role separation
	if !isOwner {
		// Admins cannot see user justifications (only system/user management)
		if isAdmin && !isReviewer {
			return nil, fmt.Errorf("permission denied: admins cannot access user justifications")
		}

		// Reviewers can see user justifications for submitted/in_review/reviewed/discussion status
		if isReviewer {
			// Cannot review draft or closed assessments
			if assessment.Status == "draft" || assessment.Status == "closed" || assessment.Status == "archived" {
				return nil, fmt.Errorf("permission denied: cannot review %s assessments", assessment.Status)
			}
			// Prevent self-review
			if assessment.UserID == userID {
				return nil, fmt.Errorf("permission denied: cannot review your own assessment")
			}
			// Reviewer has access - continue to load responses
		} else {
			return nil, fmt.Errorf("permission denied: can only view own assessment responses")
		}
	}

	// Get all responses
	responses, err := s.responseRepo.GetAllByAssessment(assessmentID)
	if err != nil {
		return nil, err
	}

	// Decrypt justifications if encryption service is available
	if s.encryptedResponseSvc != nil {
		for i := range responses {
			if responses[i].EncryptedJustificationID != nil {
				decrypted, err := s.encryptedResponseSvc.DecryptJustification(*responses[i].EncryptedJustificationID)
				if err != nil {
					// Log error but continue - don't fail the whole request
					slog.Error("Failed to decrypt justification",
						"error", err,
						"encrypted_justification_id", *responses[i].EncryptedJustificationID,
						"response_id", responses[i].ID)
					responses[i].Justification = "[Decryption failed]"
				} else {
					responses[i].Justification = decrypted
				}
			}
		}
	}

	return responses, nil
}

// GetCompleteness calculates the completeness of an assessment
func (s *SelfAssessmentService) GetCompleteness(userID uint, assessmentID uint) (*models.AssessmentCompleteness, error) {
	// Get assessment and verify ownership
	if _, err := s.getAssessmentAndCheckOwnership(assessmentID, userID); err != nil {
		return nil, err
	}

	// Get completeness from repository
	completeness, err := s.responseRepo.GetCompleteness(assessmentID)
	if err != nil {
		return nil, err
	}

	return completeness, nil
}

// CalculateWeightedScore calculates the weighted average score for a self-assessment
func (s *SelfAssessmentService) CalculateWeightedScore(userID uint, assessmentID uint) (*models.WeightedScore, error) {
	// Verify ownership
	assessment, err := s.getAssessmentAndCheckOwnership(assessmentID, userID)
	if err != nil {
		return nil, err
	}

	// Get catalog details including weights
	catalogDetails, err := s.catalogRepo.GetCatalogWithDetails(assessment.CatalogID)
	if err != nil {
		return nil, err
	}
	if catalogDetails == nil {
		return nil, fmt.Errorf("catalog not found")
	}

	// Get all responses for this assessment with details (includes level_number)
	responses, err := s.responseRepo.GetAllByAssessment(assessmentID)
	if err != nil {
		return nil, err
	}

	// Build map of responses by category ID
	responseMap := make(map[uint]models.AssessmentResponseWithDetails)
	for _, response := range responses {
		responseMap[response.CategoryID] = response
	}

	// Calculate weighted score
	weightedSum := 0.0
	hasAllResponses := true

	for _, category := range catalogDetails.Categories {
		response, exists := responseMap[category.ID]
		if !exists || category.Weight == nil {
			hasAllResponses = false
			continue
		}

		// weighted_sum += level_number * weight
		weightedSum += float64(response.LevelNumber) * (*category.Weight)
	}

	// Determine overall level based on weighted average
	overallLevelNumber := int(weightedSum + 0.5) // Round to nearest integer
	if overallLevelNumber < 1 {
		overallLevelNumber = 1
	}
	if overallLevelNumber > len(catalogDetails.Levels) {
		overallLevelNumber = len(catalogDetails.Levels)
	}

	// Find the level name (letter)
	overallLevelName := ""
	for _, level := range catalogDetails.Levels {
		if level.LevelNumber == overallLevelNumber {
			overallLevelName = level.Name
			break
		}
	}

	return &models.WeightedScore{
		WeightedAverage: weightedSum,
		OverallLevel:    overallLevelName,
		LevelNumber:     overallLevelNumber,
		IsComplete:      hasAllResponses,
	}, nil
}

// filterAssessmentsByReviewCompletion filters out assessments in review_consolidation or later phases
// if the reviewer has not completed their review for that assessment
func (s *SelfAssessmentService) filterAssessmentsByReviewCompletion(assessments []models.SelfAssessmentWithDetails, reviewerUserID uint) ([]models.SelfAssessmentWithDetails, error) {
	var filtered []models.SelfAssessmentWithDetails

	consolidationPhases := map[string]bool{
		"review_consolidation": true,
		"reviewed":             true,
		"discussion":           true,
		"archived":             true,
	}

	for _, assessment := range assessments {
		// If assessment is in consolidation phase or later, check if reviewer completed their review
		if consolidationPhases[assessment.Status] {
			// Check if reviewer has completed all categories
			hasComplete, err := s.hasCompleteReview(assessment.ID, reviewerUserID)
			if err != nil {
				return nil, err
			}

			// Only include if reviewer has complete review
			if hasComplete {
				filtered = append(filtered, assessment)
			}
		} else {
			// For other statuses, include the assessment
			filtered = append(filtered, assessment)
		}
	}

	return filtered, nil
}

// hasCompleteReview checks if a reviewer has completed their review for an assessment
func (s *SelfAssessmentService) hasCompleteReview(assessmentID, reviewerUserID uint) (bool, error) {
	// Get assessment to find catalog
	assessment, err := s.selfAssessmentRepo.GetByID(assessmentID)
	if err != nil {
		return false, err
	}
	if assessment == nil {
		return false, fmt.Errorf("assessment not found")
	}

	// Get categories for this catalog
	categories, err := s.catalogRepo.GetCategoriesByCatalogID(assessment.CatalogID)
	if err != nil {
		return false, err
	}

	totalCategories := len(categories)

	// Get reviewer's responses
	responses, err := s.reviewerRepo.GetByAssessmentAndReviewer(assessmentID, reviewerUserID)
	if err != nil {
		return false, err
	}

	reviewedCategories := len(responses)

	return reviewedCategories >= totalCategories && totalCategories > 0, nil
}

// SubmitAssessment submits an assessment for review (changes status from draft to submitted)
func (s *SelfAssessmentService) SubmitAssessment(userID, assessmentID uint) error {
	// Get assessment and verify ownership and status
	assessment, err := s.getAssessmentAndCheckOwnership(assessmentID, userID)
	if err != nil {
		return err
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
	s.auditSvc.Log(userID, "submit", "self_assessment",
		fmt.Sprintf("Submitted self-assessment %d for review", assessmentID))

	return nil
}
