package service

import (
	"database/sql"
	"fmt"
	"log/slog"

	"new-pay/internal/keymanager"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/securestore"
)

// ReviewerService handles business logic for reviewer responses
type ReviewerService struct {
	db             *sql.DB
	reviewerRepo   *repository.ReviewerResponseRepository
	assessmentRepo *repository.SelfAssessmentRepository
	responseRepo   *repository.AssessmentResponseRepository
	keyManager     *keymanager.KeyManager
	secureStore    *securestore.SecureStore
}

// NewReviewerService creates a new reviewer service
func NewReviewerService(
	db *sql.DB,
	reviewerRepo *repository.ReviewerResponseRepository,
	assessmentRepo *repository.SelfAssessmentRepository,
	responseRepo *repository.AssessmentResponseRepository,
	keyManager *keymanager.KeyManager,
	secureStore *securestore.SecureStore,
) *ReviewerService {
	return &ReviewerService{
		db:             db,
		reviewerRepo:   reviewerRepo,
		assessmentRepo: assessmentRepo,
		responseRepo:   responseRepo,
		keyManager:     keyManager,
		secureStore:    secureStore,
	}
}

// CreateOrUpdateResponse creates or updates a reviewer response with encryption
func (s *ReviewerService) CreateOrUpdateResponse(response *models.ReviewerResponse, reviewerUserID uint) error {
	// Check assessment status - reviews cannot be created/modified in review_consolidation status
	assessment, err := s.assessmentRepo.GetByID(response.AssessmentID)
	if err != nil {
		return fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}
	if assessment.Status == "review_consolidation" || assessment.Status == "reviewed" || assessment.Status == "discussion" || assessment.Status == "archived" {
		return fmt.Errorf("cannot create or modify reviews for assessments in %s status", assessment.Status)
	}

	// Ensure user key exists for reviewer
	if err := s.ensureUserKey(int64(reviewerUserID)); err != nil {
		return fmt.Errorf("failed to ensure reviewer key: %w", err)
	}

	// Ensure process key exists for this assessment
	processID := fmt.Sprintf("assessment-%d", response.AssessmentID)
	if err := s.ensureProcessKey(processID); err != nil {
		return fmt.Errorf("failed to ensure process key: %w", err)
	}

	// Encrypt justification if provided
	if response.Justification != "" {
		data := &securestore.PlainData{
			Fields: map[string]interface{}{
				"justification": response.Justification,
			},
			Metadata: map[string]string{
				"assessment_id": fmt.Sprintf("%d", response.AssessmentID),
				"category_id":   fmt.Sprintf("%d", response.CategoryID),
				"reviewer_id":   fmt.Sprintf("%d", reviewerUserID),
			},
		}

		record, err := s.secureStore.CreateRecord(
			processID,
			int64(reviewerUserID),
			"REVIEWER_JUSTIFICATION",
			data,
			"",
		)
		if err != nil {
			return fmt.Errorf("failed to encrypt justification: %w", err)
		}

		response.EncryptedJustificationID = &record.ID
		response.Justification = "" // Clear plaintext
	}

	response.ReviewerUserID = reviewerUserID

	// Transition from submitted to in_review on any reviewer response
	if assessment.Status == "submitted" {
		if err := s.assessmentRepo.UpdateStatus(response.AssessmentID, "in_review"); err != nil {
			slog.Error("Failed to update assessment status to in_review", "error", err, "assessment_id", response.AssessmentID)
			// Don't fail the response creation if status update fails
		} else {
			slog.Info("Automatically transitioned assessment to in_review", "assessment_id", response.AssessmentID, "reviewer_id", reviewerUserID)
		}
	}

	return s.reviewerRepo.CreateOrUpdate(response)
}

// GetResponsesByAssessment retrieves reviewer responses for an assessment
// Only returns responses from the specified reviewer (no admin override - strict role separation)
func (s *ReviewerService) GetResponsesByAssessment(assessmentID, reviewerUserID uint) ([]models.ReviewerResponse, error) {
	var responses []models.ReviewerResponse
	var err error

	// Always filter by reviewer - no exceptions
	responses, err = s.reviewerRepo.GetByAssessmentAndReviewer(assessmentID, reviewerUserID)

	if err != nil {
		return nil, err
	}

	// Decrypt justifications
	for i := range responses {
		if err := s.decryptJustification(&responses[i]); err != nil {
			slog.Error("Failed to decrypt justification", "error", err, "response_id", responses[i].ID)
			// Continue with other responses even if one fails
		}
	}

	return responses, nil
}

// GetResponseByCategory retrieves a specific reviewer response
func (s *ReviewerService) GetResponseByCategory(assessmentID, categoryID, reviewerUserID uint) (*models.ReviewerResponse, error) {
	response, err := s.reviewerRepo.GetByCategoryAndReviewer(assessmentID, categoryID, reviewerUserID)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, nil
	}

	// Decrypt justification
	if err := s.decryptJustification(response); err != nil {
		return nil, fmt.Errorf("failed to decrypt justification: %w", err)
	}

	return response, nil
}

// DeleteResponse deletes a reviewer response
func (s *ReviewerService) DeleteResponse(assessmentID, categoryID, reviewerUserID uint) error {
	// Get the response to find encrypted_justification_id
	response, err := s.reviewerRepo.GetByCategoryAndReviewer(assessmentID, categoryID, reviewerUserID)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("reviewer response not found")
	}

	// Delete encrypted record if exists (note: CASCADE on FK will handle this)
	// We just log if manual deletion fails
	if response.EncryptedJustificationID != nil {
		// Note: The encrypted_records will be deleted by CASCADE constraint
		slog.Info("Encrypted justification will be deleted by CASCADE", "record_id", *response.EncryptedJustificationID)
	}

	return s.reviewerRepo.Delete(assessmentID, categoryID, reviewerUserID)
}

// GetCompletionStatus returns the review completion status for an assessment
func (s *ReviewerService) GetCompletionStatus(assessmentID uint) (*models.ReviewCompletionStatus, error) {
	completeReviewers, err := s.reviewerRepo.GetCompleteReviewers(assessmentID)
	if err != nil {
		return nil, err
	}

	totalReviewers, err := s.reviewerRepo.CountTotalReviewers(assessmentID)
	if err != nil {
		return nil, err
	}

	return &models.ReviewCompletionStatus{
		TotalReviewers:               totalReviewers,
		CompleteReviews:              len(completeReviewers),
		CanConsolidate:               len(completeReviewers) >= 3,
		ReviewersWithCompleteReviews: completeReviewers,
	}, nil
}

// ValidateReviewerResponse validates a reviewer response
func (s *ReviewerService) ValidateReviewerResponse(assessmentID, categoryID, pathID, levelID uint, justification string) error {
	// Get user's response to compare
	userResponse, err := s.responseRepo.GetByAssessmentAndCategory(assessmentID, categoryID)
	if err != nil {
		return fmt.Errorf("failed to get user response: %w", err)
	}
	if userResponse == nil {
		return fmt.Errorf("no user response found for category")
	}

	// Check if justification is required (when path OR level differs)
	requiresJustification := (pathID != userResponse.PathID) || (levelID != userResponse.LevelID)

	if requiresJustification && len(justification) < 50 {
		return fmt.Errorf("justification must be at least 50 characters when deviating from user's selection")
	}

	return nil
}

// IsReviewComplete checks if a reviewer has completed all categories
func (s *ReviewerService) IsReviewComplete(assessmentID, reviewerUserID uint) (bool, error) {
	// Get total categories with user responses
	userResponses, err := s.responseRepo.GetByAssessmentID(assessmentID)
	if err != nil {
		return false, err
	}
	totalCategories := len(userResponses)

	// Get reviewer's response count
	reviewerCount, err := s.reviewerRepo.CountByAssessmentAndReviewer(assessmentID, reviewerUserID)
	if err != nil {
		return false, err
	}

	return reviewerCount >= totalCategories, nil
}

// CanTransitionToConsolidation checks if an assessment can move to review_consolidation status
func (s *ReviewerService) CanTransitionToConsolidation(assessmentID uint) (bool, error) {
	completeReviews, err := s.reviewerRepo.CountCompleteReviews(assessmentID)
	if err != nil {
		return false, err
	}
	return completeReviews >= 3, nil
}

// Helper methods

func (s *ReviewerService) ensureUserKey(userID int64) error {
	// Check if user key already exists
	_, err := s.keyManager.GetUserPublicKey(userID)
	if err == nil {
		return nil // Key exists
	}

	// Create new user key
	_, err = s.keyManager.CreateUserKey(userID)
	return err
}

func (s *ReviewerService) ensureProcessKey(processID string) error {
	// Check if process key already exists
	_, err := s.keyManager.GetProcessKeyHash(processID)
	if err == nil {
		return nil // Key exists
	}

	// Create new process key (no expiration)
	return s.keyManager.CreateProcessKey(processID, nil)
}

func (s *ReviewerService) decryptJustification(response *models.ReviewerResponse) error {
	if response.EncryptedJustificationID == nil {
		return nil
	}

	plainData, err := s.secureStore.DecryptRecord(*response.EncryptedJustificationID)
	if err != nil {
		return err
	}

	if justification, ok := plainData.Fields["justification"].(string); ok {
		response.Justification = justification
	}

	return nil
}
