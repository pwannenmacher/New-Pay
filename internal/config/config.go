package config

import (
	"fmt"
	"os"
	"strconv"
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
	OAuth     OAuthConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	App       AppConfig
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

// OAuthConfig holds OAuth-related configuration
type OAuthConfig struct {
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	FacebookClientID     string
	FacebookClientSecret string
	FacebookRedirectURL  string
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
	Env      string
	Name     string
	Version  string
	LogLevel string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

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
			SMTPFrom:         getEnv("SMTP_FROM", "noreply@newpay.com"),
			VerificationURL:  getEnv("EMAIL_VERIFICATION_URL", "http://localhost:8080/api/v1/auth/verify-email"),
			PasswordResetURL: getEnv("PASSWORD_RESET_URL", "http://localhost:8080/api/v1/auth/reset-password"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
			GoogleRedirectURL:    getEnv("GOOGLE_REDIRECT_URL", ""),
			FacebookClientID:     getEnv("FACEBOOK_CLIENT_ID", ""),
			FacebookClientSecret: getEnv("FACEBOOK_CLIENT_SECRET", ""),
			FacebookRedirectURL:  getEnv("FACEBOOK_REDIRECT_URL", ""),
		},
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
			Env:      getEnv("APP_ENV", "development"),
			Name:     getEnv("APP_NAME", "NewPay"),
			Version:  getEnv("APP_VERSION", "1.0.0"),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
	}

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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
		// Simple split by comma
		var result []string
		for _, v := range splitByComma(value) {
			if trimmed := trimSpace(v); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitByComma(s string) []string {
	var result []string
	current := ""
	for _, char := range s {
		if char == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
