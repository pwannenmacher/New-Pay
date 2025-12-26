package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"new-pay/internal/service"
)

type DiscussionHandler struct {
	discussionService *service.DiscussionService
}

func NewDiscussionHandler(discussionService *service.DiscussionService) *DiscussionHandler {
	return &DiscussionHandler{
		discussionService: discussionService,
	}
}

// GetDiscussionResult retrieves the discussion result for an assessment
// GET /api/v1/discussion/:id
func (h *DiscussionHandler) GetDiscussionResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract assessment ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	assessmentID, err := strconv.ParseUint(parts[len(parts)-1], 10, 32)
	if err != nil {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	result, err := h.discussionService.GetDiscussionResult(uint(assessmentID))
	if err != nil {
		slog.Error("Failed to get discussion result", "error", err)
		http.Error(w, "Failed to get discussion result", http.StatusInternalServerError)
		return
	}

	if result == nil {
		http.Error(w, "Discussion result not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// UpdateDiscussionNote updates the discussion note
// PUT /api/v1/discussion/:id/note
func (h *DiscussionHandler) UpdateDiscussionNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract assessment ID from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var assessmentID uint64
	var err error

	for i, part := range pathParts {
		if part == "discussion" && i+1 < len(pathParts) {
			assessmentID, err = strconv.ParseUint(pathParts[i+1], 10, 32)
			break
		}
	}

	if err != nil || assessmentID == 0 {
		http.Error(w, "Invalid assessment ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.discussionService.UpdateDiscussionNote(uint(assessmentID), req.Note); err != nil {
		slog.Error("Failed to update discussion note", "error", err)
		http.Error(w, "Failed to update discussion note", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Discussion note updated successfully"})
}
