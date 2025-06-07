package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/interfaces"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
)

const (
	stateNone                        = "none"
	stateWaitingForBloodSugar        = "waiting_for_blood_sugar"
	stateWaitingForInsulinRatio      = "waiting_for_insulin_ratio"
	stateWaitingForTimePeriod        = "waiting_for_time_period"
	stateWaitingForActiveInsulinTime = "waiting_for_active_insulin_time"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	userService     interfaces.UserServiceInterface
	foodAnalysisSvc interfaces.FoodAnalysisServiceInterface
	bloodSugarSvc   interfaces.BloodSugarServiceInterface
	insulinSvc      interfaces.InsulinServiceInterface
	userStates      map[int64]string                 // Map to track user states
	userWeights     map[int64]float64                // Map to store user-provided weights
	tempData        map[int64]map[string]interface{} // Map to store temporary data for multi-step operations
}

func NewBot(token string, userService interfaces.UserServiceInterface, foodAnalysisSvc interfaces.FoodAnalysisServiceInterface, bloodSugarSvc interfaces.BloodSugarServiceInterface, insulinSvc interfaces.InsulinServiceInterface) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	logger.Infof("Bot authorized on account %s", api.Self.UserName)
	return &Bot{
		api:             api,
		userService:     userService,
		foodAnalysisSvc: foodAnalysisSvc,
		bloodSugarSvc:   bloodSugarSvc,
		insulinSvc:      insulinSvc,
		userStates:      make(map[int64]string),
		userWeights:     make(map[int64]float64),
		tempData:        make(map[int64]map[string]interface{}),
	}, nil
}

func (b *Bot) sendMainMenu(chatID int64) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🍽️ Анализ еды", "analyze_food"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		),
	)

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
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) sendSettingsMenu(chatID int64) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Коэф. на ХЕ", "insulin_ratio"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Настройки:")
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) sendInsulinRatioMenu(chatID int64, userID uint) error {
	ratios, err := b.insulinSvc.GetUserRatios(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("failed to get insulin ratios: %w", err)
	}

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

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", "add_insulin_ratio"),
		),
	)
	if len(ratios) > 0 {
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

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err = b.api.Send(msg)
	return err
}

// Helper function to convert time string to minutes since midnight
func timeToMinutes(timeStr string) int {
	t, _ := time.Parse("15:04", timeStr)
	return t.Hour()*60 + t.Minute()
}

// Helper function to check if two time periods overlap
func doPeriodsOverlap(start1, end1, start2, end2 string) bool {
	start1Min := timeToMinutes(start1)
	end1Min := timeToMinutes(end1)
	start2Min := timeToMinutes(start2)
	end2Min := timeToMinutes(end2)

	// Handle periods that cross midnight
	if end1Min < start1Min {
		end1Min += 24 * 60
	}
	if end2Min < start2Min {
		end2Min += 24 * 60
	}

	// Check for overlap
	return (start1Min <= start2Min && end1Min > start2Min) ||
		(start1Min < end2Min && end1Min >= end2Min) ||
		(start1Min >= start2Min && end1Min <= end2Min)
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil && update.CallbackQuery == nil {
		return nil
	}

	var userID int64
	var chatID int64
	var username, firstName, lastName string

	if update.Message != nil {
		userID = update.Message.From.ID
		chatID = update.Message.Chat.ID
		username = update.Message.From.UserName
		firstName = update.Message.From.FirstName
		lastName = update.Message.From.LastName
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
		username = update.CallbackQuery.From.UserName
		firstName = update.CallbackQuery.From.FirstName
		lastName = update.CallbackQuery.From.LastName
	}

	// Register user
	user, err := b.userService.RegisterUser(
		ctx,
		userID,
		username,
		firstName,
		lastName,
	)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	logger.Infof("User registered/updated: %s (ID: %d)", user.Username, user.ID)

	// Handle callback queries (button clicks)
	if update.CallbackQuery != nil {
		// Answer callback query to remove loading state
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
		if _, err := b.api.Request(callback); err != nil {
			logger.Errorf("Failed to answer callback query: %v", err)
		}
		return b.handleCallbackQuery(ctx, update.CallbackQuery, user)
	}

	// Handle commands
	if update.Message.IsCommand() {
		return b.handleCommand(ctx, update.Message, user)
	}

	// Handle photo messages
	if update.Message.Photo != nil {
		if b.userStates[int64(user.ID)] != "analyzing_food" {
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, сначала нажмите кнопку '🍽️ Анализ еды' в меню.")
			_, err := b.api.Send(msg)
			return err
		}
		return b.handlePhoto(ctx, update.Message, user)
	}

	// Handle text messages
	if update.Message.Text != "" {
		return b.handleText(ctx, update.Message, user)
	}

	return nil
}

func (b *Bot) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery, user *database.User) error {
	switch query.Data {
	case "analyze_food":
		b.userStates[int64(user.ID)] = "analyzing_food"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
			),
		)
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Отправьте фото еды для анализа. Вы также можете указать вес блюда в граммах в подписи к фото.")
		msg.ReplyMarkup = keyboard
		_, err := b.api.Send(msg)
		return err

	case "blood_sugar":
		b.userStates[int64(user.ID)] = stateWaitingForBloodSugar
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Главное меню", "main_menu"),
			),
		)
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Введите уровень сахара в крови (ммоль/л):")
		msg.ReplyMarkup = keyboard
		_, err := b.api.Send(msg)
		return err

	case "settings":
		return b.sendSettingsMenu(query.Message.Chat.ID)

	case "insulin_ratio":
		return b.sendInsulinRatioMenu(query.Message.Chat.ID, user.ID)

	case "add_insulin_ratio":
		b.userStates[int64(user.ID)] = stateWaitingForTimePeriod
		b.tempData[int64(user.ID)] = make(map[string]interface{})
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
			),
		)
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Введите период времени в формате ЧЧ:ММ-ЧЧ:ММ (например, 08:00-12:00):")
		msg.ReplyMarkup = keyboard
		_, err := b.api.Send(msg)
		return err

	case "main_menu":
		b.userStates[int64(user.ID)] = stateNone
		return b.sendMainMenu(query.Message.Chat.ID)

	case "edit_insulin_ratio":
		ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
		if err != nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении коэффициентов")
			_, err := b.api.Send(msg)
			return err
		}

		if len(ratios) == 0 {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Нет сохраненных коэффициентов для редактирования")
			_, err := b.api.Send(msg)
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
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
		msg.ReplyMarkup = keyboard
		_, err = b.api.Send(msg)
		return err

	case "clear_and_add_ratio":
		// Delete all existing ratios
		ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
		if err != nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении коэффициентов")
			_, err := b.api.Send(msg)
			return err
		}

		for _, r := range ratios {
			if err := b.insulinSvc.DeleteRatio(context.Background(), user.ID, r.ID); err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("Ошибка при удалении коэффициента: %v", err))
				_, err := b.api.Send(msg)
				return err
			}
		}

		// Start adding new ratio
		b.userStates[int64(user.ID)] = stateWaitingForTimePeriod
		b.tempData[int64(user.ID)] = make(map[string]interface{})
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
			),
		)
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Введите период времени в формате ЧЧ:ММ-ЧЧ:ММ (например, 08:00-12:00):")
		msg.ReplyMarkup = keyboard
		_, err = b.api.Send(msg)
		return err

	case "delete_insulin_ratio":
		ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
		if err != nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении коэффициентов")
			_, err := b.api.Send(msg)
			return err
		}

		if len(ratios) == 0 {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Нет сохраненных коэффициентов для удаления")
			_, err := b.api.Send(msg)
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
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
		msg.ReplyMarkup = keyboard
		_, err = b.api.Send(msg)
		return err

	case "clear_ratios":
		// Delete all existing ratios
		ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
		if err != nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении коэффициентов")
			_, err := b.api.Send(msg)
			return err
		}

		for _, r := range ratios {
			if err := b.insulinSvc.DeleteRatio(context.Background(), user.ID, r.ID); err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("Ошибка при удалении коэффициента: %v", err))
				_, err := b.api.Send(msg)
				return err
			}
		}

		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "✅ Все коэффициенты успешно удалены")
		_, err = b.api.Send(msg)
		if err != nil {
			return err
		}

		return b.sendInsulinRatioMenu(query.Message.Chat.ID, user.ID)

	case "active_insulin_time":
		// Get current active insulin time
		activeTime, err := b.insulinSvc.GetActiveInsulinTime(context.Background(), user.ID)
		if err != nil {
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении времени активного инсулина")
			_, err := b.api.Send(msg)
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
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
		msg.ReplyMarkup = keyboard
		_, err = b.api.Send(msg)
		if err != nil {
			return err
		}

		b.userStates[int64(user.ID)] = stateWaitingForActiveInsulinTime
		return nil

	default:
		// Handle edit_ratio_X and delete_ratio_X callbacks
		if strings.HasPrefix(query.Data, "edit_ratio_") {
			ratioID, _ := strconv.ParseUint(strings.TrimPrefix(query.Data, "edit_ratio_"), 10, 32)
			ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении коэффициентов")
				_, err := b.api.Send(msg)
				return err
			}

			var selectedRatio *database.InsulinRatio
			for _, r := range ratios {
				if r.ID == uint(ratioID) {
					selectedRatio = &r
					break
				}
			}

			if selectedRatio == nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Коэффициент не найден")
				_, err := b.api.Send(msg)
				return err
			}

			b.userStates[int64(user.ID)] = stateWaitingForTimePeriod
			b.tempData[int64(user.ID)] = map[string]interface{}{
				"ratioID": ratioID,
				"isEdit":  true,
			}

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
				),
			)
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf(
				"Текущий период: %s-%s\nВведите новый период в формате ЧЧ:ММ-ЧЧ:ММ:",
				selectedRatio.StartTime, selectedRatio.EndTime,
			))
			msg.ReplyMarkup = keyboard
			_, err = b.api.Send(msg)
			return err
		}

		if strings.HasPrefix(query.Data, "delete_ratio_") {
			ratioID, err := strconv.ParseUint(strings.TrimPrefix(query.Data, "delete_ratio_"), 10, 32)
			if err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Неверный формат ID коэффициента")
				_, err := b.api.Send(msg)
				return err
			}

			// Get all ratios
			ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при получении коэффициентов")
				_, err := b.api.Send(msg)
				return err
			}

			// Find the ratio to delete and its neighbors
			var ratioToDelete *database.InsulinRatio
			var prevRatio, nextRatio *database.InsulinRatio
			for i, r := range ratios {
				if r.ID == uint(ratioID) {
					ratioToDelete = &r
					if i > 0 {
						prevRatio = &ratios[i-1]
					}
					if i < len(ratios)-1 {
						nextRatio = &ratios[i+1]
					}
					break
				}
			}

			if ratioToDelete == nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Коэффициент не найден")
				_, err := b.api.Send(msg)
				return err
			}

			// If this is the only ratio, just delete it
			if len(ratios) == 1 {
				if err := b.insulinSvc.DeleteRatio(context.Background(), user.ID, uint(ratioID)); err != nil {
					msg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("Ошибка при удалении: %v", err))
					_, err := b.api.Send(msg)
					return err
				}

				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "✅ Коэффициент успешно удален")
				_, err = b.api.Send(msg)
				if err != nil {
					return err
				}

				return b.sendInsulinRatioMenu(query.Message.Chat.ID, user.ID)
			}

			// Determine which neighbor to merge with
			var changes []string
			var ratiosToUpdate []struct {
				ID        uint
				StartTime string
				EndTime   string
				Ratio     float64
			}

			if prevRatio != nil && nextRatio != nil {
				// If both neighbors exist, merge with the one that has a closer end time
				prevEnd := timeToMinutes(prevRatio.EndTime)
				nextStart := timeToMinutes(nextRatio.StartTime)
				if prevEnd < nextStart {
					changes = append(changes, fmt.Sprintf("Изменить период %s-%s на %s-%s",
						prevRatio.StartTime, prevRatio.EndTime, prevRatio.StartTime, nextRatio.StartTime))
					ratiosToUpdate = append(ratiosToUpdate, struct {
						ID        uint
						StartTime string
						EndTime   string
						Ratio     float64
					}{prevRatio.ID, prevRatio.StartTime, nextRatio.StartTime, prevRatio.Ratio})
				} else {
					changes = append(changes, fmt.Sprintf("Изменить период %s-%s на %s-%s",
						prevRatio.StartTime, nextRatio.EndTime, prevRatio.StartTime, nextRatio.EndTime))
					ratiosToUpdate = append(ratiosToUpdate, struct {
						ID        uint
						StartTime string
						EndTime   string
						Ratio     float64
					}{nextRatio.ID, prevRatio.StartTime, nextRatio.EndTime, nextRatio.Ratio})
				}
			} else if prevRatio != nil {
				changes = append(changes, fmt.Sprintf("Изменить период %s-%s на %s-%s",
					prevRatio.StartTime, prevRatio.EndTime, prevRatio.StartTime, ratioToDelete.EndTime))
				ratiosToUpdate = append(ratiosToUpdate, struct {
					ID        uint
					StartTime string
					EndTime   string
					Ratio     float64
				}{prevRatio.ID, prevRatio.StartTime, ratioToDelete.EndTime, prevRatio.Ratio})
			} else if nextRatio != nil {
				changes = append(changes, fmt.Sprintf("Изменить период %s-%s на %s-%s",
					nextRatio.StartTime, nextRatio.EndTime, ratioToDelete.StartTime, nextRatio.EndTime))
				ratiosToUpdate = append(ratiosToUpdate, struct {
					ID        uint
					StartTime string
					EndTime   string
					Ratio     float64
				}{nextRatio.ID, ratioToDelete.StartTime, nextRatio.EndTime, nextRatio.Ratio})
			}

			if len(changes) > 0 {
				// Store changes for confirmation
				b.tempData[int64(user.ID)] = map[string]interface{}{
					"ratioID":        ratioID,
					"changes":        changes,
					"ratiosToDelete": []uint{uint(ratioID)},
					"ratiosToUpdate": ratiosToUpdate,
				}

				// Show confirmation message
				text := "Для применения изменений необходимо:\n\n"
				for _, change := range changes {
					text += "• " + change + "\n"
				}
				text += "\nПродолжить?"

				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("✅ Да", "confirm_changes"),
						tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "insulin_ratio"),
					),
				)
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
				msg.ReplyMarkup = keyboard
				_, err = b.api.Send(msg)
				return err
			}

			// If no changes needed, just delete it
			if err := b.insulinSvc.DeleteRatio(context.Background(), user.ID, uint(ratioID)); err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("Ошибка при удалении: %v", err))
				_, err := b.api.Send(msg)
				return err
			}

			msg := tgbotapi.NewMessage(query.Message.Chat.ID, "✅ Коэффициент успешно удален")
			_, err = b.api.Send(msg)
			if err != nil {
				return err
			}

			return b.sendInsulinRatioMenu(query.Message.Chat.ID, user.ID)
		}
	}

	return nil
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	logger.Infof("Handling command %s from user %d", message.Command(), user.ID)
	switch message.Command() {
	case "start":
		b.userStates[int64(user.ID)] = stateNone
		return b.sendMainMenu(message.Chat.ID)
	case "help":
		msg := tgbotapi.NewMessage(message.Chat.ID, `Доступные команды:
/start - Показать главное меню
/help - Показать это сообщение

Как указать вес блюда:
1. Нажмите кнопку "🍽️ Анализ еды"
2. Отправьте фото еды
3. В подписи к фото напишите только число - вес в граммах
Пример: "150" или "200"

Если вес не указан, бот попробует оценить его автоматически.`)
		_, err := b.api.Send(msg)
		return err
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для просмотра доступных команд.")
		_, err := b.api.Send(msg)
		return err
	}
}

func (b *Bot) handleText(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	state := b.userStates[int64(user.ID)]

	switch state {
	case stateWaitingForBloodSugar:
		value, err := strconv.ParseFloat(message.Text, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, введите корректное число (например: 5.6)")
			_, err := b.api.Send(msg)
			return err
		}

		if err := b.bloodSugarSvc.AddRecord(ctx, user.ID, value); err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при сохранении данных. Пожалуйста, попробуйте еще раз.")
			_, err := b.api.Send(msg)
			return err
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Уровень сахара %.1f ммоль/л успешно сохранен", value))
		_, err = b.api.Send(msg)
		if err != nil {
			return err
		}

		b.userStates[int64(user.ID)] = stateNone
		return b.sendMainMenu(message.Chat.ID)

	case stateWaitingForTimePeriod:
		// Parse time period
		parts := strings.Split(message.Text, "-")
		if len(parts) != 2 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат. Введите период в формате ЧЧ:ММ-ЧЧ:ММ (например, 08:00-12:00)")
			_, err := b.api.Send(msg)
			return err
		}

		startTime := strings.TrimSpace(parts[0])
		endTime := strings.TrimSpace(parts[1])

		// Validate empty values
		if startTime == "" || endTime == "" {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Время начала и окончания не могут быть пустыми")
			_, err := b.api.Send(msg)
			return err
		}

		// Validate time format
		if _, err := time.Parse("15:04", startTime); err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат времени начала. Используйте 24-часовой формат ЧЧ:ММ (например, 08:00 или 14:30)")
			_, err := b.api.Send(msg)
			return err
		}
		if _, err := time.Parse("15:04", endTime); err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат времени окончания. Используйте 24-часовой формат ЧЧ:ММ (например, 08:00 или 14:30)")
			_, err := b.api.Send(msg)
			return err
		}

		// Additional validation for 24-hour format
		startHour, _ := strconv.Atoi(strings.Split(startTime, ":")[0])
		endHour, _ := strconv.Atoi(strings.Split(endTime, ":")[0])
		if startHour < 0 || startHour > 23 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Часы начала должны быть в диапазоне 00-23")
			_, err := b.api.Send(msg)
			return err
		}
		if endHour < 0 || endHour > 24 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Часы окончания должны быть в диапазоне 00-24")
			_, err := b.api.Send(msg)
			return err
		}
		if endHour == 24 && strings.Split(endTime, ":")[1] != "00" {
			msg := tgbotapi.NewMessage(message.Chat.ID, "При использовании 24 часов, минуты должны быть 00")
			_, err := b.api.Send(msg)
			return err
		}

		// Check if this is an edit operation
		tempData := b.tempData[int64(user.ID)]
		if isEdit, ok := tempData["isEdit"].(bool); ok && isEdit {
			ratioID := tempData["ratioID"].(uint64)

			// Get all ratios to check for overlaps
			ratios, err := b.insulinSvc.GetUserRatios(context.Background(), user.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка при получении коэффициентов")
				_, err := b.api.Send(msg)
				return err
			}

			// Find affected ratios
			var affectedRatios []database.InsulinRatio
			for _, r := range ratios {
				if r.ID != uint(ratioID) {
					affectedRatios = append(affectedRatios, r)
				}
			}

			// Check for overlaps and prepare changes
			var changes []string
			var ratiosToDelete []uint
			var ratiosToUpdate []struct {
				ID        uint
				StartTime string
				EndTime   string
			}

			for _, r := range affectedRatios {
				if doPeriodsOverlap(startTime, endTime, r.StartTime, r.EndTime) {
					// If new period completely covers existing period
					if doPeriodsOverlap(startTime, endTime, r.StartTime, r.EndTime) &&
						!doPeriodsOverlap(r.StartTime, r.EndTime, startTime, endTime) {
						changes = append(changes, fmt.Sprintf("Удалить период %s-%s", r.StartTime, r.EndTime))
						ratiosToDelete = append(ratiosToDelete, r.ID)
					} else {
						// Adjust the existing period
						var newStart, newEnd string
						if timeToMinutes(startTime) <= timeToMinutes(r.StartTime) {
							newStart = endTime
							newEnd = r.EndTime
							changes = append(changes, fmt.Sprintf("Изменить период %s-%s на %s-%s",
								r.StartTime, r.EndTime, newStart, newEnd))
						} else {
							newStart = r.StartTime
							newEnd = startTime
							changes = append(changes, fmt.Sprintf("Изменить период %s-%s на %s-%s",
								r.StartTime, r.EndTime, newStart, newEnd))
						}
						ratiosToUpdate = append(ratiosToUpdate, struct {
							ID        uint
							StartTime string
							EndTime   string
						}{r.ID, newStart, newEnd})
					}
				}
			}

			if len(changes) > 0 {
				// Store changes for confirmation
				b.tempData[int64(user.ID)] = map[string]interface{}{
					"ratioID":         ratioID,
					"isEdit":          true,
					"startTime":       startTime,
					"endTime":         endTime,
					"changes":         changes,
					"ratiosToDelete":  ratiosToDelete,
					"ratiosToUpdate":  ratiosToUpdate,
					"waitingForRatio": true,
				}

				// Show confirmation message
				text := "Для применения изменений необходимо:\n\n"
				for _, change := range changes {
					text += "• " + change + "\n"
				}
				text += "\nПродолжить?"

				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("✅ Да", "confirm_changes"),
						tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "insulin_ratio"),
					),
				)
				msg := tgbotapi.NewMessage(message.Chat.ID, text)
				msg.ReplyMarkup = keyboard
				_, err := b.api.Send(msg)
				return err
			}
		}

		// Store time period and ask for ratio
		b.tempData[int64(user.ID)]["startTime"] = startTime
		b.tempData[int64(user.ID)]["endTime"] = endTime
		b.userStates[int64(user.ID)] = stateWaitingForInsulinRatio

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "insulin_ratio"),
			),
		)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите коэффициент (количество единиц инсулина на 1 ХЕ):")
		msg.ReplyMarkup = keyboard
		_, err := b.api.Send(msg)
		return err

	case stateWaitingForInsulinRatio:
		ratio, err := strconv.ParseFloat(message.Text, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, введите корректное число (например: 1.5)")
			_, err := b.api.Send(msg)
			return err
		}

		// Validate empty or zero ratio
		if ratio <= 0 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Коэффициент должен быть больше 0")
			_, err := b.api.Send(msg)
			return err
		}

		// Get stored time period
		tempData := b.tempData[int64(user.ID)]
		startTime := tempData["startTime"].(string)
		endTime := tempData["endTime"].(string)

		// Check if this is an edit operation
		if isEdit, ok := tempData["isEdit"].(bool); ok && isEdit {
			ratioID := tempData["ratioID"].(uint64)
			if err := b.insulinSvc.UpdateRatio(context.Background(), user.ID, uint(ratioID), startTime, endTime, ratio); err != nil {
				msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка при обновлении коэффициента: %v", err))
				_, err := b.api.Send(msg)
				return err
			}

			// Clear temporary data
			delete(b.tempData, int64(user.ID))
			b.userStates[int64(user.ID)] = stateNone

			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Коэффициент обновлен: %.1f ед/ХЕ для периода %s-%s", ratio, startTime, endTime))
			_, err = b.api.Send(msg)
			if err != nil {
				return err
			}

			return b.sendInsulinRatioMenu(message.Chat.ID, user.ID)
		}

		// Add insulin ratio
		if err := b.insulinSvc.AddRatio(context.Background(), user.ID, startTime, endTime, ratio); err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка при сохранении коэффициента: %v", err))
			_, err := b.api.Send(msg)
			return err
		}

		// Clear temporary data
		delete(b.tempData, int64(user.ID))
		b.userStates[int64(user.ID)] = stateNone

		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Коэффициент %.1f ед/ХЕ для периода %s-%s успешно сохранен", ratio, startTime, endTime))
		_, err = b.api.Send(msg)
		if err != nil {
			return err
		}

		return b.sendInsulinRatioMenu(message.Chat.ID, user.ID)

	case stateWaitingForActiveInsulinTime:
		// Parse time format
		parts := strings.Split(message.Text, ":")
		if len(parts) != 2 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат. Введите время в формате ЧЧ:ММ (например, 1:30)")
			_, err := b.api.Send(msg)
			return err
		}

		hours, err := strconv.Atoi(parts[0])
		if err != nil || hours < 0 || hours > 24 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Часы должны быть числом от 0 до 24")
			_, err := b.api.Send(msg)
			return err
		}

		minutes, err := strconv.Atoi(parts[1])
		if err != nil || minutes < 0 || minutes > 59 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Минуты должны быть числом от 0 до 59")
			_, err := b.api.Send(msg)
			return err
		}

		totalMinutes := hours*60 + minutes
		if totalMinutes == 0 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Время активного инсулина не может быть равно нулю")
			_, err := b.api.Send(msg)
			return err
		}

		if err := b.insulinSvc.SetActiveInsulinTime(context.Background(), user.ID, totalMinutes); err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ошибка при сохранении времени: %v", err))
			_, err := b.api.Send(msg)
			return err
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Время активного инсулина установлено: %d:%02d", hours, minutes))
		_, err = b.api.Send(msg)
		if err != nil {
			return err
		}

		b.userStates[int64(user.ID)] = stateNone
		return b.sendSettingsMenu(message.Chat.ID)

	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, используйте меню для выбора действия.")
		_, err := b.api.Send(msg)
		return err
	}
}

func (b *Bot) handlePhoto(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	// Get the largest photo
	photo := message.Photo[len(message.Photo)-1]
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Check if weight is provided in caption
	weight := 0.0
	if message.Caption != "" {
		weight, err = strconv.ParseFloat(message.Caption, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат веса. Пожалуйста, укажите вес в граммах (например: 100).")
			_, err := b.api.Send(msg)
			return err
		}
		logger.Infof("User %d provided weight: %.1f g", user.ID, weight)
	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Вес не указан. Я попробую оценить вес блюда автоматически.")
		_, err := b.api.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send weight estimation message: %w", err)
		}
	}

	// Send "processing" message
	processingMsg := tgbotapi.NewMessage(message.Chat.ID, "Анализирую изображение...")
	sentMsg, err := b.api.Send(processingMsg)
	if err != nil {
		return fmt.Errorf("failed to send processing message: %w", err)
	}

	// Analyze the image
	logger.Infof("Starting food analysis for user %d with Gemini", user.ID)
	analysis, err := b.foodAnalysisSvc.AnalyzeFood(ctx, user.ID, file.Link(b.api.Token), weight)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Извините, произошла ошибка при анализе изображения. Пожалуйста, попробуйте еще раз через несколько минут.")
		_, err := b.api.Send(msg)
		return err
	}
	logger.Infof("Food analysis completed for user %d", user.ID)

	// Delete processing message
	deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, sentMsg.MessageID)
	b.api.Send(deleteMsg)

	// Check if no food was detected
	if analysis.Carbs == 0 && analysis.Weight == 0 && len(analysis.AnalysisText) > 0 &&
		strings.Contains(analysis.AnalysisText, "не обнаружена еда") {
		// Send a simple text message for non-food images
		msg := tgbotapi.NewMessage(message.Chat.ID, analysis.AnalysisText)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ В главное меню", "main_menu"),
			),
		)
		msg.ReplyMarkup = keyboard
		_, err = b.api.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send non-food message: %w", err)
		}
		// Reset user state
		b.userStates[int64(user.ID)] = stateNone
		return nil
	}

	// Escape only essential Markdown characters
	escapedAnalysisText := strings.ReplaceAll(analysis.AnalysisText, "_", "\\_")
	escapedAnalysisText = strings.ReplaceAll(escapedAnalysisText, "*", "\\*")
	escapedAnalysisText = strings.ReplaceAll(escapedAnalysisText, "[", "\\[")
	escapedAnalysisText = strings.ReplaceAll(escapedAnalysisText, "]", "\\]")
	escapedAnalysisText = strings.ReplaceAll(escapedAnalysisText, "`", "\\`")

	// Ensure text is valid UTF-8
	escapedAnalysisText = strings.ToValidUTF8(escapedAnalysisText, "")

	// Truncate analysis text if it's too long (Telegram has a 1024 character limit for captions)
	const maxCaptionLength = 900 // Leave some room for the rest of the message
	if len(escapedAnalysisText) > maxCaptionLength {
		escapedAnalysisText = escapedAnalysisText[:maxCaptionLength-3] + "..."
	}

	// Send analysis result with photo
	var weightText string
	if weight > 0 {
		weightText = fmt.Sprintf("⚖️ *Введенный вес:* %.1f г", weight)
	} else if analysis.Weight > 0 {
		weightText = fmt.Sprintf("⚖️ *Рассчитанный вес:* %.1f г", analysis.Weight)
	} else {
		weightText = "⚖️ *Вес:* не указан"
	}

	// Log weights for debugging
	logger.Debug("Weight comparison", "user_weight", weight, "analysis_weight", analysis.Weight)

	// Convert confidence to string representation
	var confidenceText string
	switch {
	case analysis.Confidence >= 0.8:
		confidenceText = "высокая"
	case analysis.Confidence >= 0.6:
		confidenceText = "средняя"
	default:
		confidenceText = "низкая"
	}

	// Format insulin recommendation
	var insulinText string
	if analysis.InsulinRatio > 0 {
		insulinText = fmt.Sprintf("💉 *Рекомендуемая доза инсулина:* %.1f ед.\n(%.1f ХЕ × %.1f ед/ХЕ)",
			analysis.InsulinUnits,
			analysis.BreadUnits,
			analysis.InsulinRatio)
	} else {
		insulinText = "💉 *Рекомендация по инсулину:* не настроен коэффициент для текущего времени"
	}

	resultText := fmt.Sprintf("🍽️ *Анализ блюда*\n\n"+
		"🍞 *Углеводы:* %.1f г\n"+
		"🥖 *ХЕ:* %.1f\n"+
		"%s\n"+
		"🎯 *Уверенность:* %s\n"+
		"%s\n\n"+
		"📊 *Как считали:*\n%s",
		analysis.Carbs,
		analysis.BreadUnits,
		insulinText,
		confidenceText,
		weightText,
		escapedAnalysisText,
	)

	// Ensure the entire result text is valid UTF-8
	resultText = strings.ToValidUTF8(resultText, "")

	// Create photo message with caption
	photoMsg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileID(photo.FileID))
	photoMsg.Caption = resultText
	photoMsg.ParseMode = "Markdown"

	// Add navigation buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ В главное меню", "main_menu"),
		),
	)
	photoMsg.ReplyMarkup = keyboard

	_, err = b.api.Send(photoMsg)
	if err != nil {
		// If Markdown parsing fails, try sending without Markdown
		photoMsg.ParseMode = ""
		_, err = b.api.Send(photoMsg)
		if err != nil {
			return fmt.Errorf("failed to send photo message: %w", err)
		}
	}

	// Reset user state
	b.userStates[int64(user.ID)] = stateNone
	return nil
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	log.Println("Bot is now listening for updates...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Bot is shutting down...")
			return ctx.Err()
		case update := <-updates:
			if update.Message != nil {
				logger.Debug("Received message", "user_id", update.Message.From.ID, "text", update.Message.Text)
			}
			if err := b.handleUpdate(ctx, update); err != nil {
				logger.Error("Error handling update", "error", err)
			}
		}
	}
}
