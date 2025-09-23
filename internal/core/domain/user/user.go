package user

import (
	"github.com/google/uuid"
	"time"
)

// User represents a user in the system
type Status string

const (
	StatusPending  Status = "pending"
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Email     string    `json:"email" db:"email"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Password  string    `json:"-" db:"password_hash"` // Never expose in JSON
	Role      string    `json:"role" db:"role"`
	Status    Status    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest represents the data needed to create a new user
type CreateUserRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	Email     string    `json:"email" validate:"required,email"`
	FirstName string    `json:"first_name" validate:"required"`
	LastName  string    `json:"last_name" validate:"required"`
	Password  string    `json:"password" validate:"required,min=8"`
	Role      string    `json:"role" validate:"required,oneof=admin user"`
}

// UpdateUserRequest represents the data that can be updated for a user
type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Role      *string `json:"role,omitempty" validate:"omitempty,oneof=admin user"`
	Status    *Status `json:"status,omitempty"`
}

// UserResponse represents the user data returned in API responses
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a User domain object to a UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		TenantID:  u.TenantID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// Deactivate marks a user as inactive.
func (u *User) Deactivate(reason string) error {
	u.Status = StatusInactive
	u.UpdatedAt = time.Now()
	return nil
}

// CanManageUsers checks if the user can manage other users.
func (u *User) CanManageUsers() bool {
	return u.IsAdmin()
}

// CanManageBilling checks if the user can manage billing and subscriptions.
func (u *User) CanManageBilling() bool {
	return u.IsAdmin()
}

// CanCreateTemplates checks if the user can create templates.
func (u *User) CanCreateTemplates() bool {
	return u.IsAdmin()
}

// CanPublishTemplates checks if the user can publish templates.
func (u *User) CanPublishTemplates() bool {
	return u.IsAdmin()
}

// CanInstallTemplates checks if the user can install templates.
func (u *User) CanInstallTemplates() bool {
	return u.IsAdmin()
}

// NewUser creates a new User instance
func NewUser(req CreateUserRequest, hashedPassword string) *User {
	return &User{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  hashedPassword,
		Role:      req.Role,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
