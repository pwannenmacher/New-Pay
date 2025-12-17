package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"new-pay/internal/middleware"
	"new-pay/internal/models"
	"new-pay/internal/service"
)

// ConsolidationHandler handles HTTP requests for review consolidation
type ConsolidationHandler struct {
	consolidationService *service.ConsolidationService
}

// NewConsolidationHandler creates a new consolidation handler
func NewConsolidationHandler(consolidationService *service.ConsolidationService) *ConsolidationHandler {
	return &ConsolidationHandler{
		consolidationService: consolidationService,
	}
}

// GetConsolidationData retrieves all data needed for consolidation
// @Summary Get consolidation data
// @Description Retrieves user responses, averaged reviewer responses, and overrides for consolidation
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} models.ConsolidationData
// @Failure 400 {object} map[string]string "Invalid assessment ID"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Assessment not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id} [get]
func (h *ConsolidationHandler) GetConsolidationData(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
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

	data, err := h.consolidationService.GetConsolidationData(uint(assessmentID), userID)
	if err != nil {
		// Check if it's a permission denied error
		if err.Error() == "permission denied: only reviewers who completed their review can access consolidation" {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err.Error() == "assessment not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err.Error() == "assessment must be in review_consolidation status" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, data)
}

// CreateOrUpdateOverride creates or updates a consolidation override
// @Summary Create or update consolidation override
// @Description Creates or updates a manually adjusted value during consolidation
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param override body models.ConsolidationOverride true "Override data"
// @Success 200 {object} models.ConsolidationOverride
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/override [post]
func (h *ConsolidationHandler) CreateOrUpdateOverride(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
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

	var override models.ConsolidationOverride
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure assessment ID matches
	override.AssessmentID = uint(assessmentID)

	if err := h.consolidationService.CreateOrUpdateOverride(&override, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, override)
}

// ApproveOverride approves a consolidation override
// @Summary Approve consolidation override
// @Description Approves a manually adjusted value created by another reviewer
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param categoryId path int true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Override not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/override/{categoryId}/approve [post]
func (h *ConsolidationHandler) ApproveOverride(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	categoryID, err := strconv.ParseUint(r.PathValue("categoryId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.consolidationService.ApproveOverride(uint(assessmentID), uint(categoryID), userID); err != nil {
		// Check error type for appropriate status code
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before approving overrides":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "cannot approve your own override":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "override not found":
			http.Error(w, errMsg, http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Override approved successfully"})
}

// ApproveAveragedResponse approves an averaged reviewer response
// @Summary Approve averaged response
// @Description Approves the averaged reviewer response for a category (when no override exists)
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param categoryId path int true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/averaged/{categoryId}/approve [post]
func (h *ConsolidationHandler) ApproveAveragedResponse(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	categoryID, err := strconv.ParseUint(r.PathValue("categoryId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.consolidationService.ApproveAveragedResponse(uint(assessmentID), uint(categoryID), userID); err != nil {
		// Check error type for appropriate status code
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before approving averaged responses":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "cannot approve averaged response when override exists - approve the override instead":
			http.Error(w, errMsg, http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Averaged response approved successfully"})
}

// DeleteOverride deletes a consolidation override
// @Summary Delete consolidation override
// @Description Deletes a manually adjusted value (any reviewer with complete review can delete)
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param categoryId path int true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Override not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/override/{categoryId} [delete]
func (h *ConsolidationHandler) DeleteOverride(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	categoryID, err := strconv.ParseUint(r.PathValue("categoryId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.consolidationService.DeleteOverride(uint(assessmentID), uint(categoryID), userID); err != nil {
		// Check error type for appropriate status code
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before deleting overrides":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "override not found":
			http.Error(w, errMsg, http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Override deleted successfully"})
}

// SaveFinalConsolidation creates or updates the final consolidation comment
// @Summary Save final consolidation
// @Description Creates or updates the final consolidation comment
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param request body object{comment=string} true "Final consolidation comment"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/final [post]
func (h *ConsolidationHandler) SaveFinalConsolidation(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
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

	// Parse request body
	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Comment == "" {
		http.Error(w, "Comment is required", http.StatusBadRequest)
		return
	}

	if err := h.consolidationService.CreateOrUpdateFinalConsolidation(uint(assessmentID), req.Comment, userID); err != nil {
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before saving final consolidation":
			http.Error(w, errMsg, http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Final consolidation saved successfully"})
}

// ApproveFinalConsolidation approves the final consolidation
// @Summary Approve final consolidation
// @Description Approves the final consolidation
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid assessment ID"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Final consolidation not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/final/approve [post]
func (h *ConsolidationHandler) ApproveFinalConsolidation(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
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

	if err := h.consolidationService.ApproveFinalConsolidation(uint(assessmentID), userID); err != nil {
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before approving final consolidation":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "final consolidation not found":
			http.Error(w, errMsg, http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Final consolidation approved successfully"})
}

// RevokeOverrideApproval revokes a user's approval of an override
// @Summary Revoke override approval
// @Description Revokes the current user's approval of an override
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param categoryId path int true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid IDs"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Override not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/override/{categoryId}/approve [delete]
func (h *ConsolidationHandler) RevokeOverrideApproval(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	categoryID, err := strconv.ParseUint(r.PathValue("categoryId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.consolidationService.RevokeOverrideApproval(uint(assessmentID), uint(categoryID), userID); err != nil {
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before revoking approvals":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "override not found":
			http.Error(w, errMsg, http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Override approval revoked successfully"})
}

// RevokeAveragedApproval revokes a user's approval of an averaged response
// @Summary Revoke averaged response approval
// @Description Revokes the current user's approval of an averaged response
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param categoryId path int true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid IDs"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/averaged/{categoryId}/approve [delete]
func (h *ConsolidationHandler) RevokeAveragedApproval(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	categoryID, err := strconv.ParseUint(r.PathValue("categoryId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Get current user
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.consolidationService.RevokeAveragedApproval(uint(assessmentID), uint(categoryID), userID); err != nil {
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before revoking approvals":
			http.Error(w, errMsg, http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Averaged response approval revoked successfully"})
}

// RevokeFinalApproval revokes a user's approval of the final consolidation
// @Summary Revoke final consolidation approval
// @Description Revokes the current user's approval of the final consolidation
// @Tags Consolidation
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid assessment ID"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Final consolidation not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /review/consolidation/{id}/final/approve [delete]
func (h *ConsolidationHandler) RevokeFinalApproval(w http.ResponseWriter, r *http.Request) {
	assessmentID, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
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

	if err := h.consolidationService.RevokeFinalApproval(uint(assessmentID), userID); err != nil {
		errMsg := err.Error()
		switch {
		case errMsg == "user must complete their review before revoking approvals":
			http.Error(w, errMsg, http.StatusForbidden)
		case errMsg == "final consolidation not found":
			http.Error(w, errMsg, http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, map[string]string{"message": "Final consolidation approval revoked successfully"})
}
