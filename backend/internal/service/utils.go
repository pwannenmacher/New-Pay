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
