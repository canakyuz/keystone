package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	Role      string     `json:"role" db:"role"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	LastLogin *time.Time `json:"last_login" db:"last_login"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty"`
}

// UserRole represents available user roles
type UserRole string

const (
	RoleOwner     UserRole = "owner"
	RoleAdmin     UserRole = "admin"
	RoleEditor    UserRole = "editor"
	RoleViewer    UserRole = "viewer"
	RoleDeveloper UserRole = "developer"
)

// UserInvitation represents pending user invitations
type UserInvitation struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Email     string    `json:"email" db:"email"`
	Role      string    `json:"role" db:"role"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedBy uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Relations
	Tenant    *Tenant `json:"tenant,omitempty"`
	InvitedBy *User   `json:"invited_by,omitempty"`
}

// UserSession represents user sessions
type UserSession struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	RefreshToken string    `json:"-" db:"refresh_token"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`

	// Relations
	User   *User   `json:"user,omitempty"`
	Tenant *Tenant `json:"tenant,omitempty"`
}

// UserPermission represents granular permissions
type UserPermission struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Resource  string     `json:"resource" db:"resource"`   // e.g., "templates", "users", "billing"
	Action    string     `json:"action" db:"action"`       // e.g., "read", "write", "delete", "admin"
	Condition *string    `json:"condition" db:"condition"` // JSON string for ABAC conditions
	GrantedBy uuid.UUID  `json:"granted_by" db:"granted_by"`
	GrantedAt time.Time  `json:"granted_at" db:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`

	// Relations
	User          *User   `json:"user,omitempty"`
	Tenant        *Tenant `json:"tenant,omitempty"`
	GrantedByUser *User   `json:"granted_by_user,omitempty"`
}

// Permission actions
const (
	ActionRead    = "read"
	ActionWrite   = "write"
	ActionDelete  = "delete"
	ActionAdmin   = "admin"
	ActionInstall = "install"
	ActionPublish = "publish"
)

// Resources
const (
	ResourceTenants     = "tenants"
	ResourceUsers       = "users"
	ResourceTemplates   = "templates"
	ResourceModules     = "modules"
	ResourceBilling     = "billing"
	ResourceAnalytics   = "analytics"
	ResourceSettings    = "settings"
	ResourceInvitations = "invitations"
)

// HasPermission checks if user has specific permission
func (u *User) HasPermission(resource, action string) bool {
	// Owner has all permissions
	if u.Role == string(RoleOwner) {
		return true
	}

	// Admin has most permissions except billing for non-owners
	if u.Role == string(RoleAdmin) {
		if resource == ResourceBilling && action == ActionAdmin {
			return false
		}
		return true
	}

	// Role-based permissions
	switch u.Role {
	case string(RoleDeveloper):
		return resource == ResourceTemplates || resource == ResourceModules
	case string(RoleEditor):
		return action == ActionRead || action == ActionWrite
	case string(RoleViewer):
		return action == ActionRead
	}

	return false
}

// IsOwner checks if user is tenant owner
func (u *User) IsOwner() bool {
	return u.Role == string(RoleOwner)
}

// IsAdmin checks if user is admin or owner
func (u *User) IsAdmin() bool {
	return u.Role == string(RoleAdmin) || u.Role == string(RoleOwner)
}
