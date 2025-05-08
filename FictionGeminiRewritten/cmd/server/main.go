package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"workspace/FictionGeminiRewritten/internal/handlers"
	"workspace/FictionGeminiRewritten/internal/services"

	// Assuming genai and option are correctly vendored or in GOPATH
	// "github.com/google/generative-ai-go/genai"
	// "google.golang.org/api/option"
)


// CORS Middleware
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, API-Key")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Next
		next.ServeHTTP(w, r)
	})
}

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}


	// Initialize Services
	orchestratorSvc := services.NewOrchestratorService()

	// Initialize Handlers
	generateHandler := handlers.NewGenerateHandler(orchestratorSvc)

	// Setup Router
	mux := http.NewServeMux()
	mux.Handle("/generate", enableCORS(generateHandler)) // THIS LINE IS MODIFIED

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

