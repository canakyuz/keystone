package auth

import (
	"github.com/gofiber/fiber/v2"
	"nexpaces-api/internal/middleware"
	"nexpaces-api/internal/usecase/user"
)

// Handler handles authentication HTTP requests
type Handler struct {
	userService *user.Service
}

// NewHandler creates a new auth handler
func NewHandler(userService *user.Service) *Handler {
	return &Handler{
		userService: userService,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *Handler) Register(c *fiber.Ctx) error {
	println("[DEBUG] Register handler called")
	var req user.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		println("[DEBUG] Body parse error:", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	println("[DEBUG] Body parsed, calling service...")

	result, err := h.userService.Register(c.Context(), &req)
	if err != nil {
		println("[DEBUG] Service error:", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	println("[DEBUG] Registration successful")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (h *Handler) Login(c *fiber.Ctx) error {
	println("[DEBUG] Login handler called")
	var req user.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		println("[DEBUG] Login body parse error:", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}
	println("[DEBUG] Login body parsed, calling service for email:", req.Email)

	result, err := h.userService.Login(c.Context(), &req)
	if err != nil {
		println("[DEBUG] Login service error:", err.Error())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	println("[DEBUG] Login successful")
	return c.JSON(result)
}

// GetMe returns current authenticated user
// GET /api/v1/auth/me
func (h *Handler) GetMe(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	if tenantID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	result, err := h.userService.GetByID(c.Context(), tenantID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// Logout handles user logout (client-side token removal)
// POST /api/v1/auth/logout
func (h *Handler) Logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}
