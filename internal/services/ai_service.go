package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"github.com/vladimiradmaev/diabetes-helper/internal/logger"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type AIService struct {
	geminiClient *genai.Client
	openaiClient *openai.Client
	hasOpenAIKey bool
	hasGeminiKey bool
}

type FoodAnalysisResult struct {
	FoodItems    []string `json:"food_items"`
	Carbs        float64  `json:"carbs"`
	Confidence   string   `json:"confidence"`
	AnalysisText string   `json:"analysis_text"`
	Weight       float64  `json:"weight"`
}

func NewAIService(geminiAPIKey, openaiAPIKey string) *AIService {
	service := &AIService{
		hasGeminiKey: geminiAPIKey != "",
		hasOpenAIKey: openaiAPIKey != "",
	}

	// Initialize Gemini client only if key is provided
	if service.hasGeminiKey {
		logger.Infof("Initializing Gemini client with API key (length: %d)", len(geminiAPIKey))
		geminiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(geminiAPIKey))
		if err != nil {
			logger.Errorf("Failed to create Gemini client: %v", err)
			service.hasGeminiKey = false
		} else {
			service.geminiClient = geminiClient
			logger.Info("Gemini client initialized successfully")

			// Test model availability
			_ = geminiClient.GenerativeModel("gemini-2.0-flash")
			logger.Infof("Testing Gemini model availability: %s", "gemini-2.0-flash")
		}
	} else {
		logger.Warning("Gemini API key not provided, Gemini features will be disabled")
	}

	// Initialize OpenAI client only if key is provided
	if service.hasOpenAIKey {
		service.openaiClient = openai.NewClient(openaiAPIKey)
		logger.Info("OpenAI client initialized successfully")
	} else {
		logger.Warning("OpenAI API key not provided, OpenAI features will be disabled")
	}

	// Ensure at least one AI service is available
	if !service.hasGeminiKey && !service.hasOpenAIKey {
		panic("No AI API keys provided. Please set either GEMINI_API_KEY or OPENAI_API_KEY")
	}

	return service
}

// retryWithBackoff выполняет функцию с экспоненциальной задержкой при ошибках 429
func retryWithBackoff(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Проверяем, является ли ошибка 429 (Too Many Requests)
		if googleErr, ok := err.(*googleapi.Error); ok && googleErr.Code == 429 {
			if attempt < maxRetries {
				// Экспоненциальная задержка: 2^attempt секунд + jitter
				delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				if delay > 60*time.Second {
					delay = 60 * time.Second // Максимум 60 секунд
				}

				logger.Warningf("Rate limit exceeded (429), retrying in %v... (attempt %d/%d)", delay, attempt+1, maxRetries+1)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		} else {
			// Если это не 429 ошибка, не повторяем
			logger.Errorf("Non-retryable error occurred: %v", err)
			return err
		}
	}

	logger.Errorf("Max retries exceeded for operation: %v", lastErr)
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (s *AIService) AnalyzeFoodImage(ctx context.Context, imageURL string, weight float64, useOpenAI bool) (*FoodAnalysisResult, error) {
	logger.Infof("Starting food image analysis, imageURL: %s, weight: %.1f g, useOpenAI: %v", imageURL, weight, useOpenAI)

	// Check if requested AI service is available
	if useOpenAI && !s.hasOpenAIKey {
		if s.hasGeminiKey {
			logger.Warning("OpenAI requested but not available, switching to Gemini")
			useOpenAI = false
		} else {
			return nil, fmt.Errorf("OpenAI requested but API key not provided")
		}
	}

	if !useOpenAI && !s.hasGeminiKey {
		if s.hasOpenAIKey {
			logger.Warning("Gemini requested but not available, switching to OpenAI")
			useOpenAI = true
		} else {
			return nil, fmt.Errorf("Gemini requested but API key not provided")
		}
	}

	var estimatedWeight float64
	var err error

	if weight <= 0 {
		logger.Info("No weight provided, estimating weight from image")
		// If no weight provided, estimate it first
		estimatedWeight, err = s.estimateWeight(ctx, imageURL, useOpenAI)
		if err != nil {
			logger.Errorf("Failed to estimate weight: %v", err)
			return nil, fmt.Errorf("failed to estimate weight: %w", err)
		}
		weight = estimatedWeight
		logger.Infof("Estimated weight: %.1f g", weight)
	}

	var result *FoodAnalysisResult
	if useOpenAI {
		logger.Info("Using OpenAI for food analysis")
		result, err = s.analyzeWithOpenAI(ctx, imageURL, weight)
	} else {
		logger.Info("Using Gemini for food analysis")
		result, err = s.analyzeWithGemini(ctx, imageURL, weight)
	}
	if err != nil {
		logger.Errorf("Food analysis failed: %v", err)
		return nil, err
	}

	// Ensure the weight is set in the result
	if weight > 0 {
		result.Weight = weight
	}

	logger.Infof("Food analysis completed successfully: %+v", result)
	return result, nil
}

func (s *AIService) estimateWeight(ctx context.Context, imageURL string, useOpenAI bool) (float64, error) {
	if useOpenAI && s.hasOpenAIKey {
		return s.estimateWeightWithOpenAI(ctx, imageURL)
	}
	if !useOpenAI && s.hasGeminiKey {
		return s.estimateWeightWithGemini(ctx, imageURL)
	}

	// Fallback logic
	if s.hasGeminiKey {
		logger.Warning("Falling back to Gemini for weight estimation")
		return s.estimateWeightWithGemini(ctx, imageURL)
	}
	if s.hasOpenAIKey {
		logger.Warning("Falling back to OpenAI for weight estimation")
		return s.estimateWeightWithOpenAI(ctx, imageURL)
	}

	return 0, fmt.Errorf("no AI service available for weight estimation")
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

	prompt := `Estimate food weight in grams. Consider portion sizes and plate size.
Standard portions: rice/pasta 150-200g, meat 100-150g, vegetables 100-150g.
Return ONLY the number (e.g., 150).`

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
		parsedWeight, parseErr := strconv.ParseFloat(strings.TrimSpace(string(responseText)), 64)
		if parseErr != nil {
			return fmt.Errorf("failed to parse weight: %w", parseErr)
		}

		weight = parsedWeight
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to estimate weight with retries: %w", err)
	}

	return weight, nil
}

func (s *AIService) estimateWeightWithOpenAI(ctx context.Context, imageURL string) (float64, error) {
	prompt := `Estimate food weight in grams. Consider portion sizes and plate size.
Standard portions: rice/pasta 150-200g, meat 100-150g, vegetables 100-150g.
Return ONLY the number (e.g., 150).`

	resp, err := s.openaiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4VisionPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: prompt,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL: imageURL,
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create chat completion: %w", err)
	}

	weight, err := strconv.ParseFloat(strings.TrimSpace(resp.Choices[0].Message.Content), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse weight: %w", err)
	}

	return weight, nil
}

func (s *AIService) analyzeWithGemini(ctx context.Context, imageURL string, weight float64) (*FoodAnalysisResult, error) {
	logger.Debugf("Starting Gemini analysis for image: %s", imageURL)
	model := s.geminiClient.GenerativeModel("gemini-2.0-flash")

	// Download image
	logger.Debug("Downloading image from URL")
	resp, err := http.Get(imageURL)
	if err != nil {
		logger.Errorf("Failed to download image from %s: %v", imageURL, err)
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Failed to read image data: %v", err)
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	logger.Debugf("Downloaded image data: %d bytes", len(imageData))

	prompt := fmt.Sprintf(`Analyze this image for diabetes management. Weight: %.1f g (estimate if 0).

Look carefully at the image. If you see ANY edible food items (cooked dishes, raw ingredients, snacks, drinks with calories, etc.), analyze them.

ONLY if the image contains absolutely NO food items (like empty plates, utensils only, non-food objects), return:
{"food_items":[],"carbs":0,"confidence":"low","analysis_text":"На изображении не обнаружена еда. Пожалуйста, отправьте фото блюда для анализа.","weight":0}

For ANY food items found:
- List all food items
- Calculate total carbohydrates in grams
- Include starches, sugars, breading, sauces
- Confidence: high/medium/low based on visibility
- Analysis MUST be in Russian language only: "1. Название блюда: Xг, Yг углеводов"

Return JSON: {"food_items":["item1","item2"],"carbs":X.X,"confidence":"level","analysis_text":"RUSSIAN TEXT ONLY","weight":X.X}`, weight)

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
		logger.Debugf("Detected image format: %s", imageFormat)

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
		logger.Debugf("Gemini raw response: %s", string(responseText))

		// Extract JSON from the response, handling code blocks or text wrapping
		jsonStr := extractJSON(string(responseText))
		if jsonStr == "" {
			logger.Error("No valid JSON found in Gemini response")
			return fmt.Errorf("no valid JSON found in response")
		}
		logger.Debugf("Extracted JSON: %s", jsonStr)

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

func (s *AIService) analyzeWithOpenAI(ctx context.Context, imageURL string, weight float64) (*FoodAnalysisResult, error) {
	logger.Debugf("Starting OpenAI analysis for image: %s", imageURL)

	prompt := fmt.Sprintf(`Analyze this image for diabetes management. Weight: %.1f g (estimate if 0).

Look carefully at the image. If you see ANY edible food items (cooked dishes, raw ingredients, snacks, drinks with calories, etc.), analyze them.

ONLY if the image contains absolutely NO food items (like empty plates, utensils only, non-food objects), return:
{"food_items":[],"carbs":0,"confidence":"low","analysis_text":"На изображении не обнаружена еда. Пожалуйста, отправьте фото блюда для анализа.","weight":0}

For ANY food items found:
- List all food items
- Calculate total carbohydrates in grams
- Include starches, sugars, breading, sauces
- Confidence: high/medium/low based on visibility
- Analysis MUST be in Russian language only: "1. Название блюда: Xг, Yг углеводов"

Return JSON: {"food_items":["item1","item2"],"carbs":X.X,"confidence":"level","analysis_text":"RUSSIAN TEXT ONLY","weight":X.X}`, weight)

	logger.Debug("Sending request to OpenAI API")
	resp, err := s.openaiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4VisionPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: prompt,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL: imageURL,
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		logger.Errorf("OpenAI API request failed: %v", err)
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	var result FoodAnalysisResult
	responseText := resp.Choices[0].Message.Content
	logger.Debugf("OpenAI raw response: %s", responseText)

	// Extract JSON from the response, handling code blocks or text wrapping
	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		logger.Error("No valid JSON found in OpenAI response")
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	logger.Debugf("Extracted JSON: %s", jsonStr)

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		logger.Errorf("Failed to parse JSON response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Debug("OpenAI analysis completed successfully")
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
