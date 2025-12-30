package service

import (
	"fmt"
	"log/slog"
	"math"

	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/securestore"
)

type DiscussionService struct {
	discussionRepo         *repository.DiscussionRepository
	assessmentRepo         *repository.SelfAssessmentRepository
	reviewerRespRepo       *repository.ReviewerResponseRepository
	responseRepo           *repository.AssessmentResponseRepository
	overrideRepo           *repository.ConsolidationOverrideRepository
	finalConsRepo          *repository.FinalConsolidationRepository
	catalogRepo            *repository.CatalogRepository
	userRepo               *repository.UserRepository
	categoryDiscussionRepo *repository.CategoryDiscussionCommentRepository
	confirmationRepo       *repository.DiscussionConfirmationRepository
	secureStore            *securestore.SecureStore
}

func NewDiscussionService(
	discussionRepo *repository.DiscussionRepository,
	assessmentRepo *repository.SelfAssessmentRepository,
	reviewerRespRepo *repository.ReviewerResponseRepository,
	responseRepo *repository.AssessmentResponseRepository,
	overrideRepo *repository.ConsolidationOverrideRepository,
	finalConsRepo *repository.FinalConsolidationRepository,
	catalogRepo *repository.CatalogRepository,
	userRepo *repository.UserRepository,
	categoryDiscussionRepo *repository.CategoryDiscussionCommentRepository,
	confirmationRepo *repository.DiscussionConfirmationRepository,
	secureStore *securestore.SecureStore,
) *DiscussionService {
	return &DiscussionService{
		discussionRepo:         discussionRepo,
		assessmentRepo:         assessmentRepo,
		reviewerRespRepo:       reviewerRespRepo,
		responseRepo:           responseRepo,
		overrideRepo:           overrideRepo,
		finalConsRepo:          finalConsRepo,
		catalogRepo:            catalogRepo,
		userRepo:               userRepo,
		categoryDiscussionRepo: categoryDiscussionRepo,
		confirmationRepo:       confirmationRepo,
		secureStore:            secureStore,
	}
}

// Helper functions

// decryptSecureStoreField decrypts a field from secure store
func (s *DiscussionService) decryptSecureStoreField(recordID int64, fieldName string) (string, error) {
	plainData, err := s.secureStore.DecryptRecord(recordID)
	if err != nil {
		return "", err
	}
	if value, ok := plainData.Fields[fieldName].(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("field %s not found or not a string", fieldName)
}

// findCategoryName finds a category name by ID in catalog
func (s *DiscussionService) findCategoryName(catalog *models.CatalogWithDetails, categoryID uint) string {
	for i := range catalog.Categories {
		if catalog.Categories[i].ID == categoryID {
			return catalog.Categories[i].Name
		}
	}
	return ""
}

// populateLevelNames populates level names in category results
func (s *DiscussionService) populateLevelNames(categoryResults []models.DiscussionCategoryResult, catalog *models.CatalogWithDetails) {
	for i := range categoryResults {
		// Find category name
		categoryResults[i].CategoryName = s.findCategoryName(catalog, categoryResults[i].CategoryID)

		// Find user level name
		if categoryResults[i].UserLevelID != nil {
			if level := findLevelByID(catalog, *categoryResults[i].UserLevelID); level != nil {
				categoryResults[i].UserLevelName = level.Name
			}
		}

		// Find reviewer level name
		if level := findLevelByID(catalog, categoryResults[i].ReviewerLevelID); level != nil {
			categoryResults[i].ReviewerLevelName = level.Name
		}
	}
}

// CreateDiscussionResult generates and stores discussion results when status changes to 'discussion'
func (s *DiscussionService) CreateDiscussionResult(assessmentID uint) error {
	// Check if discussion result already exists
	existing, err := s.discussionRepo.GetByAssessmentID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to check existing discussion result: %w", err)
	}
	if existing != nil {
		slog.Info("Discussion result already exists", "assessment_id", assessmentID)
		return nil // Already created
	}

	// Get assessment
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}

	// Get catalog with categories
	catalog, err := s.catalogRepo.GetCatalogWithDetails(assessment.CatalogID)
	if err != nil {
		return fmt.Errorf("failed to get catalog: %w", err)
	}
	if catalog == nil {
		return fmt.Errorf("catalog not found")
	}

	// Get user responses
	userResponses, err := s.responseRepo.GetAllByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get user responses: %w", err)
	}

	// Get overrides
	overrides, err := s.overrideRepo.GetByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get overrides: %w", err)
	}

	// Get all reviewer responses for averaged calculations
	allReviewerResponses, err := s.reviewerRespRepo.GetAllByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get reviewer responses: %w", err)
	}

	// Decrypt reviewer justifications for averaged responses
	for i := range allReviewerResponses {
		if allReviewerResponses[i].EncryptedJustificationID != nil {
			justification, err := s.decryptSecureStoreField(*allReviewerResponses[i].EncryptedJustificationID, "justification")
			if err != nil {
				slog.Warn("Failed to decrypt reviewer justification", "error", err, "record_id", *allReviewerResponses[i].EncryptedJustificationID)
				continue
			}
			allReviewerResponses[i].Justification = justification
		}
	}

	// Calculate averaged responses per category (with justifications for discussion freezing)
	averagedResponses := calculateAveragedResponses(allReviewerResponses, catalog, true)

	// Get final consolidation
	finalCons, err := s.finalConsRepo.GetByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get final consolidation: %w", err)
	}
	if finalCons == nil {
		return fmt.Errorf("final consolidation not found")
	}

	// Decrypt final comment
	if finalCons.EncryptedCommentID != nil {
		comment, err := s.decryptSecureStoreField(*finalCons.EncryptedCommentID, "comment")
		if err != nil {
			return fmt.Errorf("failed to decrypt comment: %w", err)
		}
		finalCons.Comment = comment
	}

	// Get all reviewers who completed reviews
	completedReviewers, err := s.reviewerRespRepo.GetCompleteReviewers(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get completed reviewers: %w", err)
	}

	// Get category discussion comments
	categoryComments, err := s.categoryDiscussionRepo.GetByAssessment(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get category discussion comments: %w", err)
	}

	// Decrypt category comments
	categoryCommentMap := make(map[uint]string)
	for _, comment := range categoryComments {
		if comment.EncryptedCommentID != nil {
			commentText, err := s.decryptSecureStoreField(*comment.EncryptedCommentID, "comment")
			if err != nil {
				slog.Warn("Failed to decrypt category discussion comment", "error", err, "record_id", *comment.EncryptedCommentID)
				continue
			}
			categoryCommentMap[comment.CategoryID] = commentText
		}
	}

	// Calculate weighted overall level and category results
	var totalWeight float64
	var weightedSum float64
	var categoryResults []models.DiscussionCategoryResult

	for _, category := range catalog.Categories {
		// Get user's level for this category
		var userLevelID *uint
		for _, userResp := range userResponses {
			if userResp.CategoryID == category.ID {
				userLevelID = &userResp.LevelID
				break
			}
		}

		// Determine reviewer level (override takes precedence)
		var reviewerLevelID uint
		var reviewerLevelNumber float64
		var encryptedJustificationID *int64
		var isOverride bool

		// Check for override first
		var override *models.ConsolidationOverride
		for i := range overrides {
			if overrides[i].CategoryID == category.ID {
				override = &overrides[i]
				break
			}
		}

		if override != nil {
			// Use override
			reviewerLevelID = override.LevelID
			isOverride = true

			// Find level number
			if level := findLevelByID(catalog, override.LevelID); level != nil {
				reviewerLevelNumber = float64(level.LevelNumber)
			}
		} else {
			// Use averaged response
			for _, avg := range averagedResponses {
				if avg.CategoryID == category.ID {
					// Find the level closest to average_level_number
					reviewerLevelNumber = avg.AverageLevelNumber

					var closestLevel *models.Level
					minDiff := math.MaxFloat64
					for _, level := range catalog.Levels {
						diff := math.Abs(float64(level.LevelNumber) - avg.AverageLevelNumber)
						if diff < minDiff {
							minDiff = diff
							closestLevel = &level
						}
					}
					if closestLevel != nil {
						reviewerLevelID = closestLevel.ID
					}
					break
				}
			}
		}

		// Encrypt category discussion comment if available (public comment)
		if publicComment, exists := categoryCommentMap[category.ID]; exists && publicComment != "" {
			processID := fmt.Sprintf("assessment-%d", assessmentID)
			plainData := securestore.PlainData{
				Fields: map[string]interface{}{
					"justification": publicComment,
				},
			}

			record, err := s.secureStore.CreateRecord(
				processID,
				int64(assessment.UserID),
				"CATEGORY_JUSTIFICATION",
				&plainData,
				"",
			)
			if err != nil {
				return fmt.Errorf("failed to encrypt category justification: %w", err)
			}
			encryptedJustificationID = &record.ID
		}

		// Add to weighted calculation
		weight := float64(1)
		if category.Weight != nil && *category.Weight > 0 {
			weight = *category.Weight
		}
		totalWeight += weight
		weightedSum += reviewerLevelNumber * weight

		// Create category result
		categoryResults = append(categoryResults, models.DiscussionCategoryResult{
			CategoryID:               category.ID,
			UserLevelID:              userLevelID,
			ReviewerLevelID:          reviewerLevelID,
			ReviewerLevelNumber:      reviewerLevelNumber,
			EncryptedJustificationID: encryptedJustificationID,
			IsOverride:               isOverride,
		})
	}

	// Calculate overall weighted level
	averageLevel := weightedSum / totalWeight

	// Find the level closest to the average
	var overallLevelID uint
	minDiff := math.MaxFloat64
	for _, level := range catalog.Levels {
		diff := math.Abs(float64(level.LevelNumber) - averageLevel)
		if diff < minDiff {
			minDiff = diff
			overallLevelID = level.ID
		}
	}

	// Encrypt final comment using secureStore
	processID := fmt.Sprintf("assessment-%d", assessmentID)
	plainData := &securestore.PlainData{
		Fields: map[string]interface{}{
			"comment": finalCons.Comment,
		},
		Metadata: map[string]string{
			"assessment_id": fmt.Sprintf("%d", assessmentID),
			"type":          "final_discussion_comment",
		},
	}

	record, err := s.secureStore.CreateRecord(
		processID,
		int64(assessment.UserID),
		"FINAL_DISCUSSION_COMMENT",
		plainData,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to encrypt final comment: %w", err)
	}

	// Create discussion result
	discussionResult := &models.DiscussionResult{
		AssessmentID:            assessmentID,
		WeightedOverallLevelNum: averageLevel,
		WeightedOverallLevelID:  overallLevelID,
		EncryptedFinalCommentID: &record.ID,
	}

	if err := s.discussionRepo.Create(discussionResult); err != nil {
		return fmt.Errorf("failed to create discussion result: %w", err)
	}

	// Create category results
	for i := range categoryResults {
		categoryResults[i].DiscussionResultID = discussionResult.ID
		if err := s.discussionRepo.CreateCategoryResult(&categoryResults[i]); err != nil {
			return fmt.Errorf("failed to create category result: %w", err)
		}
	}

	// Create reviewer records
	for _, reviewer := range completedReviewers {
		user, err := s.userRepo.GetByID(reviewer.ReviewerID)
		if err != nil {
			slog.Warn("Failed to get reviewer user", "user_id", reviewer.ReviewerID, "error", err)
			continue
		}

		reviewerRecord := &models.DiscussionReviewer{
			DiscussionResultID: discussionResult.ID,
			ReviewerUserID:     reviewer.ReviewerID,
			ReviewerName:       fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		}
		if err := s.discussionRepo.CreateReviewer(reviewerRecord); err != nil {
			slog.Warn("Failed to create reviewer record", "error", err)
		}
	}

	slog.Info("Discussion result created successfully", "assessment_id", assessmentID, "discussion_result_id", discussionResult.ID)
	return nil
}

// GetDiscussionResult retrieves discussion result with all data
func (s *DiscussionService) GetDiscussionResult(assessmentID uint) (*models.DiscussionResult, error) {
	result, err := s.discussionRepo.GetByAssessmentID(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discussion result: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	// Get assessment to get catalog ID
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return nil, fmt.Errorf("assessment not found")
	}

	// Set assessment status in result
	result.AssessmentStatus = assessment.Status

	// Get catalog with levels and categories
	catalog, err := s.catalogRepo.GetCatalogWithDetails(assessment.CatalogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}
	if catalog == nil {
		return nil, fmt.Errorf("catalog not found")
	}

	// Decrypt final comment from secure store
	if result.EncryptedFinalCommentID != nil {
		comment, err := s.decryptSecureStoreField(*result.EncryptedFinalCommentID, "comment")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt final comment: %w", err)
		}
		result.FinalComment = comment

	}

	// Decrypt discussion note from secure store
	if result.EncryptedDiscussionNoteID != nil {
		note, err := s.decryptSecureStoreField(*result.EncryptedDiscussionNoteID, "note")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt discussion note: %w", err)
		}
		result.DiscussionNote = note
	}

	// Set weighted overall level name
	if level := findLevelByID(catalog, result.WeightedOverallLevelID); level != nil {
		result.WeightedOverallLevelName = level.Name
	}

	// Get category results
	categoryResults, err := s.discussionRepo.GetCategoryResults(result.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category results: %w", err)
	}

	// Decrypt justifications from secure store
	for i := range categoryResults {
		if categoryResults[i].EncryptedJustificationID != nil {
			justification, err := s.decryptSecureStoreField(*categoryResults[i].EncryptedJustificationID, "justification")
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt justification: %w", err)
			}
			categoryResults[i].Justification = justification
		}
	}

	// Populate category and level names
	s.populateLevelNames(categoryResults, catalog)

	result.CategoryResults = categoryResults

	// Get reviewers
	reviewers, err := s.discussionRepo.GetReviewers(result.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}
	result.Reviewers = reviewers

	// Get confirmations
	confirmations, err := s.confirmationRepo.GetByAssessment(assessmentID)
	if err != nil {
		slog.Error("Failed to get discussion confirmations", "error", err)
		confirmations = []models.DiscussionConfirmation{} // Empty slice on error
	}

	// Populate user names for confirmations
	for i := range confirmations {
		user, err := s.userRepo.GetByID(confirmations[i].UserID)
		if err == nil && user != nil {
			confirmations[i].UserName = user.FirstName + " " + user.LastName
			confirmations[i].UserEmail = user.Email
		}
	}

	result.Confirmations = confirmations

	return result, nil
}

// UpdateDiscussionNote updates the discussion note
func (s *DiscussionService) UpdateDiscussionNote(assessmentID uint, note string) error {
	// Get assessment to check status
	assessment, err := s.assessmentRepo.GetByID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("assessment not found")
	}

	// Prevent changes if assessment is archived
	if assessment.Status == "archived" {
		return fmt.Errorf("cannot modify notes: assessment is archived")
	}

	result, err := s.discussionRepo.GetByAssessmentID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get discussion result: %w", err)
	}
	if result == nil {
		return fmt.Errorf("discussion result not found")
	}

	// Encrypt note using secure store if note is not empty
	var encryptedNoteID *int64
	if note != "" {
		processID := fmt.Sprintf("assessment-%d", assessmentID)
		plainData := securestore.PlainData{
			Fields: map[string]interface{}{
				"note": note,
			},
		}

		record, err := s.secureStore.CreateRecord(
			processID,
			int64(assessment.UserID),
			"DISCUSSION_NOTE",
			&plainData,
			"",
		)
		if err != nil {
			return fmt.Errorf("failed to encrypt discussion note: %w", err)
		}
		encryptedNoteID = &record.ID
	}

	return s.discussionRepo.UpdateDiscussionNote(result.ID, encryptedNoteID, nil)
}
