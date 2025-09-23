package middleware

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/storage/redis/v3"

	"nexspaces-api/internal/shared/errors"
)

// SecurityConfig contains configuration for security middleware
type SecurityConfig struct {
	AllowedOrigins    []string
	AllowCredentials  bool
	MaxRequestSize    string
	RateLimitRequests int
	RateLimitDuration time.Duration
	RedisURL          string
	JWTSecret         string
	EnableCSP         bool
	EnableHSTS        bool
	TrustedProxies    []string
}

// NewSecurityMiddleware creates and configures security middlewares
func NewSecurityMiddleware(config SecurityConfig) []fiber.Handler {
	var middlewares []fiber.Handler

	// 1. Recovery middleware - should be first
	middlewares = append(middlewares, recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			// Log the error with full stack trace
			// TODO: Add proper logging
			fmt.Printf("Panic recovered: %v\n", e)
		},
	}))

	// 2. Helmet middleware for security headers
	helmetConfig := helmet.Config{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		PermissionsPolicy:  "geolocation=(), microphone=(), camera=()",
	}

	if config.EnableHSTS {
		helmetConfig.HSTSMaxAge = 31536000 // 1 year
		helmetConfig.HSTSIncludeSubdomains = true
		helmetConfig.HSTSPreloadEnabled = true
	}

	if config.EnableCSP {
		helmetConfig.ContentSecurityPolicy = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' https:; connect-src 'self' https:; frame-ancestors 'none';"
	}

	middlewares = append(middlewares, helmet.New(helmetConfig))

	// 3. CORS middleware
	middlewares = append(middlewares, cors.New(cors.Config{
		AllowOrigins:     strings.Join(config.AllowedOrigins, ","),
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-Tenant-ID",
		AllowCredentials: config.AllowCredentials,
		MaxAge:           86400, // 24 hours
	}))

	// 4. Rate limiting middleware
	var storage fiber.Storage
	if config.RedisURL != "" {
		storage = redis.New(redis.Config{
			URL: config.RedisURL,
		})
	}

	middlewares = append(middlewares, limiter.New(limiter.Config{
		Max:        config.RateLimitRequests,
		Expiration: config.RateLimitDuration,
		Storage:    storage,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Generate rate limit key based on IP and tenant
			ip := c.IP()
			tenantID := c.Get("X-Tenant-ID")
			if tenantID == "" {
				if tid, ok := c.Locals("tenant_id").(string); ok {
					tenantID = tid
				}
			}
			return fmt.Sprintf("rate_limit:%s:%s", ip, tenantID)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return errors.NewHTTPError(fiber.StatusTooManyRequests, "Rate limit exceeded", nil)
		},
	}))

	// 5. Request size limiting
	middlewares = append(middlewares, func(c *fiber.Ctx) error {
		if len(c.Body()) > parseSize(config.MaxRequestSize) {
			return errors.NewHTTPError(fiber.StatusRequestEntityTooLarge, "Request body too large", nil)
		}
		return c.Next()
	})

	return middlewares
}

// TenantAwareRateLimit creates a rate limiter that's tenant-aware
func TenantAwareRateLimit(config limiter.Config, storage fiber.Storage) fiber.Handler {
	if config.KeyGenerator == nil {
		config.KeyGenerator = func(c *fiber.Ctx) string {
			ip := c.IP()
			tenantID := "unknown"

			// Try to get tenant ID from context
			if tid, ok := c.Locals("tenant_id").(string); ok {
				tenantID = tid
			}

			return fmt.Sprintf("tenant_rate_limit:%s:%s", tenantID, ip)
		}
	}

	config.Storage = storage
	return limiter.New(config)
}

// APIKeyMiddleware validates API keys for external integrations
func APIKeyMiddleware(validAPIKeys map[string]string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return errors.NewHTTPError(fiber.StatusUnauthorized, "API key required", nil)
		}

		tenantID, exists := validAPIKeys[apiKey]
		if !exists {
			return errors.NewHTTPError(fiber.StatusUnauthorized, "Invalid API key", nil)
		}

		// Set tenant context from API key
		c.Locals("api_key_tenant_id", tenantID)
		c.Locals("auth_method", "api_key")

		return c.Next()
	}
}

// IPWhitelistMiddleware restricts access to specific IP addresses
func IPWhitelistMiddleware(allowedIPs []string) fiber.Handler {
	ipMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		ipMap[ip] = true
	}

	return func(c *fiber.Ctx) error {
		clientIP := c.IP()

		if !ipMap[clientIP] {
			return errors.NewHTTPError(fiber.StatusForbidden, "IP address not allowed", nil)
		}

		return c.Next()
	}
}

// RequestValidationMiddleware validates common request parameters
func RequestValidationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Validate User-Agent header
		userAgent := c.Get("User-Agent")
		if userAgent == "" {
			return errors.NewHTTPError(fiber.StatusBadRequest, "User-Agent header required", nil)
		}

		// Check for suspicious patterns in User-Agent
		suspiciousPatterns := []string{
			"sqlmap", "nikto", "nmap", "masscan", "zap", "burp",
		}
		userAgentLower := strings.ToLower(userAgent)
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(userAgentLower, pattern) {
				return errors.NewHTTPError(fiber.StatusForbidden, "Suspicious user agent detected", nil)
			}
		}

		// Validate Content-Type for POST/PUT requests
		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			contentType := c.Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") &&
				!strings.Contains(contentType, "multipart/form-data") &&
				!strings.Contains(contentType, "application/x-www-form-urlencoded") {
				return errors.NewHTTPError(fiber.StatusUnsupportedMediaType, "Unsupported content type", nil)
			}
		}

		return c.Next()
	}
}

// XSSProtectionMiddleware adds additional XSS protection
func XSSProtectionMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check for potential XSS in query parameters
		for key, values := range c.Queries() {
			for _, value := range values {
				if containsXSS(value) {
					return errors.NewHTTPError(fiber.StatusBadRequest, fmt.Sprintf("Potential XSS detected in parameter: %s", key), nil)
				}
			}
		}

		// Check for XSS in form data if present
		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			contentType := c.Get("Content-Type")
			if strings.Contains(contentType, "application/x-www-form-urlencoded") {
				form, err := c.MultipartForm()
				if err == nil {
					for key, values := range form.Value {
						for _, value := range values {
							if containsXSS(value) {
								return errors.NewHTTPError(fiber.StatusBadRequest, fmt.Sprintf("Potential XSS detected in form field: %s", key), nil)
							}
						}
					}
				}
			}
		}

		return c.Next()
	}
}

// SQLInjectionProtectionMiddleware checks for SQL injection patterns
func SQLInjectionProtectionMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check query parameters for SQL injection patterns
		for key, values := range c.Queries() {
			for _, value := range values {
				if containsSQLInjection(value) {
					return errors.NewHTTPError(fiber.StatusBadRequest, fmt.Sprintf("Potential SQL injection detected in parameter: %s", key), nil)
				}
			}
		}

		return c.Next()
	}
}

// containsXSS checks for common XSS patterns
func containsXSS(input string) bool {
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "onload=", "onerror=", "onclick=",
		"onmouseover=", "onfocus=", "onblur=", "eval(", "alert(", "confirm(",
		"prompt(", "document.cookie", "document.write", "innerHTML", "outerHTML",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(inputLower, pattern) {
			return true
		}
	}

	return false
}

// containsSQLInjection checks for common SQL injection patterns
func containsSQLInjection(input string) bool {
	sqlPatterns := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "union", "select",
		"insert", "update", "delete", "drop", "create", "alter", "exec",
		"execute", "declare", "cast", "char", "nchar", "varchar", "nvarchar",
		"syscolumns", "sysobjects", "information_schema",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range sqlPatterns {
		if strings.Contains(inputLower, pattern) {
			return true
		}
	}

	return false
}

// parseSize parses size strings like "10MB", "1GB" etc.
func parseSize(size string) int {
	if size == "" {
		return 10 * 1024 * 1024 // Default 10MB
	}

	size = strings.ToUpper(size)
	multiplier := 1

	if strings.HasSuffix(size, "KB") {
		multiplier = 1024
		size = strings.TrimSuffix(size, "KB")
	} else if strings.HasSuffix(size, "MB") {
		multiplier = 1024 * 1024
		size = strings.TrimSuffix(size, "MB")
	} else if strings.HasSuffix(size, "GB") {
		multiplier = 1024 * 1024 * 1024
		size = strings.TrimSuffix(size, "GB")
	}

	num, err := strconv.Atoi(size)
	if err != nil {
		return 10 * 1024 * 1024 // Default 10MB on error
	}

	return num * multiplier
}

// ErrorHandlerMiddleware provides consistent error handling
func ErrorHandlerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		if err != nil {
			// Handle HTTPError
			if httpErr, ok := err.(*errors.HTTPError); ok {
				return c.Status(httpErr.Status).JSON(errors.ToErrorResponse(httpErr))
			}

			// Handle Fiber errors
			if fiberErr, ok := err.(*fiber.Error); ok {
				httpErr := errors.NewHTTPError(fiberErr.Code, fiberErr.Message, err)
				return c.Status(httpErr.Status).JSON(errors.ToErrorResponse(httpErr))
			}

			// Handle unknown errors
			httpErr := errors.NewHTTPError(fiber.StatusInternalServerError, "Internal server error", err)
			return c.Status(httpErr.Status).JSON(errors.ToErrorResponse(httpErr))
		}

		return nil
	}
}
