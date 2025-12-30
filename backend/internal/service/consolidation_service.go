package service

import (
	"database/sql"
	"fmt"
	"log/slog"
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
	categoryDiscussionRepo *repository.CategoryDiscussionCommentRepository
	encryptedResponseSvc   *EncryptedResponseService
	keyManager             *keymanager.KeyManager
	secureStore            *securestore.SecureStore
	emailService           *email.Service
	llmService             *LLMService
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
	categoryDiscussionRepo *repository.CategoryDiscussionCommentRepository,
	encryptedResponseSvc *EncryptedResponseService,
	keyManager *keymanager.KeyManager,
	secureStore *securestore.SecureStore,
	emailService *email.Service,
	llmService *LLMService,
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
		categoryDiscussionRepo: categoryDiscussionRepo,
		encryptedResponseSvc:   encryptedResponseSvc,
		keyManager:             keyManager,
		secureStore:            secureStore,
		emailService:           emailService,
		llmService:             llmService,
	}
}

// Helper functions

// getAssessment loads an assessment and checks if it exists
func (s *ConsolidationService) getAssessment(assessmentID uint) (*models.SelfAssessment, error) {
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return nil, fmt.Errorf("assessment not found")
	}
	return assessment, nil
}

// decryptField decrypts a field from secure store
func (s *ConsolidationService) decryptField(recordID int64, fieldName string) (string, error) {
	plainData, err := s.secureStore.DecryptRecord(recordID)
	if err != nil {
		return "", err
	}
	if value, ok := plainData.Fields[fieldName].(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("field %s not found or not a string", fieldName)
}

// decryptJustifications decrypts justifications for reviewer responses in place
func (s *ConsolidationService) decryptJustifications(responses []models.ReviewerResponse) {
	for i := range responses {
		if responses[i].EncryptedJustificationID != nil {
			justification, err := s.decryptField(*responses[i].EncryptedJustificationID, "justification")
			if err != nil {
				slog.Warn("Failed to decrypt reviewer justification", "error", err, "record_id", *responses[i].EncryptedJustificationID)
				continue
			}
			responses[i].Justification = justification
		}
	}
}

// decryptOverrideJustifications decrypts justifications for overrides in place
func (s *ConsolidationService) decryptOverrideJustifications(overrides []models.ConsolidationOverride) {
	for i := range overrides {
		if overrides[i].EncryptedJustificationID != nil {
			justification, err := s.decryptField(*overrides[i].EncryptedJustificationID, "justification")
			if err != nil {
				slog.Warn("Failed to decrypt override justification", "error", err, "record_id", *overrides[i].EncryptedJustificationID)
				continue
			}
			overrides[i].Justification = justification
		}
	}
}

// decryptCategoryDiscussionComments decrypts comments in place
func (s *ConsolidationService) decryptCategoryDiscussionComments(comments []models.CategoryDiscussionComment) {
	for i := range comments {
		if comments[i].EncryptedCommentID != nil {
			comment, err := s.decryptField(*comments[i].EncryptedCommentID, "comment")
			if err != nil {
				slog.Warn("Failed to decrypt category discussion comment", "error", err, "record_id", *comments[i].EncryptedCommentID)
				continue
			}
			comments[i].Comment = comment
		}
	}
}

// GetConsolidationData retrieves all data needed for consolidation page
func (s *ConsolidationService) GetConsolidationData(assessmentID uint, currentUserID uint) (*models.ConsolidationData, error) {
	// Get assessment
	assessment, err := s.getAssessment(assessmentID)
	if err != nil {
		return nil, err
	}

	// Check that assessment is in consolidation, reviewed, or discussion status
	if assessment.Status != "review_consolidation" && assessment.Status != "reviewed" && assessment.Status != "discussion" {
		return nil, fmt.Errorf("assessment must be in review_consolidation, reviewed, or discussion status")
	}

	// Check if current user has completed a review for this assessment
	hasCompleteReview, err := s.HasCompleteReview(assessmentID, currentUserID)
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
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
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
	s.decryptJustifications(reviewerResponses)

	// Calculate averaged responses per category (without justifications for security)
	averagedResponses := calculateAveragedResponses(reviewerResponses, catalog, false)

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
	s.decryptOverrideJustifications(overrides)

	for i := range overrides {

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
	s.decryptJustifications(currentUserResponses)

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
			comment, err := s.decryptField(*fc.EncryptedCommentID, "comment")
			if err != nil {
				slog.Error("Failed to decrypt final consolidation comment", "error", err)
			} else {
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

	// Get category discussion comments
	categoryDiscussionComments, err := s.GetCategoryDiscussionComments(assessmentID)
	if err != nil {
		slog.Error("Failed to get category discussion comments", "error", err)
		categoryDiscussionComments = []models.CategoryDiscussionComment{} // Empty slice on error
	}

	return &models.ConsolidationData{
		Assessment:                 *assessment,
		UserResponses:              userResponses,
		AveragedResponses:          averagedResponses,
		Overrides:                  overrides,
		Catalog:                    *catalog,
		CurrentUserResponses:       currentUserResponses,
		FinalConsolidation:         finalConsolidation,
		CategoryDiscussionComments: categoryDiscussionComments,
		AllCategoriesApproved:      allCategoriesApproved,
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
	assessment, err := s.getAssessment(assessmentID)
	if err != nil {
		return err
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
	assessment, err := s.getAssessment(assessmentID)
	if err != nil {
		return err
	}

	if assessment.Status != "review_consolidation" {
		return fmt.Errorf("editing only allowed in review_consolidation status")
	}

	return nil
}

// HasCompleteReview checks if a user has completed their review for an assessment
func (s *ConsolidationService) HasCompleteReview(assessmentID, userID uint) (bool, error) {
	// Get assessment to find catalog
	assessment, err := s.getAssessment(assessmentID)
	if err != nil {
		return false, err
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

// CreateOrUpdateOverride creates or updates a consolidation override with encryption
func (s *ConsolidationService) CreateOrUpdateOverride(override *models.ConsolidationOverride, userID uint) error {
	// Check if editing is allowed
	if err := s.checkEditingAllowed(override.AssessmentID); err != nil {
		return err
	}

	// Verify assessment is in review_consolidation status
	assessment, err := s.getAssessment(override.AssessmentID)
	if err != nil {
		return err
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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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

	processID := fmt.Sprintf("assessment-%d", assessmentID)

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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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
		assessment, err := s.getAssessment(assessmentID)
		if err != nil {
			return err
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
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
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

// SaveCategoryDiscussionComment saves or updates a category-specific discussion comment
func (s *ConsolidationService) SaveCategoryDiscussionComment(assessmentID, categoryID, userID uint, comment string) error {
	// Check if editing is allowed
	if err := s.checkEditingAllowed(assessmentID); err != nil {
		return err
	}

	// Verify user has completed their review
	hasComplete, err := s.HasCompleteReview(assessmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to check review completion: %w", err)
	}
	if !hasComplete {
		return fmt.Errorf("user must complete their review before adding discussion comments")
	}

	return s.saveCategoryDiscussionCommentInternal(assessmentID, categoryID, userID, comment)
}

func (s *ConsolidationService) saveCategoryDiscussionCommentInternal(assessmentID, categoryID, userID uint, comment string) error {
	// Get assessment for process ID
	assessment, err := s.getAssessment(assessmentID)
	if err != nil {
		return err
	}
	processID := fmt.Sprintf("assessment-%d", assessment.ID)

	// Store the comment in secure store
	plainData := &securestore.PlainData{
		Fields: map[string]interface{}{
			"comment": comment,
		},
		Metadata: map[string]string{
			"assessment_id": fmt.Sprintf("%d", assessmentID),
			"category_id":   fmt.Sprintf("%d", categoryID),
			"user_id":       fmt.Sprintf("%d", userID),
			"type":          "category_discussion_comment",
		},
	}

	record, err := s.secureStore.CreateRecord(
		processID,
		int64(userID),
		"CATEGORY_DISCUSSION_COMMENT",
		plainData,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to encrypt comment: %w", err)
	}

	// Check if comment already exists
	existing, err := s.categoryDiscussionRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to check existing comment: %w", err)
	}

	if existing != nil {
		// Update existing comment
		existing.EncryptedCommentID = &record.ID
		return s.categoryDiscussionRepo.Update(existing)
	}

	// Create new comment
	newComment := &models.CategoryDiscussionComment{
		AssessmentID:       assessmentID,
		CategoryID:         categoryID,
		EncryptedCommentID: &record.ID,
		CreatedByUserID:    userID,
	}
	return s.categoryDiscussionRepo.Create(newComment)
}

// GenerateConsolidationProposals generates proposals for category discussion comments using LLM
func (s *ConsolidationService) GenerateConsolidationProposals(assessmentID uint) error {
	// Get all categories for the assessment
	assessment, err := s.getAssessment(assessmentID)
	if err != nil {
		return err
	}

	catalog, err := s.catalogRepo.GetCatalogWithDetails(assessment.CatalogID)
	if err != nil {
		return err
	}

	// For each category, get reviewer comments
	for _, category := range catalog.Categories {
		// Get reviewer responses for this category
		responses, err := s.reviewerRepo.GetByAssessmentAndCategory(assessmentID, category.ID)
		if err != nil {
			slog.Error("Failed to get reviewer responses", "error", err, "category_id", category.ID)
			continue
		}

		// Decrypt justifications
		s.decryptJustifications(responses)

		var comments []string
		var creatorID uint
		for _, r := range responses {
			if r.Justification != "" {
				comments = append(comments, r.Justification)
			}
			// Use the first reviewer as the creator if not set
			if creatorID == 0 {
				creatorID = r.ReviewerUserID
			}
		}

		// If no creator found (no responses), use system user ID 1 or skip
		if creatorID == 0 {
			slog.Warn("No reviewer responses found for category, trying to find any reviewer", "category_id", category.ID, "assessment_id", assessmentID)
			// Try to get any reviewer who reviewed this assessment
			allResponses, err := s.reviewerRepo.GetAllByAssessment(assessmentID)
			if err == nil && len(allResponses) > 0 {
				creatorID = allResponses[0].ReviewerUserID
			}
		}

		if len(comments) == 0 {
			slog.Info("No reviewer comments found for category", "category_id", category.ID, "assessment_id", assessmentID)
			comments = append(comments, "Alle Reviewer stimmen der Einschätzung des Mitarbeiters zu.") // Default comment
		}

		// Generate summary
		summary, err := s.llmService.SummarizeComments(comments)
		if err != nil {
			slog.Error("Failed to summarize comments", "error", err, "category_id", category.ID)
			continue
		}

		// Save as category discussion comment
		if creatorID != 0 {
			// Prepend "Vorschlag (KI): " to indicate it's AI generated
			summary = "Vorschlag (KI): " + summary
			if err := s.saveCategoryDiscussionCommentInternal(assessmentID, category.ID, creatorID, summary); err != nil {
				slog.Error("Failed to save generated proposal", "error", err, "category_id", category.ID)
			} else {
				slog.Info("Successfully saved consolidation proposal", "category_id", category.ID, "assessment_id", assessmentID)
			}
		} else {
			slog.Error("No creator ID available, skipping proposal save", "category_id", category.ID, "assessment_id", assessmentID)
		}
	}

	return nil
}

// GetCategoryDiscussionComments retrieves all category discussion comments for an assessment (decrypted)
func (s *ConsolidationService) GetCategoryDiscussionComments(assessmentID uint) ([]models.CategoryDiscussionComment, error) {
	comments, err := s.categoryDiscussionRepo.GetByAssessment(assessmentID)
	if err != nil {
		return nil, err
	}

	// Decrypt comments
	s.decryptCategoryDiscussionComments(comments)

	return comments, nil
}

// GenerateFinalConsolidationProposal generates a final consolidation comment from category comments using LLM
func (s *ConsolidationService) GenerateFinalConsolidationProposal(assessmentID uint, userID uint) error {
	// Get all category discussion comments
	categoryComments, err := s.GetCategoryDiscussionComments(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get category comments: %w", err)
	}

	if len(categoryComments) == 0 {
		return fmt.Errorf("no category comments found to summarize")
	}

	// Extract comment texts
	var comments []string
	for _, comment := range categoryComments {
		if comment.Comment != "" {
			comments = append(comments, comment.Comment)
		}
	}

	if len(comments) == 0 {
		return fmt.Errorf("no category comment texts found to summarize")
	}

	// Generate summary using LLM
	summary, err := s.llmService.SummarizeComments(comments)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Prepend "Vorschlag (KI): " to indicate it's AI generated
	summary = "Vorschlag (KI): " + summary

	// Save as final consolidation
	if err := s.CreateOrUpdateFinalConsolidation(assessmentID, summary, userID); err != nil {
		return fmt.Errorf("failed to save final consolidation: %w", err)
	}

	slog.Info("Successfully generated final consolidation proposal", "assessment_id", assessmentID, "user_id", userID)
	return nil
}
