package user

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"nexpaces-api/internal/domain/user"
	userRepo "nexpaces-api/internal/repository/user"
	"nexpaces-api/pkg/logger"
	"nexpaces-api/pkg/validator"
)

// Service handles user business logic
type Service struct {
	repo      userRepo.Repository
	validator *validator.Validator
	logger    *logger.Logger
	jwtSecret string
}

// NewService creates a new user service
func NewService(repo userRepo.Repository, val *validator.Validator, log *logger.Logger, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		validator: val,
		logger:    log,
		jwtSecret: jwtSecret,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Check if user already exists
	exists, err := s.repo.ExistsByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to check user existence")
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, user.ErrEmailAlreadyTaken
	}

	// Create user domain entity
	u, err := user.New(req.TenantID, req.Email, req.Password, req.FirstName, req.LastName, user.RoleViewer)
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repo.Create(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
		"email":     u.Email,
	}).Info("User registered successfully")

	return ToResponse(u), nil
}

// Login authenticates a user and returns JWT token
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Get user by email
	var u *user.User
	var err error

	if req.TenantID != "" {
		u, err = s.repo.GetByEmail(ctx, req.TenantID, req.Email)
	} else {
		u, err = s.repo.GetByEmailGlobal(ctx, req.Email)
	}

	if err != nil {
		return nil, user.ErrInvalidCredentials
	}

	// Check password
	if !u.CheckPassword(req.Password) {
		s.logger.WithFields(logger.Fields{
			"email":     req.Email,
			"tenant_id": u.TenantID,
		}).Warn("Failed login attempt - invalid password")
		return nil, user.ErrInvalidCredentials
	}

	// Check if user is active
	if !u.IsActive() {
		if u.IsSuspended() {
			return nil, user.ErrUserSuspended
		}
		if !u.IsEmailVerified() {
			return nil, user.ErrEmailNotVerified
		}
		return nil, user.ErrUserInactive
	}

	// Update last login
	if err := s.repo.UpdateLastLogin(ctx, u.TenantID, u.ID); err != nil {
		s.logger.ErrorWithErr(err, "failed to update last login")
	}

	// Generate JWT token
	token, expiresIn, err := s.generateToken(u)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to generate token")
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
		"email":     u.Email,
	}).Info("User logged in successfully")

	return &LoginResponse{
		User:        ToResponse(u),
		AccessToken: token,
		ExpiresIn:   expiresIn,
	}, nil
}

// Create creates a new user (admin only)
func (s *Service) Create(ctx context.Context, tenantID string, req *CreateUserRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Check if user already exists
	exists, err := s.repo.ExistsByEmail(ctx, tenantID, req.Email)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to check user existence")
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, user.ErrEmailAlreadyTaken
	}

	// Create user domain entity
	u, err := user.New(tenantID, req.Email, req.Password, req.FirstName, req.LastName, user.UserRole(req.Role))
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repo.Create(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
		"email":     u.Email,
		"role":      u.Role,
	}).Info("User created successfully")

	return ToResponse(u), nil
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(ctx context.Context, tenantID, userID string) (*UserResponse, error) {
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	return ToResponse(u), nil
}

// GetByEmail retrieves a user by email
func (s *Service) GetByEmail(ctx context.Context, tenantID, email string) (*UserResponse, error) {
	u, err := s.repo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	return ToResponse(u), nil
}

// List retrieves all users with pagination
func (s *Service) List(ctx context.Context, tenantID string, page, perPage int, role, status, search string) (*UserListResponse, error) {
	// Set defaults
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Build filters
	filters := userRepo.ListFilters{
		Limit:     perPage,
		Offset:    (page - 1) * perPage,
		Search:    search,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	if role != "" {
		r := user.UserRole(role)
		filters.Role = &r
	}

	if status != "" {
		st := user.UserStatus(status)
		filters.Status = &st
	}

	// Fetch from repository
	users, total, err := s.repo.List(ctx, tenantID, filters)
	if err != nil {
		s.logger.ErrorWithErr(err, "failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return ToResponseList(users, total, page, perPage), nil
}

// Update updates a user
func (s *Service) Update(ctx context.Context, tenantID, userID string, req *UpdateUserRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Get existing user
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// Update profile
	u.UpdateProfile(req.FirstName, req.LastName, req.Phone, req.Timezone, req.Locale)

	// Save to repository
	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to update user")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
	}).Info("User updated successfully")

	return ToResponse(u), nil
}

// UpdatePassword updates user password
func (s *Service) UpdatePassword(ctx context.Context, tenantID, userID string, req *UpdatePasswordRequest) error {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return err
	}

	// Get user
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	// Check current password
	if !u.CheckPassword(req.CurrentPassword) {
		return user.ErrInvalidCredentials
	}

	// Update password
	if err := u.UpdatePassword(req.NewPassword); err != nil {
		return err
	}

	// Save to repository
	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to update password")
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
	}).Info("User password updated")

	return nil
}

// UpdateRole updates user role (admin only)
func (s *Service) UpdateRole(ctx context.Context, tenantID, userID string, req *UpdateRoleRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Get user
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// Prevent changing owner role
	if u.IsOwner() {
		return nil, user.ErrCannotModifyOwner
	}

	// Update role
	if err := u.UpdateRole(user.UserRole(req.Role)); err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to update role")
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
		"new_role":  req.Role,
	}).Info("User role updated")

	return ToResponse(u), nil
}

// Suspend suspends a user
func (s *Service) Suspend(ctx context.Context, tenantID, userID, reason string) error {
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	// Prevent suspending owner
	if u.IsOwner() {
		return user.ErrCannotModifyOwner
	}

	if err := u.Suspend(reason); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to suspend user")
		return fmt.Errorf("failed to suspend user: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
		"reason":    reason,
	}).Warn("User suspended")

	return nil
}

// Activate activates a user
func (s *Service) Activate(ctx context.Context, tenantID, userID string) error {
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	if err := u.Activate(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to activate user")
		return fmt.Errorf("failed to activate user: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
	}).Info("User activated")

	return nil
}

// VerifyEmail verifies user email
func (s *Service) VerifyEmail(ctx context.Context, tenantID, userID string) error {
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	if err := u.VerifyEmail(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.ErrorWithErr(err, "failed to verify email")
		return fmt.Errorf("failed to verify email: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
	}).Info("User email verified")

	return nil
}

// Delete soft deletes a user
func (s *Service) Delete(ctx context.Context, tenantID, userID string) error {
	// Check if user is owner
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	if u.IsOwner() {
		return user.ErrCannotModifyOwner
	}

	if err := s.repo.Delete(ctx, tenantID, userID); err != nil {
		s.logger.ErrorWithErr(err, "failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"user_id":   userID,
		"tenant_id": tenantID,
	}).Warn("User deleted")

	return nil
}

// GetStats retrieves user statistics for a tenant
func (s *Service) GetStats(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total users
	total, err := s.repo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	stats["total"] = total

	// Count by role
	for _, role := range []user.UserRole{user.RoleOwner, user.RoleAdmin, user.RoleEditor, user.RoleViewer} {
		count, err := s.repo.CountByRole(ctx, tenantID, role)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}
		stats[string(role)] = count
	}

	// Count by status
	for _, status := range []user.UserStatus{user.UserStatusActive, user.UserStatusInactive, user.UserStatusSuspended, user.UserStatusPending} {
		count, err := s.repo.CountByStatus(ctx, tenantID, status)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}
		stats[string(status)] = count
	}

	return stats, nil
}

// generateToken generates JWT token for user
func (s *Service) generateToken(u *user.User) (string, int64, error) {
	expiresIn := int64(24 * 60 * 60) // 24 hours
	expirationTime := time.Now().Add(time.Duration(expiresIn) * time.Second)

	claims := &jwt.MapClaims{
		"user_id":   u.ID,
		"tenant_id": u.TenantID,
		"email":     u.Email,
		"role":      u.Role,
		"exp":       expirationTime.Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresIn, nil
}
