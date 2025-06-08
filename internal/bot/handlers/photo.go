package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vladimiradmaev/diabetes-helper/internal/bot/state"
	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
)

// PhotoHandler handles photo messages
type PhotoHandler struct {
	api          *tgbotapi.BotAPI
	deps         Dependencies
	stateManager state.StateManager
}

// NewPhotoHandler creates a new photo handler
func NewPhotoHandler(api *tgbotapi.BotAPI, deps Dependencies, stateManager state.StateManager) *PhotoHandler {
	return &PhotoHandler{
		api:          api,
		deps:         deps,
		stateManager: stateManager,
	}
}

// Handle processes a photo message
func (h *PhotoHandler) Handle(ctx context.Context, message *tgbotapi.Message, user *database.User) error {
	// Get the largest photo
	photo := message.Photo[len(message.Photo)-1]
	file, err := h.api.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Check if weight is provided in caption or saved from state
	weight := 0.0

	// First check for saved weight from the food analysis flow
	savedWeight := h.stateManager.GetUserWeight(user.TelegramID)
	if savedWeight > 0 {
		weight = savedWeight
		logger.Infof("User %d using saved weight: %.1f g", user.ID, weight)
		// Clear saved weight after use
		h.stateManager.SetUserWeight(user.TelegramID, 0)
	} else if message.Caption != "" {
		weight, err = strconv.ParseFloat(message.Caption, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неверный формат веса. Пожалуйста, укажите вес в граммах (например: 100).")
			_, err := h.api.Send(msg)
			return err
		}
		logger.Infof("User %d provided weight in caption: %.1f g", user.ID, weight)
	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Вес не указан. Я попробую оценить вес блюда автоматически.")
		_, err := h.api.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send weight estimation message: %w", err)
		}
	}

	// Send "processing" message
	processingMsg := tgbotapi.NewMessage(message.Chat.ID, "Анализирую изображение...")
	sentMsg, err := h.api.Send(processingMsg)
	if err != nil {
		return fmt.Errorf("failed to send processing message: %w", err)
	}

	// Analyze the image
	logger.Infof("Starting food analysis for user %d with Gemini", user.ID)
	analysis, err := h.deps.FoodAnalysisSvc.AnalyzeFood(ctx, user.ID, file.Link(h.api.Token), weight)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Извините, произошла ошибка при анализе изображения. Пожалуйста, попробуйте еще раз через несколько минут.")
		_, err := h.api.Send(msg)
		return err
	}
	logger.Infof("Food analysis completed for user %d", user.ID)

	// Delete processing message
	deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, sentMsg.MessageID)
	h.api.Send(deleteMsg)

	// Check if no food was detected (independent of weight)
	if analysis.Carbs == 0 && len(analysis.AnalysisText) > 0 &&
		strings.Contains(analysis.AnalysisText, "не обнаружена еда") {
		// Send a simple text message for non-food images with proper navigation
		msg := tgbotapi.NewMessage(message.Chat.ID, "На изображении не обнаружена еда. Пожалуйста, отправьте фото блюда для анализа.")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Новый анализ", "analyze_food"),
			),
		)
		msg.ReplyMarkup = keyboard
		_, err = h.api.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send non-food message: %w", err)
		}
		// Reset user state
		h.stateManager.SetUserState(user.TelegramID, state.None)
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
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Новый анализ", "analyze_food"),
		),
	)
	photoMsg.ReplyMarkup = keyboard

	_, err = h.api.Send(photoMsg)
	if err != nil {
		// If Markdown parsing fails, try sending without Markdown
		photoMsg.ParseMode = ""
		_, err = h.api.Send(photoMsg)
		if err != nil {
			return fmt.Errorf("failed to send photo message: %w", err)
		}
	}

	// Reset user state
	h.stateManager.SetUserState(user.TelegramID, state.None)
	return nil
}
