package auth

import (
	"errors"

	domainUser "github.com/canakyuz/keystone/internal/domain/user"
	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/canakyuz/keystone/internal/usecase/user"
	"github.com/gofiber/fiber/v2"
)

// Handler handles authentication HTTP requests.
//
// These endpoints deliberately carry no debug logging. They used to print the request
// email and the raw service error to stderr with the builtin println, on every login
// attempt. That put an identifier into an unstructured stream nothing rotates or
// redacts, and it leaked the reason a login failed, which tells an attacker whether an
// address exists. Failures here are reported to the caller and nowhere else.
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
	var req user.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.userService.Register(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (h *Handler) Login(c *fiber.Ctx) error {
	var req user.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.userService.Login(c.Context(), &req)
	if err != nil {
		return loginError(c, err)
	}

	return c.JSON(result)
}

// loginError maps a login failure to a response.
//
// Only the domain errors are passed through. Anything else, a repository or database
// failure, is reported as a generic 500: returning err.Error() would put the database
// message in the response body, where it describes the schema to whoever asked.
//
// The credentials case says only that the pair was wrong. The service already returns
// one error for both "no such address" and "wrong password", and this keeps that
// property at the boundary: a response that distinguishes them tells an attacker which
// addresses are registered.
func loginError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domainUser.ErrInvalidCredentials):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid email or password",
		})
	case errors.Is(err, domainUser.ErrUserSuspended):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "account suspended",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "could not process the request",
		})
	}
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
