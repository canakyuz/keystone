package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port         int
	Host         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // in minutes
}

type RedisConfig struct {
	URL      string
	Host     string
	Port     int
	Username string
	Password string
	Database int
}

type AuthConfig struct {
	JWTSecret           string
	JWTExpiry           time.Duration
	RefreshTokenExpiry  time.Duration
	InviteTokenExpiry   time.Duration
	PasswordResetExpiry time.Duration
}

type SecurityConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxRequestSize   string
	EnableCSP        bool
	EnableHSTS       bool
	TrustedProxies   []string
	RateLimit        RateLimitConfig
}

type RateLimitConfig struct {
	Requests int
	Duration int // in minutes
}

func Load() *Config {
	// Parse numeric values
	serverPort, _ := strconv.Atoi(getEnv("PORT", "8080"))
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	connMaxLifetime, _ := strconv.Atoi(getEnv("DB_CONN_MAX_LIFETIME", "30"))

	rateLimitRequests, _ := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "100"))
	rateLimitDuration, _ := strconv.Atoi(getEnv("RATE_LIMIT_DURATION", "1"))

	// Parse durations
	jwtExpiry, _ := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))
	refreshExpiry, _ := time.ParseDuration(getEnv("REFRESH_TOKEN_EXPIRY", "168h")) // 7 days
	inviteExpiry, _ := time.ParseDuration(getEnv("INVITE_TOKEN_EXPIRY", "72h"))    // 3 days
	passwordResetExpiry, _ := time.ParseDuration(getEnv("PASSWORD_RESET_EXPIRY", "1h"))

	// Parse server timeouts
	readTimeout, _ := time.ParseDuration(getEnv("SERVER_READ_TIMEOUT", "30s"))
	writeTimeout, _ := time.ParseDuration(getEnv("SERVER_WRITE_TIMEOUT", "30s"))
	idleTimeout, _ := time.ParseDuration(getEnv("SERVER_IDLE_TIMEOUT", "120s"))

	// Parse arrays
	allowedOrigins := parseStringSlice(getEnv("ALLOWED_ORIGINS", "*"))
	trustedProxies := parseStringSlice(getEnv("TRUSTED_PROXIES", ""))

	// Parse booleans
	allowCredentials := getEnv("ALLOW_CREDENTIALS", "true") == "true"
	enableCSP := getEnv("ENABLE_CSP", "true") == "true"
	enableHSTS := getEnv("ENABLE_HSTS", "false") == "true"

	return &Config{
		Server: ServerConfig{
			Port:         serverPort,
			Host:         getEnv("HOST", "0.0.0.0"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            dbPort,
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "password"),
			Name:            getEnv("DB_NAME", "nexspaces_dev"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     redisPort,
			Username: getEnv("REDIS_USERNAME", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			Database: redisDB,
		},
		Auth: AuthConfig{
			JWTSecret:           getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			JWTExpiry:           jwtExpiry,
			RefreshTokenExpiry:  refreshExpiry,
			InviteTokenExpiry:   inviteExpiry,
			PasswordResetExpiry: passwordResetExpiry,
		},
		Security: SecurityConfig{
			AllowedOrigins:   allowedOrigins,
			AllowCredentials: allowCredentials,
			MaxRequestSize:   getEnv("MAX_REQUEST_SIZE", "10MB"),
			EnableCSP:        enableCSP,
			EnableHSTS:       enableHSTS,
			TrustedProxies:   trustedProxies,
			RateLimit: RateLimitConfig{
				Requests: rateLimitRequests,
				Duration: rateLimitDuration,
			},
		},
	}
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

func (r *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// parseStringSlice parses a comma-separated string into a slice
func parseStringSlice(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	var result []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Auth.JWTSecret == "your-secret-key-change-in-production" && c.Server.Environment == "production" {
		return fmt.Errorf("JWT secret must be changed in production")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

// IsProduction checks if the environment is production
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment checks if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}
