package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog with application-specific functionality
type Logger struct {
	logger zerolog.Logger
}

// Fields represents structured logging fields
type Fields map[string]interface{}

// Config represents logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Environment string // development, staging, production
	Output      io.Writer
}

// New creates a new logger instance
func New(cfg Config) *Logger {
	// Set log level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure output format based on environment
	var output io.Writer = cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Pretty print for development, JSON for production
	if cfg.Environment == "development" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	// Create logger with common fields
	logger := zerolog.New(output).
		With().
		Timestamp().
		Str("service", "keystone").
		Str("environment", cfg.Environment).
		Logger()

	return &Logger{logger: logger}
}

// Default returns the global logger instance
func Default() *Logger {
	return &Logger{logger: log.Logger}
}

// WithContext adds logger to context
func (l *Logger) WithContext(ctx context.Context) context.Context {
	return l.logger.WithContext(ctx)
}

// FromContext retrieves logger from context
func FromContext(ctx context.Context) *Logger {
	return &Logger{logger: zerolog.Ctx(ctx).With().Logger()}
}

// WithTenant adds tenant_id to all subsequent logs
func (l *Logger) WithTenant(tenantID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("tenant_id", tenantID).Logger(),
	}
}

// WithUser adds user_id to all subsequent logs
func (l *Logger) WithUser(userID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("user_id", userID).Logger(),
	}
}

// WithCorrelation adds correlation_id for request tracing
func (l *Logger) WithCorrelation(correlationID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("correlation_id", correlationID).Logger(),
	}
}

// WithFields adds multiple fields at once
func (l *Logger) WithFields(fields Fields) *Logger {
	ctx := l.logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	return &Logger{logger: ctx.Logger()}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// ErrorWithErr logs an error with error details
func (l *Logger) ErrorWithErr(err error, msg string) {
	l.logger.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// HTTPRequest logs HTTP request details (for middleware)
func (l *Logger) HTTPRequest(method, path string, statusCode, latencyMs int) {
	l.logger.Info().
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Int("latency_ms", latencyMs).
		Msg("HTTP request")
}

// HTTPRequestWithFields logs HTTP request with additional fields
func (l *Logger) HTTPRequestWithFields(method, path string, statusCode, latencyMs int, fields Fields) {
	event := l.logger.Info().
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Int("latency_ms", latencyMs)

	for key, value := range fields {
		event = event.Interface(key, value)
	}

	event.Msg("HTTP request")
}

// DatabaseQuery logs database query details
func (l *Logger) DatabaseQuery(query string, durationMs int) {
	l.logger.Debug().
		Str("query", query).
		Int("duration_ms", durationMs).
		Msg("Database query executed")
}

// AuditLog logs security-sensitive actions
func (l *Logger) AuditLog(action, resource string, success bool, metadata Fields) {
	event := l.logger.Info().
		Str("action", action).
		Str("resource", resource).
		Bool("success", success).
		Str("log_type", "audit")

	for key, value := range metadata {
		event = event.Interface(key, value)
	}

	event.Msg("Audit log")
}

// SecurityEvent logs security-related events
func (l *Logger) SecurityEvent(eventType, description string, severity string) {
	l.logger.Warn().
		Str("event_type", eventType).
		Str("description", description).
		Str("severity", severity).
		Str("log_type", "security").
		Msg("Security event")
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
