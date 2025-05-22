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
	Weight       float64  `json:"weight"`
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
	var estimatedWeight float64
	var err error

	if weight <= 0 {
		// If no weight provided, estimate it first
		estimatedWeight, err = s.estimateWeight(ctx, imageURL, useOpenAI)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate weight: %w", err)
		}
		weight = estimatedWeight
	}

	var result *FoodAnalysisResult
	if useOpenAI {
		result, err = s.analyzeWithOpenAI(ctx, imageURL, weight)
	} else {
		result, err = s.analyzeWithGemini(ctx, imageURL, weight)
	}
	if err != nil {
		return nil, err
	}

	// Ensure the weight is set in the result
	if weight > 0 {
		result.Weight = weight
	}

	return result, nil
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
- Consider standard portion sizes and plate sizes
- Account for the plate/bowl size if visible
- Use these reference points for common items:
  * Standard dinner plate: 27-30 cm diameter
  * Standard soup bowl: 500-600 ml capacity
  * Standard portion sizes:
    - Rice/pasta: 150-200 г
    - Meat/fish: 100-150 г
    - Vegetables: 100-150 г
    - Soup: 300-400 г
    - Salad: 100-150 г
- Consider the depth/height of the food
- Account for any visible serving dishes or containers
- If multiple items, estimate each separately and sum them
- Round to the nearest 10 grams
- Return ONLY a number representing the total weight in grams
- Do not include any text, units, or explanations

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
- Consider standard portion sizes and plate sizes
- Account for the plate/bowl size if visible
- Use these reference points for common items:
  * Standard dinner plate: 27-30 cm diameter
  * Standard soup bowl: 500-600 ml capacity
  * Standard portion sizes:
    - Rice/pasta: 150-200 г
    - Meat/fish: 100-150 г
    - Vegetables: 100-150 г
    - Soup: 300-400 г
    - Salad: 100-150 г
- Consider the depth/height of the food
- Account for any visible serving dishes or containers
- If multiple items, estimate each separately and sum them
- Round to the nearest 10 grams
- Return ONLY a number representing the total weight in grams
- Do not include any text, units, or explanations

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
4. If no weight is provided, estimate the total weight of the food
5. Provide the information in a specific JSON format

CRITICAL MATH CONSISTENCY REQUIREMENTS:
- The total carbs value MUST be exactly equal to the sum of all component carbs
- Follow these steps in order:
  1. Calculate carbs for each component
  2. Sum up all component carbs
  3. Use that exact sum as the total carbs value
  4. Do not round or adjust the total
  5. Double-check that total equals sum of components
- Example calculation:
  * Component 1: 10g carbs
  * Component 2: 25g carbs
  * Component 3: 5g carbs
  * Total MUST be: 10 + 25 + 5 = 40g
- If you need to round:
  * Round each component first
  * Then sum the rounded values
  * Use that sum as the total
- VALIDATION:
  * After calculating, verify that total = sum of components
  * If they don't match, recalculate until they do
  * Never proceed with mismatched values

CRITICAL CONFIDENCE ASSESSMENT REQUIREMENTS:
- Set confidence based on these EXACT criteria:
  * HIGH confidence when:
    - Food items are clearly visible and identifiable
    - Portion sizes are clear and measurable
    - No hidden ingredients or sauces
    - Standard preparation methods
    - Weight is provided by user
  * MEDIUM confidence when:
    - Most food items are visible
    - Some portion sizes are approximate
    - Minor hidden ingredients possible
    - Weight is estimated but reasonable
  * LOW confidence when:
    - Food items are partially obscured
    - Portion sizes are very approximate
    - Significant hidden ingredients likely
    - Complex preparation methods
    - Weight is highly uncertain
- Do not default to "low" - assess each case carefully
- Consider the image quality and clarity
- Consider the complexity of the dish
- Consider the presence of nutritional labels

CRITICAL CARBOHYDRATE CALCULATION REQUIREMENTS:
- Use your knowledge of nutritional databases to calculate exact carb content
- Consider ALL possible sources of carbohydrates:
  * Main ingredients
  * Side dishes
  * Sauces and dressings
  * Breading and coatings
  * Hidden ingredients (flour, starch, etc.)
- For each component:
  * Identify the exact food item
  * Determine its weight
  * Calculate carbs based on standard nutritional values
  * Account for cooking methods that affect carb content
- For mixed dishes:
  * Break down into individual components
  * Calculate carbs for each component separately
  * Sum up the total
- When in doubt:
  * Round UP to ensure safety
  * Set confidence to "low"
  * Be explicit about uncertainties
- Double-check all calculations before providing the final result

WEIGHT CONSISTENCY REQUIREMENTS:
- The sum of all component weights MUST equal the total weight
- If user provided weight is %.1f grams:
  * Distribute this weight among components proportionally
  * Ensure the sum of component weights equals %.1f grams
- If estimating weight:
  * Estimate each component's weight
  * Sum up to get total weight
  * Ensure the sum matches the total weight

CRITICAL SAFETY REQUIREMENTS:
- For meat/fish with breading (котлеты):
  * Account for breading (панировка) which adds significant carbs
  * Include flour/breadcrumbs in carb calculation
  * Set confidence to "low" if breading amount is uncertain
- For starchy sides (картофельное пюре, рис, etc.):
  * Use exact nutritional values from database
  * Account for cooking method (boiled, mashed, etc.)
  * Include any added ingredients (milk, butter, etc.)
- For vegetables:
  * Most fresh vegetables have very low carb content
  * Pickled vegetables (соленые огурцы) have almost no carbs
  * Only count significant carb sources
- For sauces and dressings:
  * Include any flour, starch, or sugar
  * Account for thickening agents
  * Consider portion size

REQUIREMENTS:
- Be medically precise in your carbohydrate estimation
- Include both visible ingredients and likely hidden ingredients that contain carbs
- Consider portion sizes carefully
- Account for various cooking methods that might affect carbohydrate content
- If the image contains nutritional information or packaging, prioritize that data
- IMPORTANT: Provide all text responses in Russian language for Russian users
- Food names should be in Russian
- Reasoning/descriptions should be in Russian

ANALYSIS TEXT REQUIREMENTS:
- Keep the analysis text VERY concise and structured
- Use this exact format:
  "1. [Food item 1]: [weight] г, [carbs] г углеводов
   2. [Food item 2]: [weight] г, [carbs] г углеводов
   ..."
- Include only the main components that contribute to carbs
- Skip any ingredients with negligible carb content
- Do not include explanations or additional text
- Maximum 3-4 main components
- Total length should not exceed 200 characters

IMPORTANT WEIGHT INFORMATION:
- The user has specified that the food weighs %.1f grams
- If the weight is 0, you must estimate the total weight of the food
- Adjust your carbohydrate calculation based on the provided or estimated weight

CRITICAL JSON FORMAT REQUIREMENTS:
- Your response MUST be a valid JSON object
- Do not include any markdown formatting, bullet points, or dashes
- Do not include any explanatory text before or after the JSON
- The JSON must have these exact fields:
  {
    "food_items": ["item1", "item2"],
    "carbs": 123.45,
    "confidence": "low|medium|high",
    "analysis_text": "Your analysis in Russian",
    "weight": 350.0
  }`, weight, weight, weight)

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
4. If no weight is provided, estimate the total weight of the food
5. Provide the information in a specific JSON format

CRITICAL MATH CONSISTENCY REQUIREMENTS:
- The total carbs value MUST be exactly equal to the sum of all component carbs
- Follow these steps in order:
  1. Calculate carbs for each component
  2. Sum up all component carbs
  3. Use that exact sum as the total carbs value
  4. Do not round or adjust the total
  5. Double-check that total equals sum of components
- Example calculation:
  * Component 1: 10g carbs
  * Component 2: 25g carbs
  * Component 3: 5g carbs
  * Total MUST be: 10 + 25 + 5 = 40g
- If you need to round:
  * Round each component first
  * Then sum the rounded values
  * Use that sum as the total
- VALIDATION:
  * After calculating, verify that total = sum of components
  * If they don't match, recalculate until they do
  * Never proceed with mismatched values

CRITICAL CONFIDENCE ASSESSMENT REQUIREMENTS:
- Set confidence based on these EXACT criteria:
  * HIGH confidence when:
    - Food items are clearly visible and identifiable
    - Portion sizes are clear and measurable
    - No hidden ingredients or sauces
    - Standard preparation methods
    - Weight is provided by user
  * MEDIUM confidence when:
    - Most food items are visible
    - Some portion sizes are approximate
    - Minor hidden ingredients possible
    - Weight is estimated but reasonable
  * LOW confidence when:
    - Food items are partially obscured
    - Portion sizes are very approximate
    - Significant hidden ingredients likely
    - Complex preparation methods
    - Weight is highly uncertain
- Do not default to "low" - assess each case carefully
- Consider the image quality and clarity
- Consider the complexity of the dish
- Consider the presence of nutritional labels

CRITICAL CARBOHYDRATE CALCULATION REQUIREMENTS:
- Use your knowledge of nutritional databases to calculate exact carb content
- Consider ALL possible sources of carbohydrates:
  * Main ingredients
  * Side dishes
  * Sauces and dressings
  * Breading and coatings
  * Hidden ingredients (flour, starch, etc.)
- For each component:
  * Identify the exact food item
  * Determine its weight
  * Calculate carbs based on standard nutritional values
  * Account for cooking methods that affect carb content
- For mixed dishes:
  * Break down into individual components
  * Calculate carbs for each component separately
  * Sum up the total
- When in doubt:
  * Round UP to ensure safety
  * Set confidence to "low"
  * Be explicit about uncertainties
- Double-check all calculations before providing the final result

WEIGHT CONSISTENCY REQUIREMENTS:
- The sum of all component weights MUST equal the total weight
- If user provided weight is %.1f grams:
  * Distribute this weight among components proportionally
  * Ensure the sum of component weights equals %.1f grams
- If estimating weight:
  * Estimate each component's weight
  * Sum up to get total weight
  * Ensure the sum matches the total weight

CRITICAL SAFETY REQUIREMENTS:
- For meat/fish with breading (котлеты):
  * Account for breading (панировка) which adds significant carbs
  * Include flour/breadcrumbs in carb calculation
  * Set confidence to "low" if breading amount is uncertain
- For starchy sides (картофельное пюре, рис, etc.):
  * Use exact nutritional values from database
  * Account for cooking method (boiled, mashed, etc.)
  * Include any added ingredients (milk, butter, etc.)
- For vegetables:
  * Most fresh vegetables have very low carb content
  * Pickled vegetables (соленые огурцы) have almost no carbs
  * Only count significant carb sources
- For sauces and dressings:
  * Include any flour, starch, or sugar
  * Account for thickening agents
  * Consider portion size

REQUIREMENTS:
- Be medically precise in your carbohydrate estimation
- Include both visible ingredients and likely hidden ingredients that contain carbs
- Consider portion sizes carefully
- Account for various cooking methods that might affect carbohydrate content
- If the image contains nutritional information or packaging, prioritize that data
- IMPORTANT: Provide all text responses in Russian language for Russian users
- Food names should be in Russian
- Reasoning/descriptions should be in Russian

ANALYSIS TEXT REQUIREMENTS:
- Keep the analysis text VERY concise and structured
- Use this exact format:
  "1. [Food item 1]: [weight] г, [carbs] г углеводов
   2. [Food item 2]: [weight] г, [carbs] г углеводов
   ..."
- Include only the main components that contribute to carbs
- Skip any ingredients with negligible carb content
- Do not include explanations or additional text
- Maximum 3-4 main components
- Total length should not exceed 200 characters

IMPORTANT WEIGHT INFORMATION:
- The user has specified that the food weighs %.1f grams
- If the weight is 0, you must estimate the total weight of the food
- Adjust your carbohydrate calculation based on the provided or estimated weight

CRITICAL JSON FORMAT REQUIREMENTS:
- Your response MUST be a valid JSON object
- Do not include any markdown formatting, bullet points, or dashes
- Do not include any explanatory text before or after the JSON
- The JSON must have these exact fields:
  {
    "food_items": ["item1", "item2"],
    "carbs": 123.45,
    "confidence": "low|medium|high",
    "analysis_text": "Your analysis in Russian",
    "weight": 350.0
  }`, weight, weight, weight)

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
