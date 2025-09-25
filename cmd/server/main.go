package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	_ "nexspaces-api/docs"
	"nexspaces-api/internal/app"
	"nexspaces-api/internal/config"
)

// @title NexSpaces API
// @version 1.0
// @description This is a production-ready multi-tenant SaaS backend built with modern Clean Architecture + DDD patterns.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url https://www.nexspaces.com/support
// @contact.email support@nexspaces.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
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
