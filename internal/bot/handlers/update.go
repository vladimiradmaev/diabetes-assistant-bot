package handlers

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/state"
	"github.com/vladimiradmaev/diabetes-helper/internal/interfaces"
)

// UpdateHandler handles telegram updates and coordinates other handlers
type UpdateHandler struct {
	api             *tgbotapi.BotAPI
	userService     interfaces.UserServiceInterface
	stateManager    state.StateManager
	callbackHandler *CallbackHandler
	commandHandler  *CommandHandler
	textHandler     *TextHandler
	photoHandler    *PhotoHandler
}

// NewUpdateHandler creates a new update handler
func NewUpdateHandler(
	api *tgbotapi.BotAPI,
	userService interfaces.UserServiceInterface,
	deps Dependencies,
	stateManager state.StateManager,
) *UpdateHandler {
	return &UpdateHandler{
		api:             api,
		userService:     userService,
		stateManager:    stateManager,
		callbackHandler: NewCallbackHandler(api, deps, stateManager),
		commandHandler:  NewCommandHandler(api, stateManager),
		textHandler:     NewTextHandler(api, deps, stateManager),
		photoHandler:    NewPhotoHandler(api, deps, stateManager),
	}
}

// Handle processes a telegram update
func (h *UpdateHandler) Handle(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil && update.CallbackQuery == nil {
		return nil
	}

	var userID int64

	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	}

	// Get or create user
	user, err := h.userService.RegisterUser(ctx, userID, "", "", "")
	if err != nil {
		log.Printf("Error getting/creating user: %v", err)
		return fmt.Errorf("failed to get/create user: %w", err)
	}

	// Handle different update types
	if update.CallbackQuery != nil {
		return h.callbackHandler.Handle(ctx, update.CallbackQuery, user)
	}

	if update.Message != nil {
		if update.Message.IsCommand() {
			return h.commandHandler.Handle(ctx, update.Message, user)
		}

		if update.Message.Text != "" {
			return h.textHandler.Handle(ctx, update.Message, user)
		}

		if len(update.Message.Photo) > 0 {
			return h.photoHandler.Handle(ctx, update.Message, user)
		}
	}

	return nil
}
