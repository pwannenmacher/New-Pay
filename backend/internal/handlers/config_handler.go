package handlers

import (
	"net/http"

	"new-pay/internal/config"
)

// ConfigHandler handles configuration requests
type ConfigHandler struct {
	config      *config.Config
	authService AuthServiceInterface
}

// AuthServiceInterface defines the methods needed from AuthService
type AuthServiceInterface interface {
	CountAllUsers() (int, error)
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(cfg *config.Config, authService AuthServiceInterface) *ConfigHandler {
	return &ConfigHandler{
		config:      cfg,
		authService: authService,
	}
}

// GetOAuthConfig returns the OAuth configuration for the frontend
// @Summary Get OAuth configuration
// @Description Get public OAuth configuration (all enabled providers)
// @Tags Configuration
// @Produce json
// @Success 200 {object} map[string]interface{} "OAuth configuration"
// @Router /config/oauth [get]
func (h *ConfigHandler) GetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build response with enabled providers
	type ProviderInfo struct {
		Name string `json:"name"`
	}

	var enabledProviders []ProviderInfo
	for _, provider := range h.config.OAuth.Providers {
		if provider.Enabled {
			enabledProviders = append(enabledProviders, ProviderInfo{
				Name: provider.Name,
			})
		}
	}

	oauthConfig := map[string]interface{}{
		"enabled":   len(enabledProviders) > 0,
		"providers": enabledProviders,
	}

	respondWithJSON(w, http.StatusOK, oauthConfig)
}

// GetAppConfig returns the public app configuration for the frontend
// @Summary Get app configuration
// @Description Get public app configuration (registration settings)
// @Tags Configuration
// @Produce json
// @Success 200 {object} map[string]interface{} "App configuration"
// @Router /config/app [get]
func (h *ConfigHandler) GetAppConfig(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if registration should be enabled
	// Registration is always enabled if no users exist (for first user)
	enableRegistration := h.config.App.EnableRegistration
	if !enableRegistration {
		// Check if database is empty - if yes, allow registration for first user
		userCount, err := h.authService.CountAllUsers()
		if err == nil && userCount == 0 {
			enableRegistration = true
		}
	}

	appConfig := map[string]interface{}{
		"enable_registration":       enableRegistration,
		"enable_oauth_registration": h.config.App.EnableOAuthRegistration,
	}

	respondWithJSON(w, http.StatusOK, appConfig)
}
