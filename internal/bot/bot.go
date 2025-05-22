package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/services"
)

const (
	stateNone                 = "none"
	stateWaitingForBloodSugar = "waiting_for_blood_sugar"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	userService     *services.UserService
	foodAnalysisSvc *services.FoodAnalysisService
	bloodSugarSvc   *services.BloodSugarService
	userStates      map[int64]string  // Map to track user states
	userWeights     map[int64]float64 // Map to store user-provided weights
}

func NewBot(token string, userService *services.UserService, foodAnalysisSvc *services.FoodAnalysisService, bloodSugarSvc *services.BloodSugarService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	log.Printf("Bot authorized on account %s", api.Self.UserName)
	return &Bot{
		api:             api,
		userService:     userService,
		foodAnalysisSvc: foodAnalysisSvc,
		bloodSugarSvc:   bloodSugarSvc,
		userStates:      make(map[int64]string),
		userWeights:     make(map[int64]float64),
	}, nil
}

func (b *Bot) sendMainMenu(chatID int64) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🍽️ Анализ еды", "analyze_food"),
			tgbotapi.NewInlineKeyboardButtonData("🩸 Уровень сахара", "blood_sugar"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
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
	log.Printf("User registered/updated: %s (ID: %d)", user.Username, user.ID)

	// Handle callback queries (button clicks)
	if update.CallbackQuery != nil {
		// Answer callback query to remove loading state
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
		if _, err := b.api.Request(callback); err != nil {
			log.Printf("Failed to answer callback query: %v", err)
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

	case "main_menu":
		b.userStates[int64(user.ID)] = stateNone
		return b.sendMainMenu(query.Message.Chat.ID)
	}

	return nil
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	log.Printf("Handling command %s from user %d", message.Command(), user.ID)
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

		// Create keyboard for navigation
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🍽️ Анализ еды", "analyze_food"),
				tgbotapi.NewInlineKeyboardButtonData("🩸 Уровень сахара", "blood_sugar"),
			),
		)

		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Уровень сахара %.1f ммоль/л успешно сохранен", value))
		msg.ReplyMarkup = keyboard
		_, err = b.api.Send(msg)
		if err != nil {
			return err
		}

		b.userStates[int64(user.ID)] = stateNone
		return nil

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
		log.Printf("User %d provided weight: %.1f g", user.ID, weight)
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
	log.Printf("Starting food analysis for user %d with Gemini", user.ID)
	analysis, err := b.foodAnalysisSvc.AnalyzeFood(ctx, user.ID, file.Link(b.api.Token), weight, false)
	if err != nil {
		log.Printf("Gemini analysis failed for user %d, trying OpenAI: %v", user.ID, err)
		// Try OpenAI if Gemini fails
		analysis, err = b.foodAnalysisSvc.AnalyzeFood(ctx, user.ID, file.Link(b.api.Token), weight, true)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Извините, произошла ошибка при анализе изображения. Пожалуйста, попробуйте еще раз.")
			_, err := b.api.Send(msg)
			return err
		}
		log.Printf("OpenAI analysis completed for user %d", user.ID)
	} else {
		log.Printf("Gemini analysis completed for user %d", user.ID)
	}

	// Delete processing message
	deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, sentMsg.MessageID)
	b.api.Send(deleteMsg)

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
	log.Printf("User weight: %.1f, Analysis weight: %.1f", weight, analysis.Weight)

	resultText := fmt.Sprintf("🍽️ *Анализ блюда*\n\n"+
		"🍞 *Углеводы:* %.1f г\n"+
		"🎯 *Уверенность:* %s\n"+
		"%s\n\n"+
		"📊 *Как считали:*\n%s",
		analysis.Carbs,
		analysis.Confidence,
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
			tgbotapi.NewInlineKeyboardButtonData("🍽️ Анализ еды", "analyze_food"),
			tgbotapi.NewInlineKeyboardButtonData("🩸 Уровень сахара", "blood_sugar"),
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
				log.Printf("Received message from user %d: %s", update.Message.From.ID, update.Message.Text)
			}
			if err := b.handleUpdate(ctx, update); err != nil {
				log.Printf("Error handling update: %v", err)
			}
		}
	}
}
