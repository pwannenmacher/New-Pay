package service

import (
	"math"
	"new-pay/internal/models"
)

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// removeString removes a specific string from a slice
func removeString(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// findLevelByID finds a level by ID in catalog
func findLevelByID(catalog *models.CatalogWithDetails, levelID uint) *models.Level {
	for i := range catalog.Levels {
		if catalog.Levels[i].ID == levelID {
			return &catalog.Levels[i]
		}
	}
	return nil
}

// findLevelNumber finds the level number for a given level ID
func findLevelNumber(catalog *models.CatalogWithDetails, levelID uint) int {
	for _, level := range catalog.Levels {
		if level.ID == levelID {
			return level.LevelNumber
		}
	}
	return 0
}

// findClosestLevelName finds the closest level name for an average level number
func findClosestLevelName(catalog *models.CatalogWithDetails, avgNumber float64) string {
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

// calculateAveragedResponses calculates averaged reviewer responses per category.
// Only includes reviewers who have completed ALL categories.
// If includeJustifications is true, collects all justifications for each category.
func calculateAveragedResponses(reviewerResponses []models.ReviewerResponse, catalog *models.CatalogWithDetails, includeJustifications bool) []models.AveragedReviewerResponse {
	// Store category info from catalog
	categoryInfo := make(map[uint]struct {
		Name      string
		SortOrder int
	})
	totalCategories := len(catalog.Categories)

	for _, cat := range catalog.Categories {
		categoryInfo[cat.ID] = struct {
			Name      string
			SortOrder int
		}{
			Name:      cat.Name,
			SortOrder: cat.SortOrder,
		}
	}

	// First, group responses by reviewer to check completeness
	reviewerCategoryMap := make(map[uint]map[uint]models.ReviewerResponse) // [reviewerID][categoryID]response

	for _, resp := range reviewerResponses {
		if reviewerCategoryMap[resp.ReviewerUserID] == nil {
			reviewerCategoryMap[resp.ReviewerUserID] = make(map[uint]models.ReviewerResponse)
		}
		reviewerCategoryMap[resp.ReviewerUserID][resp.CategoryID] = resp
	}

	// Filter reviewers who completed ALL categories
	completeReviewers := make(map[uint]bool)
	for reviewerID, categories := range reviewerCategoryMap {
		if len(categories) == totalCategories {
			completeReviewers[reviewerID] = true
		}
	}

	// Now group by category, but only include responses from complete reviewers
	categoryMap := make(map[uint][]models.ReviewerResponse)
	for _, resp := range reviewerResponses {
		if completeReviewers[resp.ReviewerUserID] {
			categoryMap[resp.CategoryID] = append(categoryMap[resp.CategoryID], resp)
		}
	}

	var averaged []models.AveragedReviewerResponse
	for categoryID, responses := range categoryMap {
		if len(responses) == 0 {
			continue
		}

		// Calculate average level number
		var sum float64
		var justifications []string

		for _, resp := range responses {
			// Find level number from catalog
			levelNumber := findLevelNumber(catalog, resp.LevelID)
			sum += float64(levelNumber)

			// Collect justifications if requested and present
			if includeJustifications && resp.Justification != "" {
				justifications = append(justifications, resp.Justification)
			}
		}

		avgLevelNumber := sum / float64(len(responses))

		// Find closest level name
		avgLevelName := findClosestLevelName(catalog, avgLevelNumber)

		info := categoryInfo[categoryID]
		avgResponse := models.AveragedReviewerResponse{
			CategoryID:         categoryID,
			CategoryName:       info.Name,
			CategorySortOrder:  info.SortOrder,
			AverageLevelNumber: math.Round(avgLevelNumber*100) / 100, // Round to 2 decimals
			AverageLevelName:   avgLevelName,
			ReviewerCount:      len(responses), // Number of complete reviewers who rated this category
		}

		if includeJustifications {
			avgResponse.ReviewerJustifications = justifications
		}

		averaged = append(averaged, avgResponse)
	}

	return averaged
}
