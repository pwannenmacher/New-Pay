package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds logger configuration
type Config struct {
	Level string
}

// Setup initializes the global logger with the specified configuration
func Setup(cfg Config) {
	level := parseLevel(cfg.Level)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format if needed
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05")),
				}
			}
			return a
		},
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// parseLevel converts a string log level to slog.Level
func parseLevel(levelStr string) slog.Level {
	level := strings.ToUpper(levelStr)
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLevel returns the normalized log level string
func GetLevel(levelStr string) string {
	level := strings.ToUpper(levelStr)
	switch level {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return level
	default:
		return "INFO"
	}
}
