package repositories

import (
	"context"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *user.User) error

	// GetByID retrieves a user by ID within tenant context
	GetByID(ctx context.Context, tenantID tenant.TenantID, id user.UserID) (*user.User, error)

	// GetByEmail retrieves a user by email within tenant context
	GetByEmail(ctx context.Context, tenantID tenant.TenantID, email string) (*user.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *user.User) error

	// Delete deletes a user (soft delete)
	Delete(ctx context.Context, tenantID tenant.TenantID, id user.UserID) error

	// List retrieves users with filtering and pagination within tenant context
	List(ctx context.Context, tenantID tenant.TenantID, filter UserFilter) (*UserList, error)

	// ExistsByEmail checks if a user with the given email exists in tenant
	ExistsByEmail(ctx context.Context, tenantID tenant.TenantID, email string) (bool, error)

	// Count returns the total number of users matching the filter within tenant
	Count(ctx context.Context, tenantID tenant.TenantID, filter UserFilter) (int64, error)

	// GetByRole retrieves users by role within tenant
	GetByRole(ctx context.Context, tenantID tenant.TenantID, role user.Role) ([]*user.User, error)

	// GetOwners retrieves tenant owners
	GetOwners(ctx context.Context, tenantID tenant.TenantID) ([]*user.User, error)

	// GetAdmins retrieves tenant admins (including owners)
	GetAdmins(ctx context.Context, tenantID tenant.TenantID) ([]*user.User, error)

	// UpdateLastLogin updates user's last login timestamp
	UpdateLastLogin(ctx context.Context, tenantID tenant.TenantID, userID user.UserID) error

	// GetActiveUsersCount returns count of active users in tenant
	GetActiveUsersCount(ctx context.Context, tenantID tenant.TenantID) (int64, error)

	// GetRecentlyActive retrieves recently active users
	GetRecentlyActive(ctx context.Context, tenantID tenant.TenantID, hours int) ([]*user.User, error)

	// BulkUpdateRole updates role for multiple users
	BulkUpdateRole(ctx context.Context, tenantID tenant.TenantID, userIDs []user.UserID, role user.Role) error

	// GetUserPermissions retrieves user's permissions (if using granular permissions)
	GetUserPermissions(ctx context.Context, tenantID tenant.TenantID, userID user.UserID) ([]Permission, error)

	// GrantPermission grants a permission to user
	GrantPermission(ctx context.Context, permission Permission) error

	// RevokePermission revokes a permission from user
	RevokePermission(ctx context.Context, tenantID tenant.TenantID, userID user.UserID, resource, action string) error

	// Cross-tenant operations (for system admins only)

	// GetUserAcrossTenantsForAuth retrieves user across all tenants for authentication
	GetUserAcrossTenantsForAuth(ctx context.Context, email string) ([]*user.User, error)

	// GetUserTenantsCount returns number of tenants user belongs to
	GetUserTenantsCount(ctx context.Context, email string) (int64, error)
}

// UserFilter represents filtering options for user queries
type UserFilter struct {
	// Status filtering
	Status *shared.Status

	// Role filtering
	Role *user.Role

	// Search term (searches in name and email)
	Search string

	// Created date range
	CreatedAfter  *shared.Timestamp
	CreatedBefore *shared.Timestamp

	// Last login range
	LastLoginAfter  *shared.Timestamp
	LastLoginBefore *shared.Timestamp

	// Active users only
	ActiveOnly bool

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // name, email, created_at, last_login, role
	SortOrder string // asc, desc

	// Include related data
	IncludePermissions bool
	IncludeMetadata    bool
}

// UserList represents a paginated list of users
type UserList struct {
	Items      []*user.User `json:"items"`
	Total      int64        `json:"total"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
	HasMore    bool         `json:"has_more"`
	TotalPages int          `json:"total_pages"`
}

// Permission represents a granular permission
type Permission struct {
	ID        shared.ID         `json:"id"`
	UserID    user.UserID       `json:"user_id"`
	TenantID  tenant.TenantID   `json:"tenant_id"`
	Resource  string            `json:"resource"`  // templates, users, billing, etc.
	Action    string            `json:"action"`    // read, write, delete, admin
	Condition *string           `json:"condition"` // JSON string for ABAC conditions
	GrantedBy user.UserID       `json:"granted_by"`
	GrantedAt shared.Timestamp  `json:"granted_at"`
	ExpiresAt *shared.Timestamp `json:"expires_at"`
}

// UserInvitation represents a user invitation
type UserInvitation struct {
	ID         shared.ID         `json:"id"`
	TenantID   tenant.TenantID   `json:"tenant_id"`
	Email      string            `json:"email"`
	Role       user.Role         `json:"role"`
	Token      string            `json:"token"`
	ExpiresAt  shared.Timestamp  `json:"expires_at"`
	CreatedBy  user.UserID       `json:"created_by"`
	CreatedAt  shared.Timestamp  `json:"created_at"`
	AcceptedAt *shared.Timestamp `json:"accepted_at"`
	Status     InvitationStatus  `json:"status"`
}

// InvitationStatus represents invitation status
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationExpired  InvitationStatus = "expired"
	InvitationRevoked  InvitationStatus = "revoked"
)

// UserInvitationRepository defines the interface for user invitation data access
type UserInvitationRepository interface {
	// Create creates a new invitation
	Create(ctx context.Context, invitation *UserInvitation) error

	// GetByToken retrieves invitation by token
	GetByToken(ctx context.Context, token string) (*UserInvitation, error)

	// GetByEmail retrieves pending invitations for email
	GetByEmail(ctx context.Context, email string) ([]*UserInvitation, error)

	// GetByTenant retrieves invitations for tenant
	GetByTenant(ctx context.Context, tenantID tenant.TenantID) ([]*UserInvitation, error)

	// Update updates invitation status
	Update(ctx context.Context, invitation *UserInvitation) error

	// Delete deletes invitation
	Delete(ctx context.Context, id shared.ID) error

	// ExpireOldInvitations marks old invitations as expired
	ExpireOldInvitations(ctx context.Context) error

	// GetPendingCount returns count of pending invitations for tenant
	GetPendingCount(ctx context.Context, tenantID tenant.TenantID) (int64, error)
}

// Validation methods
func (f *UserFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 1000 {
		return shared.NewValidationError("limit must be between 0 and 1000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	if f.SortBy != "" {
		validSortFields := []string{"name", "email", "created_at", "last_login", "role", "status"}
		valid := false
		for _, field := range validSortFields {
			if f.SortBy == field {
				valid = true
				break
			}
		}
		if !valid {
			return shared.NewValidationError("invalid sort field")
		}
	}

	if f.SortOrder != "" && f.SortOrder != "asc" && f.SortOrder != "desc" {
		return shared.NewValidationError("sort order must be 'asc' or 'desc'")
	}

	return nil
}

// Apply default values
func (f *UserFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	if f.SortBy == "" {
		f.SortBy = "created_at"
	}

	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}

// Permission helper methods
func (p *Permission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return shared.Now().After(*p.ExpiresAt)
}

func (p *Permission) IsValid() bool {
	return !p.IsExpired()
}

// Invitation helper methods
func (i *UserInvitation) IsExpired() bool {
	return shared.Now().After(i.ExpiresAt)
}

func (i *UserInvitation) IsPending() bool {
	return i.Status == InvitationPending && !i.IsExpired()
}

func (i *UserInvitation) Accept() error {
	if !i.IsPending() {
		return shared.NewBusinessRuleError("invitation is not pending or has expired")
	}

	now := shared.Now()
	i.Status = InvitationAccepted
	i.AcceptedAt = &now

	return nil
}

func (i *UserInvitation) Revoke() error {
	if i.Status != InvitationPending {
		return shared.NewBusinessRuleError("only pending invitations can be revoked")
	}

	i.Status = InvitationRevoked
	return nil
}
