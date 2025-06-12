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
	stateManager state.StateManager
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(api *tgbotapi.BotAPI, deps Dependencies, stateManager state.StateManager) *CallbackHandler {
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
	case "help":
		return h.handleHelp(query.Message.Chat.ID)
	case "food_examples":
		return h.handleFoodExamples(query.Message.Chat.ID)
	default:
		return h.handleUnknownCallback(query.Message.Chat.ID)
	}
}

// handleAnalyzeFood handles analyze food callback
func (h *CallbackHandler) handleAnalyzeFood(chatID int64, user *database.User) error {
	h.stateManager.SetUserState(user.TelegramID, "analyzing_food")

	text := `📷 *Отправьте фото еды для анализа*

💡 *Для точного расчета:*
• Укажите вес в подписи к фото (например: "150")
• Сфотографируйте блюдо целиком
• Убедитесь, что освещение хорошее

🤖 *Бот определит:*
• Количество углеводов
• Хлебные единицы (ХЕ)
• Рекомендуемую дозу инсулина`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💡 Примеры", "food_examples"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Помощь", "help"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
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

// handleHelp handles help callback
func (h *CallbackHandler) handleHelp(chatID int64) error {
	text := `🤖 *Справка по использованию бота*

*🍽️ Анализ еды:*
• Отправьте фото блюда
• В подписи можете указать вес в граммах (например: "150")
• Если вес не указан, ИИ попробует определить его самостоятельно, но результат может быть менее точным
• Получите информацию об углеводах, ХЕ и дозе инсулина

*⚙️ Настройки:*
• Установите коэффициенты инсулина на ХЕ для разного времени суток
• Это повысит точность расчета дозы инсулина

*💡 Советы:*
• Указывайте точный вес блюда для наиболее точного расчета
• Настройте коэффициенты для персонализированных рекомендаций
• Всегда консультируйтесь с врачом!`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, err := h.api.Send(msg)
	return err
}

// handleFoodExamples handles food examples callback
func (h *CallbackHandler) handleFoodExamples(chatID int64) error {
	text := `📸 *Примеры фотографий еды:*

✅ *Хорошие фото:*
• Целое блюдо на тарелке
• Хорошее освещение
• Видны все компоненты
• Указан вес: "200"

❌ *Плохие фото:*
• Слишком темно
• Частично съеденное блюдо
• Слишком далеко/близко
• Неясные компоненты

🥘 *Хорошо распознается:*
• Каши, гарниры
• Мясо, рыба
• Овощи, салаты
• Супы
• Хлеб, выпечка

⚠️ *Сложно распознается:*
• Смешанные блюда
• Соусы внутри
• Мелко нарезанное`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "analyze_food"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, err := h.api.Send(msg)
	return err
}

// handleUnknownCallback handles unknown callbacks
func (h *CallbackHandler) handleUnknownCallback(chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда")
	_, err := h.api.Send(msg)
	return err
}
