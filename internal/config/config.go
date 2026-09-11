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
	Payment  PaymentConfig
	Tracing  TracingConfig
	GRPC     GRPCConfig
}

// GRPCConfig configures the typed service surface.
type GRPCConfig struct {
	// Addr is where the gRPC server listens. Empty disables it, so a deployment that has
	// no gRPC clients does not open a port it does not use.
	Addr string

	// Reflection exposes the service descriptors to tools like grpcurl.
	//
	// Off unless explicitly enabled. Reflection hands anyone who can reach the port a
	// complete list of methods and message shapes, which is useful in development and is
	// a map of the attack surface anywhere else.
	Reflection bool
}

// parseFloat reads a ratio from the environment.
//
// A malformed value yields zero, which disables sampling rather than defaulting to
// everything: a typo in a deployment variable should not quietly start exporting every
// span in production.
func parseFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	return parsed
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string
	Port         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// MetricsAddr is where the worker publishes its Prometheus metrics.
	//
	// The API serves /metrics on its own port, alongside the API itself. The worker has
	// no HTTP server of its own, so it needs an address to bind one. Empty disables the
	// listener entirely, which is what the tests want.
	//
	// It defaults to a loopback address rather than 0.0.0.0: metrics are unauthenticated
	// by convention, so the default should not be reachable from outside the host.
	MetricsAddr string
}

// TracingConfig configures distributed tracing.
//
// Tracing is off unless an endpoint is set. Off means a no-op tracer rather than a
// branch at every call site, so the traced and untraced builds run the same code.
type TracingConfig struct {
	// Endpoint is the OTLP/HTTP collector address. Empty disables tracing.
	Endpoint string

	// SampleRatio is the head sampling ratio, 0 to 1.
	//
	// It defaults to a fraction rather than 1. Sampling everything is affordable in
	// development and is not in production, and a default that only works in development
	// is the kind that reaches production unnoticed.
	SampleRatio float64
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	// WorkerUser and WorkerPassword are the worker's own credentials; see WorkerDatabase.
	WorkerUser     string
	WorkerPassword string

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

// PaymentConfig holds payment provider configuration
type PaymentConfig struct {
	Iyzico   IyzicoConfig
	Checkout CheckoutConfig
	Stripe   StripeConfig
}

// IyzicoConfig holds iyzico (Turkey) payment configuration
type IyzicoConfig struct {
	Enabled         bool
	APIKey          string
	SecretKey       string
	BaseURL         string
	ThreeDSCallback string
	Currency        string
}

// CheckoutConfig holds Checkout.com (Global) payment configuration
type CheckoutConfig struct {
	Enabled     bool
	PublicKey   string
	SecretKey   string
	BaseURL     string
	CallbackURL string
	Currency    string
}

// StripeConfig holds Stripe payment configuration
type StripeConfig struct {
	Enabled        bool
	PublishableKey string
	SecretKey      string
	WebhookSecret  string
	Currency       string
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
			MetricsAddr:  getEnv("WORKER_METRICS_ADDR", "127.0.0.1:9091"),
		},
		GRPC: GRPCConfig{
			Addr:       getEnv("GRPC_ADDR", ""),
			Reflection: getEnv("GRPC_REFLECTION", "false") == "true",
		},
		Tracing: TracingConfig{
			Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			SampleRatio: parseFloat(getEnv("OTEL_TRACES_SAMPLER_ARG", "0.1")),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			WorkerUser:      getEnv("WORKER_DB_USER", ""),
			WorkerPassword:  getEnv("WORKER_DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "keystone_dev"),
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
		Payment: PaymentConfig{
			Iyzico: IyzicoConfig{
				Enabled:         parseBool(getEnv("IYZICO_ENABLED", "false")),
				APIKey:          getEnv("IYZICO_API_KEY", ""),
				SecretKey:       getEnv("IYZICO_SECRET_KEY", ""),
				BaseURL:         getEnv("IYZICO_BASE_URL", "https://api.iyzipay.com"),
				ThreeDSCallback: getEnv("IYZICO_3DS_CALLBACK_URL", ""),
				Currency:        getEnv("IYZICO_CURRENCY", "TRY"),
			},
			Checkout: CheckoutConfig{
				Enabled:     parseBool(getEnv("CHECKOUT_ENABLED", "false")),
				PublicKey:   getEnv("CHECKOUT_PUBLIC_KEY", ""),
				SecretKey:   getEnv("CHECKOUT_SECRET_KEY", ""),
				BaseURL:     getEnv("CHECKOUT_BASE_URL", "https://api.checkout.com"),
				CallbackURL: getEnv("CHECKOUT_CALLBACK_URL", ""),
				Currency:    getEnv("CHECKOUT_CURRENCY", "USD"),
			},
			Stripe: StripeConfig{
				Enabled:        parseBool(getEnv("STRIPE_ENABLED", "false")),
				PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
				SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
				WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
				Currency:       getEnv("STRIPE_CURRENCY", "USD"),
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

// WorkerDatabase returns the connection settings the worker uses.
//
// The worker connects as its own role, one holding keystone_worker (migration 039). The
// process that provisions tenants and delivers webhooks then holds neither the API's
// privileges nor the ability to read tenant data, and the API does not hold the worker's.
//
// Outside production an unset worker user falls back to the API's credentials, so a local
// run needs no extra setup. In production that fallback is refused, and so is a worker
// user equal to the API's: a shared role is the boundary not existing.
func (c *Config) WorkerDatabase() (DatabaseConfig, error) {
	db := c.Database

	if c.Server.Environment == "production" && (db.WorkerUser == "" || db.WorkerUser == db.User) {
		return DatabaseConfig{}, fmt.Errorf("WORKER_DB_USER must name a role separate from DB_USER in production")
	}

	if db.WorkerUser != "" {
		db.User = db.WorkerUser
		db.Password = db.WorkerPassword
	}

	return db, nil
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
