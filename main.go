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
	"github.com/vladimiradmaev/diabetes-helper/internal/interfaces"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
	"github.com/vladimiradmaev/diabetes-helper/internal/services"
)

func main() {
	// Initialize basic logger first
	if err := logger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		logger.Warning("Warning: .env file not found", "error", err.Error())
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Reinitialize logger with config
	if err := logger.InitWithConfig(logger.Config{
		Level:      cfg.Logger.Level,
		OutputPath: cfg.Logger.OutputPath,
		Format:     cfg.Logger.Format,
	}); err != nil {
		logger.Error("Failed to reinitialize logger with config", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting Diabetes Helper Bot...",
		"version", "1.0.0",
		"log_level", cfg.Logger.Level,
		"log_format", cfg.Logger.Format)
	logger.Info("Configuration loaded successfully")

	db, err := database.NewPostgresDB(cfg.DB)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.Info("Database connection established and migrations completed")

	// Initialize AI service
	aiService := services.NewAIService(cfg.GeminiAPIKey)

	// Initialize services implementing interfaces
	var userService interfaces.UserServiceInterface = services.NewUserService(db)
	var foodAnalysisService interfaces.FoodAnalysisServiceInterface = services.NewFoodAnalysisService(aiService, db)
	var bloodSugarService interfaces.BloodSugarServiceInterface = services.NewBloodSugarService(db)
	var insulinService interfaces.InsulinServiceInterface = services.NewInsulinService(db)
	logger.Info("Services initialized successfully")

	// Initialize bot with interfaces
	telegramBot, err := bot.NewBot(cfg.TelegramToken, userService, foodAnalysisService, bloodSugarService, insulinService)
	if err != nil {
		logger.Error("Failed to create bot", "error", err)
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
				logger.Error("Bot stopped with error", "error", err)
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
