package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"nexspaces-api/internal/app"
	"nexspaces-api/internal/config"
)

func main() {
	// Load environment variables from .env file (if exists)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration validation failed:", err)
	}

	// Create and initialize application
	application, err := app.NewApplication(cfg)
	if err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	// Start the application
	if err := application.Start(); err != nil {
		log.Printf("Application shutdown with error: %v", err)
		os.Exit(1)
	}
}