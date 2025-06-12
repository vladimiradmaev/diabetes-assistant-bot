package menus

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/keyboards"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
)

// SendMainMenu sends the main menu to a chat
func SendMainMenu(api *tgbotapi.BotAPI, chatID int64) error {
	text := `🤖 *ДиаАИ* — твой помощник для управления диабетом

🍽️ Отправь фото еды, и я:
• Определю количество углеводов
• Рассчитаю хлебные единицы (ХЕ)  
• Предложу дозу инсулина

🤖 *ИИ модели:*
• Gemini 2.0 Flash (до 1500 запросов/день)
• Автоматическое переключение на OpenAI при превышении лимитов

⚠️ *Важно:* Это справочная информация, всегда консультируйтесь с врачом!

Выберите действие:`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboards.MainMenu()
	_, err := api.Send(msg)
	return err
}

// SendSettingsMenu sends the settings menu to a chat
func SendSettingsMenu(api *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Настройки:")
	msg.ReplyMarkup = keyboards.SettingsMenu()
	_, err := api.Send(msg)
	return err
}

// SendInsulinRatioMenu sends the insulin ratio management menu
func SendInsulinRatioMenu(api *tgbotapi.BotAPI, chatID int64, ratios []database.InsulinRatio) error {
	var text string
	if len(ratios) == 0 {
		text = "У вас пока нет сохраненных коэффициентов. Нажмите 'Добавить' чтобы создать новый."
	} else {
		// Calculate total hours
		totalMinutes := 0
		for _, r := range ratios {
			start := timeToMinutes(r.StartTime)
			end := timeToMinutes(r.EndTime)
			if end < start {
				end += 24 * 60 // Handle periods crossing midnight
			}
			totalMinutes += end - start
		}
		totalHours := float64(totalMinutes) / 60.0

		text = "Ваши коэффициенты:\n\n"
		for _, r := range ratios {
			text += fmt.Sprintf("🕒 %s - %s: %.1f ед/ХЕ\n", r.StartTime, r.EndTime, r.Ratio)
		}
		text += "\n"

		if totalHours < 24 {
			text += fmt.Sprintf("⚠️ Внимание: сохранено только %.1f часов из 24\n", totalHours)
			text += "Добавьте еще периоды, чтобы покрыть все 24 часа\n"
		} else if totalHours > 24 {
			text += fmt.Sprintf("⚠️ Внимание: сохранено %.1f часов (больше 24)\n", totalHours)
			text += "Периоды перекрываются или превышают 24 часа\n"
		} else {
			text += "✅ Периоды полностью покрывают 24 часа\n"
		}
	}

	keyboard := keyboards.InsulinRatioMenu(len(ratios) > 0)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := api.Send(msg)
	return err
}

// Helper function to convert time string to minutes since midnight
func timeToMinutes(timeStr string) int {
	t, _ := time.Parse("15:04", timeStr)
	return t.Hour()*60 + t.Minute()
}
