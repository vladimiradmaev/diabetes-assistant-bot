package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
)

type AIService struct {
	geminiClient *genai.Client
	openaiClient *openai.Client
}

type FoodAnalysisResult struct {
	FoodItems    []string `json:"food_items"`
	Carbs        float64  `json:"carbs"`
	Confidence   string   `json:"confidence"`
	AnalysisText string   `json:"analysis_text"`
}

func NewAIService(geminiAPIKey, openaiAPIKey string) *AIService {
	geminiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(geminiAPIKey))
	if err != nil {
		panic(fmt.Sprintf("Failed to create Gemini client: %v", err))
	}

	openaiClient := openai.NewClient(openaiAPIKey)

	return &AIService{
		geminiClient: geminiClient,
		openaiClient: openaiClient,
	}
}

func (s *AIService) AnalyzeFoodImage(ctx context.Context, imageURL string, weight float64, useOpenAI bool) (*FoodAnalysisResult, error) {
	if weight <= 0 {
		// If no weight provided, estimate it first
		estimatedWeight, err := s.estimateWeight(ctx, imageURL, useOpenAI)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate weight: %w", err)
		}
		weight = estimatedWeight
	}

	if useOpenAI {
		return s.analyzeWithOpenAI(ctx, imageURL, weight)
	}
	return s.analyzeWithGemini(ctx, imageURL, weight)
}

func (s *AIService) estimateWeight(ctx context.Context, imageURL string, useOpenAI bool) (float64, error) {
	if useOpenAI {
		return s.estimateWeightWithOpenAI(ctx, imageURL)
	}
	return s.estimateWeightWithGemini(ctx, imageURL)
}

func (s *AIService) estimateWeightWithGemini(ctx context.Context, imageURL string) (float64, error) {
	model := s.geminiClient.GenerativeModel("gemini-1.5-flash")

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

	prompt := `You are a food weight estimation expert. Your task is to estimate the weight of the food in the image in grams.

REQUIREMENTS:
- Estimate the weight as accurately as possible
- Consider standard portion sizes
- Account for the plate/bowl size if visible
- Return ONLY a number representing the weight in grams
- Do not include any text, units, or explanations
- Round to the nearest gram

Example response format:
150`

	img := genai.ImageData("image/jpeg", imageData)
	geminiResp, err := model.GenerateContent(ctx, img, genai.Text(prompt))
	if err != nil {
		return 0, fmt.Errorf("failed to generate content: %w", err)
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].(genai.Text)
	weight, err := strconv.ParseFloat(strings.TrimSpace(string(responseText)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse weight: %w", err)
	}

	return weight, nil
}

func (s *AIService) estimateWeightWithOpenAI(ctx context.Context, imageURL string) (float64, error) {
	prompt := `You are a food weight estimation expert. Your task is to estimate the weight of the food in the image in grams.

REQUIREMENTS:
- Estimate the weight as accurately as possible
- Consider standard portion sizes
- Account for the plate/bowl size if visible
- Return ONLY a number representing the weight in grams
- Do not include any text, units, or explanations
- Round to the nearest gram

Example response format:
150`

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
	model := s.geminiClient.GenerativeModel("gemini-1.5-flash")

	// Download image
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	prompt := fmt.Sprintf(`You are a certified diabetes educator specializing in nutrition analysis. 
You will analyze the food in the image to estimate its carbohydrate content accurately for diabetes management.

TASK:
1. Identify the food items in the image
2. Estimate total carbohydrates (in grams) based on standard nutritional databases
3. Assess your confidence in this estimation (low, medium, high)
4. Provide the information in a specific JSON format

REQUIREMENTS:
- Be medically precise in your carbohydrate estimation
- Include both visible ingredients and likely hidden ingredients that contain carbs
- Consider portion sizes carefully
- Account for various cooking methods that might affect carbohydrate content
- If the image contains nutritional information or packaging, prioritize that data
- IMPORTANT: Provide all text responses in Russian language for Russian users
- Food names should be in Russian
- Reasoning/descriptions should be in Russian
- Keep the analysis text concise and focused on methodology
- Use bullet points for key points
- Avoid unnecessary explanations
- Focus on how the calculation was made

IMPORTANT WEIGHT INFORMATION:
- The user has specified that the food weighs %.1f grams
- Adjust your carbohydrate calculation based on this exact weight
- Make sure to mention the weight in your reasoning

CRITICAL JSON FORMAT REQUIREMENTS:
- Your response MUST be a valid JSON object
- Do not include any markdown formatting, bullet points, or dashes
- Do not include any explanatory text before or after the JSON
- The JSON must have these exact fields:
  {
    "food_items": ["item1", "item2"],
    "carbs": 123.45,
    "confidence": "low|medium|high",
    "analysis_text": "Your analysis in Russian"
  }`, weight)

	img := genai.ImageData("image/jpeg", imageData)
	geminiResp, err := model.GenerateContent(ctx, img, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var result FoodAnalysisResult
	responseText := geminiResp.Candidates[0].Content.Parts[0].(genai.Text)
	// Extract JSON from the response, handling code blocks or text wrapping
	jsonStr := extractJSON(string(responseText))
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

func (s *AIService) analyzeWithOpenAI(ctx context.Context, imageURL string, weight float64) (*FoodAnalysisResult, error) {
	prompt := fmt.Sprintf(`You are a certified diabetes educator specializing in nutrition analysis. 
You will analyze the food in the image to estimate its carbohydrate content accurately for diabetes management.

TASK:
1. Identify the food items in the image
2. Estimate total carbohydrates (in grams) based on standard nutritional databases
3. Assess your confidence in this estimation (low, medium, high)
4. Provide the information in a specific JSON format

REQUIREMENTS:
- Be medically precise in your carbohydrate estimation
- Include both visible ingredients and likely hidden ingredients that contain carbs
- Consider portion sizes carefully
- Account for various cooking methods that might affect carbohydrate content
- If the image contains nutritional information or packaging, prioritize that data
- IMPORTANT: Provide all text responses in Russian language for Russian users
- Food names should be in Russian
- Reasoning/descriptions should be in Russian
- Keep the analysis text concise and focused on methodology
- Use bullet points for key points
- Avoid unnecessary explanations
- Focus on how the calculation was made

IMPORTANT WEIGHT INFORMATION:
- The user has specified that the food weighs %.1f grams
- Adjust your carbohydrate calculation based on this exact weight
- Make sure to mention the weight in your reasoning

CRITICAL JSON FORMAT REQUIREMENTS:
- Your response MUST be a valid JSON object
- Do not include any markdown formatting, bullet points, or dashes
- Do not include any explanatory text before or after the JSON
- The JSON must have these exact fields:
  {
    "food_items": ["item1", "item2"],
    "carbs": 123.45,
    "confidence": "low|medium|high",
    "analysis_text": "Your analysis in Russian"
  }`, weight)

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
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	var result FoodAnalysisResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// extractJSON attempts to extract a valid JSON object from the given string.
// It handles cases where the JSON is wrapped in code blocks (```json ... ```) or other text.
func extractJSON(s string) string {
	// Try to find a JSON object (starting with '{' and ending with '}')
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end <= start {
		return ""
	}
	return s[start : end+1]
}
