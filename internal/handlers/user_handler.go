package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/pwannenmacher/New-Pay/internal/middleware"
	"github.com/pwannenmacher/New-Pay/internal/models"
	"github.com/pwannenmacher/New-Pay/internal/repository"
	"github.com/pwannenmacher/New-Pay/internal/service"
)

// UserHandler handles user management requests
type UserHandler struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
	auditMw  *middleware.AuditMiddleware
	authSvc  *service.AuthService
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	auditMw *middleware.AuditMiddleware,
	authSvc *service.AuthService,
) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
		auditMw:  auditMw,
		authSvc:  authSvc,
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

	// Get OAuth connections
	oauthConnections, _ := h.authSvc.GetUserOAuthConnections(userID)

	// Determine if user has local password
	hasLocalPassword := user.PasswordHash != ""

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":                 user.ID,
		"email":              user.Email,
		"first_name":         user.FirstName,
		"last_name":          user.LastName,
		"email_verified":     user.EmailVerified,
		"email_verified_at":  user.EmailVerifiedAt,
		"is_active":          user.IsActive,
		"last_login_at":      user.LastLoginAt,
		"created_at":         user.CreatedAt,
		"roles":              roles,
		"oauth_connections":  oauthConnections,
		"has_local_password": hasLocalPassword,
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
		_ = h.auditMw.LogAction(&userID, "user.profile.update.error", "users", "Profile update failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(&userID, "user.profile.update", "users", "Profile updated", getIP(r), r.UserAgent())

	// Get user roles
	roles, _ := h.userRepo.GetUserRoles(userID)

	// Get OAuth connections
	oauthConnections, _ := h.authSvc.GetUserOAuthConnections(userID)

	// Determine if user has local password
	hasLocalPassword := user.PasswordHash != ""

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":                 user.ID,
		"email":              user.Email,
		"first_name":         user.FirstName,
		"last_name":          user.LastName,
		"email_verified":     user.EmailVerified,
		"email_verified_at":  user.EmailVerifiedAt,
		"is_active":          user.IsActive,
		"last_login_at":      user.LastLoginAt,
		"created_at":         user.CreatedAt,
		"updated_at":         user.UpdatedAt,
		"roles":              roles,
		"oauth_connections":  oauthConnections,
		"has_local_password": hasLocalPassword,
	})
}

// UpdateUserActiveStatus toggles a user's active status (admin only)
// @Summary Update user active status
// @Description Toggle a user's active/inactive status (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID and active status"
// @Success 200 {object} map[string]string "Status updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User not found"
// @Router /admin/users/update-status [post]
func (h *UserHandler) UpdateUserActiveStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   uint `json:"user_id"`
		IsActive bool `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// If deactivating user, check if they are the last active admin
	if !req.IsActive {
		isLastAdmin, err := h.userRepo.IsLastActiveAdmin(req.UserID)
		if err != nil {
			actorID, _ := middleware.GetUserID(r)
			_ = h.auditMw.LogAction(&actorID, "user.status.update.error", "users", "Failed to check admin status: "+err.Error(), getIP(r), r.UserAgent())
			respondWithError(w, http.StatusInternalServerError, "Failed to verify admin status")
			return
		}

		if isLastAdmin {
			respondWithError(w, http.StatusBadRequest, "Cannot deactivate the last active admin")
			return
		}
	}

	// Update active status
	if err := h.userRepo.UpdateActiveStatus(req.UserID, req.IsActive); err != nil {
		actorID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&actorID, "user.status.update.error", "users", "User status update failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to update user status")
		return
	}

	// Log the action
	actorID, _ := middleware.GetUserID(r)
	statusStr := "inactive"
	if req.IsActive {
		statusStr = "active"
	}
	_ = h.auditMw.LogAction(&actorID, "update_user_status", "user",
		"User "+user.Email+" status changed to "+statusStr, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "User status updated successfully",
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

	// Get OAuth connections
	oauthConnections, _ := h.authSvc.GetUserOAuthConnections(uint(id))

	// Determine if user has local password
	hasLocalPassword := user.PasswordHash != ""

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":                 user.ID,
		"email":              user.Email,
		"first_name":         user.FirstName,
		"last_name":          user.LastName,
		"email_verified":     user.EmailVerified,
		"email_verified_at":  user.EmailVerifiedAt,
		"is_active":          user.IsActive,
		"last_login_at":      user.LastLoginAt,
		"created_at":         user.CreatedAt,
		"roles":              roles,
		"oauth_connections":  oauthConnections,
		"has_local_password": hasLocalPassword,
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
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "user.role.assign.error", "users", "Role assignment failed: "+err.Error(), getIP(r), r.UserAgent())
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

	// Check if we're removing the Admin role
	role, err := h.roleRepo.GetByID(req.RoleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Role not found")
		return
	}

	// If removing Admin role, check if user is the last active admin
	if role.Name == "Admin" {
		isLastAdmin, err := h.userRepo.IsLastActiveAdmin(req.UserID)
		if err != nil {
			adminID, _ := middleware.GetUserID(r)
			_ = h.auditMw.LogAction(&adminID, "user.role.remove.error", "users", "Failed to check admin status: "+err.Error(), getIP(r), r.UserAgent())
			respondWithError(w, http.StatusInternalServerError, "Failed to verify admin status")
			return
		}

		if isLastAdmin {
			respondWithError(w, http.StatusBadRequest, "Cannot remove Admin role from the last active admin")
			return
		}
	}

	// Remove role
	if err := h.userRepo.RemoveRole(req.UserID, req.RoleID); err != nil {
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "user.role.remove.error", "users", "Role removal failed: "+err.Error(), getIP(r), r.UserAgent())
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

// ListUsers lists all users with pagination (admin only)
// @Summary List all users
// @Description Get a paginated list of all users (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {array} map[string]interface{} "List of users"
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/users/list [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Get pagination parameters
	page := 1
	limit := 20

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
	filters := repository.UserFilters{
		Search:    r.URL.Query().Get("search"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
	}

	// Parse role IDs filter
	if roleIDsStr := r.URL.Query().Get("role_ids"); roleIDsStr != "" {
		roleIDsStrs := strings.Split(roleIDsStr, ",")
		for _, idStr := range roleIDsStrs {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				filters.RoleIDs = append(filters.RoleIDs, id)
			}
		}
	}

	// Parse active filter
	if activeStr := r.URL.Query().Get("is_active"); activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			filters.IsActive = &active
		}
	}

	// Parse email verified filter
	if verifiedStr := r.URL.Query().Get("email_verified"); verifiedStr != "" {
		if verified, err := strconv.ParseBool(verifiedStr); err == nil {
			filters.EmailVerified = &verified
		}
	}

	// Get total count
	totalCount, err := h.userRepo.CountWithFilters(filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	// Get users with filters
	users, err := h.userRepo.GetAllWithFilters(filters, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	// Build response with user roles, OAuth connections, and password status
	var userList []map[string]interface{}
	for _, user := range users {
		roles, err := h.userRepo.GetUserRoles(user.ID)
		if err != nil {
			// Log error but continue with empty roles array
			roles = []models.Role{}
		}

		// Get OAuth connections
		oauthConnections, err := h.authSvc.GetUserOAuthConnections(user.ID)
		if err != nil {
			oauthConnections = []models.OAuthConnection{}
		}

		// Check if user has local password
		hasLocalPassword := user.PasswordHash != ""

		userList = append(userList, map[string]interface{}{
			"id":                 user.ID,
			"email":              user.Email,
			"first_name":         user.FirstName,
			"last_name":          user.LastName,
			"email_verified":     user.EmailVerified,
			"email_verified_at":  user.EmailVerifiedAt,
			"is_active":          user.IsActive,
			"last_login_at":      user.LastLoginAt,
			"created_at":         user.CreatedAt,
			"updated_at":         user.UpdatedAt,
			"roles":              roles,
			"oauth_connections":  oauthConnections,
			"has_local_password": hasLocalPassword,
		})
	}

	// Calculate total pages
	totalPages := (totalCount + limit - 1) / limit

	response := map[string]interface{}{
		"users":       userList,
		"total":       totalCount,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// ListRoles lists all available roles (admin only)
// @Summary List all roles
// @Description Get a list of all available roles (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Role "List of roles"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/roles/list [get]
func (h *UserHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.roleRepo.GetAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve roles")
		return
	}

	respondWithJSON(w, http.StatusOK, roles)
}

// UpdateUser updates a user's basic information (admin only)
// @Summary Update user information
// @Description Update a user's email, first name, and last name (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID and updated fields"
// @Success 200 {object} map[string]string "User updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User not found"
// @Router /admin/users/update [post]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    uint   `json:"user_id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing user
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if new email is already taken by another user
	if req.Email != user.Email {
		if existingUser, _ := h.userRepo.GetByEmail(req.Email); existingUser != nil && existingUser.ID != req.UserID {
			respondWithError(w, http.StatusBadRequest, "Email already in use")
			return
		}
	}

	// Update user fields
	user.Email = req.Email
	user.FirstName = req.FirstName
	user.LastName = req.LastName

	if err := h.userRepo.Update(user); err != nil {
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "update_user.error", "users", "User update failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Log the action
	actorID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&actorID, "update_user", "user",
		"Updated user "+user.Email, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "User updated successfully",
	})
}

// SetUserPassword sets a new password for a user (admin only)
// @Summary Set user password
// @Description Set a new password for any user (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID and new password"
// @Success 200 {object} map[string]string "Password updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User not found"
// @Router /admin/users/set-password [post]
func (h *UserHandler) SetUserPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   uint   `json:"user_id"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate password
	if len(req.Password) < 8 {
		respondWithError(w, http.StatusBadRequest, "Password must be at least 8 characters long")
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Hash the new password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "set_user_password.error", "users", "Password hash failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Update password
	if err := h.userRepo.UpdatePassword(req.UserID, string(hashedBytes)); err != nil {
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "set_user_password.error", "users", "Password update failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Log the action
	actorID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&actorID, "set_user_password", "user",
		"Set password for user "+user.Email, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Password updated successfully",
	})
}

// DeleteUser deletes a user (admin only)
// @Summary Delete user
// @Description Delete a user from the system (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID"
// @Success 200 {object} map[string]string "User deleted successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User not found"
// @Router /admin/users/delete [post]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Prevent deleting yourself
	actorID, _ := middleware.GetUserID(r)
	if actorID == req.UserID {
		respondWithError(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	// Check if they are the last active admin
	isLastAdmin, err := h.userRepo.IsLastActiveAdmin(req.UserID)
	if err != nil {
		_ = h.auditMw.LogAction(&actorID, "delete_user.error", "users", "Failed to check admin status: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to verify admin status")
		return
	}

	if isLastAdmin {
		respondWithError(w, http.StatusBadRequest, "Cannot delete the last active admin")
		return
	}

	// Delete user
	if err := h.userRepo.Delete(req.UserID); err != nil {
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "delete_user.error", "users", "User deletion failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	// Log the action
	_ = h.auditMw.LogAction(&actorID, "delete_user", "user",
		"Deleted user "+user.Email, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}
