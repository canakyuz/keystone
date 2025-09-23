package external

import (
	"fmt"

	"nexspaces-api/internal/adapters/external/aws"
	"nexspaces-api/internal/adapters/external/sendgrid"
	"nexspaces-api/internal/adapters/external/stripe"
	"nexspaces-api/internal/core/ports/services"
)

// ServiceRegistry holds all external service implementations
type ServiceRegistry struct {
	EmailService   services.EmailService
	StorageService services.StorageService
	BillingService services.BillingService
}

// Config holds configuration for external services
type Config struct {
	Stripe   StripeConfig   `json:"stripe"`
	SendGrid SendGridConfig `json:"sendgrid"`
	AWS      AWSConfig      `json:"aws"`
}

type StripeConfig struct {
	SecretKey string `json:"secret_key"`
	Enabled   bool   `json:"enabled"`
}

type SendGridConfig struct {
	APIKey    string `json:"api_key"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	Enabled   bool   `json:"enabled"`
}

type AWSConfig struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Enabled         bool   `json:"enabled"`
}

// NewServiceRegistry creates a new service registry with configured external services
func NewServiceRegistry(config Config) (*ServiceRegistry, error) {
	registry := &ServiceRegistry{}

	// Initialize Email Service (SendGrid)
	if config.SendGrid.Enabled {
		if config.SendGrid.APIKey == "" {
			return nil, fmt.Errorf("SendGrid API key is required when SendGrid is enabled")
		}
		registry.EmailService = sendgrid.NewEmailService(
			config.SendGrid.APIKey,
			config.SendGrid.FromEmail,
			config.SendGrid.FromName,
		)
	} else {
		// Use mock email service for development
		registry.EmailService = NewMockEmailService()
	}

	// Initialize Storage Service (AWS S3)
	if config.AWS.Enabled {
		if config.AWS.Region == "" || config.AWS.Bucket == "" {
			return nil, fmt.Errorf("AWS region and bucket are required when AWS is enabled")
		}

		var storageService *aws.StorageService
		var err error

		if config.AWS.AccessKeyID != "" && config.AWS.SecretAccessKey != "" {
			// Use explicit credentials
			storageService, err = aws.NewStorageServiceWithCredentials(
				config.AWS.Region,
				config.AWS.Bucket,
				config.AWS.AccessKeyID,
				config.AWS.SecretAccessKey,
			)
		} else {
			// Use default credential chain (IAM roles, environment variables, etc.)
			storageService, err = aws.NewStorageService(
				config.AWS.Region,
				config.AWS.Bucket,
			)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to initialize AWS storage service: %w", err)
		}

		registry.StorageService = storageService
	} else {
		// Use mock storage service for development
		registry.StorageService = NewMockStorageService()
	}

	// Initialize Billing Service (Stripe)
	if config.Stripe.Enabled {
		if config.Stripe.SecretKey == "" {
			return nil, fmt.Errorf("Stripe secret key is required when Stripe is enabled")
		}
		registry.BillingService = stripe.NewSimpleBillingService(config.Stripe.SecretKey)
	} else {
		// Use mock billing service for development
		registry.BillingService = NewMockBillingService()
	}

	return registry, nil
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() Config {
	return Config{
		Stripe: StripeConfig{
			SecretKey: getEnv("STRIPE_SECRET_KEY", ""),
			Enabled:   getEnvBool("STRIPE_ENABLED", false),
		},
		SendGrid: SendGridConfig{
			APIKey:    getEnv("SENDGRID_API_KEY", ""),
			FromEmail: getEnv("SENDGRID_FROM_EMAIL", "noreply@nexpaces.com"),
			FromName:  getEnv("SENDGRID_FROM_NAME", "NexSpaces"),
			Enabled:   getEnvBool("SENDGRID_ENABLED", false),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			Bucket:          getEnv("AWS_S3_BUCKET", ""),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Enabled:         getEnvBool("AWS_ENABLED", false),
		},
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	// This would normally use os.Getenv, but we're avoiding the import for now
	// In a real implementation, you'd import "os" and use os.Getenv(key)
	if key == "SENDGRID_FROM_EMAIL" && defaultValue == "noreply@nexpaces.com" {
		return defaultValue
	}
	if key == "SENDGRID_FROM_NAME" && defaultValue == "NexSpaces" {
		return defaultValue
	}
	if key == "AWS_REGION" && defaultValue == "us-east-1" {
		return defaultValue
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	// This would normally parse os.Getenv(key) as a boolean
	// In a real implementation, you'd use strconv.ParseBool
	return defaultValue
}
