package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"new-pay/internal/middleware"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/service"
)

// ReviewerHandler handles reviewer-related HTTP requests
type ReviewerHandler struct {
	reviewerService *service.ReviewerService
	assessmentRepo  *repository.SelfAssessmentRepository
}

// NewReviewerHandler creates a new reviewer handler
func NewReviewerHandler(
	reviewerService *service.ReviewerService,
	assessmentRepo *repository.SelfAssessmentRepository,
) *ReviewerHandler {
	return &ReviewerHandler{
		reviewerService: reviewerService,
		assessmentRepo:  assessmentRepo,
	}
}

// GetResponses retrieves reviewer responses for an assessment
// GET /api/v1/review/assessment/:id/responses
func (h *ReviewerHandler) GetResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get assessment ID from URL
	assessmentID, err := extractAssessmentID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is trying to review their own assessment
	assessment, err := h.assessmentRepo.GetByID(uint(assessmentID))
	if err != nil {
		slog.Error("Failed to get assessment", "error", err)
		http.Error(w, "Failed to get assessment", http.StatusInternalServerError)
		return
	}
	if assessment == nil {
		http.Error(w, "Assessment not found", http.StatusNotFound)
		return
	}

	// Prevent self-review
	if assessment.UserID == userID {
		http.Error(w, "Cannot review your own assessment", http.StatusForbidden)
		return
	}

	// Get responses (always filtered by reviewer - no admin override)
	responses, err := h.reviewerService.GetResponsesByAssessment(uint(assessmentID), userID)
	if err != nil {
		slog.Error("Failed to get reviewer responses", "error", err)
		http.Error(w, "Failed to get responses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// CreateOrUpdateResponse creates or updates a reviewer response
// POST /api/v1/review/assessment/:id/responses
func (h *ReviewerHandler) CreateOrUpdateResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get assessment ID from URL
	assessmentID, err := extractAssessmentID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is trying to review their own assessment
	assessment, err := h.assessmentRepo.GetByID(uint(assessmentID))
	if err != nil {
		slog.Error("Failed to get assessment", "error", err)
		http.Error(w, "Failed to get assessment", http.StatusInternalServerError)
		return
	}
	if assessment == nil {
		http.Error(w, "Assessment not found", http.StatusNotFound)
		return
	}

	// Prevent self-review
	if assessment.UserID == userID {
		http.Error(w, "Cannot review your own assessment", http.StatusForbidden)
		return
	}

	// Parse request body
	var req struct {
		CategoryID    uint   `json:"category_id"`
		PathID        uint   `json:"path_id"`
		LevelID       uint   `json:"level_id"`
		Justification string `json:"justification"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CategoryID == 0 || req.PathID == 0 || req.LevelID == 0 {
		http.Error(w, "category_id, path_id, and level_id are required", http.StatusBadRequest)
		return
	}

	// Validate justification requirements
	if err := h.reviewerService.ValidateReviewerResponse(
		uint(assessmentID),
		req.CategoryID,
		req.PathID,
		req.LevelID,
		req.Justification,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create response
	response := &models.ReviewerResponse{
		AssessmentID:  uint(assessmentID),
		CategoryID:    req.CategoryID,
		PathID:        req.PathID,
		LevelID:       req.LevelID,
		Justification: req.Justification,
	}

	if err := h.reviewerService.CreateOrUpdateResponse(response, userID); err != nil {
		slog.Error("Failed to create/update reviewer response", "error", err)
		http.Error(w, "Failed to save response", http.StatusInternalServerError)
		return
	}

	// Load the saved response with decrypted justification
	savedResponse, err := h.reviewerService.GetResponseByCategory(uint(assessmentID), req.CategoryID, userID)
	if err != nil {
		slog.Error("Failed to load saved response", "error", err)
		http.Error(w, "Response saved but failed to reload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(savedResponse)
}

// DeleteResponse deletes a reviewer response
// DELETE /api/v1/review/assessment/:id/responses/:category_id
func (h *ReviewerHandler) DeleteResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get assessment ID and category ID from URL
	assessmentID, categoryID, err := extractAssessmentAndCategoryID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid URL parameters", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is trying to review their own assessment
	assessment, err := h.assessmentRepo.GetByID(uint(assessmentID))
	if err != nil {
		slog.Error("Failed to get assessment", "error", err)
		http.Error(w, "Failed to get assessment", http.StatusInternalServerError)
		return
	}
	if assessment == nil {
		http.Error(w, "Assessment not found", http.StatusNotFound)
		return
	}

	// Prevent self-review
	if assessment.UserID == userID {
		http.Error(w, "Cannot review your own assessment", http.StatusForbidden)
		return
	}

	// Delete response
	if err := h.reviewerService.DeleteResponse(uint(assessmentID), uint(categoryID), userID); err != nil {
		slog.Error("Failed to delete reviewer response", "error", err)
		http.Error(w, "Failed to delete response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Reviewer response deleted successfully",
	})
}

// CompleteReview marks a reviewer's review as complete and optionally changes assessment status
// POST /api/v1/review/assessment/:id/complete
func (h *ReviewerHandler) CompleteReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get assessment ID from URL
	assessmentID, err := extractAssessmentID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is trying to review their own assessment
	assessment, err := h.assessmentRepo.GetByID(uint(assessmentID))
	if err != nil {
		slog.Error("Failed to get assessment", "error", err)
		http.Error(w, "Failed to get assessment", http.StatusInternalServerError)
		return
	}
	if assessment == nil {
		http.Error(w, "Assessment not found", http.StatusNotFound)
		return
	}

	// Prevent self-review
	if assessment.UserID == userID {
		http.Error(w, "Cannot review your own assessment", http.StatusForbidden)
		return
	}

	// Check if review is complete
	isComplete, err := h.reviewerService.IsReviewComplete(uint(assessmentID), userID)
	if err != nil {
		slog.Error("Failed to check review completeness", "error", err)
		http.Error(w, "Failed to check review status", http.StatusInternalServerError)
		return
	}

	if !isComplete {
		http.Error(w, "Review is incomplete. All categories must have responses.", http.StatusBadRequest)
		return
	}

	// Parse request body for new status (optional)
	var req struct {
		NewStatus string `json:"new_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.NewStatus != "" {
		// Validate status transition
		if req.NewStatus == "review_consolidation" {
			canConsolidate, err := h.reviewerService.CanTransitionToConsolidation(uint(assessmentID))
			if err != nil {
				slog.Error("Failed to check consolidation eligibility", "error", err)
				http.Error(w, "Failed to check consolidation status", http.StatusInternalServerError)
				return
			}
			if !canConsolidate {
				http.Error(w, "Cannot transition to review_consolidation: at least 3 complete reviews required", http.StatusBadRequest)
				return
			}
		}

		// Update assessment status (only reviewers can do this, not admins)
		// Note: Status changes should be controlled by review workflow, not admin override
		if err := h.assessmentRepo.UpdateStatus(uint(assessmentID), req.NewStatus); err != nil {
			slog.Error("Failed to update assessment status", "error", err)
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
			return
		}
		assessment.Status = req.NewStatus
	}

	response := map[string]interface{}{
		"message": "Review completed successfully",
		"assessment": map[string]interface{}{
			"id":     assessment.ID,
			"status": assessment.Status,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetCompletionStatus returns the review completion status for an assessment
// GET /api/v1/review/assessment/:id/completion-status
func (h *ReviewerHandler) GetCompletionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get assessment ID from URL
	assessmentID, err := extractAssessmentID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	status, err := h.reviewerService.GetCompletionStatus(uint(assessmentID))
	if err != nil {
		slog.Error("Failed to get completion status", "error", err)
		http.Error(w, "Failed to get completion status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Helper functions

func extractAssessmentID(path string) (int, error) {
	// Extract ID from path like "/api/v1/review/assessment/123/responses"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "assessment" && i+1 < len(parts) {
			return strconv.Atoi(parts[i+1])
		}
	}
	return 0, http.ErrAbortHandler
}

func extractAssessmentAndCategoryID(path string) (int, int, error) {
	// Extract IDs from path like "/api/v1/review/assessment/123/responses/456"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var assessmentID, categoryID int
	var err error

	for i, part := range parts {
		if part == "assessment" && i+1 < len(parts) {
			assessmentID, err = strconv.Atoi(parts[i+1])
			if err != nil {
				return 0, 0, err
			}
		}
		if part == "responses" && i+1 < len(parts) {
			categoryID, err = strconv.Atoi(parts[i+1])
			if err != nil {
				return 0, 0, err
			}
		}
	}

	if assessmentID == 0 || categoryID == 0 {
		return 0, 0, http.ErrAbortHandler
	}

	return assessmentID, categoryID, nil
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
