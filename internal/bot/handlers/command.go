package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/menus"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/state"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
)

// CommandHandler handles bot commands
type CommandHandler struct {
	api          *tgbotapi.BotAPI
	stateManager *state.Manager
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(api *tgbotapi.BotAPI, stateManager *state.Manager) *CommandHandler {
	return &CommandHandler{
		api:          api,
		stateManager: stateManager,
	}
}

// Handle processes a command message
func (h *CommandHandler) Handle(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	logger.Infof("Handling command %s from user %d", message.Command(), user.ID)

	switch message.Command() {
	case "start":
		h.stateManager.SetUserState(user.TelegramID, state.None)
		return menus.SendMainMenu(h.api, message.Chat.ID)
	case "help":
		return h.handleHelp(message.Chat.ID)
	default:
		return h.handleUnknownCommand(message.Chat.ID)
	}
}

// handleHelp handles the /help command
func (h *CommandHandler) handleHelp(chatID int64) error {
	text := `Доступные команды:
/start - Показать главное меню
/help - Показать это сообщение

Как указать вес блюда:
1. Нажмите кнопку "🍽️ Анализ еды"
2. Отправьте фото еды
3. В подписи к фото напишите только число - вес в граммах
Пример: "150" или "200"

Если вес не указан, бот попробует оценить его автоматически.`

	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.api.Send(msg)
	return err
}

// handleUnknownCommand handles unknown commands
func (h *CommandHandler) handleUnknownCommand(chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /help для просмотра доступных команд.")
	_, err := h.api.Send(msg)
	return err
}
