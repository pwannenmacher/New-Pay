package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"new-pay/internal/middleware"
	"new-pay/internal/models"
	"new-pay/internal/repository"
	"new-pay/internal/service"
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
		respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorized)
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
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
		respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorized)
		return
	}

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
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

// ChangePassword allows a user to change their own password
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Current and new password"
// @Success 200 {object} map[string]string "Password changed successfully"
// @Failure 400 {object} map[string]string "Invalid request or password too short"
// @Failure 401 {object} map[string]string "Unauthorized or incorrect current password"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/password/change [post]
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Validate new password
	if len(req.NewPassword) < 8 {
		respondWithError(w, http.StatusBadRequest, "New password must be at least 8 characters long")
		return
	}

	// Get current user
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
		return
	}

	// Check if user has a local password
	if user.PasswordHash == "" {
		respondWithError(w, http.StatusBadRequest, "User has no local password set. Please set a password first.")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		_ = h.auditMw.LogAction(&userID, "user.password.change.failed", "users", "Incorrect current password", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash the new password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		_ = h.auditMw.LogAction(&userID, "user.password.change.error", "users", "Password hash failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, ErrMsgFailedToHashPassword)
		return
	}

	// Update password
	if err := h.userRepo.UpdatePassword(userID, string(hashedBytes)); err != nil {
		_ = h.auditMw.LogAction(&userID, "user.password.change.error", "users", "Password update failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Log successful password change
	_ = h.auditMw.LogAction(&userID, "user.password.change", "users", "Password changed successfully", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})
}

// Helper methods to reduce cognitive complexity

// assignRolesToNewUser assigns roles to a newly created user and logs the actions
func (h *UserHandler) assignRolesToNewUser(userID uint, roleIDs []uint, userEmail string, adminUserID uint, r *http.Request) {
	for _, roleID := range roleIDs {
		if err := h.userRepo.AssignRole(userID, roleID); err != nil {
			_ = h.auditMw.LogAction(&adminUserID, "user.role.assign.error", "users",
				fmt.Sprintf("Failed to assign role %d to user %s", roleID, userEmail), getIP(r), r.UserAgent())
		} else {
			role, _ := h.roleRepo.GetByID(roleID)
			if role != nil {
				_ = h.auditMw.LogAction(&adminUserID, "user.role.assigned", "users",
					fmt.Sprintf("Assigned role '%s' to user %s", role.Name, userEmail), getIP(r), r.UserAgent())
			}
		}
	}
}

// sendVerificationEmailIfRequested sends verification email if requested and logs the action
func (h *UserHandler) sendVerificationEmailIfRequested(userID uint, userEmail string, sendEmail bool, adminUserID uint, r *http.Request) {
	if !sendEmail {
		return
	}

	if err := h.authSvc.SendVerificationEmailToUser(userID); err != nil {
		_ = h.auditMw.LogAction(&adminUserID, "email.verification.error", "users",
			fmt.Sprintf("Failed to send verification email to %s: %v", userEmail, err), getIP(r), r.UserAgent())
	} else {
		_ = h.auditMw.LogAction(&adminUserID, "email.verification.sent", "users",
			fmt.Sprintf("Verification email sent to %s", userEmail), getIP(r), r.UserAgent())
	}
}

// buildUserListResponse builds the response for listing users with roles and OAuth connections
func (h *UserHandler) buildUserListResponse(users []models.User) []map[string]interface{} {
	var userList []map[string]interface{}
	for _, user := range users {
		roles, err := h.userRepo.GetUserRoles(user.ID)
		if err != nil {
			roles = []models.Role{}
		}

		oauthConnections, err := h.authSvc.GetUserOAuthConnections(user.ID)
		if err != nil {
			oauthConnections = []models.OAuthConnection{}
		}

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
	return userList
}

// parsePaginationParams parses and validates pagination parameters from the request
func parsePaginationParams(r *http.Request) (page, limit, offset int) {
	page = 1
	limit = 20

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

	offset = (page - 1) * limit
	return page, limit, offset
}

// parseUserFilters parses filter parameters from the request
func parseUserFilters(r *http.Request) repository.UserFilters {
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

	return filters
}

// CreateUser creates a new user (admin only)
// @Summary Create a new user
// @Description Create a new user account (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "User details"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /admin/users/create [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsActive  bool   `json:"is_active"`
		SendEmail bool   `json:"send_email"`
		RoleIDs   []uint `json:"role_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{
			"error": ErrMsgInvalidRequestBody,
		})
		return
	}

	// Validate required fields
	if req.Email == "" || req.FirstName == "" || req.LastName == "" {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Email, first name, and last name are required",
		})
		return
	}

	// Get admin user ID for audit logging
	adminUserID, _ := middleware.GetUserID(r)

	// Check if user already exists
	existingUser, _ := h.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{
			"error": "User with this email already exists",
		})
		_ = h.auditMw.LogAction(&adminUserID, "user.create.error", "users",
			fmt.Sprintf("Failed to create user %s: email already exists", req.Email), getIP(r), r.UserAgent())
		return
	}

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{
				"error": ErrMsgFailedToHashPassword,
			})
			return
		}
		passwordHash = string(hash)
	}

	// Create user
	user := &models.User{
		Email:         req.Email,
		PasswordHash:  passwordHash,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		IsActive:      req.IsActive,
		EmailVerified: false, // Admin can verify later if needed
	}

	if err := h.userRepo.Create(user); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
		_ = h.auditMw.LogAction(&adminUserID, "user.create.error", "users",
			fmt.Sprintf("Failed to create user %s: %v", req.Email, err), getIP(r), r.UserAgent())
		return
	}

	// Assign roles if provided
	h.assignRolesToNewUser(user.ID, req.RoleIDs, req.Email, adminUserID, r)

	// Send verification email if requested
	h.sendVerificationEmailIfRequested(user.ID, req.Email, req.SendEmail, adminUserID, r)

	// Log user creation
	_ = h.auditMw.LogAction(&adminUserID, "user.create", "users",
		fmt.Sprintf("User created: %s (ID: %d)", req.Email, user.ID), getIP(r), r.UserAgent())

	// Get user with roles for response
	roles, _ := h.authSvc.GetUserRoles(user.ID)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"user": map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"is_active":      user.IsActive,
			"email_verified": user.EmailVerified,
			"roles":          roles,
		},
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
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
		return
	}

	// If deactivating user, check if they are the last active admin
	if !req.IsActive {
		isLastAdmin, err := h.userRepo.IsLastActiveAdmin(req.UserID)
		if err != nil {
			actorID, _ := middleware.GetUserID(r)
			_ = h.auditMw.LogAction(&actorID, "user.status.update.error", "users", ErrMsgFailedToCheckAdminStatus+err.Error(), getIP(r), r.UserAgent())
			respondWithError(w, http.StatusInternalServerError, ErrMsgFailedToVerifyAdminStatus)
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
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
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
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Prevent admins from modifying their own roles
	adminID, _ := middleware.GetUserID(r)
	if adminID == req.UserID {
		_ = h.auditMw.LogAction(&adminID, "user.role.assign.error", "users", "Attempted to assign role to self", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusForbidden, "Cannot modify your own roles")
		return
	}

	// Verify user exists
	_, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
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
		_ = h.auditMw.LogAction(&adminID, "user.role.assign.error", "users", "Role assignment failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to assign role")
		return
	}

	// Log audit event
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
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Prevent admins from modifying their own roles
	adminID, _ := middleware.GetUserID(r)
	if adminID == req.UserID {
		_ = h.auditMw.LogAction(&adminID, "user.role.remove.error", "users", "Attempted to remove role from self", getIP(r), r.UserAgent())
		respondWithError(w, http.StatusForbidden, "Cannot modify your own roles")
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
			_ = h.auditMw.LogAction(&adminID, "user.role.remove.error", "users", ErrMsgFailedToCheckAdminStatus+err.Error(), getIP(r), r.UserAgent())
			respondWithError(w, http.StatusInternalServerError, ErrMsgFailedToVerifyAdminStatus)
			return
		}

		if isLastAdmin {
			respondWithError(w, http.StatusBadRequest, "Cannot remove Admin role from the last active admin")
			return
		}
	}

	// Remove role
	if err := h.userRepo.RemoveRole(req.UserID, req.RoleID); err != nil {
		_ = h.auditMw.LogAction(&adminID, "user.role.remove.error", "users", "Role removal failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to remove role")
		return
	}

	// Log audit event
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
	page, limit, offset := parsePaginationParams(r)

	// Build filters
	filters := parseUserFilters(r)

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
	userList := h.buildUserListResponse(users)

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
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Get existing user
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
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
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
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
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
		return
	}

	// Hash the new password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		adminID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&adminID, "set_user_password.error", "users", "Password hash failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, ErrMsgFailedToHashPassword)
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
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ErrMsgUserNotFound)
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
		_ = h.auditMw.LogAction(&actorID, "delete_user.error", "users", ErrMsgFailedToCheckAdminStatus+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, ErrMsgFailedToVerifyAdminStatus)
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

// ResendVerificationEmail resends verification email for the current user
// @Summary Resend verification email
// @Description Resend email verification link for current user if not yet verified
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Verification email sent"
// @Failure 400 {object} map[string]string "Email already verified"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/resend-verification [post]
func (h *UserHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorized)
		return
	}

	if err := h.authSvc.ResendVerificationEmail(userID); err != nil {
		if err.Error() == "email already verified" {
			respondWithError(w, http.StatusBadRequest, "Email already verified")
			return
		}
		_ = h.auditMw.LogAction(&userID, "user.resend_verification.error", "users", "Failed to resend verification: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to send verification email")
		return
	}

	_ = h.auditMw.LogAction(&userID, "user.resend_verification", "users", "Verification email resent", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent successfully",
	})
}

// AdminSendVerificationEmail sends verification email to any user (admin only)
// @Summary Send verification email to user
// @Description Admin endpoint to send verification email to any user
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID"
// @Success 200 {object} map[string]string "Verification email sent"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Failure 404 {object} map[string]string "User not found"
// @Router /admin/users/send-verification [post]
func (h *UserHandler) AdminSendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	if err := h.authSvc.SendVerificationEmailToUser(req.UserID); err != nil {
		actorID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&actorID, "admin.send_verification.error", "users", "Failed to send verification: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to send verification email")
		return
	}

	actorID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&actorID, "admin.send_verification", "users", fmt.Sprintf("Sent verification email to user ID %d", req.UserID), getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent successfully",
	})
}

// AdminCancelVerification cancels pending email verification for a user (admin only)
// @Summary Cancel pending email verification
// @Description Admin endpoint to cancel all pending verification tokens for a user
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID"
// @Success 200 {object} map[string]string "Verification cancelled"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/users/cancel-verification [post]
func (h *UserHandler) AdminCancelVerification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	if err := h.authSvc.CancelEmailVerification(req.UserID); err != nil {
		actorID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&actorID, "admin.cancel_verification.error", "users", "Failed to cancel verification: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to cancel verification")
		return
	}

	actorID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&actorID, "admin.cancel_verification", "users", fmt.Sprintf("Cancelled verification for user ID %d", req.UserID), getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Email verification cancelled successfully",
	})
}

// AdminRevokeVerification marks a user's email as unverified (admin only)
// @Summary Revoke email verification
// @Description Admin endpoint to mark a user's email as unverified
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "User ID"
// @Success 200 {object} map[string]string "Verification revoked"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/users/revoke-verification [post]
func (h *UserHandler) AdminRevokeVerification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody)
		return
	}

	if err := h.authSvc.RevokeEmailVerification(req.UserID); err != nil {
		actorID, _ := middleware.GetUserID(r)
		_ = h.auditMw.LogAction(&actorID, "admin.revoke_verification.error", "users", "Failed to revoke verification: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke verification")
		return
	}

	actorID, _ := middleware.GetUserID(r)
	_ = h.auditMw.LogAction(&actorID, "admin.revoke_verification", "users", fmt.Sprintf("Revoked verification for user ID %d", req.UserID), getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Email verification revoked successfully",
	})
}
