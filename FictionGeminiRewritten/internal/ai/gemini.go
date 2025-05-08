package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// CallGeminiAPI sends a prompt to the Gemini API using the provided API key and model name.
// It creates a new client for each call to ensure the correct API key is used.
func CallGeminiAPI(ctx context.Context, apiKey string, modelName string, prompt string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("API key is required for CallGeminiAPI")
	}
	if modelName == "" {
		return "", fmt.Errorf("model name is required for CallGeminiAPI")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing temporary Gemini client: %v", err)
		}
	}()

	model := client.GenerativeModel(modelName)
	// model.GenerationConfig.ResponseMIMEType = "application/json" // If needed globally

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content using model %s: %w", modelName, err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		// Check for safety ratings / finish reason if content is empty
		if resp != nil && len(resp.Candidates) > 0 && resp.Candidates[0].FinishReason != genai.FinishReasonStop {
			return "", fmt.Errorf("AI content generation stopped due to %s. Safety ratings: %v", resp.Candidates[0].FinishReason, resp.Candidates[0].SafetyRatings)
		}
		// It's possible to get an empty Parts array with FinishReasonStop if the prompt itself was empty or invalid.
		finishReason := genai.FinishReasonUnspecified
		if resp != nil && len(resp.Candidates) > 0 {
			finishReason = resp.Candidates[0].FinishReason
		}
		return "", fmt.Errorf("no content received from AI for model %s: empty response or parts. FinishReason: %s", modelName, finishReason)
	}
	
	responsePart := resp.Candidates[0].Content.Parts[0]
	if responseText, ok := responsePart.(genai.Text); ok {
		return string(responseText), nil
	}

	return "", fmt.Errorf("unexpected response part type: %T for model %s", responsePart, modelName)
}
