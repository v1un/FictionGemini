package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"workspace/FictionGeminiRewritten/internal/ai"
	"workspace/FictionGeminiRewritten/internal/handlers"
	"workspace/FictionGeminiRewritten/internal/services"

	// Assuming genai and option are correctly vendored or in GOPATH
	// "github.com/google/generative-ai-go/genai"
	// "google.golang.org/api/option"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatalf("GEMINI_API_KEY environment variable not set. The application cannot start.")
	}

	ctx := context.Background()

	// Initialize AI Client (once)
	geminiClient, err := ai.NewGeminiClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini AI client: %v", err)
	}
	defer geminiClient.Close() // Ensure client is closed when main exits

	// Initialize Services
	orchestratorSvc := services.NewOrchestratorService(geminiClient)

	// Initialize Handlers
	generateHandler := handlers.NewGenerateHandler(orchestratorSvc)

	// Setup Router
	mux := http.NewServeMux()
	mux.Handle("/generate", generateHandler)

	// Root handler for basic check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status": "FictionGeminiRewritten server is running"}`)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second, // Increased to allow for long AI generation times
		IdleTimeout:  150 * time.Second,
	}

	log.Printf("FictionGeminiRewritten server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", port, err)
	}

	log.Println("Server stopped")
}

