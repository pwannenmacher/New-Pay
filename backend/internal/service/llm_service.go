package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const fallbackSummaryMessage = "Automatische Zusammenfassung nicht verfügbar."

// LLMService handles interaction with the Language Model
type LLMService struct {
	baseURL string
	model   string
	enabled bool
	client  *http.Client
}

// NewLLMService creates a new LLM service
func NewLLMService(baseURL, model string, enabled bool) *LLMService {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	}
	return &LLMService{
		baseURL: baseURL,
		model:   model,
		enabled: enabled,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// SummarizeComments summarizes a list of comments into a single proposal
func (s *LLMService) SummarizeComments(comments []string) (string, error) {
	if len(comments) == 0 {
		return "", nil
	}

	// Fallback if service is disabled
	if !s.enabled {
		return fallbackSummaryMessage, nil
	}

	prompt := s.buildPrompt(comments)

	reqBody := ollamaRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fallbackSummaryMessage, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.client.Post(fmt.Sprintf("%s/api/generate", s.baseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("LLM service unreachable", "error", err)
		return fallbackSummaryMessage, nil // Return fallback instead of error
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("LLM service returned non-200 status", "status", resp.StatusCode, "body", string(bodyBytes))

		// If model not found, try to pull it
		if resp.StatusCode == http.StatusNotFound && strings.Contains(string(bodyBytes), "model") {
			go s.PullModel()
		}

		return fallbackSummaryMessage, nil
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		slog.Error("Failed to decode LLM response", "error", err)
		return fallbackSummaryMessage, nil
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

func (s *LLMService) buildPrompt(comments []string) string {
	var sb strings.Builder
	sb.WriteString("Fasse die folgenden Kommentare von Reviewern zu einer Kategorie zusammen. ")
	sb.WriteString("Erstelle einen konstruktiven, zusammenfassenden Vorschlag für das Feedback an den Mitarbeiter. ")
	sb.WriteString("Es soll klar, prägnant und hilfreich sein. Vermeide Wiederholungen und fasse ähnliche Punkte zusammen. ")
	sb.WriteString("Nutze eine positive und unterstützende Sprache. ")
	sb.WriteString("Es dürfen keine direkten Zitate aus den Kommentaren verwendet werden. ")
	sb.WriteString("Es darf kein Rückschluss auf einzelne Reviewer gezogen werden. ")
	sb.WriteString("Antworte NUR mit dem zusammenfassenden Text auf Deutsch.\n\n")

	for i, c := range comments {
		sb.WriteString(fmt.Sprintf("Kommentar %d: %s\n", i+1, c))
	}

	return sb.String()
}

// PullModel triggers a model pull
func (s *LLMService) PullModel() {
	slog.Info("Attempting to pull LLM model", "model", s.model)

	reqBody := map[string]string{
		"name": s.model,
	}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := s.client.Post(fmt.Sprintf("%s/api/pull", s.baseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("Failed to trigger model pull", "error", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("Failed to pull model", "status", resp.StatusCode, "body", string(bodyBytes))
		return
	}

	slog.Info("Model pull triggered successfully", "model", s.model)
}
