package user

import (
	"time"

	"nexpaces-api/internal/domain/user"
)

// RegisterRequest represents user registration request
type RegisterRequest struct {
	TenantID  string `json:"tenant_id" validate:"required,uuid"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TenantID string `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
}

// CreateUserRequest represents request to create a user
type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	Role      string `json:"role" validate:"required,oneof=owner admin editor viewer"`
}

// UpdateUserRequest represents request to update a user
type UpdateUserRequest struct {
	FirstName string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Phone     string `json:"phone,omitempty" validate:"omitempty,max=20"`
	Timezone  string `json:"timezone,omitempty" validate:"omitempty,max=50"`
	Locale    string `json:"locale,omitempty" validate:"omitempty,max=10"`
}

// UpdatePasswordRequest represents request to update password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
}

// UpdateRoleRequest represents request to update user role
type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=owner admin editor viewer"`
}

// UpdateStatusRequest represents request to update user status
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active inactive suspended pending"`
	Reason string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// UserResponse represents user response (without password)
type UserResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Authentication
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`

	// Profile
	Avatar   string `json:"avatar,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Locale   string `json:"locale,omitempty"`

	// Security
	TwoFactorEnabled  bool       `json:"two_factor_enabled"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`

	// Permissions
	Permissions []string `json:"permissions"`
}

// LoginResponse represents login response
type LoginResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	ExpiresIn    int64         `json:"expires_in"` // seconds
}

// UserListResponse represents paginated user list response
type UserListResponse struct {
	Data       []*UserResponse `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int             `json:"total_pages"`
}

// ToResponse converts domain user to response DTO
func ToResponse(u *user.User) *UserResponse {
	permissions := user.UserRole(u.Role).GetPermissions()

	return &UserResponse{
		ID:                u.ID,
		TenantID:          u.TenantID,
		Email:             u.Email,
		FirstName:         u.FirstName,
		LastName:          u.LastName,
		FullName:          u.FullName(),
		Role:              string(u.Role),
		Status:            string(u.Status),
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
		EmailVerified:     u.EmailVerified,
		EmailVerifiedAt:   u.EmailVerifiedAt,
		LastLoginAt:       u.LastLoginAt,
		Avatar:            u.Avatar,
		Phone:             u.Phone,
		Timezone:          u.Timezone,
		Locale:            u.Locale,
		TwoFactorEnabled:  u.TwoFactorEnabled,
		PasswordChangedAt: u.PasswordChangedAt,
		Permissions:       permissions,
	}
}

// ToResponseList converts domain users to response list
func ToResponseList(users []*user.User, total int64, page, perPage int) *UserListResponse {
	data := make([]*UserResponse, len(users))
	for i, u := range users {
		data[i] = ToResponse(u)
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &UserListResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}
