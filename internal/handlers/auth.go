package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"nexspaces-api/internal/auth"
	"nexspaces-api/internal/models"
	"nexspaces-api/pkg/database"
)

type AuthHandler struct {
	db    *database.DB
	redis *database.RedisClient
	jwt   *auth.JWTService
}

type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	TenantSlug string `json:"tenant_slug,omitempty"`
}

type RegisterRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	Name       string `json:"name" validate:"required,min=2"`
	TenantSlug string `json:"tenant_slug,omitempty"`
}

type AuthResponse struct {
	Token     string             `json:"token"`
	User      *models.TenantUser `json:"user"`
	ExpiresAt time.Time          `json:"expires_at"`
}

func NewAuthHandler(db *database.DB, redis *database.RedisClient, jwt *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		db:    db,
		redis: redis,
		jwt:   jwt,
	}
}

// ValidateToken validates Better Auth token and returns user info
func (h *AuthHandler) ValidateToken(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	claims, err := h.jwt.ValidateBetterAuthToken(authHeader)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid token",
			"details": err.Error(),
		})
	}

	// Get user details from database
	user, err := h.getUserByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"valid":  true,
		"user":   user,
		"claims": claims,
	})
}

// RefreshToken generates a new token from a valid existing token
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	claims, err := h.jwt.ValidateBetterAuthToken(authHeader)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Generate new token
	newToken, err := h.jwt.RefreshToken(claims)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"token":      newToken,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
}

// Me returns current user information
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	// Get user from middleware context
	claims, ok := c.Locals("user").(*auth.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Get full user details
	user, err := h.getUserByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Get tenant information
	tenant, err := h.getTenantByID(claims.TenantID)
	if err != nil {
		// User might not be associated with a tenant yet
		tenant = nil
	}

	return c.JSON(fiber.Map{
		"user":   user,
		"tenant": tenant,
		"permissions": fiber.Map{
			"role":        claims.Role,
			"tenant_role": claims.TenantRole,
		},
	})
}

// Logout invalidates the current session
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*auth.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Invalidate session in Redis if session ID exists
	if claims.SessionID != "" {
		sessionKey := "session:" + claims.SessionID
		err := h.redis.Del(context.Background(), sessionKey).Err()
		if err != nil {
			// Log error but don't fail the logout
			// logger.Error("Failed to delete session from Redis", "error", err)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Successfully logged out",
	})
}

// SwitchTenant allows user to switch between tenants they have access to
func (h *AuthHandler) SwitchTenant(c *fiber.Ctx) error {
	type SwitchTenantRequest struct {
		TenantID string `json:"tenant_id" validate:"required"`
	}

	var req SwitchTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	claims, ok := c.Locals("user").(*auth.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Verify user has access to the requested tenant
	hasAccess, role, err := h.checkTenantAccess(claims.UserID, req.TenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check tenant access",
		})
	}

	if !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied to this tenant",
		})
	}

	// Generate new token with new tenant context
	newToken, err := h.jwt.GenerateToken(
		claims.UserID,
		req.TenantID,
		claims.Email,
		claims.Role,
		role,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(fiber.Map{
		"token":       newToken,
		"tenant_id":   req.TenantID,
		"tenant_role": role,
		"expires_at":  time.Now().Add(24 * time.Hour),
	})
}

// Helper functions
func (h *AuthHandler) getUserByID(userID string) (*models.TenantUser, error) {
	query := `
		SELECT id, tenant_id, email, name, role, last_login, created_at, updated_at
		FROM tenant_users 
		WHERE id = $1
	`

	var user models.TenantUser
	err := h.db.QueryRow(query, userID).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.Name,
		&user.Role, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (h *AuthHandler) getTenantByID(tenantID string) (*models.Tenant, error) {
	query := `
		SELECT id, slug, name, custom_domain, plan_id, status, settings, created_at, updated_at
		FROM tenants 
		WHERE id = $1
	`

	var tenant models.Tenant
	err := h.db.QueryRow(query, tenantID).Scan(
		&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.CustomDomain,
		&tenant.PlanID, &tenant.Status, &tenant.Settings,
		&tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (h *AuthHandler) checkTenantAccess(userID, tenantID string) (bool, string, error) {
	query := `
		SELECT role FROM tenant_users 
		WHERE id = $1 AND tenant_id = $2
	`

	var role string
	err := h.db.QueryRow(query, userID, tenantID).Scan(&role)
	if err != nil {
		return false, "", err
	}

	return true, role, nil
}
