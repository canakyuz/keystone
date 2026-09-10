package main

import (
	"log"

	"github.com/canakyuz/keystone/internal/app"
	"github.com/canakyuz/keystone/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (optional in production with env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Initialize application
	application, err := app.NewApplication(cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	// Start server
	if err := application.Start(); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
