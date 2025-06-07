package config

import (
	"os"
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

func Load() (*Config, error) {
	return &Config{
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
	}, nil
}
