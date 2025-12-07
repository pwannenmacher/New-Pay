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
// @Success 200 {array} models.AuditLog "List of audit logs"
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

	logs, err := h.auditRepo.GetAll(limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve audit logs")
		return
	}

	respondWithJSON(w, http.StatusOK, logs)
}
