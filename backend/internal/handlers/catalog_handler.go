package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"new-pay/internal/middleware"
	"new-pay/internal/models"
	"new-pay/internal/service"
)

// CatalogRequest represents the request body for creating/updating catalogs
type CatalogRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	ValidFrom   string  `json:"valid_from"`  // Date string in YYYY-MM-DD format
	ValidUntil  string  `json:"valid_until"` // Date string in YYYY-MM-DD format
	Phase       *string `json:"phase,omitempty"`
}

// CatalogHandler handles criteria catalog requests
type CatalogHandler struct {
	catalogService *service.CatalogService
	auditMw        *middleware.AuditMiddleware
}

// NewCatalogHandler creates a new catalog handler
func NewCatalogHandler(
	catalogService *service.CatalogService,
	auditMw *middleware.AuditMiddleware,
) *CatalogHandler {
	return &CatalogHandler{
		catalogService: catalogService,
		auditMw:        auditMw,
	}
}

// GetAllCatalogs retrieves all catalogs visible to the user
// @Summary Get all catalogs
// @Description Get all catalogs based on user permissions
// @Tags Catalogs
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.CriteriaCatalog
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /catalogs [get]
func (h *CatalogHandler) GetAllCatalogs(w http.ResponseWriter, r *http.Request) {
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{} // Default to empty roles if not found
	}

	catalogs, err := h.catalogService.GetVisibleCatalogs(userRoles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, catalogs)
}

// GetCatalogByID retrieves a catalog by ID
// @Summary Get catalog by ID
// @Description Get a specific catalog with all details
// @Tags Catalogs
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Success 200 {object} models.CatalogWithDetails
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Catalog not found"
// @Router /catalogs/{id} [get]
func (h *CatalogHandler) GetCatalogByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	catalog, err := h.catalogService.GetCatalogWithDetails(uint(id), userRoles)
	if err != nil {
		if err.Error() == "catalog not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if err.Error() == "permission denied" || strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, catalog)
}

// CreateCatalog creates a new catalog
// @Summary Create catalog
// @Description Create a new criteria catalog (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param catalog body CatalogRequest true "Catalog data"
// @Success 201 {object} models.CriteriaCatalog
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs [post]
func (h *CatalogHandler) CreateCatalog(w http.ResponseWriter, r *http.Request) {
	var req CatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse date strings
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		http.Error(w, "Invalid valid_from date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	validUntil, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		http.Error(w, "Invalid valid_until date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Create catalog model
	catalog := models.CriteriaCatalog{
		Name:        req.Name,
		Description: req.Description,
		ValidFrom:   validFrom,
		ValidUntil:  validUntil,
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	if err := h.catalogService.CreateCatalog(&catalog, userID); err != nil {
		if strings.Contains(err.Error(), "overlaps") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, catalog)
}

// UpdateCatalog updates an existing catalog
// @Summary Update catalog
// @Description Update a catalog (admin only, restricted in archived phase)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param catalog body CatalogRequest true "Updated catalog data"
// @Success 200 {object} models.CriteriaCatalog
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id} [put]
func (h *CatalogHandler) UpdateCatalog(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	var req CatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse date strings
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		http.Error(w, "Invalid valid_from date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	validUntil, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		http.Error(w, "Invalid valid_until date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Create catalog model
	catalog := models.CriteriaCatalog{
		ID:          uint(id),
		Name:        req.Name,
		Description: req.Description,
		ValidFrom:   validFrom,
		ValidUntil:  validUntil,
	}

	if req.Phase != nil {
		catalog.Phase = *req.Phase
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.UpdateCatalog(&catalog, userID, userRoles); err != nil {
		var statusCode int
		if strings.Contains(err.Error(), "permission denied") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(err.Error(), "overlaps") {
			statusCode = http.StatusConflict
		} else {
			statusCode = http.StatusBadRequest
		}

		w.WriteHeader(statusCode)
		JSONResponse(w, map[string]string{"error": err.Error()})
		return
	}

	JSONResponse(w, catalog)
}

// UpdateCatalogValidUntil updates only the valid_until date of an active catalog
// @Summary Update catalog end date
// @Description Update the valid_until date of an active catalog (admin only, can only shorten)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param validUntil body object{valid_until=string} true "New end date"
// @Success 200 {object} models.CriteriaCatalog
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/catalogs/{id}/valid-until [put]
func (h *CatalogHandler) UpdateCatalogValidUntil(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ValidUntil string `json:"valid_until"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	catalog, err := h.catalogService.UpdateCatalogValidUntil(uint(id), req.ValidUntil)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "only active") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, catalog)
}

// DeleteCatalog deletes a catalog
// @Summary Delete catalog
// @Description Delete a catalog (admin only, only in draft phase)
// @Tags Catalogs
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id} [delete]
func (h *CatalogHandler) DeleteCatalog(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	if err := h.catalogService.DeleteCatalog(uint(id), userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TransitionToActive transitions a catalog to active phase
// @Summary Transition to active
// @Description Move catalog from draft to active phase (admin only)
// @Tags Catalogs
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/transition-to-active [post]
func (h *CatalogHandler) TransitionToActive(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.TransitionToActive(uint(id), userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Catalog transitioned to active phase",
	})
}

// TransitionToArchived transitions a catalog to archived phase
// @Summary Transition to archived
// @Description Move catalog from active to archived phase (admin only)
// @Tags Catalogs
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/transition-to-archived [post]
func (h *CatalogHandler) TransitionToArchived(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.TransitionToArchived(uint(id), userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Catalog transitioned to archived phase",
	})
}

// CreateCategory creates a new category
// @Summary Create category
// @Description Create a new category in a catalog (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param category body models.Category true "Category data"
// @Success 201 {object} models.Category
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/categories [post]
func (h *CatalogHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	category.CatalogID = uint(catalogID)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.CreateCategory(&category, userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, category)
}

// UpdateCategory updates a category
// @Summary Update category
// @Description Update an existing category (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param categoryId path int true "Category ID"
// @Param category body models.Category true "Updated category data"
// @Success 200 {object} models.Category
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/categories/{categoryId} [put]
func (h *CatalogHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	categoryIDStr := r.PathValue("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	category.ID = uint(categoryID)
	category.CatalogID = uint(catalogID)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.UpdateCategory(&category, userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, category)
}

// DeleteCategory deletes a category
// @Summary Delete category
// @Description Delete a category (admin only)
// @Tags Catalogs
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param categoryId path int true "Category ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/categories/{categoryId} [delete]
func (h *CatalogHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	categoryIDStr := r.PathValue("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	if err := h.catalogService.DeleteCategory(uint(categoryID), uint(catalogID), userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateLevel creates a new level
// @Summary Create level
// @Description Create a new level in a catalog (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param level body models.Level true "Level data"
// @Success 201 {object} models.Level
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/levels [post]
func (h *CatalogHandler) CreateLevel(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	var level models.Level
	if err := json.NewDecoder(r.Body).Decode(&level); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	level.CatalogID = uint(catalogID)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.CreateLevel(&level, userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, level)
}

// UpdateLevel updates a level
// @Summary Update level
// @Description Update an existing level (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param levelId path int true "Level ID"
// @Param level body models.Level true "Updated level data"
// @Success 200 {object} models.Level
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/levels/{levelId} [put]
func (h *CatalogHandler) UpdateLevel(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	levelIDStr := r.PathValue("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid level ID", http.StatusBadRequest)
		return
	}

	var level models.Level
	if err := json.NewDecoder(r.Body).Decode(&level); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	level.ID = uint(levelID)
	level.CatalogID = uint(catalogID)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.UpdateLevel(&level, userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, level)
}

// DeleteLevel deletes a level
// @Summary Delete level
// @Description Delete a level (admin only)
// @Tags Catalogs
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param levelId path int true "Level ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/levels/{levelId} [delete]
func (h *CatalogHandler) DeleteLevel(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	levelIDStr := r.PathValue("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid level ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	if err := h.catalogService.DeleteLevel(uint(levelID), uint(catalogID), userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreatePath creates a new path
// @Summary Create path
// @Description Create a new path in a category (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param categoryId path int true "Category ID"
// @Param path body models.Path true "Path data"
// @Success 201 {object} models.Path
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/categories/{categoryId}/paths [post]
func (h *CatalogHandler) CreatePath(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	categoryIDStr := r.PathValue("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var path models.Path
	if err := json.NewDecoder(r.Body).Decode(&path); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	path.CategoryID = uint(categoryID)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.CreatePath(&path, userID, userRoles, uint(catalogID)); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, path)
}

// UpdatePath updates a path
// @Summary Update path
// @Description Update an existing path (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param categoryId path int true "Category ID"
// @Param pathId path int true "Path ID"
// @Param path body models.Path true "Updated path data"
// @Success 200 {object} models.Path
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/categories/{categoryId}/paths/{pathId} [put]
func (h *CatalogHandler) UpdatePath(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	categoryIDStr := r.PathValue("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	pathIDStr := r.PathValue("pathId")
	pathID, err := strconv.ParseUint(pathIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid path ID", http.StatusBadRequest)
		return
	}

	var path models.Path
	if err := json.NewDecoder(r.Body).Decode(&path); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	path.ID = uint(pathID)
	path.CategoryID = uint(categoryID)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.UpdatePath(&path, userID, userRoles, uint(catalogID)); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, path)
}

// DeletePath deletes a path
// @Summary Delete path
// @Description Delete a path (admin only)
// @Tags Catalogs
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param categoryId path int true "Category ID"
// @Param pathId path int true "Path ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/categories/{categoryId}/paths/{pathId} [delete]
func (h *CatalogHandler) DeletePath(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	pathIDStr := r.PathValue("pathId")
	pathID, err := strconv.ParseUint(pathIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid path ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	if err := h.catalogService.DeletePath(uint(pathID), uint(catalogID), userID, userRoles); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateOrUpdateDescription creates or updates a path-level description
// @Summary Create or update description
// @Description Create or update a description for a path-level combination (admin only)
// @Tags Catalogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Param description body models.PathLevelDescription true "Description data"
// @Success 200 {object} models.PathLevelDescription
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/descriptions [post]
func (h *CatalogHandler) CreateOrUpdateDescription(w http.ResponseWriter, r *http.Request) {
	catalogIDStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(catalogIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	var description models.PathLevelDescription
	if err := json.NewDecoder(r.Body).Decode(&description); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	if err := h.catalogService.CreateOrUpdateDescription(&description, userID, userRoles, uint(catalogID)); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	JSONResponse(w, description)
}

// GetChanges retrieves change log for a catalog
// @Summary Get catalog changes
// @Description Get all changes made to a catalog (admin only)
// @Tags Catalogs
// @Produce json
// @Security BearerAuth
// @Param id path int true "Catalog ID"
// @Success 200 {array} models.CatalogChange
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Permission denied"
// @Router /admin/catalogs/{id}/changes [get]
func (h *CatalogHandler) GetChanges(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	catalogID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid catalog ID", http.StatusBadRequest)
		return
	}

	userRoles, ok := middleware.GetUserRoles(r)
	if !ok {
		userRoles = []string{}
	}

	changes, err := h.catalogService.GetChangesByCatalogID(uint(catalogID), userRoles)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	JSONResponse(w, changes)
}
