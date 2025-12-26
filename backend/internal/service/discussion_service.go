package service

import (
	"fmt"
	"log/slog"
	"math"

	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/securestore"
	"new-pay/internal/vault"
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
			plainData, err := s.secureStore.DecryptRecord(*allReviewerResponses[i].EncryptedJustificationID)
			if err != nil {
				slog.Warn("Failed to decrypt reviewer justification", "error", err, "record_id", *allReviewerResponses[i].EncryptedJustificationID)
				continue
			}
			if justification, ok := plainData.Fields["justification"].(string); ok {
				allReviewerResponses[i].Justification = justification
			}
		}
	}

	// Calculate averaged responses per category
	averagedResponses := s.calculateAveragedResponses(allReviewerResponses, catalog)

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
		plainData, err := s.secureStore.DecryptRecord(*finalCons.EncryptedCommentID)
		if err != nil {
			return fmt.Errorf("failed to decrypt comment: %w", err)
		}
		if comment, ok := plainData.Fields["comment"].(string); ok {
			finalCons.Comment = comment
		}
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
			plainData, err := s.secureStore.DecryptRecord(*comment.EncryptedCommentID)
			if err != nil {
				slog.Warn("Failed to decrypt category discussion comment", "error", err, "record_id", *comment.EncryptedCommentID)
				continue
			}
			if commentText, ok := plainData.Fields["comment"].(string); ok {
				categoryCommentMap[comment.CategoryID] = commentText
			}
		}
	}

	// Calculate weighted overall level and category results
	var totalWeight float64
	var weightedSum float64
	categoryResults := []models.DiscussionCategoryResult{}

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
		var justificationPlain *string
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
			for _, level := range catalog.Levels {
				if level.ID == override.LevelID {
					reviewerLevelNumber = float64(level.LevelNumber)
					break
				}
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

		// Use category discussion comment if available (public comment), otherwise internal notes
		if publicComment, exists := categoryCommentMap[category.ID]; exists {
			justificationPlain = &publicComment
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
			CategoryID:          category.ID,
			UserLevelID:         userLevelID,
			ReviewerLevelID:     reviewerLevelID,
			ReviewerLevelNumber: reviewerLevelNumber,
			JustificationPlain:  justificationPlain,
			IsOverride:          isOverride,
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

	// Encrypt final comment using simple encryption
	// Use assessment ID as additional context
	additionalData := []byte(fmt.Sprintf("discussion:assessment:%d", assessmentID))

	// Derive a key from the assessment ID (simple approach for discussion data)
	// In production, this should use proper key management
	keyMaterial := []byte(fmt.Sprintf("discussion-key-%d", assessmentID))
	encryptionKey := vault.DeriveKey(keyMaterial, nil, "discussion-final-comment", 32)

	encryptedData, nonce, err := vault.EncryptLocal([]byte(finalCons.Comment), encryptionKey, additionalData)
	if err != nil {
		return fmt.Errorf("failed to encrypt final comment: %w", err)
	}

	// Create discussion result
	discussionResult := &models.DiscussionResult{
		AssessmentID:            assessmentID,
		WeightedOverallLevelNum: averageLevel,
		WeightedOverallLevelID:  overallLevelID,
		FinalCommentEncrypted:   encryptedData,
		FinalCommentNonce:       nonce,
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

// calculateAveragedResponses calculates averaged reviewer responses per category
func (s *DiscussionService) calculateAveragedResponses(reviewerResponses []models.ReviewerResponse, catalog *models.CatalogWithDetails) []models.AveragedReviewerResponse {
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
		justifications := []string{}

		for _, resp := range responses {
			// Find level number from catalog
			levelNumber := s.findLevelNumber(catalog, resp.LevelID)
			sum += float64(levelNumber)

			// Collect justifications if present
			if resp.Justification != "" {
				justifications = append(justifications, resp.Justification)
			}
		}

		avgLevelNumber := sum / float64(len(responses))

		// Find closest level name
		avgLevelName := s.findClosestLevelName(catalog, avgLevelNumber)

		info := categoryInfo[categoryID]
		averaged = append(averaged, models.AveragedReviewerResponse{
			CategoryID:             categoryID,
			CategoryName:           info.Name,
			CategorySortOrder:      info.SortOrder,
			AverageLevelNumber:     math.Round(avgLevelNumber*100) / 100, // Round to 2 decimals
			AverageLevelName:       avgLevelName,
			ReviewerCount:          len(responses),
			ReviewerJustifications: justifications, // Include all justifications for discussion freezing
		})
	}

	return averaged
}

// findLevelNumber finds the level number for a given level ID
func (s *DiscussionService) findLevelNumber(catalog *models.CatalogWithDetails, levelID uint) int {
	for _, level := range catalog.Levels {
		if level.ID == levelID {
			return level.LevelNumber
		}
	}
	return 0
}

// findClosestLevelName finds the closest level name for an average level number
func (s *DiscussionService) findClosestLevelName(catalog *models.CatalogWithDetails, avgNumber float64) string {
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

	// Get catalog with levels and categories
	catalog, err := s.catalogRepo.GetCatalogWithDetails(assessment.CatalogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	// Decrypt final comment
	additionalData := []byte(fmt.Sprintf("discussion:assessment:%d", assessmentID))
	keyMaterial := []byte(fmt.Sprintf("discussion-key-%d", assessmentID))
	encryptionKey := vault.DeriveKey(keyMaterial, nil, "discussion-final-comment", 32)

	decrypted, err := vault.DecryptLocal(result.FinalCommentEncrypted, encryptionKey, result.FinalCommentNonce, additionalData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt final comment: %w", err)
	}
	result.FinalComment = string(decrypted)

	// Set weighted overall level name
	for _, level := range catalog.Levels {
		if level.ID == result.WeightedOverallLevelID {
			result.WeightedOverallLevelName = level.Name
			break
		}
	}

	// Get category results
	categoryResults, err := s.discussionRepo.GetCategoryResults(result.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category results: %w", err)
	}

	// Populate category and level names
	for i := range categoryResults {
		// Find category name
		for _, category := range catalog.Categories {
			if category.ID == categoryResults[i].CategoryID {
				categoryResults[i].CategoryName = category.Name
				break
			}
		}

		// Find user level name
		if categoryResults[i].UserLevelID != nil {
			for _, level := range catalog.Levels {
				if level.ID == *categoryResults[i].UserLevelID {
					categoryResults[i].UserLevelName = level.Name
					break
				}
			}
		}

		// Find reviewer level name
		for _, level := range catalog.Levels {
			if level.ID == categoryResults[i].ReviewerLevelID {
				categoryResults[i].ReviewerLevelName = level.Name
				break
			}
		}
	}

	// Category results already have plain text justifications stored
	// No need to decrypt - justifications are stored as plain text in discussion results
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
	result, err := s.discussionRepo.GetByAssessmentID(assessmentID)
	if err != nil {
		return fmt.Errorf("failed to get discussion result: %w", err)
	}
	if result == nil {
		return fmt.Errorf("discussion result not found")
	}

	return s.discussionRepo.UpdateDiscussionNote(result.ID, note, nil)
}
