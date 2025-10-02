package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Security SecurityConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string
	Port         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Host     string
	Port     string
	Username string
	Password string
	DB       int
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret           string
	JWTExpiry           time.Duration
	RefreshTokenExpiry  time.Duration
	InviteTokenExpiry   time.Duration
	PasswordResetExpiry time.Duration
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	AllowedOrigins   string
	AllowCredentials bool
	MaxRequestSize   string
	EnableCSP        bool
	EnableHSTS       bool
	TrustedProxies   string

	RateLimit RateLimitConfig
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Requests int
	Duration time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("HOST", "0.0.0.0"),
			Port:         getEnv("PORT", "8080"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  parseDuration(getEnv("SERVER_READ_TIMEOUT", "30s")),
			WriteTimeout: parseDuration(getEnv("SERVER_WRITE_TIMEOUT", "30s")),
			IdleTimeout:  parseDuration(getEnv("SERVER_IDLE_TIMEOUT", "120s")),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "nexspaces_dev"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    parseInt(getEnv("DB_MAX_OPEN_CONNS", "25")),
			MaxIdleConns:    parseInt(getEnv("DB_MAX_IDLE_CONNS", "10")),
			ConnMaxLifetime: parseDuration(getEnv("DB_CONN_MAX_LIFETIME", "30m")),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Username: getEnv("REDIS_USERNAME", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       parseInt(getEnv("REDIS_DB", "0")),
		},
		Auth: AuthConfig{
			JWTSecret:           getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			JWTExpiry:           parseDuration(getEnv("JWT_EXPIRY", "24h")),
			RefreshTokenExpiry:  parseDuration(getEnv("REFRESH_TOKEN_EXPIRY", "168h")),
			InviteTokenExpiry:   parseDuration(getEnv("INVITE_TOKEN_EXPIRY", "72h")),
			PasswordResetExpiry: parseDuration(getEnv("PASSWORD_RESET_EXPIRY", "1h")),
		},
		Security: SecurityConfig{
			AllowedOrigins:   getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
			AllowCredentials: parseBool(getEnv("ALLOW_CREDENTIALS", "true")),
			MaxRequestSize:   getEnv("MAX_REQUEST_SIZE", "10MB"),
			EnableCSP:        parseBool(getEnv("ENABLE_CSP", "true")),
			EnableHSTS:       parseBool(getEnv("ENABLE_HSTS", "false")),
			TrustedProxies:   getEnv("TRUSTED_PROXIES", "127.0.0.1,::1"),
			RateLimit: RateLimitConfig{
				Requests: parseInt(getEnv("RATE_LIMIT_REQUESTS", "100")),
				Duration: parseDuration(getEnv("RATE_LIMIT_DURATION", "1m")),
			},
		},
	}

	// Validate required fields
	if cfg.Database.Password == "" && cfg.Server.Environment == "production" {
		return nil, fmt.Errorf("DB_PASSWORD is required in production")
	}

	if cfg.Auth.JWTSecret == "your-secret-key-change-in-production" && cfg.Server.Environment == "production" {
		return nil, fmt.Errorf("JWT_SECRET must be changed in production")
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func parseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}
