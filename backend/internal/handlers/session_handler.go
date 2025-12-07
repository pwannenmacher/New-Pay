package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"new-pay/internal/middleware"
	"new-pay/internal/repository"
	"new-pay/internal/service"
)

// SessionHandler handles session management requests
type SessionHandler struct {
	sessionRepo *repository.SessionRepository
	authService *service.AuthService
	auditMw     *middleware.AuditMiddleware
	db          *sql.DB
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(
	sessionRepo *repository.SessionRepository,
	authService *service.AuthService,
	auditMw *middleware.AuditMiddleware,
	db *sql.DB,
) *SessionHandler {
	return &SessionHandler{
		sessionRepo: sessionRepo,
		authService: authService,
		auditMw:     auditMw,
		db:          db,
	}
}

// GetMySessions gets the current user's active sessions
// @Summary Get user sessions
// @Description Get all active sessions for the authenticated user
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} object "List of active sessions"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /users/sessions [get]
func (h *SessionHandler) GetMySessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessions, err := h.sessionRepo.GetByUserID(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get sessions")
		return
	}

	// Group sessions by session_id and return only unique sessions
	sessionMap := make(map[string]map[string]interface{})
	for _, session := range sessions {
		if _, exists := sessionMap[session.SessionID]; !exists {
			sessionMap[session.SessionID] = map[string]interface{}{
				"session_id":       session.SessionID,
				"created_at":       session.CreatedAt,
				"last_activity_at": session.LastActivityAt,
				"ip_address":       session.IPAddress,
				"user_agent":       session.UserAgent,
				"expires_at":       session.ExpiresAt,
			}
		} else {
			// Update with the latest activity time
			if session.LastActivityAt.After(sessionMap[session.SessionID]["last_activity_at"].(time.Time)) {
				sessionMap[session.SessionID]["last_activity_at"] = session.LastActivityAt
			}
		}
	}

	// Convert map to array
	var result []map[string]interface{}
	for _, sessionData := range sessionMap {
		result = append(result, sessionData)
	}

	respondWithJSON(w, http.StatusOK, result)
}

// DeleteMySession deletes a specific session for the current user
// @Summary Delete user session
// @Description Delete a specific session by session_id
// @Tags Sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param session_id query string true "Session ID to delete"
// @Success 200 {object} map[string]string "Session deleted successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /users/sessions/delete [delete]
func (h *SessionHandler) DeleteMySession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		respondWithError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Verify session belongs to user
	sessions, err := h.sessionRepo.GetByUserID(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to verify session")
		return
	}

	found := false
	for _, session := range sessions {
		if session.SessionID == sessionID {
			found = true
			break
		}
	}

	if !found {
		respondWithError(w, http.StatusForbidden, "Session not found or access denied")
		return
	}

	// Delete the session
	if err := h.sessionRepo.DeleteBySessionID(sessionID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(&userID, "session.delete", "sessions", "User deleted their own session", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Session deleted successfully",
	})
}

// DeleteAllMySessions deletes all sessions except the current one
// @Summary Delete all user sessions except current
// @Description Delete all active sessions for the user except the current one
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "All other sessions deleted"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /users/sessions/delete-all [delete]
func (h *SessionHandler) DeleteAllMySessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get current session ID from the access token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusUnauthorized, "No authorization header")
		return
	}

	// Extract token (remove "Bearer " prefix)
	token := authHeader[7:]
	currentJTI, err := h.authService.ExtractJTI(token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	// Get current session
	currentSession, err := h.sessionRepo.GetByJTI(currentJTI)
	if err != nil {
		_ = h.auditMw.LogAction(&userID, "session.get.error", "sessions", "Get current session failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to get current session")
		return
	}

	// Get all user sessions
	sessions, err := h.sessionRepo.GetByUserID(userID)
	if err != nil {
		_ = h.auditMw.LogAction(&userID, "session.get.error", "sessions", "Get sessions failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to get sessions")
		return
	}

	// Delete all sessions except the current one
	for _, session := range sessions {
		if session.SessionID != currentSession.SessionID {
			_ = h.sessionRepo.DeleteBySessionID(session.SessionID)
		}
	}

	// Log audit event
	_ = h.auditMw.LogAction(&userID, "session.delete_all_others", "sessions", "User deleted all other sessions", getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "All other sessions deleted successfully",
	})
}

// GetAllSessions gets all active sessions (admin only)
// @Summary Get all sessions
// @Description Get all active sessions for all users (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {array} object "List of all active sessions"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/sessions [get]
func (h *SessionHandler) GetAllSessions(w http.ResponseWriter, r *http.Request) {
	// Get all sessions from database
	query := `
		SELECT DISTINCT ON (session_id) 
			s.session_id, s.user_id, s.created_at, s.last_activity_at, s.ip_address, s.user_agent, s.expires_at,
			u.email, u.first_name, u.last_name
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.expires_at > NOW()
		ORDER BY s.session_id, s.last_activity_at DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get sessions")
		return
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var sessionID, ipAddress, userAgent, email, firstName, lastName string
		var userID uint
		var createdAt, lastActivityAt, expiresAt time.Time

		if err := rows.Scan(&sessionID, &userID, &createdAt, &lastActivityAt, &ipAddress, &userAgent, &expiresAt, &email, &firstName, &lastName); err != nil {
			continue
		}

		sessions = append(sessions, map[string]interface{}{
			"session_id":       sessionID,
			"user_id":          userID,
			"user_email":       email,
			"user_name":        firstName + " " + lastName,
			"created_at":       createdAt,
			"last_activity_at": lastActivityAt,
			"ip_address":       ipAddress,
			"user_agent":       userAgent,
			"expires_at":       expiresAt,
		})
	}

	respondWithJSON(w, http.StatusOK, sessions)
}

// DeleteUserSession deletes a specific session for any user (admin only)
// @Summary Delete user session
// @Description Delete a specific session by session_id (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param session_id query string true "Session ID to delete"
// @Success 200 {object} map[string]string "Session deleted successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/sessions/delete [delete]
func (h *SessionHandler) DeleteUserSession(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		respondWithError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Delete the session
	if err := h.sessionRepo.DeleteBySessionID(sessionID); err != nil {
		_ = h.auditMw.LogAction(&adminID, "admin.session.delete.error", "sessions", "Admin session deletion failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(&adminID, "admin.session.delete", "sessions", "Admin deleted session: "+sessionID, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Session deleted successfully",
	})
}

// DeleteAllUserSessions deletes all sessions for a specific user (admin only)
// @Summary Delete all sessions for a user
// @Description Delete all active sessions for a specific user (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id query int true "User ID"
// @Success 200 {object} map[string]string "All user sessions deleted"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin only"
// @Router /admin/sessions/delete-all [delete]
func (h *SessionHandler) DeleteAllUserSessions(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Delete all sessions for the user
	if err := h.sessionRepo.DeleteAllUserSessions(uint(userID)); err != nil {
		_ = h.auditMw.LogAction(&adminID, "admin.session.delete_all_user.error", "sessions", "Delete all user sessions failed: "+err.Error(), getIP(r), r.UserAgent())
		respondWithError(w, http.StatusInternalServerError, "Failed to delete sessions")
		return
	}

	// Log audit event
	_ = h.auditMw.LogAction(&adminID, "admin.session.delete_all_user", "sessions", "Admin deleted all sessions for user ID: "+userIDStr, getIP(r), r.UserAgent())

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "All user sessions deleted successfully",
	})
}
