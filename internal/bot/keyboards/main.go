package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MainMenu creates the main menu keyboard
func MainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🍽️ Анализ еды", "analyze_food"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		),
	)
}

// SettingsMenu creates the settings menu keyboard
func SettingsMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Коэф. на ХЕ", "insulin_ratio"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
		),
	)
}

// InsulinRatioMenu creates the insulin ratio management keyboard
func InsulinRatioMenu(hasRatios bool) tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", "add_insulin_ratio"),
		),
	)

	if hasRatios {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "edit_insulin_ratio"),
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить", "delete_insulin_ratio"),
			),
		)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "settings"),
		),
	)

	return keyboard
}
