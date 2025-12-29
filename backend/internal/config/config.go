package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	Session   SessionConfig
	Email     EmailConfig
	OAuth     OAuthProvidersConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	App       AppConfig
	Log       LogConfig
	Scheduler SchedulerConfig
	Vault     VaultConfig
	LLM       LLMConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string
	Port         string
	TimeoutRead  time.Duration
	TimeoutWrite time.Duration
	TimeoutIdle  time.Duration
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	RefreshExpiration time.Duration
}

// SessionConfig holds session-related configuration
type SessionConfig struct {
	Timeout time.Duration
}

// EmailConfig holds email-related configuration
type EmailConfig struct {
	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	SMTPFrom         string
	VerificationURL  string
	PasswordResetURL string
}

// OAuthProviderConfig holds configuration for a single OAuth provider
type OAuthProviderConfig struct {
	Name         string
	Enabled      bool
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	GroupMapping map[string]string // Maps OAuth groups to internal roles (e.g., "admin-group": "admin")
	GroupsClaim  string            // Claim name containing groups (default: "groups")
	DefaultRole  string            // Default role to assign if no groups match (optional, e.g., "user")
}

// OAuthProvidersConfig holds configuration for all OAuth providers
type OAuthProvidersConfig struct {
	RedirectURL         string
	FrontendCallbackURL string
	Providers           []OAuthProviderConfig
}

// CORSConfig holds CORS-related configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled  bool
	Requests int
	Duration time.Duration
}

// AppConfig holds general application configuration
type AppConfig struct {
	Env                     string
	Name                    string
	Version                 string
	LogLevel                string
	EnableRegistration      bool
	EnableOAuthRegistration bool
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level string
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	DraftReminderCron     string // e.g., "0 9 * * 1" (Monday 9 AM)
	ReviewerSummaryCron   string // e.g., "0 8 * * *" (Daily 8 AM)
	ReminderIntervalMins  int    // Interval in minutes for draft reminders (default: 10080 = 7 days)
	EnableDraftReminders  bool   // Enable/disable draft reminders
	EnableReviewerSummary bool   // Enable/disable reviewer summaries
}

// VaultConfig holds Vault-related configuration
type VaultConfig struct {
	Address      string
	Token        string
	TransitMount string
	Enabled      bool
}

// LLMConfig holds LLM-related configuration
type LLMConfig struct {
	BaseURL string
	Model   string
	Enabled bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	// Try to load from most specific to least specific
	// godotenv doesn't override already-set variables, so order matters
	_ = godotenv.Load("backend/.env") // When running from project root (local dev)
	_ = godotenv.Load(".env")         // When running from backend dir or Docker
	_ = godotenv.Load("../.env")      // Fallback

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "localhost"),
			Port:         getEnv("SERVER_PORT", "8080"),
			TimeoutRead:  getDurationEnv("SERVER_TIMEOUT_READ", 15*time.Second),
			TimeoutWrite: getDurationEnv("SERVER_TIMEOUT_WRITE", 15*time.Second),
			TimeoutIdle:  getDurationEnv("SERVER_TIMEOUT_IDLE", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "newpay"),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "newpay_db"),
			SSLMode:         getEnv("DB_SSLMODE", "prefer"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", ""),
			Expiration:        getDurationEnv("JWT_EXPIRATION", 24*time.Hour),
			RefreshExpiration: getDurationEnv("JWT_REFRESH_EXPIRATION", 168*time.Hour),
		},
		Session: SessionConfig{
			Timeout: getDurationEnv("SESSION_TIMEOUT", 30*time.Minute),
		},
		Email: EmailConfig{
			SMTPHost:         getEnv("SMTP_HOST", ""),
			SMTPPort:         getEnv("SMTP_PORT", "587"),
			SMTPUsername:     getEnv("SMTP_USERNAME", ""),
			SMTPPassword:     getEnv("SMTP_PASSWORD", ""),
			SMTPFrom:         getEnv("SMTP_FROM", "noreply@example.com"),
			VerificationURL:  getEnv("EMAIL_VERIFICATION_URL", "http://localhost:8080/api/v1/auth/verify-email"),
			PasswordResetURL: getEnv("PASSWORD_RESET_URL", "http://localhost:8080/api/v1/auth/reset-password"),
		},
		OAuth: loadOAuthProviders(),
		CORS: CORSConfig{
			AllowedOrigins:   getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			AllowedMethods:   getSliceEnv("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getSliceEnv("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type"}),
			ExposedHeaders:   getSliceEnv("CORS_EXPOSED_HEADERS", []string{"Link"}),
			AllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getIntEnv("CORS_MAX_AGE", 300),
		},
		RateLimit: RateLimitConfig{
			Enabled:  getBoolEnv("RATE_LIMIT_ENABLED", true),
			Requests: getIntEnv("RATE_LIMIT_REQUESTS", 100),
			Duration: getDurationEnv("RATE_LIMIT_DURATION", 1*time.Minute),
		},
		App: AppConfig{
			Env:                     getEnv("APP_ENV", "development"),
			Name:                    getEnv("APP_NAME", "NewPay"),
			Version:                 getEnv("APP_VERSION", "1.0.0"),
			LogLevel:                getEnv("LOG_LEVEL", "info"),
			EnableRegistration:      getBoolEnv("ENABLE_REGISTRATION", false),
			EnableOAuthRegistration: getBoolEnv("ENABLE_OAUTH_REGISTRATION", false),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Scheduler: SchedulerConfig{
			DraftReminderCron:     getEnv("SCHEDULER_DRAFT_REMINDER_CRON", "0 9 * * 1"),   // Monday 9 AM
			ReviewerSummaryCron:   getEnv("SCHEDULER_REVIEWER_SUMMARY_CRON", "0 8 * * *"), // Daily 8 AM
			ReminderIntervalMins:  getIntEnv("SCHEDULER_REMINDER_INTERVAL_MINS", 10080),   // 7 days = 10080 minutes
			EnableDraftReminders:  getBoolEnv("SCHEDULER_ENABLE_DRAFT_REMINDERS", true),
			EnableReviewerSummary: getBoolEnv("SCHEDULER_ENABLE_REVIEWER_SUMMARY", true),
		},
		Vault: VaultConfig{
			Address:      getEnv("VAULT_ADDR", "http://localhost:8200"),
			Token:        getEnv("VAULT_TOKEN", ""),
			TransitMount: getEnv("VAULT_TRANSIT_MOUNT", "transit"),
			Enabled:      getBoolEnv("VAULT_ENABLED", true),
		},
		LLM: LLMConfig{
			BaseURL: getEnv("LLM_BASE_URL", "http://localhost:11434"),
			Model:   getEnv("LLM_MODEL", "llama3"),
			Enabled: getBoolEnv("LLM_ENABLED", true),
		},
	}

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadOAuthProviders loads all OAuth provider configurations from environment variables
func loadOAuthProviders() OAuthProvidersConfig {
	redirectURL := getEnv("OAUTH_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/callback")
	frontendCallbackURL := getEnv("OAUTH_FRONTEND_CALLBACK_URL", "http://localhost:3001/oauth/callback")

	var providers []OAuthProviderConfig

	// Scan up to 50 providers (reasonable maximum)
	for i := 1; i <= 50; i++ {
		prefix := fmt.Sprintf("OAUTH_%d_", i)

		// Check if this provider is configured (at minimum needs a name)
		name := getEnv(prefix+"NAME", "")
		if name == "" {
			continue
		}

		enabled := getBoolEnv(prefix+"ENABLED", true)

		provider := OAuthProviderConfig{
			Name:         name,
			Enabled:      enabled,
			ClientID:     getEnv(prefix+"CLIENT_ID", ""),
			ClientSecret: getEnv(prefix+"CLIENT_SECRET", ""),
			AuthURL:      getEnv(prefix+"AUTH_URL", ""),
			TokenURL:     getEnv(prefix+"TOKEN_URL", ""),
			UserInfoURL:  getEnv(prefix+"USER_INFO_URL", ""),
			GroupMapping: parseGroupMapping(getEnv(prefix+"GROUP_MAPPING", "")),
			GroupsClaim:  getEnv(prefix+"GROUPS_CLAIM", "groups"),
			DefaultRole:  getEnv(prefix+"DEFAULT_ROLE", ""),
		}

		// Only add provider if it has all required fields
		if provider.ClientID != "" && provider.ClientSecret != "" &&
			provider.AuthURL != "" && provider.TokenURL != "" &&
			provider.UserInfoURL != "" {
			providers = append(providers, provider)
		}
	}

	return OAuthProvidersConfig{
		RedirectURL:         redirectURL,
		FrontendCallbackURL: frontendCallbackURL,
		Providers:           providers,
	}
}

// parseGroupMapping parses the group mapping configuration
// Format: "oauth-group-1:role-1,oauth-group-2:role-2"
// Example: "admins:admin,developers:user,reviewers:reviewer"
func parseGroupMapping(mappingStr string) map[string]string {
	mapping := make(map[string]string)
	if mappingStr == "" {
		return mapping
	}

	pairs := strings.Split(mappingStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			oauthGroup := strings.TrimSpace(parts[0])
			role := strings.TrimSpace(parts[1])
			if oauthGroup != "" && role != "" {
				mapping[oauthGroup] = role
			}
		}
	}

	return mapping
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Database.Password == "" && c.App.Env == "production" {
		return fmt.Errorf("DB_PASSWORD is required in production")
	}
	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma and trim whitespace
		parts := strings.Split(value, ",")
		var result []string
		for _, v := range parts {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
