package handlers

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/menus"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/state"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
)

// CallbackHandler handles callback query messages
type CallbackHandler struct {
	api          *tgbotapi.BotAPI
	deps         Dependencies
	stateManager *state.Manager
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(api *tgbotapi.BotAPI, deps Dependencies, stateManager *state.Manager) *CallbackHandler {
	return &CallbackHandler{
		api:          api,
		deps:         deps,
		stateManager: stateManager,
	}
}

// Handle processes a callback query
func (h *CallbackHandler) Handle(ctx context.Context, query *tgbotapi.CallbackQuery, user *database.User) error {
	// Answer the callback query first
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := h.api.Request(callback); err != nil {
		return err
	}

	switch query.Data {
	case "analyze_food":
		return h.handleAnalyzeFood(query.Message.Chat.ID, user)
	case "blood_sugar":
		return h.handleBloodSugar(query.Message.Chat.ID, user)
	case "settings":
		return h.handleSettings(query.Message.Chat.ID)
	case "insulin_ratio":
		return h.handleInsulinRatio(query.Message.Chat.ID, user)
	case "add_insulin_ratio":
		return h.handleAddInsulinRatio(query.Message.Chat.ID, user)
	case "main_menu":
		return h.handleMainMenu(query.Message.Chat.ID, user)
	case "edit_insulin_ratio":
		return h.handleEditInsulinRatio(query.Message.Chat.ID, user)
	case "clear_and_add_ratio":
		return h.handleClearAndAddRatio(query.Message.Chat.ID, user)
	case "delete_insulin_ratio":
		return h.handleDeleteInsulinRatio(query.Message.Chat.ID, user)
	case "clear_ratios":
		return h.handleClearRatios(query.Message.Chat.ID, user)
	case "active_insulin_time":
		return h.handleActiveInsulinTime(query.Message.Chat.ID, user)
	default:
		return h.handleUnknownCallback(query.Message.Chat.ID)
	}
}

// handleAnalyzeFood handles analyze food callback
func (h *CallbackHandler) handleAnalyzeFood(chatID int64, user *database.User) error {
	h.stateManager.SetUserState(user.TelegramID, "analyzing_food")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Отправьте фото еды для анализа. Вы также можете указать вес блюда в граммах в подписи к фото.")
	msg.ReplyMarkup = keyboard
	_, err := h.api.Send(msg)
	return err
}

// handleBloodSugar handles blood sugar callback
func (h *CallbackHandler) handleBloodSugar(chatID int64, user *database.User) error {
	h.stateManager.SetUserState(user.TelegramID, state.WaitingForBloodSugar)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Введите уровень сахара в крови (ммоль/л):")
	msg.ReplyMarkup = keyboard
	_, err := h.api.Send(msg)
	return err
}

// handleSettings handles settings callback
func (h *CallbackHandler) handleSettings(chatID int64) error {
	return menus.SendSettingsMenu(h.api, chatID)
}

// handleInsulinRatio handles insulin ratio callback
func (h *CallbackHandler) handleInsulinRatio(chatID int64, user *database.User) error {
	ratios, err := h.deps.InsulinSvc.GetUserRatios(context.Background(), user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении коэффициентов")
		_, sendErr := h.api.Send(msg)
		return sendErr
	}
	return menus.SendInsulinRatioMenu(h.api, chatID, ratios)
}

// handleAddInsulinRatio handles add insulin ratio callback
func (h *CallbackHandler) handleAddInsulinRatio(chatID int64, user *database.User) error {
	h.stateManager.SetUserState(user.TelegramID, state.WaitingForTimePeriod)
	h.stateManager.ClearTempData(user.TelegramID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Введите период времени в формате ЧЧ:ММ-ЧЧ:ММ (например, 08:00-12:00):")
	msg.ReplyMarkup = keyboard
	_, err := h.api.Send(msg)
	return err
}

// handleMainMenu handles main menu callback
func (h *CallbackHandler) handleMainMenu(chatID int64, user *database.User) error {
	h.stateManager.SetUserState(user.TelegramID, state.None)
	return menus.SendMainMenu(h.api, chatID)
}

// handleEditInsulinRatio handles edit insulin ratio callback
func (h *CallbackHandler) handleEditInsulinRatio(chatID int64, user *database.User) error {
	ratios, err := h.deps.InsulinSvc.GetUserRatios(context.Background(), user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении коэффициентов")
		_, err := h.api.Send(msg)
		return err
	}

	if len(ratios) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет сохраненных коэффициентов для редактирования")
		_, err := h.api.Send(msg)
		return err
	}

	// Show confirmation message
	text := "⚠️ Внимание!\n\nРедактирование коэффициентов удалит все существующие периоды.\n\n"
	text += "Текущие периоды:\n"
	for _, r := range ratios {
		text += fmt.Sprintf("• %s-%s: %.1f ед/ХЕ\n", r.StartTime, r.EndTime, r.Ratio)
	}
	text += "\nПродолжить?"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить все", "clear_and_add_ratio"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "insulin_ratio"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err = h.api.Send(msg)
	return err
}

// handleClearAndAddRatio handles clear and add ratio callback
func (h *CallbackHandler) handleClearAndAddRatio(chatID int64, user *database.User) error {
	// Delete all existing ratios
	ratios, err := h.deps.InsulinSvc.GetUserRatios(context.Background(), user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении коэффициентов")
		_, err := h.api.Send(msg)
		return err
	}

	for _, r := range ratios {
		if err := h.deps.InsulinSvc.DeleteRatio(context.Background(), user.ID, r.ID); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при удалении коэффициента: %v", err))
			_, err := h.api.Send(msg)
			return err
		}
	}

	// Start adding new ratio
	h.stateManager.SetUserState(user.TelegramID, state.WaitingForTimePeriod)
	h.stateManager.ClearTempData(user.TelegramID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Введите период времени в формате ЧЧ:ММ-ЧЧ:ММ (например, 08:00-12:00):")
	msg.ReplyMarkup = keyboard
	_, err = h.api.Send(msg)
	return err
}

// handleDeleteInsulinRatio handles delete insulin ratio callback
func (h *CallbackHandler) handleDeleteInsulinRatio(chatID int64, user *database.User) error {
	ratios, err := h.deps.InsulinSvc.GetUserRatios(context.Background(), user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении коэффициентов")
		_, err := h.api.Send(msg)
		return err
	}

	if len(ratios) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет сохраненных коэффициентов для удаления")
		_, err := h.api.Send(msg)
		return err
	}

	// Show confirmation message
	text := "⚠️ Внимание!\n\nУдаление коэффициента удалит все существующие периоды.\n\n"
	text += "Текущие периоды:\n"
	for _, r := range ratios {
		text += fmt.Sprintf("• %s-%s: %.1f ед/ХЕ\n", r.StartTime, r.EndTime, r.Ratio)
	}
	text += "\nПродолжить?"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить все", "clear_ratios"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "insulin_ratio"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err = h.api.Send(msg)
	return err
}

// handleClearRatios handles clear ratios callback
func (h *CallbackHandler) handleClearRatios(chatID int64, user *database.User) error {
	// Delete all existing ratios
	ratios, err := h.deps.InsulinSvc.GetUserRatios(context.Background(), user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении коэффициентов")
		_, err := h.api.Send(msg)
		return err
	}

	for _, r := range ratios {
		if err := h.deps.InsulinSvc.DeleteRatio(context.Background(), user.ID, r.ID); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при удалении коэффициента: %v", err))
			_, err := h.api.Send(msg)
			return err
		}
	}

	msg := tgbotapi.NewMessage(chatID, "✅ Все коэффициенты успешно удалены")
	_, err = h.api.Send(msg)
	if err != nil {
		return err
	}

	ratios, err = h.deps.InsulinSvc.GetUserRatios(context.Background(), user.ID)
	if err != nil {
		return err
	}
	return menus.SendInsulinRatioMenu(h.api, chatID, ratios)
}

// handleActiveInsulinTime handles active insulin time callback
func (h *CallbackHandler) handleActiveInsulinTime(chatID int64, user *database.User) error {
	// Get current active insulin time
	activeTime, err := h.deps.InsulinSvc.GetActiveInsulinTime(context.Background(), user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении времени активного инсулина")
		_, err := h.api.Send(msg)
		return err
	}

	var text string
	if activeTime == 0 {
		text = "Время активного инсулина не установлено.\n\n"
	} else {
		hours := int(activeTime) / 60
		minutes := int(activeTime) % 60
		text = fmt.Sprintf("Текущее время активного инсулина: %d:%02d\n\n", hours, minutes)
	}
	text += "Введите время активного инсулина в формате ЧЧ:ММ (например, 1:30 для 1 часа и 30 минут):"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "settings"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err = h.api.Send(msg)
	return err
}

// handleUnknownCallback handles unknown callbacks
func (h *CallbackHandler) handleUnknownCallback(chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда")
	_, err := h.api.Send(msg)
	return err
}
