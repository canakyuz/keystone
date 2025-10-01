package user

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"nexspaces-api/internal/middleware"
	"nexspaces-api/internal/usecase/user"
)

// Handler handles user HTTP requests
type Handler struct {
	userService *user.Service
}

// NewHandler creates a new user handler
func NewHandler(userService *user.Service) *Handler {
	return &Handler{
		userService: userService,
	}
}

// Create creates a new user (admin only)
// POST /api/v1/users
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req user.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.userService.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}

// GetByID retrieves a user by ID
// GET /api/v1/users/:id
func (h *Handler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")

	if tenantID == "" {
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

// List retrieves all users with pagination
// GET /api/v1/users
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	role := c.Query("role")
	status := c.Query("status")
	search := c.Query("search")

	result, err := h.userService.List(c.Context(), tenantID, page, perPage, role, status, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// Update updates a user
// PATCH /api/v1/users/:id
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req user.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.userService.Update(c.Context(), tenantID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// UpdatePassword updates user password
// POST /api/v1/users/:id/password
func (h *Handler) UpdatePassword(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")
	currentUserID := middleware.GetUserID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Users can only update their own password
	if userID != currentUserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden",
		})
	}

	var req user.UpdatePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.userService.UpdatePassword(c.Context(), tenantID, userID, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}

// UpdateRole updates user role (admin only)
// POST /api/v1/users/:id/role
func (h *Handler) UpdateRole(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req user.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := h.userService.UpdateRole(c.Context(), tenantID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// Suspend suspends a user (admin only)
// POST /api/v1/users/:id/suspend
func (h *Handler) Suspend(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.userService.Suspend(c.Context(), tenantID, userID, req.Reason); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User suspended successfully",
	})
}

// Activate activates a user (admin only)
// POST /api/v1/users/:id/activate
func (h *Handler) Activate(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.userService.Activate(c.Context(), tenantID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User activated successfully",
	})
}

// VerifyEmail verifies user email
// POST /api/v1/users/:id/verify-email
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.userService.VerifyEmail(c.Context(), tenantID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}

// Delete deletes a user (admin only)
// DELETE /api/v1/users/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := c.Params("id")
	currentUserID := middleware.GetUserID(c)

	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Users cannot delete themselves
	if userID == currentUserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	if err := h.userService.Delete(c.Context(), tenantID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetStats retrieves user statistics
// GET /api/v1/users/stats
func (h *Handler) GetStats(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	stats, err := h.userService.GetStats(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": stats,
	})
}
