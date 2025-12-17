package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"

	"new-pay/internal/email"
	"new-pay/internal/keymanager"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/securestore"
)

// ConsolidationService handles business logic for review consolidation
type ConsolidationService struct {
	db                     *sql.DB
	consolidationRepo      *repository.ConsolidationOverrideRepository
	approvalRepo           *repository.ConsolidationOverrideApprovalRepository
	averagedApprovalRepo   *repository.ConsolidationAveragedApprovalRepository
	finalConsolidationRepo *repository.FinalConsolidationRepository
	finalApprovalRepo      *repository.FinalConsolidationApprovalRepository
	assessmentRepo         *repository.SelfAssessmentRepository
	responseRepo           *repository.AssessmentResponseRepository
	reviewerRepo           *repository.ReviewerResponseRepository
	catalogRepo            *repository.CatalogRepository
	encryptedResponseSvc   *EncryptedResponseService
	keyManager             *keymanager.KeyManager
	secureStore            *securestore.SecureStore
	emailService           *email.Service
}

// NewConsolidationService creates a new consolidation service
func NewConsolidationService(
	db *sql.DB,
	consolidationRepo *repository.ConsolidationOverrideRepository,
	approvalRepo *repository.ConsolidationOverrideApprovalRepository,
	averagedApprovalRepo *repository.ConsolidationAveragedApprovalRepository,
	finalConsolidationRepo *repository.FinalConsolidationRepository,
	finalApprovalRepo *repository.FinalConsolidationApprovalRepository,
	assessmentRepo *repository.SelfAssessmentRepository,
	responseRepo *repository.AssessmentResponseRepository,
	reviewerRepo *repository.ReviewerResponseRepository,
	catalogRepo *repository.CatalogRepository,
	encryptedResponseSvc *EncryptedResponseService,
	keyManager *keymanager.KeyManager,
	secureStore *securestore.SecureStore,
	emailService *email.Service,
) *ConsolidationService {
	return &ConsolidationService{
		db:                     db,
		consolidationRepo:      consolidationRepo,
		approvalRepo:           approvalRepo,
		averagedApprovalRepo:   averagedApprovalRepo,
		finalConsolidationRepo: finalConsolidationRepo,
		finalApprovalRepo:      finalApprovalRepo,
		assessmentRepo:         assessmentRepo,
		responseRepo:           responseRepo,
		reviewerRepo:           reviewerRepo,
		catalogRepo:            catalogRepo,
		encryptedResponseSvc:   encryptedResponseSvc,
		keyManager:             keyManager,
		secureStore:            secureStore,
		emailService:           emailService,
	}
}

// GetConsolidationData retrieves all data needed for consolidation page
func (s *ConsolidationService) GetConsolidationData(assessmentID uint, currentUserID uint) (*models.ConsolidationData, error) {
	// Get assessment
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return nil, fmt.Errorf("assessment not found")
	}

	// Check that assessment is in consolidation, reviewed, or discussion status
	if assessment.Status != "review_consolidation" && assessment.Status != "reviewed" && assessment.Status != "discussion" {
		return nil, fmt.Errorf("assessment must be in review_consolidation, reviewed, or discussion status")
	}

	// Check if current user has completed a review for this assessment
	hasCompleteReview, err := s.hasCompleteReview(assessmentID, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasCompleteReview {
		return nil, fmt.Errorf("permission denied: only reviewers who completed their review can access consolidation")
	}

	// Get catalog with details
	catalog, err := s.catalogRepo.GetCatalogWithDetails(assessment.CatalogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	// Get user responses with details (decrypted)
	userResponsesPtr, err := s.encryptedResponseSvc.GetResponsesWithDetailsByAssessment(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user responses: %w", err)
	}

	// Convert from []*models.AssessmentResponseWithDetails to []models.AssessmentResponseWithDetails
	userResponses := make([]models.AssessmentResponseWithDetails, len(userResponsesPtr))
	for i, resp := range userResponsesPtr {
		userResponses[i] = *resp
	}

	// Get all reviewer responses for this assessment
	reviewerResponses, err := s.reviewerRepo.GetAllByAssessment(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewer responses: %w", err)
	}

	// Decrypt reviewer justifications
	for i := range reviewerResponses {
		if reviewerResponses[i].EncryptedJustificationID != nil {
			plainData, err := s.secureStore.DecryptRecord(*reviewerResponses[i].EncryptedJustificationID)
			if err != nil {
				slog.Error("Failed to decrypt justification", "error", err, "record_id", *reviewerResponses[i].EncryptedJustificationID)
				continue
			}

			if justification, ok := plainData.Fields["justification"].(string); ok {
				reviewerResponses[i].Justification = justification
			}
		}
	}

	// Calculate averaged responses per category
	averagedResponses := s.calculateAveragedResponses(reviewerResponses, catalog)

	// Load approvals for averaged responses
	allAveragedApprovals, err := s.averagedApprovalRepo.GetApprovalsByAssessment(assessmentID)
	if err != nil {
		slog.Error("Failed to load averaged approvals", "error", err)
	} else {
		// Group approvals by category
		approvalsByCategory := make(map[uint][]models.ConsolidationAveragedApproval)
		for _, approval := range allAveragedApprovals {
			approvalsByCategory[approval.CategoryID] = append(approvalsByCategory[approval.CategoryID], approval)
		}

		// Attach approvals to averaged responses
		for i := range averagedResponses {
			approvals := approvalsByCategory[averagedResponses[i].CategoryID]
			averagedResponses[i].Approvals = approvals
			averagedResponses[i].ApprovalCount = len(approvals)
			averagedResponses[i].IsApproved = len(approvals) >= 2 // Need at least 2 approvals
		}
	}

	// Get consolidation overrides
	overrides, err := s.consolidationRepo.GetByAssessment(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overrides: %w", err)
	}

	// Decrypt override justifications and load approvals
	for i := range overrides {
		if overrides[i].EncryptedJustificationID != nil {
			plainData, err := s.secureStore.DecryptRecord(*overrides[i].EncryptedJustificationID)
			if err != nil {
				slog.Error("Failed to decrypt justification", "error", err, "record_id", *overrides[i].EncryptedJustificationID)
				continue
			}

			if justification, ok := plainData.Fields["justification"].(string); ok {
				overrides[i].Justification = justification
			}
		}

		// Load approvals for this override
		approvals, err := s.approvalRepo.GetApprovalsByOverride(overrides[i].ID)
		if err != nil {
			slog.Error("Failed to load approvals", "error", err, "override_id", overrides[i].ID)
			continue
		}
		overrides[i].Approvals = approvals
		overrides[i].ApprovalCount = len(approvals)
		overrides[i].IsApproved = len(approvals) > 0
	}

	// Get current user's own reviewer responses (decrypted)
	currentUserResponses, err := s.reviewerRepo.GetByAssessmentAndReviewer(assessmentID, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user responses: %w", err)
	}

	// Decrypt current user's justifications
	for i := range currentUserResponses {
		if currentUserResponses[i].EncryptedJustificationID != nil {
			plainData, err := s.secureStore.DecryptRecord(*currentUserResponses[i].EncryptedJustificationID)
			if err != nil {
				slog.Error("Failed to decrypt justification", "error", err, "record_id", *currentUserResponses[i].EncryptedJustificationID)
				continue
			}

			if justification, ok := plainData.Fields["justification"].(string); ok {
				currentUserResponses[i].Justification = justification
			}
		}
	}

	// Check if all categories are approved
	allCategoriesApproved := s.areAllCategoriesApproved(catalog.Categories, averagedResponses, overrides)

	// Get final consolidation if it exists
	var finalConsolidation *models.FinalConsolidation
	fc, err := s.finalConsolidationRepo.GetByAssessment(assessmentID)
	if err != nil {
		slog.Error("Failed to get final consolidation", "error", err)
	} else if fc != nil {
		// Decrypt comment
		if fc.EncryptedCommentID != nil {
			plainData, err := s.secureStore.DecryptRecord(*fc.EncryptedCommentID)
			if err != nil {
				slog.Error("Failed to decrypt final consolidation comment", "error", err)
			} else if comment, ok := plainData.Fields["comment"].(string); ok {
				fc.Comment = comment
			}
		}

		// Load approvals
		approvals, err := s.finalApprovalRepo.GetApprovalsByAssessment(assessmentID)
		if err != nil {
			slog.Error("Failed to load final consolidation approvals", "error", err)
		} else {
			fc.Approvals = approvals
			fc.ApprovalCount = len(approvals)
		}

		// Get required approval count (number of reviewers who completed review)
		reviewers, err := s.reviewerRepo.GetCompleteReviewers(assessmentID)
		if err != nil {
			slog.Error("Failed to get reviewer count", "error", err)
		} else {
			fc.RequiredApprovals = len(reviewers)
			fc.IsFullyApproved = fc.ApprovalCount >= fc.RequiredApprovals
		}

		finalConsolidation = fc
	}

	return &models.ConsolidationData{
		Assessment:            *assessment,
		UserResponses:         userResponses,
		AveragedResponses:     averagedResponses,
		Overrides:             overrides,
		Catalog:               *catalog,
		CurrentUserResponses:  currentUserResponses,
		FinalConsolidation:    finalConsolidation,
		AllCategoriesApproved: allCategoriesApproved,
	}, nil
}

// areAllCategoriesApproved checks if all categories have required approvals
func (s *ConsolidationService) areAllCategoriesApproved(categories []models.CategoryWithPaths, averagedResponses []models.AveragedReviewerResponse, overrides []models.ConsolidationOverride) bool {
	for _, category := range categories {
		// Check if override exists and is approved
		overrideApproved := false
		for _, override := range overrides {
			if override.CategoryID == category.ID && override.IsApproved {
				overrideApproved = true
				break
			}
		}

		if overrideApproved {
			continue
		}

		// Check if averaged response has 2+ approvals
		averagedApproved := false
		for _, averaged := range averagedResponses {
			if averaged.CategoryID == category.ID && averaged.IsApproved {
				averagedApproved = true
				break
			}
		}

		if !averagedApproved {
			return false // This category is not approved
		}
	}

	return true
}

// checkRevocationAllowed checks if approval revocation is allowed based on assessment status and time
func (s *ConsolidationService) checkRevocationAllowed(assessmentID uint) error {
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}

	// Allow revocation in review_consolidation status
	if assessment.Status == "review_consolidation" {
		return nil
	}

	// In reviewed status, only allow within 1 hour
	if assessment.Status == "reviewed" {
		if assessment.ReviewedAt == nil {
			return fmt.Errorf("reviewed_at timestamp not set")
		}

		oneHourAgo := time.Now().Add(-1 * time.Hour)
		if assessment.ReviewedAt.Before(oneHourAgo) {
			return fmt.Errorf("revocation period expired (only allowed within 1 hour after review completion)")
		}
		return nil
	}

	// Not allowed in other statuses
	return fmt.Errorf("approval revocation not allowed in current status")
}

// checkEditingAllowed checks if editing is allowed (only in review_consolidation status)
func (s *ConsolidationService) checkEditingAllowed(assessmentID uint) error {
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}

	if assessment.Status != "review_consolidation" {
		return fmt.Errorf("editing only allowed in review_consolidation status")
	}

	return nil
}

// hasCompleteReview checks if a user has completed their review for an assessment
func (s *ConsolidationService) hasCompleteReview(assessmentID, userID uint) (bool, error) {
	// Get assessment to find catalog
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
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

	// Get user's reviewer responses
	responses, err := s.reviewerRepo.GetByAssessmentAndReviewer(assessmentID, userID)
	if err != nil {
		return false, err
	}

	reviewedCategories := len(responses)

	return reviewedCategories >= totalCategories && totalCategories > 0, nil
}

// calculateAveragedResponses calculates averaged reviewer responses per category
func (s *ConsolidationService) calculateAveragedResponses(reviewerResponses []models.ReviewerResponse, catalog *models.CatalogWithDetails) []models.AveragedReviewerResponse {
	// Group by category
	categoryMap := make(map[uint][]models.ReviewerResponse)
	categoryInfo := make(map[uint]struct {
		Name      string
		SortOrder int
	})

	// Store category info from catalog
	for _, cat := range catalog.Categories {
		categoryInfo[cat.ID] = struct {
			Name      string
			SortOrder int
		}{
			Name:      cat.Name,
			SortOrder: cat.SortOrder,
		}
	}

	for _, resp := range reviewerResponses {
		categoryMap[resp.CategoryID] = append(categoryMap[resp.CategoryID], resp)
	}

	var averaged []models.AveragedReviewerResponse
	for categoryID, responses := range categoryMap {
		if len(responses) == 0 {
			continue
		}

		// Calculate average level number
		var sum float64
		for _, resp := range responses {
			// Find level number from catalog
			levelNumber := s.findLevelNumber(catalog, resp.LevelID)
			sum += float64(levelNumber)
		}

		avgLevelNumber := sum / float64(len(responses))

		// Find closest level name
		avgLevelName := s.findClosestLevelName(catalog, avgLevelNumber)

		info := categoryInfo[categoryID]
		averaged = append(averaged, models.AveragedReviewerResponse{
			CategoryID:         categoryID,
			CategoryName:       info.Name,
			CategorySortOrder:  info.SortOrder,
			AverageLevelNumber: math.Round(avgLevelNumber*100) / 100, // Round to 2 decimals
			AverageLevelName:   avgLevelName,
			ReviewerCount:      len(responses),
			// ReviewerJustifications intentionally omitted for security - reviewers should not see each other's comments
		})
	}

	return averaged
}

// findLevelNumber finds the level number for a given level ID
func (s *ConsolidationService) findLevelNumber(catalog *models.CatalogWithDetails, levelID uint) int {
	for _, level := range catalog.Levels {
		if level.ID == levelID {
			return level.LevelNumber
		}
	}
	return 0
}

// findClosestLevelName finds the closest level name for an average level number
func (s *ConsolidationService) findClosestLevelName(catalog *models.CatalogWithDetails, avgNumber float64) string {
	if len(catalog.Levels) == 0 {
		return ""
	}

	closestLevel := catalog.Levels[0]
	minDiff := math.Abs(float64(closestLevel.LevelNumber) - avgNumber)

	for _, level := range catalog.Levels[1:] {
		diff := math.Abs(float64(level.LevelNumber) - avgNumber)
		if diff < minDiff {
			minDiff = diff
			closestLevel = level
		}
	}

	return closestLevel.Name
}

// CreateOrUpdateOverride creates or updates a consolidation override with encryption
func (s *ConsolidationService) CreateOrUpdateOverride(override *models.ConsolidationOverride, userID uint) error {
	// Check if editing is allowed
	if err := s.checkEditingAllowed(override.AssessmentID); err != nil {
		return err
	}

	// Verify assessment is in review_consolidation status
	assessment, err := s.assessmentRepo.GetByID(override.AssessmentID)
	if err != nil {
		return fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}
	if assessment.Status != "review_consolidation" {
		return fmt.Errorf("assessment must be in review_consolidation status")
	}

	// Check if override already exists to detect edits
	existingOverride, err := s.consolidationRepo.GetByAssessmentAndCategory(override.AssessmentID, override.CategoryID)
	if err != nil && err.Error() != "override not found" {
		return fmt.Errorf("failed to check existing override: %w", err)
	}

	// If override exists and either the user is different OR the data changed, reset approvals
	if existingOverride != nil {
		authorChanged := existingOverride.CreatedByUserID != userID
		dataChanged := existingOverride.PathID != override.PathID ||
			existingOverride.LevelID != override.LevelID ||
			existingOverride.Justification != override.Justification

		if authorChanged || dataChanged {
			// Delete all approvals for this override
			if err := s.approvalRepo.DeleteAllApprovalsForOverride(existingOverride.ID); err != nil {
				return fmt.Errorf("failed to reset approvals: %w", err)
			}

			// Update the author to the current user (whoever makes changes becomes the new author)
			override.CreatedByUserID = userID
		}
	} else {
		// This is a new override - delete any averaged approvals for this category
		if err := s.averagedApprovalRepo.DeleteAllApprovalsForCategory(override.AssessmentID, override.CategoryID); err != nil {
			slog.Error("Failed to delete averaged approvals", "error", err)
		}
	}

	// Ensure user key exists
	if err := s.ensureUserKey(int64(userID)); err != nil {
		return fmt.Errorf("failed to ensure user key: %w", err)
	}

	// Ensure process key exists
	processID := fmt.Sprintf("assessment-%d", override.AssessmentID)
	if err := s.ensureProcessKey(processID); err != nil {
		return fmt.Errorf("failed to ensure process key: %w", err)
	}

	// Encrypt justification if provided
	if override.Justification != "" {
		data := &securestore.PlainData{
			Fields: map[string]interface{}{
				"justification": override.Justification,
			},
			Metadata: map[string]string{
				"assessment_id": fmt.Sprintf("%d", override.AssessmentID),
				"category_id":   fmt.Sprintf("%d", override.CategoryID),
				"user_id":       fmt.Sprintf("%d", userID),
				"type":          "consolidation_override",
			},
		}

		record, err := s.secureStore.CreateRecord(
			processID,
			int64(userID),
			"CONSOLIDATION_JUSTIFICATION",
			data,
			"",
		)
		if err != nil {
			return fmt.Errorf("failed to encrypt justification: %w", err)
		}

		override.EncryptedJustificationID = &record.ID
		override.Justification = "" // Clear plaintext
	}

	// Set author for new overrides
	if existingOverride == nil {
		override.CreatedByUserID = userID
	}

	return s.consolidationRepo.CreateOrUpdate(override)
}

// ensureUserKey ensures a user encryption key exists
func (s *ConsolidationService) ensureUserKey(userID int64) error {
	// Check if user key exists
	_, err := s.keyManager.GetUserPublicKey(userID)
	if err != nil {
		// Create user key if it doesn't exist
		_, err = s.keyManager.CreateUserKey(userID)
		return err
	}
	return nil
}

// ensureProcessKey ensures a process encryption key exists
func (s *ConsolidationService) ensureProcessKey(processID string) error {
	// Check if process key exists
	_, err := s.keyManager.GetProcessKey(processID)
	if err != nil {
		// Create process key if it doesn't exist
		return s.keyManager.CreateProcessKey(processID, nil)
	}
	return nil
}

// ApproveOverride approves a consolidation override
func (s *ConsolidationService) ApproveOverride(assessmentID, categoryID, userID uint) error {
	// Check if editing is allowed
	if err := s.checkEditingAllowed(assessmentID); err != nil {
		return err
	}

	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before approving overrides")
	}

	// Get the override
	override, err := s.consolidationRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to get override: %w", err)
	}
	if override == nil {
		return fmt.Errorf("override not found")
	}

	// Verify user is not the override author
	if override.CreatedByUserID == userID {
		return fmt.Errorf("cannot approve your own override")
	}

	// Create approval (idempotent due to ON CONFLICT DO NOTHING)
	return s.approvalRepo.CreateApproval(override.ID, userID)
}

// ApproveAveragedResponse approves an averaged reviewer response (when no override exists)
func (s *ConsolidationService) ApproveAveragedResponse(assessmentID, categoryID, userID uint) error {
	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before approving averaged responses")
	}

	// Verify no override exists for this category
	override, err := s.consolidationRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil && err.Error() != "override not found" {
		return fmt.Errorf("failed to check for override: %w", err)
	}
	if override != nil {
		return fmt.Errorf("cannot approve averaged response when override exists - approve the override instead")
	}

	// Create approval (idempotent due to ON CONFLICT DO NOTHING)
	return s.averagedApprovalRepo.CreateApproval(assessmentID, categoryID, userID)
}

// DeleteOverride deletes a consolidation override (any reviewer can delete)
func (s *ConsolidationService) DeleteOverride(assessmentID, categoryID, userID uint) error {
	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before deleting overrides")
	}

	// Get the override
	override, err := s.consolidationRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to get override: %w", err)
	}
	if override == nil {
		return fmt.Errorf("override not found")
	}

	// Delete all approvals for this override first
	if err := s.approvalRepo.DeleteAllApprovalsForOverride(override.ID); err != nil {
		return fmt.Errorf("failed to delete approvals: %w", err)
	}

	// Delete the override
	return s.consolidationRepo.Delete(override.ID)
}

// RevokeOverrideApproval removes a user's approval from an override
func (s *ConsolidationService) RevokeOverrideApproval(assessmentID, categoryID, userID uint) error {
	// Check if revocation is allowed (within 1 hour if reviewed)
	if err := s.checkRevocationAllowed(assessmentID); err != nil {
		return err
	}

	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before revoking approvals")
	}

	// Get the override
	override, err := s.consolidationRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to get override: %w", err)
	}
	if override == nil {
		return fmt.Errorf("override not found")
	}

	// Delete the user's approval
	return s.approvalRepo.DeleteApproval(override.ID, userID)
}

// RevokeAveragedApproval removes a user's approval from an averaged response
func (s *ConsolidationService) RevokeAveragedApproval(assessmentID, categoryID, userID uint) error {
	// Check if revocation is allowed (within 1 hour if reviewed)
	if err := s.checkRevocationAllowed(assessmentID); err != nil {
		return err
	}

	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before revoking approvals")
	}

	// Delete the user's approval
	return s.averagedApprovalRepo.DeleteApproval(assessmentID, categoryID, userID)
}

// CreateOrUpdateFinalConsolidation creates or updates the final consolidation comment
func (s *ConsolidationService) CreateOrUpdateFinalConsolidation(assessmentID uint, comment string, userID uint) error {
	// Check if editing is allowed
	if err := s.checkEditingAllowed(assessmentID); err != nil {
		return err
	}

	processID := fmt.Sprintf("assessment_%d", assessmentID)

	// Ensure process key exists
	if err := s.ensureProcessKey(processID); err != nil {
		return fmt.Errorf("failed to ensure process key: %w", err)
	}

	// Ensure user key exists
	if err := s.ensureUserKey(int64(userID)); err != nil {
		return fmt.Errorf("failed to ensure user key: %w", err)
	}

	// Encrypt the comment
	plainData := &securestore.PlainData{
		Fields: map[string]interface{}{
			"comment": comment,
		},
	}

	encryptedRecord, err := s.secureStore.CreateRecord(processID, int64(userID), "final_consolidation", plainData, "active")
	if err != nil {
		return fmt.Errorf("failed to encrypt comment: %w", err)
	}

	encryptedID := encryptedRecord.ID

	// Check if final consolidation already exists
	existing, err := s.finalConsolidationRepo.GetByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to check existing final consolidation: %w", err)
	}

	// If it exists and comment changed, delete all approvals
	if existing != nil {
		if err := s.finalApprovalRepo.DeleteAllApprovalsForAssessment(assessmentID); err != nil {
			return fmt.Errorf("failed to delete approvals: %w", err)
		}
	}

	// Create or update final consolidation
	finalConsolidation := &models.FinalConsolidation{
		AssessmentID:       assessmentID,
		EncryptedCommentID: &encryptedID,
		CreatedByUserID:    userID,
	}

	return s.finalConsolidationRepo.CreateOrUpdate(finalConsolidation)
}

// ApproveFinalConsolidation approves the final consolidation
func (s *ConsolidationService) ApproveFinalConsolidation(assessmentID, userID uint) error {
	// Check if editing is allowed
	if err := s.checkEditingAllowed(assessmentID); err != nil {
		return err
	}

	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before approving final consolidation")
	}

	// Verify final consolidation exists
	finalConsolidation, err := s.finalConsolidationRepo.GetByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get final consolidation: %w", err)
	}
	if finalConsolidation == nil {
		return fmt.Errorf("final consolidation not found")
	}

	// Create the approval
	if err := s.finalApprovalRepo.CreateApproval(assessmentID, userID); err != nil {
		return fmt.Errorf("failed to create approval: %w", err)
	}

	// Check if all required reviewers have approved
	approvalCount, err := s.finalApprovalRepo.GetApprovalCount(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get approval count: %w", err)
	}

	// Get required approval count (number of reviewers who completed review)
	reviewers, err := s.reviewerRepo.GetCompleteReviewers(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get reviewers: %w", err)
	}

	requiredApprovals := len(reviewers)

	// If all reviewers approved, change status to 'reviewed'
	if approvalCount >= requiredApprovals {
		assessment, err := s.assessmentRepo.GetByID(assessmentID)
		if err != nil {
			return fmt.Errorf("failed to get assessment: %w", err)
		}

		// Get assessment details with user info for email
		assessmentDetails, err := s.assessmentRepo.GetByIDWithDetails(assessmentID)
		if err != nil {
			slog.Error("Failed to get assessment details for email notification", "error", err)
		}

		// Get catalog info for email
		catalog, err := s.catalogRepo.GetCatalogByID(assessment.CatalogID)
		if err != nil {
			slog.Error("Failed to get catalog for email notification", "error", err)
		}

		// Update status to 'reviewed'
		assessment.Status = "reviewed"
		if err := s.assessmentRepo.Update(assessment); err != nil {
			return fmt.Errorf("failed to update assessment status: %w", err)
		}

		// Send notification email to the assessed user
		if assessmentDetails != nil && catalog != nil {
			if err := s.emailService.SendReviewCompletedNotification(assessmentDetails.UserEmail, assessmentDetails.UserName, catalog.Name, assessmentID); err != nil {
				slog.Error("Failed to send review completion email", "error", err, "assessment_id", assessmentID, "user_email", assessmentDetails.UserEmail)
				// Don't fail the approval if email sending fails
			} else {
				slog.Info("Assessment review completed and notification sent", "assessment_id", assessmentID, "user_id", assessment.UserID, "user_email", assessmentDetails.UserEmail)
			}
		} else {
			slog.Info("Assessment review completed (no email sent - details not available)", "assessment_id", assessmentID, "user_id", assessment.UserID)
		}
	}

	return nil
}

// RevokeFinalApproval removes a user's approval from the final consolidation
func (s *ConsolidationService) RevokeFinalApproval(assessmentID, userID uint) error {
	// Check if revocation is allowed (within 1 hour if reviewed)
	if err := s.checkRevocationAllowed(assessmentID); err != nil {
		return err
	}

	// Verify user has completed their review
	hasComplete, err := s.hasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before revoking approvals")
	}

	// Verify final consolidation exists
	finalConsolidation, err := s.finalConsolidationRepo.GetByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get final consolidation: %w", err)
	}
	if finalConsolidation == nil {
		return fmt.Errorf("final consolidation not found")
	}

	// Delete the user's approval
	return s.finalApprovalRepo.DeleteApproval(assessmentID, userID)
}
