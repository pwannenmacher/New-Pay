package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"new-pay/internal/middleware"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/service"
	"strconv"
	"strings"
	"time"
)

// SelfAssessmentHandler handles self-assessment HTTP requests
type SelfAssessmentHandler struct {
	selfAssessmentService *service.SelfAssessmentService
	discussionService     *service.DiscussionService
	confirmationRepo      *repository.DiscussionConfirmationRepository
	assessmentRepo        *repository.SelfAssessmentRepository
	consolidationService  *service.ConsolidationService
}

// NewSelfAssessmentHandler creates a new self-assessment handler
func NewSelfAssessmentHandler(
	selfAssessmentService *service.SelfAssessmentService,
	discussionService *service.DiscussionService,
	confirmationRepo *repository.DiscussionConfirmationRepository,
	assessmentRepo *repository.SelfAssessmentRepository,
	consolidationService *service.ConsolidationService,
) *SelfAssessmentHandler {
	return &SelfAssessmentHandler{
		selfAssessmentService: selfAssessmentService,
		discussionService:     discussionService,
		confirmationRepo:      confirmationRepo,
		assessmentRepo:        assessmentRepo,
		consolidationService:  consolidationService,
	}
}

// Helper methods to reduce cognitive complexity

// checkUserIsAdminOrReviewer checks if a user has admin or reviewer role
func checkUserIsAdminOrReviewer(userRoles []string) bool {
	for _, role := range userRoles {
		if role == "admin" || role == "reviewer" {
			return true
		}
	}
	return false
}

// handleAssessmentError handles different types of assessment errors
func handleAssessmentError(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
		http.Error(w, err.Error(), http.StatusForbidden)
	} else if strings.Contains(err.Error(), ErrMsgNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetActiveCatalogs returns catalogs that are active and valid for current date
// @Summary Get active catalogs
// @Description Retrieve catalogs that users can create self-assessments for
// @Tags Self-Assessments
// @Security BearerAuth
// @Success 200 {array} models.CriteriaCatalog
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /self-assessments/active-catalogs [get]
func (h *SelfAssessmentHandler) GetActiveCatalogs(w http.ResponseWriter, r *http.Request) {
	catalogs, err := h.selfAssessmentService.GetActiveCatalogs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, catalogs)
}

// CreateSelfAssessment creates a new self-assessment
// @Summary Create self-assessment
// @Description Create a new self-assessment for an active catalog
// @Tags Self-Assessments
// @Security BearerAuth
// @Param catalogId path int true "Catalog ID"
// @Success 201 {object} models.SelfAssessment
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /catalogs/{catalogId}/self-assessments [post]
func (h *SelfAssessmentHandler) CreateSelfAssessment(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("catalogId")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	assessment, err := h.selfAssessmentService.CreateSelfAssessment(uint(catalogID), userID)
	if err != nil {
		slog.Error("Failed to create self-assessment", "error", err, "catalog_id", catalogID, "user_id", userID)
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else if strings.Contains(err.Error(), "not valid") || strings.Contains(err.Error(), "can only create") {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, assessment)
}

// GetUserSelfAssessments returns all self-assessments for the current user
// @Summary Get user's self-assessments
// @Description Retrieve all self-assessments created by the current user
// @Tags Self-Assessments
// @Security BearerAuth
// @Success 200 {array} models.SelfAssessmentWithDetails
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /self-assessments/my [get]
func (h *SelfAssessmentHandler) GetUserSelfAssessments(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	assessments, err := h.selfAssessmentService.GetUserSelfAssessments(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, assessments)
}

// GetSelfAssessment returns a specific self-assessment
// @Summary Get self-assessment
// @Description Retrieve a self-assessment by ID (with permission checks)
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} models.SelfAssessment
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Not found"
// @Router /self-assessments/{id} [get]
func (h *SelfAssessmentHandler) GetSelfAssessment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	isAdminOrReviewer := checkUserIsAdminOrReviewer(userRoles)

	if isAdminOrReviewer {
		// Return with details for admin/reviewer
		assessment, err := h.selfAssessmentService.GetSelfAssessmentWithDetails(uint(id), userID, userRoles)
		if err != nil {
			handleAssessmentError(w, err)
			return
		}
		JSONResponse(w, assessment)
	} else {
		// Return basic info for regular users
		assessment, err := h.selfAssessmentService.GetSelfAssessment(uint(id), userID, userRoles)
		if err != nil {
			handleAssessmentError(w, err)
			return
		}
		JSONResponse(w, assessment)
	}
}

// UpdateStatus updates the status of a self-assessment
// @Summary Update self-assessment status
// @Description Transition a self-assessment to a new status
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param body body object true "Status update request"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/status [put]
func (h *SelfAssessmentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.selfAssessmentService.UpdateSelfAssessmentStatus(uint(id), req.Status, userID, userRoles); err != nil {
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// If status changes to 'review_consolidation', generate consolidation proposals
	if req.Status == "review_consolidation" && h.consolidationService != nil {
		go func(id uint) {
			if err := h.consolidationService.GenerateConsolidationProposals(id); err != nil {
				slog.Error("Failed to generate consolidation proposals", "assessmentID", id, "error", err)
			} else {
				slog.Info("Consolidation proposals generated", "assessmentID", id)
			}
		}(uint(id))
	}

	// If status changes to 'discussion', create discussion results
	if req.Status == "discussion" && h.discussionService != nil {
		if err := h.discussionService.CreateDiscussionResult(uint(id)); err != nil {
			slog.Error("Failed to create discussion result", "assessmentID", id, "error", err)
			// Don't fail the request - discussion can be regenerated later
		} else {
			slog.Info("Discussion result created", "assessmentID", id)
		}
	}

	JSONResponse(w, map[string]string{
		"message": "Assessment status updated successfully",
	})
}

// GetAllSelfAssessmentsAdmin retrieves all self-assessments with filters (admin only)
// @Summary Get all self-assessments (admin)
// @Description Retrieve all self-assessments with optional filters (admin only)
// @Tags Self-Assessments
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param username query string false "Filter by username (email, first name, or last name)"
// @Param from_date query string false "Filter by creation date from (RFC3339)"
// @Param to_date query string false "Filter by creation date to (RFC3339)"
// @Success 200 {array} models.SelfAssessment
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /admin/self-assessments [get]
func (h *SelfAssessmentHandler) GetAllSelfAssessmentsAdmin(w http.ResponseWriter, r *http.Request) {
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	// Check if user is admin
	isAdmin := false
	for _, role := range userRoles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	username := r.URL.Query().Get("username")

	var fromDate, toDate *time.Time
	if fromStr := r.URL.Query().Get("from_date"); fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromDate = &parsed
		}
	}
	if toStr := r.URL.Query().Get("to_date"); toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			toDate = &parsed
		}
	}

	assessments, err := h.selfAssessmentService.GetAllSelfAssessmentsWithFiltersAndDetails(status, username, fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, assessments)
}

// GetOpenAssessmentsForReview retrieves all open self-assessments for reviewers
// @Summary Get open assessments for review
// @Description Retrieve self-assessments with status submitted, in_review, reviewed, or discussion (reviewer only)
// @Tags Self-Assessments
// @Security BearerAuth
// @Param catalog_id query int false "Filter by catalog ID"
// @Param username query string false "Filter by username (email, first name, or last name)"
// @Param status query string false "Filter by status"
// @Param from_date query string false "Filter by creation date from (RFC3339)"
// @Param to_date query string false "Filter by creation date to (RFC3339)"
// @Param from_submitted_date query string false "Filter by submission date from (RFC3339)"
// @Param to_submitted_date query string false "Filter by submission date to (RFC3339)"
// @Success 200 {array} models.SelfAssessmentWithDetails
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /review/open-assessments [get]
func (h *SelfAssessmentHandler) GetOpenAssessmentsForReview(w http.ResponseWriter, r *http.Request) {
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	// Check if user is reviewer or admin
	isReviewer := false
	for _, role := range userRoles {
		if role == "reviewer" || role == "admin" {
			isReviewer = true
			break
		}
	}
	if !isReviewer {
		http.Error(w, "Reviewer or Admin access required", http.StatusForbidden)
		return
	}

	// Parse query parameters
	catalogIDStr := r.URL.Query().Get("catalog_id")
	username := r.URL.Query().Get("username")
	status := r.URL.Query().Get("status")

	var catalogID *int
	if catalogIDStr != "" {
		if parsed, err := strconv.Atoi(catalogIDStr); err == nil {
			catalogID = &parsed
		}
	}

	var fromDate, toDate, fromSubmittedDate, toSubmittedDate *time.Time
	if fromStr := r.URL.Query().Get("from_date"); fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromDate = &parsed
		}
	}
	if toStr := r.URL.Query().Get("to_date"); toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			toDate = &parsed
		}
	}
	if fromSubStr := r.URL.Query().Get("from_submitted_date"); fromSubStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromSubStr); err == nil {
			fromSubmittedDate = &parsed
		}
	}
	if toSubStr := r.URL.Query().Get("to_submitted_date"); toSubStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toSubStr); err == nil {
			toSubmittedDate = &parsed
		}
	}

	assessments, err := h.selfAssessmentService.GetOpenAssessmentsForReview(catalogID, username, status, fromDate, toDate, fromSubmittedDate, toSubmittedDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, assessments)
}

// GetCompletedAssessmentsForReview retrieves archived assessments for reviewers
// @Summary Get completed assessments for review
// @Description Retrieves archived assessments. Admins see all, reviewers see only those they participated in
// @Tags Self-Assessments
// @Security BearerAuth
// @Param catalog_id query int false "Filter by catalog ID"
// @Param username query string false "Filter by username (email, first name, or last name)"
// @Param from_date query string false "Filter by creation date from (RFC3339)"
// @Param to_date query string false "Filter by creation date to (RFC3339)"
// @Param from_submitted_date query string false "Filter by submission date from (RFC3339)"
// @Param to_submitted_date query string false "Filter by submission date to (RFC3339)"
// @Success 200 {array} models.SelfAssessmentWithDetails
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /review/completed-assessments [get]
func (h *SelfAssessmentHandler) GetCompletedAssessmentsForReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	// Check if user is reviewer or admin
	isReviewer := false
	isAdmin := false
	for _, role := range userRoles {
		if role == "reviewer" {
			isReviewer = true
		}
		if role == "admin" {
			isAdmin = true
		}
	}
	if !isReviewer && !isAdmin {
		http.Error(w, "Reviewer or Admin access required", http.StatusForbidden)
		return
	}

	// Parse query parameters
	catalogIDStr := r.URL.Query().Get("catalog_id")
	username := r.URL.Query().Get("username")

	var catalogID *int
	if catalogIDStr != "" {
		if parsed, err := strconv.Atoi(catalogIDStr); err == nil {
			catalogID = &parsed
		}
	}

	var fromDate, toDate, fromSubmittedDate, toSubmittedDate *time.Time
	if fromStr := r.URL.Query().Get("from_date"); fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromDate = &parsed
		}
	}
	if toStr := r.URL.Query().Get("to_date"); toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			toDate = &parsed
		}
	}
	if fromSubStr := r.URL.Query().Get("from_submitted_date"); fromSubStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromSubStr); err == nil {
			fromSubmittedDate = &parsed
		}
	}
	if toSubStr := r.URL.Query().Get("to_submitted_date"); toSubStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toSubStr); err == nil {
			toSubmittedDate = &parsed
		}
	}

	assessments, err := h.selfAssessmentService.GetCompletedAssessmentsForReview(userID, isAdmin, catalogID, username, fromDate, toDate, fromSubmittedDate, toSubmittedDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, assessments)
}

// DeleteSelfAssessment deletes a self-assessment (admin only, closed without submission)
// @Summary Delete self-assessment
// @Description Delete a closed self-assessment that was never submitted (admin only)
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /admin/self-assessments/{id} [delete]
func (h *SelfAssessmentHandler) DeleteSelfAssessment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.selfAssessmentService.DeleteSelfAssessment(uint(id), userID, userRoles); err != nil {
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Assessment deleted successfully",
	})
}

// SaveResponse saves or updates an assessment response
// @Summary Save assessment response
// @Description Save or update a response for a category in a self-assessment
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param response body models.AssessmentResponse true "Response data"
// @Success 200 {object} models.AssessmentResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/responses [post]
func (h *SelfAssessmentHandler) SaveResponse(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	assessmentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	var response models.AssessmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	savedResponse, err := h.selfAssessmentService.SaveResponse(userID, uint(assessmentID), &response)
	if err != nil {
		slog.Error("Failed to save response", "error", err, "assessment_id", assessmentID, "user_id", userID)
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if strings.Contains(err.Error(), ErrMsgNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, savedResponse)
}

// DeleteResponse deletes an assessment response
// @Summary Delete assessment response
// @Description Delete a response for a category in a self-assessment
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Param categoryId path int true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/responses/{categoryId} [delete]
func (h *SelfAssessmentHandler) DeleteResponse(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	assessmentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	categoryIDStr := r.PathValue("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	if err := h.selfAssessmentService.DeleteResponse(userID, uint(assessmentID), uint(categoryID)); err != nil {
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if strings.Contains(err.Error(), ErrMsgNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Response deleted successfully",
	})
}

// GetResponses retrieves all responses for an assessment
// @Summary Get assessment responses
// @Description Retrieve all responses for a self-assessment
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {array} models.AssessmentResponseWithDetails
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/responses [get]
func (h *SelfAssessmentHandler) GetResponses(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	assessmentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	responses, err := h.selfAssessmentService.GetResponses(userID, uint(assessmentID), userRoles)
	if err != nil {
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if strings.Contains(err.Error(), ErrMsgNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, responses)
}

// GetCompleteness retrieves the completeness status of an assessment
// @Summary Get assessment completeness
// @Description Retrieve the completion status and progress of a self-assessment
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} models.AssessmentCompleteness
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/completeness [get]
func (h *SelfAssessmentHandler) GetCompleteness(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	assessmentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	completeness, err := h.selfAssessmentService.GetCompleteness(userID, uint(assessmentID))
	if err != nil {
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if strings.Contains(err.Error(), ErrMsgNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, completeness)
}

// GetWeightedScore retrieves the weighted average score for an assessment
// @Summary Get weighted score
// @Description Calculate and retrieve the weighted average score for a self-assessment
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} models.WeightedScore
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/weighted-score [get]
func (h *SelfAssessmentHandler) GetWeightedScore(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	assessmentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	score, err := h.selfAssessmentService.CalculateWeightedScore(userID, uint(assessmentID))
	if err != nil {
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if strings.Contains(err.Error(), ErrMsgNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, score)
}

// SubmitAssessment submits an assessment for review
// @Summary Submit self-assessment
// @Description Submit a self-assessment for review (changes status from draft to submitted)
// @Tags Self-Assessments
// @Security BearerAuth
// @Param id path int true "Assessment ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request or incomplete assessment"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /self-assessments/{id}/submit [put]
func (h *SelfAssessmentHandler) SubmitAssessment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	assessmentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, ErrMsgInvalidAssessmentID, http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, ErrMsgUserIDNotFound, http.StatusUnauthorized)
		return
	}

	if err := h.selfAssessmentService.SubmitAssessment(userID, uint(assessmentID)); err != nil {
		slog.Error("Failed to submit assessment", "error", err, "assessment_id", assessmentID, "user_id", userID)
		if strings.Contains(err.Error(), ErrMsgPermissionDenied) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if strings.Contains(err.Error(), ErrMsgNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Assessment submitted successfully",
	})
}

// ArchiveAssessment archives an assessment after confirmations
func (h *SelfAssessmentHandler) ArchiveAssessment(w http.ResponseWriter, r *http.Request) {
	// Get assessment ID from URL
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var assessmentID uint64
	var parseErr error

	for i, part := range pathParts {
		if part == "assessments" && i+1 < len(pathParts) {
			assessmentID, parseErr = strconv.ParseUint(pathParts[i+1], 10, 32)
			break
		}
	}

	if parseErr != nil || assessmentID == 0 {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	// Get user roles
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is admin or reviewer
	if !checkUserIsAdminOrReviewer(userRoles) {
		http.Error(w, "Only admins and reviewers can archive assessments", http.StatusForbidden)
		return
	}

	// Get assessment
	assessment, err := h.assessmentRepo.GetByID(uint(assessmentID))
	if err != nil {
		http.Error(w, "Assessment not found", http.StatusNotFound)
		return
	}

	// Check if assessment is in discussion status
	if assessment.Status != "discussion" {
		http.Error(w, "Assessment must be in discussion status to be archived", http.StatusBadRequest)
		return
	}

	// Prevent re-archiving
	if assessment.Status == "archived" {
		http.Error(w, "Assessment is already archived", http.StatusBadRequest)
		return
	}

	// Check if both reviewer and owner have confirmed
	hasReviewerConf, err := h.confirmationRepo.HasReviewerConfirmation(uint(assessmentID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !hasReviewerConf {
		http.Error(w, "At least one reviewer must confirm before archiving", http.StatusBadRequest)
		return
	}

	hasOwnerConf, err := h.confirmationRepo.HasOwnerConfirmation(uint(assessmentID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !hasOwnerConf {
		http.Error(w, "Owner must confirm before archiving", http.StatusBadRequest)
		return
	}

	// Get userID for audit
	userID, _ := middleware.GetUserID(r)

	// Update status to archived
	if err := h.selfAssessmentService.UpdateSelfAssessmentStatus(uint(assessmentID), "archived", userID, userRoles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Assessment archived successfully",
	})
}
