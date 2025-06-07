package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/handlers"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/state"
	"github.com/vladimiradmaev/diabetes-helper/internal/interfaces"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
)

// Bot represents the main bot structure
type Bot struct {
	api           *tgbotapi.BotAPI
	updateHandler *handlers.UpdateHandler
}

// NewBot creates a new bot instance
func NewBot(
	token string,
	redisHost, redisPort string,
	userService interfaces.UserServiceInterface,
	foodAnalysisSvc interfaces.FoodAnalysisServiceInterface,
	bloodSugarSvc interfaces.BloodSugarServiceInterface,
	insulinSvc interfaces.InsulinServiceInterface,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	logger.Infof("Bot authorized on account %s", api.Self.UserName)

	// Create dependencies for handlers
	deps := handlers.Dependencies{
		UserService:     userService,
		FoodAnalysisSvc: foodAnalysisSvc,
		BloodSugarSvc:   bloodSugarSvc,
		InsulinSvc:      insulinSvc,
	}

	// Create Redis state manager
	stateManager, err := state.NewRedisManager(redisHost, redisPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis state manager: %w", err)
	}

	// Create update handler
	updateHandler := handlers.NewUpdateHandler(api, userService, deps, stateManager)

	return &Bot{
		api:           api,
		updateHandler: updateHandler,
	}, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	logger.Info("Starting bot...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Bot is shutting down...")
			b.api.StopReceivingUpdates()
			logger.Info("Bot stopped gracefully")
			return nil
		case update := <-updates:
			go func(update tgbotapi.Update) {
				if err := b.updateHandler.Handle(ctx, update); err != nil {
					logger.Errorf("Error handling update: %v", err)
				}
			}(update)
		}
	}
}
