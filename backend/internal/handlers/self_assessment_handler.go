package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"new-pay/internal/middleware"
	"new-pay/internal/service"
	"strconv"
	"strings"
	"time"
)

// SelfAssessmentHandler handles self-assessment HTTP requests
type SelfAssessmentHandler struct {
	selfAssessmentService *service.SelfAssessmentService
}

// NewSelfAssessmentHandler creates a new self-assessment handler
func NewSelfAssessmentHandler(selfAssessmentService *service.SelfAssessmentService) *SelfAssessmentHandler {
	return &SelfAssessmentHandler{
		selfAssessmentService: selfAssessmentService,
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalogs)
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
// @Router /self-assessments/catalog/{catalogId} [post]
func (h *SelfAssessmentHandler) CreateSelfAssessment(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("catalogId")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(assessment)
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
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	assessments, err := h.selfAssessmentService.GetUserSelfAssessments(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assessments)
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
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	// Check if user is admin or reviewer to return details
	isAdminOrReviewer := false
	for _, role := range userRoles {
		if role == "admin" || role == "reviewer" {
			isAdminOrReviewer = true
			break
		}
	}

	if isAdminOrReviewer {
		// Return with details for admin/reviewer
		assessment, err := h.selfAssessmentService.GetSelfAssessmentWithDetails(uint(id), userID, userRoles)
		if err != nil {
			if strings.Contains(err.Error(), "permission denied") {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assessment)
	} else {
		// Return basic info for regular users
		assessment, err := h.selfAssessmentService.GetSelfAssessment(uint(id), userID, userRoles)
		if err != nil {
			if strings.Contains(err.Error(), "permission denied") {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assessment)
	}
}

// GetVisibleSelfAssessments returns self-assessments visible to the user based on role
// @Summary Get visible self-assessments
// @Description Retrieve self-assessments based on user role (user: own, reviewer: submitted+, admin: metadata only)
// @Tags Self-Assessments
// @Security BearerAuth
// @Success 200 {array} models.SelfAssessment
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /self-assessments [get]
func (h *SelfAssessmentHandler) GetVisibleSelfAssessments(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	assessments, err := h.selfAssessmentService.GetVisibleSelfAssessments(userID, userRoles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assessments)
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
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
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
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.selfAssessmentService.UpdateSelfAssessmentStatus(uint(id), req.Status, userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Status updated successfully",
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assessments)
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
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.selfAssessmentService.DeleteSelfAssessment(uint(id), userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Self-assessment deleted successfully",
	})
}
