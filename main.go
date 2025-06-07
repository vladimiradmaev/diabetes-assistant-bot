package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot"
	"github.com/vladimiradmaev/diabetes-helper/internal/config"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
	"github.com/vladimiradmaev/diabetes-helper/internal/services"
)

func main() {
	// Initialize logger first
	if err := logger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	logger.Info("Starting Diabetes Helper Bot...")

	if err := godotenv.Load(); err != nil {
		logger.Warning("Warning: .env file not found")
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Errorf("Failed to load config: %v", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully")

	db, err := database.NewPostgresDB(cfg.DB)
	if err != nil {
		logger.Errorf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	logger.Info("Database connection established and migrations completed")

	// Initialize AI service
	aiService := services.NewAIService(cfg.GeminiAPIKey)
	userService := services.NewUserService(db)
	foodAnalysisService := services.NewFoodAnalysisService(aiService, db)
	bloodSugarService := services.NewBloodSugarService(db)
	insulinService := services.NewInsulinService(db)
	logger.Info("Services initialized successfully")

	// Initialize bot
	telegramBot, err := bot.NewBot(cfg.TelegramToken, userService, foodAnalysisService, bloodSugarService, insulinService)
	if err != nil {
		logger.Errorf("Failed to create bot: %v", err)
		os.Exit(1)
	}
	logger.Info("Bot initialized successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start bot in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting bot...")
		if err := telegramBot.Start(ctx); err != nil {
			if err != context.Canceled {
				logger.Errorf("Bot stopped with error: %v", err)
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Bot is running. Press Ctrl+C to stop.")
	<-sigChan
	logger.Info("Received shutdown signal, stopping bot...")

	cancel()
	wg.Wait()
	logger.Info("Bot stopped gracefully")
}
