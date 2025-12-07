package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pwannenmacher/New-Pay/internal/middleware"
	"github.com/pwannenmacher/New-Pay/internal/repository"
)

// UserHandler handles user management requests
type UserHandler struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
	auditMw  *middleware.AuditMiddleware
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	auditMw *middleware.AuditMiddleware,
) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
		auditMw:  auditMw,
	}
}

// GetProfile gets the current user's profile
// @Summary Get user profile
// @Description Get authenticated user's profile information
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile with roles"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Get user roles
	roles, _ := h.userRepo.GetUserRoles(userID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":                user.ID,
		"email":             user.Email,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"email_verified":    user.EmailVerified,
		"email_verified_at": user.EmailVerifiedAt,
		"is_active":         user.IsActive,
		"last_login_at":     user.LastLoginAt,
		"created_at":        user.CreatedAt,
		"roles":             roles,
	})
}

// UpdateProfile updates the current user's profile
// @Summary Update user profile
// @Description Update authenticated user's profile information
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Profile update (first_name, last_name)"
// @Success 200 {object} map[string]interface{} "Profile updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /users/profile/update [post]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName

	if err := h.userRepo.Update(user); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(&userID, "user.profile.update", "users", "Profile updated", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Profile updated successfully",
		"user": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
		},
	})
}

// GetUser gets a user by ID (admin only)
// @Summary Get user by ID
// @Description Get any user's information by ID (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id query int true "User ID"
// @Success 200 {object} map[string]interface{} "User information with roles"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User not found"
// @Router /admin/users/get [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL parameter
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Get user roles
	roles, _ := h.userRepo.GetUserRoles(uint(id))

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":                user.ID,
		"email":             user.Email,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"email_verified":    user.EmailVerified,
		"email_verified_at": user.EmailVerifiedAt,
		"is_active":         user.IsActive,
		"last_login_at":     user.LastLoginAt,
		"created_at":        user.CreatedAt,
		"roles":             roles,
	})
}

// AssignRole assigns a role to a user (admin only)
// @Summary Assign role to user
// @Description Assign a role to a user (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "user_id and role_id"
// @Success 200 {object} map[string]string "Role assigned successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User or role not found"
// @Router /admin/users/assign-role [post]
func (h *UserHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint `json:"user_id"`
		RoleID uint `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify user exists
	_, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify role exists
	_, err = h.roleRepo.GetByID(req.RoleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Role not found")
		return
	}

	// Assign role
	if err := h.userRepo.AssignRole(req.UserID, req.RoleID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to assign role")
		return
	}

	// Log audit event
	adminID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&adminID, "user.role.assign", "users", "Role assigned to user", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Role assigned successfully",
	})
}

// RemoveRole removes a role from a user (admin only)
// @Summary Remove role from user
// @Description Remove a role from a user (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "user_id and role_id"
// @Success 200 {object} map[string]string "Role removed successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/users/remove-role [post]
func (h *UserHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint `json:"user_id"`
		RoleID uint `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Remove role
	if err := h.userRepo.RemoveRole(req.UserID, req.RoleID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to remove role")
		return
	}

	// Log audit event
	adminID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&adminID, "user.role.remove", "users", "Role removed from user", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Role removed successfully",
	})
}
