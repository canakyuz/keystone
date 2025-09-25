package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/usecases/user"
	"nexspaces-api/internal/shared/errors"
	"nexspaces-api/internal/shared/validation"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	createUserUC     *user.CreateUserUseCase
	updateUserUC     *user.UpdateUserUseCase
	deactivateUserUC *user.DeactivateUserUseCase
	validator        *validation.Validator
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	createUserUC *user.CreateUserUseCase,
	updateUserUC *user.UpdateUserUseCase,
	deactivateUserUC *user.DeactivateUserUseCase,
	validator *validation.Validator,
) *UserHandler {
	return &UserHandler{
		createUserUC:     createUserUC,
		updateUserUC:     updateUserUC,
		deactivateUserUC: deactivateUserUC,
		validator:        validator,
	}
}

// CreateUserRequest represents the HTTP request for creating a user
type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	Role      string `json:"role" validate:"required,oneof=owner admin editor viewer billing_admin"`
}

// UpdateUserRequest represents the HTTP request for updating a user
type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Role      *string `json:"role,omitempty" validate:"omitempty,oneof=owner admin editor viewer billing_admin"`
}

// DeactivateUserRequest represents the HTTP request for deactivating a user
type DeactivateUserRequest struct {
	Reason string `json:"reason,omitempty" validate:"max=500"`
}

// CreateUser handles POST /api/tenants/{tenantId}/users
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Extract current user from context (set by auth middleware)
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Verify tenant context (set by tenant middleware)
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Parse role
	role, err := user.ParseRole(req.Role)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid role", err)
	}

	// Convert to use case request
	ucReq := user.CreateUserRequest{
		TenantID:  tenantID,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      role,
		CreatedBy: shared.UserID(currentUserID),
	}

	// Execute use case
	result, err := h.createUserUC.Execute(c.Context(), ucReq)
	if err != nil {
		return errors.HandleDomainError(err)
	}

	// Convert to response DTO
	response := fiber.Map{
		"user":         convertUserToDTO(result.User),
		"invite_token": result.InviteToken,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    response,
		"message": "User created successfully",
	})
}

// UpdateUser handles PUT /api/tenants/{tenantId}/users/{userId}
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Extract user ID from URL
	userIDStr := c.Params("userId")
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid user ID format", err)
	}
	userID := shared.UserID(userUUID)

	// Extract current user from context
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Convert to use case request
	ucReq := user.UpdateUserRequest{
		UserID:    userID,
		TenantID:  tenantID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		UpdatedBy: shared.UserID(currentUserID),
	}

	// Parse role if provided
	if req.Role != nil {
		role, err := user.ParseRole(*req.Role)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid role", err)
		}
		ucReq.Role = &role
	}

	// Execute use case
	result, err := h.updateUserUC.Execute(c.Context(), ucReq)
	if err != nil {
		return errors.HandleDomainError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    convertUserToDTO(result),
		"message": "User updated successfully",
	})
}

// DeactivateUser handles DELETE /api/tenants/{tenantId}/users/{userId}
func (h *UserHandler) DeactivateUser(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}
	tenantID := shared.TenantID(tenantUUID)

	// Extract user ID from URL
	userIDStr := c.Params("userId")
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid user ID format", err)
	}
	userID := shared.UserID(userUUID)

	// Extract current user from context
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User context not found", nil)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	var req DeactivateUserRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional for DELETE, ignore parse errors
		req = DeactivateUserRequest{}
	}

	// Validate request if body provided
	if err := h.validator.Validate(req); err != nil {
		return errors.NewValidationError("Validation failed", err)
	}

	// Convert to use case request
	ucReq := user.DeactivateUserRequest{
		UserID:        userID,
		TenantID:      tenantID,
		DeactivatedBy: shared.UserID(currentUserID),
		Reason:        req.Reason,
	}

	// Execute use case
	if err := h.deactivateUserUC.Execute(c.Context(), ucReq); err != nil {
		return errors.HandleDomainError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deactivated successfully",
	})
}

// GetUsers handles GET /api/tenants/{tenantId}/users
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	// Parse query parameters
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := c.QueryInt("page_size", 20)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.Query("status")
	role := c.Query("role")
	search := c.Query("search")

	// TODO: Implement GetUsersUseCase and execute here
	// For now, return placeholder response
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"users":       []interface{}{},
			"total_count": 0,
			"page":        page,
			"page_size":   pageSize,
			"has_next":    false,
		},
		"message": "Users retrieved successfully",
	})
}

// GetUser handles GET /api/tenants/{tenantId}/users/{userId}
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	// Extract tenant ID from URL
	tenantIDStr := c.Params("tenantId")
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
	}

	// Extract user ID from URL
	userIDStr := c.Params("userId")
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid user ID format", err)
	}

	// Verify tenant context
	contextTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok || contextTenantID != tenantUUID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant context mismatch", nil)
	}

	// TODO: Implement GetUserUseCase and execute here
	// For now, return placeholder response
	return c.JSON(fiber.Map{
		"success": true,
		"data":    nil,
		"message": "User retrieved successfully",
	})
}
