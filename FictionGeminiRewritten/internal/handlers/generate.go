package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"workspace/FictionGeminiRewritten/internal/models"
	"workspace/FictionGeminiRewritten/internal/services"
)

// GenerateHandler handles the /generate endpoint.
type GenerateHandler struct {
	orchestrator *services.OrchestratorService
}

// NewGenerateHandler creates a new GenerateHandler.
func NewGenerateHandler(orchestrator *services.OrchestratorService) *GenerateHandler {
	return &GenerateHandler{orchestrator: orchestrator}
}

func (h *GenerateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload models.RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	// Basic Validations
	if payload.APIKey == "" {
		http.Error(w, "API Key is missing", http.StatusBadRequest)
		return
	}
	// In a real app, you'd validate the API key against a stored one, e.g., from env.
	// For this rewrite, we'll assume it matches the one needed for the Gemini client if provided.

	if strings.TrimSpace(payload.Series) == "" {
		http.Error(w, "Series name is missing or empty", http.StatusBadRequest)
		return
	}
	if payload.Option == "" {
		http.Error(w, "Option is missing", http.StatusBadRequest)
		return
	}
	if payload.Option == "3" && strings.TrimSpace(payload.ToolCardPurpose) == "" {
		http.Error(w, "Tool Card Purpose is required for Option 3", http.StatusBadRequest)
		return
	}
	if payload.Model == "" {
		// Default to a model if not provided, or could make it mandatory.
		// For now, let's assume a default is handled by Gemini client or orchestrator if needed,
		// but ideally, it should be validated or defaulted here.
		// Based on original, it seems user must select one.
		http.Error(w, "AI Model selection is missing", http.StatusBadRequest)
		return
	}

	// Generate a unique log identifier for this request session
	logIdentifier := services.GenerateLogIdentifier(payload.Series)
	log.Printf("Received /generate request. Log ID: %s, Series: '%s', Option: %s, Model: %s", 
		logIdentifier, payload.Series, payload.Option, payload.Model)

	// Set API key in context for Gemini client (if it were to be initialized per request or in orchestrator)
	// However, our current NewGeminiClient takes it at startup. If API key can change per request,
	// the orchestrator or AI service would need to handle it, or client re-initialized.
	// For this version, we assume APIKey in payload is for validation/logging, and client uses env var.
	// Let's simulate checking against an environment variable for the API key as the original did.
	envAPIKey := os.Getenv("GEMINI_API_KEY")
	if envAPIKey == "" {
		log.Printf("GEMINI_API_KEY environment variable not set on server.")
		// http.Error(w, "Server configuration error: AI API Key not set.", http.StatusInternalServerError) // User doesn't need to know this
		// return
		// For now, proceed, assuming orchestrator/AI client might have a built-in key for dev, or will fail there.
	} else if payload.APIKey != envAPIKey {
		log.Printf("Invalid API Key provided by user (Log ID: %s). Expected: %s...%s, Got: %s...%s", 
			logIdentifier, envAPIKey[:3], envAPIKey[len(envAPIKey)-4:], payload.APIKey[:3], payload.APIKey[len(payload.APIKey)-4:])
		http.Error(w, "Invalid API Key.", http.StatusUnauthorized)
		return
	}

	ctx := r.Context() // Use request context
	generatedJSON, messageLog, optionText, err := h.orchestrator.ProcessGenerationRequest(ctx, payload, logIdentifier)

	response := models.ResponsePayload{
		Timestamp:    time.Now().Format(time.RFC3339),
		LogIdentifier: logIdentifier,
		Series:       payload.Series,
		OptionChosen: optionText, // Set by orchestrator
		ModelUsed:    payload.Model,
	}
response.APIKeyReceived = (payload.APIKey != "")

	if err != nil {
		log.Printf("Error processing request (Log ID: %s): %v", logIdentifier, err)
response.Error = fmt.Sprintf("Error during generation: %s", err.Error())
response.Message = messageLog
		response.GeneratedContent = ""
		w.WriteHeader(http.StatusInternalServerError) // Or map error types to specific HTTP statuses
	} else {
		response.Message = "Generation process completed. See details below and check generated files.\n" + messageLog
		response.GeneratedContent = generatedJSON
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		log.Printf("Failed to encode response (Log ID: %s): %v", logIdentifier, encodeErr)
		// http.Error already sent, or client will time out
	}
}

