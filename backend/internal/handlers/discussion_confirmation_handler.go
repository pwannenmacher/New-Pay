package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"new-pay/internal/middleware"
	"new-pay/internal/models"
	"new-pay/internal/repository"
)

type DiscussionConfirmationHandler struct {
	confirmationRepo *repository.DiscussionConfirmationRepository
	assessmentRepo   *repository.SelfAssessmentRepository
	userRepo         *repository.UserRepository
}

func NewDiscussionConfirmationHandler(
	confirmationRepo *repository.DiscussionConfirmationRepository,
	assessmentRepo *repository.SelfAssessmentRepository,
	userRepo *repository.UserRepository,
) *DiscussionConfirmationHandler {
	return &DiscussionConfirmationHandler{
		confirmationRepo: confirmationRepo,
		assessmentRepo:   assessmentRepo,
		userRepo:         userRepo,
	}
}

// CreateConfirmation creates a new discussion confirmation
func (h *DiscussionConfirmationHandler) CreateConfirmation(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Get assessment ID from URL
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var assessmentID uint64
	var parseErr error

	for i, part := range pathParts {
		if part == "discussion" && i+1 < len(pathParts) {
			assessmentID, parseErr = strconv.ParseUint(pathParts[i+1], 10, 32)
			break
		}
	}

	if parseErr != nil || assessmentID == 0 {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	// Get assessment
	assessment, err := h.assessmentRepo.GetByID(uint(assessmentID))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Assessment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if assessment is in discussion status
	if assessment.Status != "discussion" {
		http.Error(w, "Assessment must be in discussion status", http.StatusBadRequest)
		return
	}

	// Determine user type
	// Get user roles from repository
	roles, err := h.userRepo.GetUserRoles(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userRoleNames := make([]string, len(roles))
	for i, role := range roles {
		userRoleNames[i] = role.Name
	}

	userType := ""
	isOwner := assessment.UserID == user.ID
	isReviewer := contains(userRoleNames, "reviewer")

	if isReviewer && !isOwner {
		userType = "reviewer"
	} else if isOwner {
		userType = "owner"
	} else {
		http.Error(w, "User must be either reviewer or owner of assessment", http.StatusForbidden)
		return
	}

	// If owner, check if at least one reviewer has confirmed
	if userType == "owner" {
		hasReviewerConf, err := h.confirmationRepo.HasReviewerConfirmation(uint(assessmentID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !hasReviewerConf {
			http.Error(w, "At least one reviewer must confirm first", http.StatusBadRequest)
			return
		}
	}

	// Check if user already confirmed
	existing, err := h.confirmationRepo.GetByAssessmentAndUser(uint(assessmentID), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "User has already confirmed", http.StatusConflict)
		return
	}

	// Create confirmation
	confirmation := &models.DiscussionConfirmation{
		AssessmentID: uint(assessmentID),
		UserID:       user.ID,
		UserType:     userType,
	}

	if err := h.confirmationRepo.Create(confirmation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(confirmation)
}

// GetConfirmations retrieves all confirmations for an assessment
func (h *DiscussionConfirmationHandler) GetConfirmations(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (just for auth check)
	_, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get assessment ID from URL
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var assessmentID uint64
	var parseErr error

	for i, part := range pathParts {
		if part == "discussion" && i+1 < len(pathParts) {
			assessmentID, parseErr = strconv.ParseUint(pathParts[i+1], 10, 32)
			break
		}
	}

	if parseErr != nil || assessmentID == 0 {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	// Get confirmations
	confirmations, err := h.confirmationRepo.GetByAssessment(uint(assessmentID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(confirmations)
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
