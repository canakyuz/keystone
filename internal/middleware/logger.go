package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"nexspaces-api/pkg/logger"
)

// Logger creates a request logging middleware
func Logger(log *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Generate correlation ID
		correlationID := c.Get("X-Request-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set("X-Request-ID", correlationID)

		// Store correlation ID in context
		c.Locals("correlation_id", correlationID)

		// Create request-scoped logger
		reqLogger := log.WithCorrelation(correlationID)

		// Get tenant and user from context (if authenticated)
		if tenantID := GetTenantID(c); tenantID != "" {
			reqLogger = reqLogger.WithTenant(tenantID)
		}
		if userID := GetUserID(c); userID != "" {
			reqLogger = reqLogger.WithUser(userID)
		}

		// Store logger in context
		c.Locals("logger", reqLogger)

		// Process request
		err := c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Build log fields
		fields := logger.Fields{
			"method":         c.Method(),
			"path":           c.Path(),
			"status":         statusCode,
			"latency_ms":     latency.Milliseconds(),
			"ip":             c.IP(),
			"user_agent":     c.Get("User-Agent"),
			"correlation_id": correlationID,
		}

		// Add tenant and user if available
		if tenantID := GetTenantID(c); tenantID != "" {
			fields["tenant_id"] = tenantID
		}
		if userID := GetUserID(c); userID != "" {
			fields["user_id"] = userID
		}

		// Log based on status code
		if err != nil {
			reqLogger.WithFields(fields).ErrorWithErr(err, "Request failed")
		} else if statusCode >= 500 {
			reqLogger.WithFields(fields).Error("Server error")
		} else if statusCode >= 400 {
			reqLogger.WithFields(fields).Warn("Client error")
		} else {
			reqLogger.WithFields(fields).Info("Request completed")
		}

		return err
	}
}

// GetLogger retrieves logger from context
func GetLogger(c *fiber.Ctx) *logger.Logger {
	if log := c.Locals("logger"); log != nil {
		return log.(*logger.Logger)
	}
	return logger.Default()
}
