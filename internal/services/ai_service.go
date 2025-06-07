package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	apperrors "github.com/vladimiradmaev/diabetes-helper/internal/errors"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type AIService struct {
	geminiClient *genai.Client
	logger       *slog.Logger
}

type FoodAnalysisResult struct {
	FoodItems    []string `json:"food_items"`
	Carbs        float64  `json:"carbs"`
	Confidence   string   `json:"confidence"`
	AnalysisText string   `json:"analysis_text"`
	Weight       float64  `json:"weight"`
}

func NewAIService(geminiAPIKey string) *AIService {
	service := &AIService{
		logger: logger.GetLogger(),
	}

	// Initialize Gemini client
	if geminiAPIKey != "" {
		service.logger.Info("Initializing Gemini client", "api_key_length", len(geminiAPIKey))
		ctx := context.Background()
		client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
		if err != nil {
			service.logger.Error("Failed to initialize Gemini client", "error", err)
		} else {
			service.geminiClient = client
			service.logger.Info("Gemini client initialized successfully")
			service.logger.Info("Testing Gemini model", "model", "gemini-2.0-flash")
		}
	} else {
		service.logger.Error("Gemini API key not provided")
	}

	return service
}

func retryWithBackoff(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err

			// Check if it's a retryable error
			if googleErr, ok := err.(*googleapi.Error); ok {
				if googleErr.Code == 429 || googleErr.Code >= 500 {
					// Rate limit or server error - retry with backoff
					backoff := time.Duration(i+1) * time.Second
					logger.Warningf("Retryable error occurred (attempt %d/%d): %v. Retrying in %v", i+1, maxRetries, err, backoff)

					select {
					case <-time.After(backoff):
						continue
					case <-ctx.Done():
						return ctx.Err()
					}
				} else {
					// Non-retryable error
					logger.Errorf("Non-retryable error occurred: %v", err)
					return err
				}
			} else {
				// Other errors - retry with backoff
				backoff := time.Duration(i+1) * time.Second
				logger.Warningf("Error occurred (attempt %d/%d): %v. Retrying in %v", i+1, maxRetries, err, backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		} else {
			return nil
		}
	}
	return lastErr
}

func (s *AIService) AnalyzeFoodImage(ctx context.Context, imageURL string, weight float64) (*FoodAnalysisResult, error) {
	s.logger.InfoContext(ctx, "Starting food image analysis",
		"image_url", imageURL,
		"weight", weight)

	if s.geminiClient == nil {
		return nil, apperrors.NewExternalAPIError(
			fmt.Errorf("Gemini client not available"),
			"Gemini").WithContext("operation", "analyze_food_image")
	}

	var estimatedWeight float64
	var err error

	if weight <= 0 {
		s.logger.InfoContext(ctx, "No weight provided, estimating weight from image")
		// If no weight provided, estimate it first
		estimatedWeight, err = s.estimateWeight(ctx, imageURL)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to estimate weight", "error", err)
			// Не возвращаем ошибку, а продолжаем анализ без веса
			s.logger.InfoContext(ctx, "Continuing analysis without weight estimation")
		} else {
			weight = estimatedWeight
			s.logger.InfoContext(ctx, "Estimated weight", "weight", weight)
		}
	}

	result, err := s.analyzeWithGemini(ctx, imageURL, weight)
	if err != nil {
		return nil, apperrors.NewExternalAPIError(err, "Gemini").
			WithContext("operation", "analyze_with_gemini").
			WithContext("image_url", imageURL).
			WithContext("weight", weight)
	}

	// Ensure the weight is set in the result
	if weight > 0 {
		result.Weight = weight
	}

	s.logger.InfoContext(ctx, "Food analysis completed successfully",
		"carbs", result.Carbs,
		"confidence", result.Confidence,
		"food_items_count", len(result.FoodItems))
	return result, nil
}

func (s *AIService) estimateWeight(ctx context.Context, imageURL string) (float64, error) {
	if s.geminiClient == nil {
		return 0, fmt.Errorf("Gemini client not available for weight estimation")
	}
	return s.estimateWeightWithGemini(ctx, imageURL)
}

func (s *AIService) estimateWeightWithGemini(ctx context.Context, imageURL string) (float64, error) {
	model := s.geminiClient.GenerativeModel("gemini-2.0-flash")

	// Download image
	resp, err := http.Get(imageURL)
	if err != nil {
		return 0, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read image data: %w", err)
	}

	prompt := `Оцени вес еды в граммах, используя визуальные подсказки:

РЕФЕРЕНСНЫЕ ОБЪЕКТЫ для масштаба:
- Тарелка стандартная: диаметр 24-26см
- Столовая ложка: длина 20см
- Вилка: длина 20см  
- Стакан: высота 10-12см, диаметр 7-8см
- Чашка кофе: диаметр 8-9см
- Монета (если видна): диаметр 2-2.5см

ТИПИЧНЫЕ ПОРЦИИ:
- Рис/гречка/макароны: 150-250г (размер кулака)
- Мясо/рыба: 100-200г (размер ладони)
- Овощи свежие: 100-200г
- Хлеб (ломтик): 25-30г
- Картофель (средний): 100-150г
- Яйцо: 50-60г
- Сыр (кусок): 30-50г

АНАЛИЗИРУЙ:
1. Размер порции относительно тарелки/посуды
2. Толщину/высоту блюда
3. Плотность продуктов (мясо тяжелее овощей)
4. Количество компонентов

Верни ТОЛЬКО число в граммах (например: 180)`

	var weight float64
	err = retryWithBackoff(ctx, 3, func() error {
		// Detect image format
		imageFormat := "image/jpeg"
		if len(imageData) > 4 {
			if imageData[0] == 0x89 && imageData[1] == 0x50 && imageData[2] == 0x4E && imageData[3] == 0x47 {
				imageFormat = "image/png"
			} else if imageData[0] == 0x47 && imageData[1] == 0x49 && imageData[2] == 0x46 {
				imageFormat = "image/gif"
			} else if imageData[0] == 0xFF && imageData[1] == 0xD8 {
				imageFormat = "image/jpeg"
			}
		}

		img := genai.ImageData(imageFormat, imageData)
		geminiResp, err := model.GenerateContent(ctx, img, genai.Text(prompt))
		if err != nil {
			return err
		}

		// Check response structure
		if len(geminiResp.Candidates) == 0 {
			return fmt.Errorf("no candidates in Gemini response")
		}
		if geminiResp.Candidates[0].Content == nil {
			return fmt.Errorf("no content in Gemini candidate")
		}
		if len(geminiResp.Candidates[0].Content.Parts) == 0 {
			return fmt.Errorf("no parts in Gemini content")
		}

		responseText := geminiResp.Candidates[0].Content.Parts[0].(genai.Text)
		responseStr := strings.TrimSpace(string(responseText))

		// Проверяем, содержит ли ответ только число
		if strings.Contains(strings.ToLower(responseStr), "невозможно") ||
			strings.Contains(strings.ToLower(responseStr), "нет еды") ||
			strings.Contains(strings.ToLower(responseStr), "не видно") ||
			len(responseStr) > 10 { // Если ответ слишком длинный, это не число
			return fmt.Errorf("AI не смог определить вес: %s", responseStr)
		}

		parsedWeight, parseErr := strconv.ParseFloat(responseStr, 64)
		if parseErr != nil {
			return fmt.Errorf("failed to parse weight from response '%s': %w", responseStr, parseErr)
		}

		weight = parsedWeight
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to estimate weight with retries: %w", err)
	}

	return weight, nil
}

func (s *AIService) analyzeWithGemini(ctx context.Context, imageURL string, weight float64) (*FoodAnalysisResult, error) {
	s.logger.DebugContext(ctx, "Starting Gemini analysis", "image_url", imageURL, "weight", weight)
	model := s.geminiClient.GenerativeModel("gemini-2.0-flash")

	// Download image
	s.logger.DebugContext(ctx, "Downloading image from URL")
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, apperrors.NewExternalAPIError(err, "HTTP").
			WithContext("image_url", imageURL).
			WithContext("operation", "download_image")
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewInternalError(err).
			WithContext("operation", "read_image_data")
	}
	s.logger.DebugContext(ctx, "Downloaded image data", "bytes", len(imageData))

	prompt := fmt.Sprintf(`Вы — точный ассистент по анализу продуктов питания для контроля диабета. Ваша основная задача — распознавать продукты на изображении, оценивать их вес, если он не указан, и рассчитывать общее количество углеводов.

**Входные данные:** Изображение еды. Вес: %.1f г (если 0 - оцените самостоятельно).

**Процесс:**
1. **Определите ВСЕ съедобные продукты.** Сюда входят приготовленные блюда, сырые ингредиенты, закуски и калорийные напитки.
2. **Если еда отсутствует:** (например, пустые тарелки, только столовые приборы, объекты, не являющиеся едой), верните JSON-структуру "НЕТ ЕДЫ", указанную ниже.
3. **Для каждого найденного продукта:**
   * Оцените его индивидуальный вес в граммах, если общий вес равен 0 или требует уточнения.
   * Рассчитайте содержание углеводов в граммах, включая крахмалы, сахара и углеводы из панировки, соусов или глазури.
4. **Рассчитайте общее количество углеводов** для всех найденных продуктов.
5. **Определите уровень достоверности:** "high" (высокий), если продукты четко видны и легко идентифицируются; "medium" (средний), если есть некоторые неясности; "low" (низкий), если идентификация очень сложна или частична.

**Формат вывода (ТОЛЬКО JSON):**

**A. Если еда не обнаружена:**
{"food_items":[],"carbs":0,"confidence":"low","analysis_text":"На изображении не обнаружена еда. Пожалуйста, отправьте фото блюда для анализа.","weight":0}

**B. Если еда найдена:**
{"food_items":["продукт1","продукт2"],"carbs":X.X,"confidence":"high/medium/low","analysis_text":"ПОДРОБНЫЙ АНАЛИЗ НА РУССКОМ: 1. Название блюда: Xг, Yг углеводов","weight":X.X}

Анализируйте внимательно и возвращайте точный JSON.`, weight)

	var result FoodAnalysisResult
	logger.Debug("Sending request to Gemini API")
	err = retryWithBackoff(ctx, 3, func() error {
		// Detect image format from content
		imageFormat := "image/jpeg"
		if len(imageData) > 4 {
			if imageData[0] == 0x89 && imageData[1] == 0x50 && imageData[2] == 0x4E && imageData[3] == 0x47 {
				imageFormat = "image/png"
			} else if imageData[0] == 0x47 && imageData[1] == 0x49 && imageData[2] == 0x46 {
				imageFormat = "image/gif"
			} else if imageData[0] == 0xFF && imageData[1] == 0xD8 {
				imageFormat = "image/jpeg"
			}
		}
		s.logger.DebugContext(ctx, "Detected image format", "format", imageFormat)

		img := genai.ImageData(imageFormat, imageData)
		geminiResp, err := model.GenerateContent(ctx, img, genai.Text(prompt))
		if err != nil {
			logger.Errorf("Gemini API request failed: %v", err)
			return err
		}

		// Check if response has candidates
		if len(geminiResp.Candidates) == 0 {
			logger.Error("Gemini response has no candidates")
			return fmt.Errorf("no candidates in Gemini response")
		}

		// Check if candidate has content
		if geminiResp.Candidates[0].Content == nil {
			logger.Error("Gemini candidate has no content")
			return fmt.Errorf("no content in Gemini candidate")
		}

		// Check if content has parts
		if len(geminiResp.Candidates[0].Content.Parts) == 0 {
			logger.Error("Gemini content has no parts")
			return fmt.Errorf("no parts in Gemini content")
		}

		responseText := geminiResp.Candidates[0].Content.Parts[0].(genai.Text)
		s.logger.DebugContext(ctx, "Gemini raw response", "response", string(responseText))

		// Extract JSON from the response, handling code blocks or text wrapping
		jsonStr := extractJSON(string(responseText))
		if jsonStr == "" {
			logger.Error("No valid JSON found in Gemini response")
			return fmt.Errorf("no valid JSON found in response")
		}
		s.logger.DebugContext(ctx, "Extracted JSON", "json", jsonStr)

		if parseErr := json.Unmarshal([]byte(jsonStr), &result); parseErr != nil {
			logger.Errorf("Failed to parse JSON response: %v", parseErr)
			return fmt.Errorf("failed to parse response: %w", parseErr)
		}
		return nil
	})

	if err != nil {
		logger.Errorf("Gemini analysis failed after retries: %v", err)
		return nil, fmt.Errorf("failed to analyze with retries: %w", err)
	}

	return &result, nil
}

func extractJSON(s string) string {
	// Remove markdown code blocks
	s = strings.ReplaceAll(s, "```json", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.TrimSpace(s)

	// Find JSON object boundaries
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}

	// Find the matching closing brace
	braceCount := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			braceCount++
		} else if s[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return s[start : i+1]
			}
		}
	}

	return ""
}
