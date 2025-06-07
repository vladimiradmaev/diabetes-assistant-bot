package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/vladimiradmaev/diabetes-helper/internal/config"
)

func main() {
	fmt.Println("🔍 Проверка конфигурации...")

	// Загружаем .env файл если есть
	if err := godotenv.Load(); err != nil {
		fmt.Printf("⚠️  .env файл не найден: %v\n", err)
	}

	// Загружаем и валидируем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Ошибка валидации конфигурации:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Конфигурация валидна!")
	fmt.Printf("📋 Детали конфигурации:\n")
	fmt.Printf("  - Telegram Token: %s\n", maskToken(cfg.TelegramToken))
	fmt.Printf("  - Gemini API Key: %s\n", maskToken(cfg.GeminiAPIKey))
	fmt.Printf("  - DB Host: %s\n", cfg.DB.Host)
	fmt.Printf("  - DB Port: %s\n", cfg.DB.Port)
	fmt.Printf("  - DB User: %s\n", cfg.DB.User)
	fmt.Printf("  - DB Name: %s\n", cfg.DB.DBName)
	fmt.Printf("  - Log Level: %v\n", cfg.Logger.Level)
	fmt.Printf("  - Log Output: %s\n", cfg.Logger.OutputPath)
	fmt.Printf("  - Log Format: %s\n", cfg.Logger.Format)
}

func maskToken(token string) string {
	if token == "" {
		return "<не установлен>"
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
