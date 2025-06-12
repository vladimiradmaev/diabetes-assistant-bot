package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
)

type Config struct {
	TelegramToken string
	GeminiAPIKey  string
	DB            DBConfig
	Logger        LoggerConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type LoggerConfig struct {
	Level      logger.LogLevel
	OutputPath string
	Format     string
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation failed for field '%s' (value: '%s'): %s", e.Field, e.Value, e.Message)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return logger.LevelDebug
	case "info":
		return logger.LevelInfo
	case "warn", "warning":
		return logger.LevelWarn
	case "error":
		return logger.LevelError
	default:
		return logger.LevelInfo
	}
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	var errors []ValidationError

	// Validate required fields
	if c.TelegramToken == "" {
		errors = append(errors, ValidationError{
			Field:   "TELEGRAM_BOT_TOKEN",
			Value:   "",
			Message: "telegram bot token is required",
		})
	} else if !isValidTelegramToken(c.TelegramToken) {
		errors = append(errors, ValidationError{
			Field:   "TELEGRAM_BOT_TOKEN",
			Value:   maskSensitiveValue(c.TelegramToken),
			Message: "telegram bot token format is invalid (should be like '123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11')",
		})
	}

	if c.GeminiAPIKey == "" {
		errors = append(errors, ValidationError{
			Field:   "GEMINI_API_KEY",
			Value:   "",
			Message: "gemini API key is required",
		})
	} else if !isValidGeminiAPIKey(c.GeminiAPIKey) {
		errors = append(errors, ValidationError{
			Field:   "GEMINI_API_KEY",
			Value:   maskSensitiveValue(c.GeminiAPIKey),
			Message: "gemini API key format is invalid (should start with 'AIza')",
		})
	}

	// Validate database configuration
	if dbErrors := c.DB.Validate(); len(dbErrors) > 0 {
		errors = append(errors, dbErrors...)
	}

	// Validate logger configuration
	if logErrors := c.Logger.Validate(); len(logErrors) > 0 {
		errors = append(errors, logErrors...)
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", formatValidationErrors(errors))
	}

	return nil
}

// Validate validates database configuration
func (db *DBConfig) Validate() []ValidationError {
	var errors []ValidationError

	// Validate host
	if db.Host == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_HOST",
			Value:   db.Host,
			Message: "database host cannot be empty",
		})
	}

	// Validate port
	if db.Port == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_PORT",
			Value:   db.Port,
			Message: "database port cannot be empty",
		})
	} else {
		if port, err := strconv.Atoi(db.Port); err != nil {
			errors = append(errors, ValidationError{
				Field:   "DB_PORT",
				Value:   db.Port,
				Message: "database port must be a valid number",
			})
		} else if port < 1 || port > 65535 {
			errors = append(errors, ValidationError{
				Field:   "DB_PORT",
				Value:   db.Port,
				Message: "database port must be between 1 and 65535",
			})
		}
	}

	// Validate user
	if db.User == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_USER",
			Value:   db.User,
			Message: "database user cannot be empty",
		})
	}

	// Validate database name
	if db.DBName == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_NAME",
			Value:   db.DBName,
			Message: "database name cannot be empty",
		})
	}

	// Validate host format (basic check)
	if db.Host != "localhost" && db.Host != "db" && net.ParseIP(db.Host) == nil {
		// Check if it's a valid hostname
		if !isValidHostname(db.Host) {
			errors = append(errors, ValidationError{
				Field:   "DB_HOST",
				Value:   db.Host,
				Message: "database host must be a valid IP address or hostname",
			})
		}
	}

	return errors
}

// Validate validates logger configuration
func (l *LoggerConfig) Validate() []ValidationError {
	var errors []ValidationError

	// Validate output path
	if l.OutputPath == "" {
		errors = append(errors, ValidationError{
			Field:   "LOG_OUTPUT",
			Value:   l.OutputPath,
			Message: "log output path cannot be empty",
		})
	}

	// Validate format
	if l.Format != "json" && l.Format != "text" {
		errors = append(errors, ValidationError{
			Field:   "LOG_FORMAT",
			Value:   l.Format,
			Message: "log format must be 'json' or 'text'",
		})
	}

	return errors
}

// Helper functions
func formatValidationErrors(errors []ValidationError) string {
	var messages []string
	for _, err := range errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// Remove trailing dot
	if hostname[len(hostname)-1] == '.' {
		hostname = hostname[:len(hostname)-1]
	}

	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		for _, r := range label {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}
	return true
}

func isValidTelegramToken(token string) bool {
	// Telegram bot token format: bot_id:auth_token
	// Example: 123456789:AAHdqwcvCH1vGWzxfSeofSAs0l5PALDsaw
	if len(token) < 35 || len(token) > 50 {
		return false
	}

	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}

	// Check bot_id (should be numeric)
	botID := parts[0]
	if len(botID) < 8 || len(botID) > 12 {
		return false
	}
	for _, r := range botID {
		if r < '0' || r > '9' {
			return false
		}
	}

	// Check auth_token (should be alphanumeric with some special chars)
	authToken := parts[1]
	if len(authToken) < 25 || len(authToken) > 40 {
		return false
	}

	return true
}

func isValidGeminiAPIKey(key string) bool {
	// Gemini API keys typically start with "AIza" and are about 39 characters long
	return len(key) >= 35 && len(key) <= 45 && strings.HasPrefix(key, "AIza")
}

func maskSensitiveValue(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func Load() (*Config, error) {
	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		DB: DBConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			DBName:   getEnvOrDefault("DB_NAME", "diabetes_helper"),
		},
		Logger: LoggerConfig{
			Level:      parseLogLevel(getEnvOrDefault("LOG_LEVEL", "info")),
			OutputPath: getEnvOrDefault("LOG_OUTPUT", "logs/app.log"),
			Format:     getEnvOrDefault("LOG_FORMAT", "json"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
