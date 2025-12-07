package handlers

import (
	"net/http"
	"strconv"

	"github.com/pwannenmacher/New-Pay/internal/repository"
)

// AuditHandler handles audit log requests
type AuditHandler struct {
	auditRepo *repository.AuditRepository
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditRepo *repository.AuditRepository) *AuditHandler {
	return &AuditHandler{
		auditRepo: auditRepo,
	}
}

// ListAuditLogs lists all audit logs with pagination (admin only)
// @Summary List audit logs
// @Description Get a paginated list of all audit logs (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param user_id query int false "Filter by user ID"
// @Param action query string false "Filter by action"
// @Param resource query string false "Filter by resource"
// @Param sort_by query string false "Sort by field (id, user_id, action, resource, created_at)"
// @Param sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {object} map[string]interface{} "Paginated audit logs"
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/audit-logs/list [get]
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Get pagination parameters
	page := 1
	limit := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build filters
	filters := repository.AuditFilters{
		Action:    r.URL.Query().Get("action"),
		Resource:  r.URL.Query().Get("resource"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
	}

	// Parse user ID filter
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			uid := uint(userID)
			filters.UserID = &uid
		}
	}

	// Get total count
	totalCount, err := h.auditRepo.CountWithFilters(filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to count audit logs")
		return
	}

	// Get logs with filters
	logs, err := h.auditRepo.GetAllWithFilters(filters, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve audit logs")
		return
	}

	// Calculate total pages
	totalPages := (totalCount + limit - 1) / limit

	response := map[string]interface{}{
		"logs":        logs,
		"total":       totalCount,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	}

	respondWithJSON(w, http.StatusOK, response)
}
