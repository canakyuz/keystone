package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/user"
	"nexspaces-api/internal/core/ports/repositories"
	apperrors "nexspaces-api/internal/shared/errors"
)

// AuthMiddleware handles JWT authentication and sets user context
type AuthMiddleware struct {
	userRepo  repositories.UserRepository
	jwtSecret string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(userRepo repositories.UserRepository, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// JWTClaims represents the JWT claims structure
type JWTClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Authenticate validates JWT token and sets user context
func (m *AuthMiddleware) Authenticate(c *fiber.Ctx) error {
	// Extract token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Authorization header required", nil)
	}

	// Check for Bearer token format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid authorization format", nil)
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Token required", nil)
	}

	// Parse and validate JWT token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid signing method", nil)
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid token", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid token claims", nil)
	}

	// Validate token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Token expired", nil)
	}

	// Parse UUIDs
	userUUID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid user ID in token", err)
	}

	tenantUUID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid tenant ID in token", err)
	}

	// Verify user exists and is active
	user, err := m.userRepo.GetByID(c.Context(), tenantUUID, userUUID)
	if err != nil {
		if errors.Is(err, shared.ErrUserNotFound) {
			return apperrors.NewHTTPError(fiber.StatusUnauthorized, "User not found", err)
		}
		return apperrors.NewHTTPError(fiber.StatusInternalServerError, "Failed to verify user", err)
	}

	// Check user status
	if user.Status != "active" {
		return apperrors.NewHTTPError(fiber.StatusForbidden, "User account is not active", nil)
	}

	// Verify tenant ID matches
	if user.TenantID != tenantUUID {
		return apperrors.NewHTTPError(fiber.StatusForbidden, "Token tenant mismatch", nil)
	}

	// Set user context for downstream handlers
	c.Locals("user_id", userUUID)
	c.Locals("tenant_id", tenantUUID)
	c.Locals("user_email", claims.Email)
	c.Locals("user_role", claims.Role)
	c.Locals("user", user)

	return c.Next()
}

// RequireRole creates a middleware that requires specific user roles
func (m *AuthMiddleware) RequireRole(allowedRoles ...string) fiber.Handler {
	allowedRoleMap := make(map[string]bool)
	for _, role := range allowedRoles {
		allowedRoleMap[role] = true
	}

	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("user_role").(string)
		if !ok {
			return apperrors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
		}

		if !allowedRoleMap[userRole] {
			return apperrors.NewHTTPError(fiber.StatusForbidden, "Insufficient role permissions", nil)
		}

		return c.Next()
	}
}

// RequirePermission creates a middleware that checks specific permissions
func (m *AuthMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userObj, ok := c.Locals("user").(interface{})
		if !ok {
			return apperrors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
		}

		// Type assertion to get user entity
		userEntity, ok := userObj.(*user.User)
		if !ok {
			return apperrors.NewHTTPError(fiber.StatusInternalServerError, "Invalid user context", nil)
		}

		// Check permission based on user role and the specific permission
		hasPermission := m.checkPermission(userEntity, permission)
		if !hasPermission {
			return apperrors.NewHTTPError(fiber.StatusForbidden, "Insufficient permissions", nil)
		}

		return c.Next()
	}
}

// checkPermission checks if a user has a specific permission
func (m *AuthMiddleware) checkPermission(user *user.User, permission string) bool {
	// Define permission matrix based on roles
	permissions := map[string]map[string]bool{
		"owner": {
			"manage_users":        true,
			"manage_billing":      true,
			"manage_templates":    true,
			"install_templates":   true,
			"publish_templates":   true,
			"manage_settings":     true,
			"view_analytics":      true,
			"manage_integrations": true,
		},
		"admin": {
			"manage_users":        true,
			"manage_templates":    true,
			"install_templates":   true,
			"publish_templates":   true,
			"manage_settings":     true,
			"view_analytics":      true,
			"manage_integrations": true,
		},
		"editor": {
			"manage_templates":  true,
			"install_templates": true,
			"publish_templates": true,
			"view_analytics":    false,
		},
		"viewer": {
			"install_templates": true,
			"view_analytics":    false,
		},
		"billing_admin": {
			"manage_billing": true,
			"view_analytics": true,
		},
	}

	rolePermissions, exists := permissions[string(user.Role)]
	if !exists {
		return false
	}

	return rolePermissions[permission]
}

// OptionalAuth provides optional authentication (doesn't fail if no token)
func (m *AuthMiddleware) OptionalAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		// No auth header, continue without setting user context
		return c.Next()
	}

	// Try to authenticate, but don't fail if it doesn't work
	err := m.Authenticate(c)
	if err != nil {
		// Clear any partial context that might have been set
		c.Locals("user_id", nil)
		c.Locals("tenant_id", nil)
		c.Locals("user_email", nil)
		c.Locals("user_role", nil)
		c.Locals("user", nil)
	}

	return c.Next()
}

// GenerateJWT generates a new JWT token for a user
func GenerateJWT(userID uuid.UUID, tenantID uuid.UUID, email, role, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:   userID.String(),
		TenantID: tenantID.String(),
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "nexspaces-api",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// RefreshToken refreshes a JWT token if it's close to expiry
func (m *AuthMiddleware) RefreshToken(tokenString string) (string, error) {
	// Parse existing token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return "", apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid token claims", nil)
	}

	// Check if token is close to expiry (within 15 minutes)
	if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) > 15*time.Minute {
		return tokenString, nil // Token is still valid for a while
	}

	// Generate new token with extended expiry
	userUUID, _ := uuid.Parse(claims.UserID)
	tenantUUID, _ := uuid.Parse(claims.TenantID)

	return GenerateJWT(
		userUUID,
		tenantUUID,
		claims.Email,
		claims.Role,
		m.jwtSecret,
		24*time.Hour, // 24 hour expiry for refreshed tokens
	)
}

// ValidateInviteToken validates a user invitation token
func (m *AuthMiddleware) ValidateInviteToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invalid token claims", nil)
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, apperrors.NewHTTPError(fiber.StatusUnauthorized, "Invitation token expired", nil)
	}

	return claims, nil
}
