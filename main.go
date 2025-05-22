package main

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot"
	"github.com/vladimiradmaev/diabetes-helper/internal/config"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/services"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Diabetes Helper Bot...")

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded successfully")

	db, err := database.NewPostgresDB(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established and migrations completed")

	// Initialize services
	aiService := services.NewAIService(cfg.GeminiAPIKey, cfg.OpenAIAPIKey)
	userService := services.NewUserService(db)
	foodAnalysisService := services.NewFoodAnalysisService(aiService, db)
	bloodSugarService := services.NewBloodSugarService(db)
	log.Println("Services initialized successfully")

	// Initialize bot
	telegramBot, err := bot.NewBot(cfg.TelegramToken, userService, foodAnalysisService, bloodSugarService)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Println("Bot initialized successfully")

	// Start bot in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting bot...")
		if err := telegramBot.Start(context.Background()); err != nil {
			log.Printf("Bot stopped with error: %v", err)
			os.Exit(1)
		}
	}()

	log.Println("Bot is running. Press Ctrl+C to stop.")
	wg.Wait()
}
