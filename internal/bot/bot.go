package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/services"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	userService     *services.UserService
	foodAnalysisSvc *services.FoodAnalysisService
	userStates      map[int64]string  // Map to track user states
	userWeights     map[int64]float64 // Map to store user-provided weights
}

func NewBot(token string, userService *services.UserService, foodAnalysisSvc *services.FoodAnalysisService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	log.Printf("Bot authorized on account %s", api.Self.UserName)
	return &Bot{
		api:             api,
		userService:     userService,
		foodAnalysisSvc: foodAnalysisSvc,
		userStates:      make(map[int64]string),
		userWeights:     make(map[int64]float64),
	}, nil
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

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil {
		return nil
	}

	// Register user
	user, err := b.userService.RegisterUser(
		ctx,
		update.Message.From.ID,
		update.Message.From.UserName,
		update.Message.From.FirstName,
		update.Message.From.LastName,
	)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	log.Printf("User registered/updated: %s (ID: %d)", user.Username, user.ID)

	// Handle commands
	if update.Message.IsCommand() {
		return b.handleCommand(ctx, update.Message, user)
	}

	// Handle photo messages
	if update.Message.Photo != nil {
		log.Printf("Received photo from user %d", user.ID)
		return b.handlePhoto(ctx, update.Message, user)
	}

	// Handle text messages
	if update.Message.Text != "" {
		return b.handleText(ctx, update.Message, user)
	}

	return nil
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	log.Printf("Handling command %s from user %d", message.Command(), user.ID)
	switch message.Command() {
	case "start":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Привет! Я бот для анализа углеводов в еде. Отправь мне фото еды, и я помогу определить количество углеводов.")
		_, err := b.api.Send(msg)
		return err
	case "help":
		msg := tgbotapi.NewMessage(message.Chat.ID, `Доступные команды:
/start - Начать работу с ботом
/help - Показать это сообщение

Как указать вес блюда:
1. Отправьте фото еды
2. В подписи к фото напишите только число - вес в граммах
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

	// Send analysis result with photo
	resultText := fmt.Sprintf("🍽️ *Анализ блюда*\n\n"+
		"🍞 *Углеводы:* %.1f г\n"+
		"🎯 *Уверенность:* %s\n\n"+
		"📊 *Как считали:*\n%s",
		analysis.Carbs,
		analysis.Confidence,
		analysis.AnalysisText,
	)
	if weight <= 0 {
		resultText = fmt.Sprintf("⚖️ *Оцененный вес:* %.1f г\n\n%s", analysis.Weight, resultText)
	}

	// Create photo message with caption
	photoMsg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileID(photo.FileID))
	photoMsg.Caption = resultText
	photoMsg.ParseMode = "Markdown"
	_, err = b.api.Send(photoMsg)
	return err
}

func (b *Bot) handleText(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, отправьте фото еды для анализа углеводов. Вы также можете указать вес блюда в граммах в подписи к фото.")
	_, err := b.api.Send(msg)
	return err
}
