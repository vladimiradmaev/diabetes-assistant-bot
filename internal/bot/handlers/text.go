package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/menus"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/state"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
)

// TextHandler handles text messages
type TextHandler struct {
	api          *tgbotapi.BotAPI
	deps         Dependencies
	stateManager state.StateManager
}

// NewTextHandler creates a new text handler
func NewTextHandler(api *tgbotapi.BotAPI, deps Dependencies, stateManager state.StateManager) *TextHandler {
	return &TextHandler{
		api:          api,
		deps:         deps,
		stateManager: stateManager,
	}
}

// Handle processes a text message
func (h *TextHandler) Handle(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	userState := h.stateManager.GetUserState(user.TelegramID)

	switch userState {
	case state.WaitingForBloodSugar:
		return h.handleBloodSugar(ctx, message, user)
	case state.WaitingForTimePeriod:
		return h.handleTimePeriod(ctx, message, user)
	case state.WaitingForInsulinRatio:
		return h.handleInsulinRatio(ctx, message, user)
	case state.WaitingForActiveInsulinTime:
		return h.handleActiveInsulinTime(ctx, message, user)
	default:
		return h.handleDefaultText(message.Chat.ID)
	}
}

// handleBloodSugar handles blood sugar input
func (h *TextHandler) handleBloodSugar(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	value, err := strconv.ParseFloat(message.Text, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, введите корректное число (например: 5.6)")
		_, err := h.api.Send(msg)
		return err
	}

	if err := h.deps.BloodSugarSvc.AddRecord(ctx, user.ID, value); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при сохранении данных. Пожалуйста, попробуйте еще раз.")
		_, err := h.api.Send(msg)
		return err
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Уровень сахара %.1f ммоль/л успешно сохранен", value))
	_, err = h.api.Send(msg)
	if err != nil {
		return err
	}

	h.stateManager.SetUserState(user.TelegramID, state.None)
	return menus.SendMainMenu(h.api, message.Chat.ID)
}

// handleTimePeriod handles time period input for insulin ratios
func (h *TextHandler) handleTimePeriod(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	// Parse time period
	parts := strings.Split(message.Text, "-")
	if len(parts) != 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат. Введите период в формате ЧЧ:ММ-ЧЧ:ММ (например, 08:00-12:00)")
		_, err := h.api.Send(msg)
		return err
	}

	startTime := strings.TrimSpace(parts[0])
	endTime := strings.TrimSpace(parts[1])

	// Validate empty values
	if startTime == "" || endTime == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Время начала и окончания не могут быть пустыми")
		_, err := h.api.Send(msg)
		return err
	}

	// Validate time format
	if _, err := time.Parse("15:04", startTime); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат времени начала. Используйте 24-часовой формат ЧЧ:ММ (например, 08:00 или 14:30)")
		_, err := h.api.Send(msg)
		return err
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат времени окончания. Используйте 24-часовой формат ЧЧ:ММ (например, 08:00 или 14:30)")
		_, err := h.api.Send(msg)
		return err
	}

	// Additional validation for 24-hour format
	startHour, _ := strconv.Atoi(strings.Split(startTime, ":")[0])
	endHour, _ := strconv.Atoi(strings.Split(endTime, ":")[0])
	if startHour < 0 || startHour > 23 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Часы начала должны быть в диапазоне 00-23")
		_, err := h.api.Send(msg)
		return err
	}
	if endHour < 0 || endHour > 24 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Часы окончания должны быть в диапазоне 00-24")
		_, err := h.api.Send(msg)
		return err
	}
	if endHour == 24 && strings.Split(endTime, ":")[1] != "00" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "При использовании 24 часов, минуты должны быть 00")
		_, err := h.api.Send(msg)
		return err
	}

	// Store time period and ask for ratio
	h.stateManager.SetTempData(user.TelegramID, "startTime", startTime)
	h.stateManager.SetTempData(user.TelegramID, "endTime", endTime)
	h.stateManager.SetUserState(user.TelegramID, state.WaitingForInsulinRatio)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
		),
	)
	msg := tgbotapi.NewMessage(message.Chat.ID, "Введите коэффициент (количество единиц инсулина на 1 ХЕ):")
	msg.ReplyMarkup = keyboard
	_, err := h.api.Send(msg)
	return err
}

// handleInsulinRatio handles insulin ratio input
func (h *TextHandler) handleInsulinRatio(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	ratio, err := strconv.ParseFloat(message.Text, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, введите корректное число (например: 1.5)")
		_, err := h.api.Send(msg)
		return err
	}

	// Validate empty or zero ratio
	if ratio <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Коэффициент должен быть больше 0")
		_, err := h.api.Send(msg)
		return err
	}

	// Get stored time period
	startTimeVal, ok := h.stateManager.GetTempData(user.TelegramID, "startTime")
	if !ok {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка: время начала не найдено")
		_, err := h.api.Send(msg)
		return err
	}
	endTimeVal, ok := h.stateManager.GetTempData(user.TelegramID, "endTime")
	if !ok {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка: время окончания не найдено")
		_, err := h.api.Send(msg)
		return err
	}

	startTime := startTimeVal.(string)
	endTime := endTimeVal.(string)

	// Add insulin ratio
	if err := h.deps.InsulinSvc.AddRatio(ctx, user.ID, startTime, endTime, ratio); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка при сохранении коэффициента: %v", err))
		_, err := h.api.Send(msg)
		return err
	}

	// Clear temporary data
	h.stateManager.ClearTempData(user.TelegramID)
	h.stateManager.SetUserState(user.TelegramID, state.None)

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Коэффициент %.1f ед/ХЕ для периода %s-%s успешно сохранен", ratio, startTime, endTime))
	_, err = h.api.Send(msg)
	if err != nil {
		return err
	}

	// Get updated ratios and send menu
	ratios, err := h.deps.InsulinSvc.GetUserRatios(ctx, user.ID)
	if err != nil {
		return err
	}
	return menus.SendInsulinRatioMenu(h.api, message.Chat.ID, ratios)
}

// handleActiveInsulinTime handles active insulin time input
func (h *TextHandler) handleActiveInsulinTime(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	// Parse time format
	parts := strings.Split(message.Text, ":")
	if len(parts) != 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат. Введите время в формате ЧЧ:ММ (например, 1:30)")
		_, err := h.api.Send(msg)
		return err
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours < 0 || hours > 24 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Часы должны быть числом от 0 до 24")
		_, err := h.api.Send(msg)
		return err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes < 0 || minutes > 59 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Минуты должны быть числом от 0 до 59")
		_, err := h.api.Send(msg)
		return err
	}

	totalMinutes := hours*60 + minutes
	if totalMinutes == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Время активного инсулина не может быть равно нулю")
		_, err := h.api.Send(msg)
		return err
	}

	if err := h.deps.InsulinSvc.SetActiveInsulinTime(ctx, user.ID, totalMinutes); err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка при сохранении времени: %v", err))
		_, err := h.api.Send(msg)
		return err
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Время активного инсулина установлено: %d:%02d", hours, minutes))
	_, err = h.api.Send(msg)
	if err != nil {
		return err
	}

	h.stateManager.SetUserState(user.TelegramID, state.None)
	return menus.SendSettingsMenu(h.api, message.Chat.ID)
}

// handleDefaultText handles text when no specific state is set
func (h *TextHandler) handleDefaultText(chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Пожалуйста, используйте меню для выбора действия.")
	_, err := h.api.Send(msg)
	return err
}
