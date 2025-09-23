package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	JWT      JWTConfig      `json:"jwt"`
	Security SecurityConfig `json:"security"`
}

type ServerConfig struct {
	Host         string        `json:"host"`
	Port         string        `json:"port"`
	Environment  string        `json:"environment"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            string        `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	Name            string        `json:"name"`
	SSLMode         string        `json:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

type RedisConfig struct {
	URL      string `json:"url"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type JWTConfig struct {
	Secret              string        `json:"secret"`
	Expiry              time.Duration `json:"expiry"`
	RefreshTokenExpiry  time.Duration `json:"refresh_token_expiry"`
	InviteTokenExpiry   time.Duration `json:"invite_token_expiry"`
	PasswordResetExpiry time.Duration `json:"password_reset_expiry"`
}

type SecurityConfig struct {
	AllowedOrigins   string    `json:"allowed_origins"`
	AllowCredentials bool      `json:"allow_credentials"`
	MaxRequestSize   string    `json:"max_request_size"`
	EnableCSP        bool      `json:"enable_csp"`
	EnableHSTS       bool      `json:"enable_hsts"`
	TrustedProxies   string    `json:"trusted_proxies"`
	RateLimit        RateLimit `json:"rate_limit"`
}

type RateLimit struct {
	Requests int           `json:"requests"`
	Duration time.Duration `json:"duration"`
}

func Load() *Config {
	config := &Config{
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
			Password:        getEnv("DB_PASSWORD", ""),
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
		JWT: JWTConfig{
			Secret:              getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiry:              parseDuration(getEnv("JWT_EXPIRY", "24h")),
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
			RateLimit: RateLimit{
				Requests: parseInt(getEnv("RATE_LIMIT_REQUESTS", "100")),
				Duration: parseDuration(getEnv("RATE_LIMIT_DURATION", "1m")),
			},
		},
	}

	return config
}

func (c *Config) Validate() error {
	// Validate required fields
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.JWT.Secret == "" || c.JWT.Secret == "your-secret-key-change-in-production" {
		if c.Server.Environment == "production" {
			return fmt.Errorf("JWT secret must be set for production")
		}
	}

	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// Validate Redis configuration
	if c.Redis.URL == "" && c.Redis.Host == "" {
		return fmt.Errorf("either REDIS_URL or REDIS_HOST must be set")
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
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
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

func parseBool(s string) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return false
}

func parseDuration(s string) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return 0
}
