package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
	"nexspaces-api/internal/core/ports/events"
	"nexspaces-api/internal/core/ports/repositories"
	"nexspaces-api/internal/core/ports/services"

	"github.com/google/uuid"
)

// CreateUserUseCase handles user creation with proper tenant isolation
type CreateUserUseCase struct {
	userRepo        repositories.UserRepository
	tenantRepo      repositories.TenantRepository
	authService     services.AuthService
	notificationSvc services.NotificationService
	eventBus        events.EventBus
}

func NewCreateUserUseCase(
	userRepo repositories.UserRepository,
	tenantRepo repositories.TenantRepository,
	authService services.AuthService,
	notificationSvc services.NotificationService,
	eventBus events.EventBus,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		userRepo:        userRepo,
		tenantRepo:      tenantRepo,
		authService:     authService,
		notificationSvc: notificationSvc,
		eventBus:        eventBus,
	}
}

type CreateUserRequest struct {
	TenantID  shared.TenantID `json:"tenant_id" validate:"required"`
	Email     string          `json:"email" validate:"required,email"`
	FirstName string          `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string          `json:"last_name" validate:"required,min=1,max=100"`
	Role      string          `json:"role" validate:"required"`
	CreatedBy shared.UserID   `json:"created_by" validate:"required"`
}

type CreateUserResponse struct {
	User        *user.User `json:"user"`
	InviteToken string     `json:"invite_token,omitempty"`
}

// Parse role validation can be added here or in the user entity

func (uc *CreateUserUseCase) Execute(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
	// Validate tenant exists and is active
	tenantEntity, err := uc.tenantRepo.GetByID(ctx, req.TenantID.UUID())
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenantEntity.Status != tenant.StatusActive {
		return nil, shared.ErrTenantNotActive
	}

	// Check if creator has permission to create users
	creator, err := uc.userRepo.GetByID(ctx, req.TenantID.UUID(), req.CreatedBy.UUID())
	if err != nil {
		return nil, fmt.Errorf("failed to get creator: %w", err)
	}

	if creator.TenantID != req.TenantID.UUID() {
		return nil, shared.ErrCrossTenantAccess
	}

	if !creator.CanManageUsers() {
		return nil, shared.ErrInsufficientPermissions
	}

	// Check if user already exists in tenant
	existingUser, err := uc.userRepo.GetByEmailAndTenant(ctx, req.Email, req.TenantID)
	if err != nil && !errors.Is(err, shared.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		return nil, shared.ErrUserAlreadyExists
	}

	// Create email value object
	email, err := shared.NewEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Create user
	userEntity := user.NewUser(user.CreateUserRequest{
		TenantID:  req.TenantID.UUID(),
		Email:     email.String(),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
	}, "") // Assuming empty password for now

	// Save to repository
	if err := uc.userRepo.Create(ctx, userEntity); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// Generate invite token
	inviteToken, err := uc.authService.GenerateInviteToken(ctx, userEntity.ID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}

	// Send invitation email
	if err := uc.notificationSvc.SendUserInvitation(ctx, services.UserInvitationRequest{
		UserID:      userEntity.ID,
		TenantID:    req.TenantID,
		Email:       req.Email,
		InviteToken: inviteToken,
		InvitedBy:   creator.FullName(),
	}); err != nil {
		// Log error but don't fail the whole operation
		// TODO: Add proper logging
	}

	// Publish domain event
	event := &events.UserCreatedEvent{
		BaseEvent: events.BaseEvent{
			ID:          uuid.New().String(),
			Type:        events.EventTypeUserCreated,
			AggregateID: userEntity.ID.String(),
			TenantID:    userEntity.TenantID.String(),
			OccurredAt:  time.Now(),
			Payload: map[string]interface{}{
				"user_email": req.Email,
				"user_role":  req.Role,
			},
		},
		UserID:    userEntity.ID.String(),
		UserEmail: req.Email,
		UserRole:  req.Role,
	}

	if err := uc.eventBus.Publish(ctx, event); err != nil {
		// Log error but don't fail
		// TODO: Add proper logging
	}

	return &CreateUserResponse{
		User:        userEntity,
		InviteToken: inviteToken,
	}, nil
}

// UpdateUserUseCase handles user updates with proper validation
type UpdateUserUseCase struct {
	userRepo   repositories.UserRepository
	tenantRepo repositories.TenantRepository
	eventBus   events.EventBus
}

func NewUpdateUserUseCase(
	userRepo repositories.UserRepository,
	tenantRepo repositories.TenantRepository,
	eventBus events.EventBus,
) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
		eventBus:   eventBus,
	}
}

type UpdateUserRequest struct {
	UserID    shared.UserID   `json:"user_id" validate:"required"`
	TenantID  shared.TenantID `json:"tenant_id" validate:"required"`
	FirstName *string         `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string         `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Role      *string         `json:"role,omitempty" validate:"omitempty"`
	UpdatedBy shared.UserID   `json:"updated_by" validate:"required"`
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, req UpdateUserRequest) (*user.User, error) {
	// Get the user to update
	userEntity, err := uc.userRepo.GetByID(ctx, req.TenantID.UUID(), req.UserID.UUID())
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify tenant isolation
	if userEntity.TenantID != req.TenantID.UUID() {
		return nil, shared.ErrCrossTenantAccess
	}

	// Get the updater
	updater, err := uc.userRepo.GetByID(ctx, req.TenantID.UUID(), req.UpdatedBy.UUID())
	if err != nil {
		return nil, fmt.Errorf("failed to get updater: %w", err)
	}

	// Verify updater belongs to same tenant
	if updater.TenantID != req.TenantID.UUID() {
		return nil, shared.ErrCrossTenantAccess
	}

	// Check permissions
	if !updater.CanManageUsers() && updater.ID != req.UserID.UUID() {
		return nil, shared.ErrInsufficientPermissions
	}

	// Apply updates
	if req.FirstName != nil {
		userEntity.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		userEntity.LastName = *req.LastName
	}

	if req.Role != nil {
		// Only admins can change roles
		if !updater.CanManageUsers() {
			return nil, shared.ErrInsufficientPermissions
		}
		role, err := user.ParseRole(*req.Role)
		if err != nil {
			return nil, fmt.Errorf("invalid role: %w", err)
		}
		userEntity.Role = role
	}

	userEntity.UpdatedAt = time.Now()

	// Save changes
	if err := uc.userRepo.Update(ctx, userEntity); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, &events.UserUpdatedEvent{
		BaseEvent: events.BaseEvent{
			ID:          uuid.New().String(),
			Type:        events.EventTypeUserUpdated,
			AggregateID: userEntity.ID.String(),
			TenantID:    req.TenantID.String(),
			OccurredAt:  time.Now(),
			Payload:     buildChangeMap(req),
		},
		UserID:    userEntity.ID.String(),
		Changes:   buildChangeMap(req),
		UpdatedBy: req.UpdatedBy.String(),
		UpdatedAt: userEntity.UpdatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return userEntity, nil
}

// DeactivateUserUseCase handles user deactivation
type DeactivateUserUseCase struct {
	userRepo repositories.UserRepository
	eventBus events.EventBus
}

func NewDeactivateUserUseCase(
	userRepo repositories.UserRepository,
	eventBus events.EventBus,
) *DeactivateUserUseCase {
	return &DeactivateUserUseCase{
		userRepo: userRepo,
		eventBus: eventBus,
	}
}

type DeactivateUserRequest struct {
	UserID        shared.UserID   `json:"user_id" validate:"required"`
	TenantID      shared.TenantID `json:"tenant_id" validate:"required"`
	DeactivatedBy shared.UserID   `json:"deactivated_by" validate:"required"`
	Reason        string          `json:"reason,omitempty"`
}

func (uc *DeactivateUserUseCase) Execute(ctx context.Context, req DeactivateUserRequest) error {
	// Get the user to deactivate
	userEntity, err := uc.userRepo.GetByID(ctx, req.TenantID.UUID(), req.UserID.UUID())
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify tenant isolation
	if userEntity.TenantID != req.TenantID.UUID() {
		return shared.ErrCrossTenantAccess
	}

	// Get the deactivator
	deactivator, err := uc.userRepo.GetByID(ctx, req.TenantID.UUID(), req.DeactivatedBy.UUID())
	if err != nil {
		return fmt.Errorf("failed to get deactivator: %w", err)
	}

	// Verify deactivator belongs to same tenant
	if deactivator.TenantID != req.TenantID.UUID() {
		return shared.ErrCrossTenantAccess
	}

	// Check permissions
	if !deactivator.CanManageUsers() {
		return shared.ErrInsufficientPermissions
	}

	// Can't deactivate yourself
	if userEntity.ID == req.DeactivatedBy.UUID() {
		return shared.ErrCannotDeactivateSelf
	}

	// Deactivate user
	if err := userEntity.Deactivate(req.Reason); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Save changes
	if err := uc.userRepo.Update(ctx, userEntity); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	// Publish event
	if err := uc.eventBus.Publish(ctx, &events.UserDeactivatedEvent{
		BaseEvent: events.BaseEvent{
			ID:          uuid.New().String(),
			Type:        events.EventTypeUserDeactivated,
			AggregateID: userEntity.ID.String(),
			TenantID:    req.TenantID.String(),
			OccurredAt:  time.Now(),
			Payload: map[string]interface{}{
				"reason": req.Reason,
			},
		},
		UserID:        userEntity.ID.String(),
		DeactivatedBy: req.DeactivatedBy.String(),
		Reason:        req.Reason,
		DeactivatedAt: userEntity.UpdatedAt,
	}); err != nil {
		// Log error but don't fail
	}

	return nil
}

func buildChangeMap(req UpdateUserRequest) map[string]interface{} {
	changes := make(map[string]interface{})

	if req.FirstName != nil {
		changes["first_name"] = *req.FirstName
	}

	if req.LastName != nil {
		changes["last_name"] = *req.LastName
	}

	if req.Role != nil {
		changes["role"] = *req.Role
	}

	return changes
}
