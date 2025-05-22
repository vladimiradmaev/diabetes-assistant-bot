package config

import (
	"os"
)

type Config struct {
	TelegramToken string
	GeminiAPIKey  string
	OpenAIAPIKey  string
	DB            DBConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func Load() (*Config, error) {
	return &Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
		},
	}, nil
}
