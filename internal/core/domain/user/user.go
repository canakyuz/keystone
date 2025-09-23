package user

import (
	"strings"
	"time"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
)

// UserID represents a unique user identifier
type UserID = shared.ID

// User represents a user in the system
type User struct {
	id        UserID
	tenantID  tenant.TenantID
	email     shared.Email
	name      string
	role      Role
	status    shared.Status
	metadata  map[string]interface{}
	lastLogin *shared.Timestamp
	createdAt shared.Timestamp
	updatedAt shared.Timestamp
	version   int64 // For optimistic locking
}

// Role represents user role with permissions
type Role string

const (
	RoleOwner     Role = "owner"     // Full control, billing access
	RoleAdmin     Role = "admin"     // Almost full control, no billing
	RoleEditor    Role = "editor"    // Can edit content and templates
	RoleViewer    Role = "viewer"    // Read-only access
	RoleDeveloper Role = "developer" // Template development access
	RoleMember    Role = "member"    // Basic member access
)

// IsValid checks if role is valid
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleEditor, RoleViewer, RoleDeveloper, RoleMember:
		return true
	default:
		return false
	}
}

// String returns string representation
func (r Role) String() string {
	return string(r)
}

// UserParams contains parameters for creating a new user
type UserParams struct {
	TenantID tenant.TenantID
	Email    string
	Name     string
	Role     Role
	Metadata map[string]interface{}
}

// NewUser creates a new user entity
func NewUser(params UserParams) (*User, error) {
	// Validate required fields
	if params.TenantID.IsZero() {
		return nil, shared.NewValidationError("tenant ID is required")
	}

	// Validate and create email value object
	email, err := shared.NewEmail(params.Email)
	if err != nil {
		return nil, shared.WrapDomainError(err, shared.ValidationError, "invalid email")
	}

	// Validate name
	if strings.TrimSpace(params.Name) == "" {
		return nil, shared.NewValidationError("user name is required")
	}

	if len(params.Name) > 255 {
		return nil, shared.NewValidationError("user name must be 255 characters or less")
	}

	// Validate role
	if !params.Role.IsValid() {
		return nil, shared.NewValidationError("invalid user role")
	}

	// Initialize metadata if nil
	if params.Metadata == nil {
		params.Metadata = make(map[string]interface{})
	}

	now := shared.Now()

	return &User{
		id:        shared.NewID(),
		tenantID:  params.TenantID,
		email:     email,
		name:      strings.TrimSpace(params.Name),
		role:      params.Role,
		status:    shared.StatusActive,
		metadata:  params.Metadata,
		lastLogin: nil,
		createdAt: now,
		updatedAt: now,
		version:   1,
	}, nil
}

// ReconstituteUser recreates a user from stored data (for repository pattern)
func ReconstituteUser(
	id UserID,
	tenantID tenant.TenantID,
	email shared.Email,
	name string,
	role Role,
	status shared.Status,
	metadata map[string]interface{},
	lastLogin *shared.Timestamp,
	createdAt shared.Timestamp,
	updatedAt shared.Timestamp,
	version int64,
) *User {
	return &User{
		id:        id,
		tenantID:  tenantID,
		email:     email,
		name:      name,
		role:      role,
		status:    status,
		metadata:  metadata,
		lastLogin: lastLogin,
		createdAt: createdAt,
		updatedAt: updatedAt,
		version:   version,
	}
}

// Getters
func (u *User) ID() UserID {
	return u.id
}

func (u *User) TenantID() tenant.TenantID {
	return u.tenantID
}

func (u *User) Email() shared.Email {
	return u.email
}

func (u *User) Name() string {
	return u.name
}

func (u *User) Role() Role {
	return u.role
}

func (u *User) Status() shared.Status {
	return u.status
}

func (u *User) Metadata() map[string]interface{} {
	return u.metadata
}

func (u *User) LastLogin() *shared.Timestamp {
	return u.lastLogin
}

func (u *User) CreatedAt() shared.Timestamp {
	return u.createdAt
}

func (u *User) UpdatedAt() shared.Timestamp {
	return u.updatedAt
}

func (u *User) Version() int64 {
	return u.version
}

// Business Methods

// UpdateName updates the user's name
func (u *User) UpdateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return shared.NewValidationError("user name cannot be empty")
	}

	if len(name) > 255 {
		return shared.NewValidationError("user name must be 255 characters or less")
	}

	u.name = strings.TrimSpace(name)
	u.updatedAt = shared.Now()
	u.version++

	return nil
}

// UpdateEmail updates the user's email
func (u *User) UpdateEmail(emailStr string) error {
	email, err := shared.NewEmail(emailStr)
	if err != nil {
		return shared.WrapDomainError(err, shared.ValidationError, "invalid email")
	}

	u.email = email
	u.updatedAt = shared.Now()
	u.version++

	return nil
}

// UpdateRole updates the user's role
func (u *User) UpdateRole(role Role) error {
	if !role.IsValid() {
		return shared.NewValidationError("invalid user role")
	}

	// Business rule: Cannot change owner role
	if u.role == RoleOwner && role != RoleOwner {
		return shared.NewBusinessRuleError("cannot change owner role")
	}

	// Business rule: Only owner can assign owner role
	if role == RoleOwner && u.role != RoleOwner {
		return shared.NewBusinessRuleError("only current owner can assign owner role")
	}

	u.role = role
	u.updatedAt = shared.Now()
	u.version++

	return nil
}

// Activate activates the user
func (u *User) Activate() error {
	if u.status == shared.StatusDeleted {
		return shared.NewBusinessRuleError("cannot activate deleted user")
	}

	if u.status == shared.StatusActive {
		return nil // Already active
	}

	u.status = shared.StatusActive
	u.updatedAt = shared.Now()
	u.version++

	return nil
}

// Deactivate deactivates the user
func (u *User) Deactivate() error {
	if u.status == shared.StatusDeleted {
		return shared.NewBusinessRuleError("cannot deactivate deleted user")
	}

	// Business rule: Cannot deactivate the only owner
	if u.role == RoleOwner {
		return shared.NewBusinessRuleError("cannot deactivate tenant owner")
	}

	u.status = shared.StatusInactive
	u.updatedAt = shared.Now()
	u.version++

	return nil
}

// Delete marks the user as deleted (soft delete)
func (u *User) Delete() error {
	if u.status == shared.StatusDeleted {
		return nil // Already deleted
	}

	// Business rule: Cannot delete the only owner
	if u.role == RoleOwner {
		return shared.NewBusinessRuleError("cannot delete tenant owner")
	}

	u.status = shared.StatusDeleted
	u.updatedAt = shared.Now()
	u.version++

	return nil
}

// RecordLogin records user login timestamp
func (u *User) RecordLogin() {
	now := shared.Now()
	u.lastLogin = &now
	u.updatedAt = now
	u.version++
}

// SetMetadata sets metadata key-value pair
func (u *User) SetMetadata(key string, value interface{}) {
	if u.metadata == nil {
		u.metadata = make(map[string]interface{})
	}

	u.metadata[key] = value
	u.updatedAt = shared.Now()
	u.version++
}

// GetMetadata retrieves metadata value
func (u *User) GetMetadata(key string) (interface{}, bool) {
	value, exists := u.metadata[key]
	return value, exists
}

// Business Logic Queries

// IsActive checks if user is active
func (u *User) IsActive() bool {
	return u.status == shared.StatusActive
}

// IsDeleted checks if user is deleted
func (u *User) IsDeleted() bool {
	return u.status == shared.StatusDeleted
}

// IsOwner checks if user is tenant owner
func (u *User) IsOwner() bool {
	return u.role == RoleOwner
}

// IsAdmin checks if user is admin or owner
func (u *User) IsAdmin() bool {
	return u.role == RoleAdmin || u.role == RoleOwner
}

// CanAccessResource checks if user can access a resource with given action
func (u *User) CanAccessResource(resource string, action string) bool {
	if !u.IsActive() {
		return false
	}

	// Owner has access to everything
	if u.IsOwner() {
		return true
	}

	// Admin has most permissions
	if u.IsAdmin() {
		// Admin cannot access billing resources
		if resource == "billing" || resource == "subscription" {
			return false
		}
		return true
	}

	// Role-based permissions
	switch u.role {
	case RoleDeveloper:
		return isDeveloperResource(resource) && isDeveloperAction(action)
	case RoleEditor:
		return isEditorResource(resource) && isEditorAction(action)
	case RoleViewer:
		return isViewerResource(resource) && action == "read"
	case RoleMember:
		return isMemberResource(resource) && isMemberAction(action)
	default:
		return false
	}
}

// Permission helper functions
func isDeveloperResource(resource string) bool {
	developerResources := []string{
		"templates", "modules", "components", "themes",
		"api", "webhooks", "integrations",
	}

	for _, r := range developerResources {
		if resource == r {
			return true
		}
	}
	return false
}

func isDeveloperAction(action string) bool {
	developerActions := []string{"read", "write", "create", "update", "delete", "publish"}

	for _, a := range developerActions {
		if action == a {
			return true
		}
	}
	return false
}

func isEditorResource(resource string) bool {
	editorResources := []string{
		"content", "pages", "posts", "media",
		"templates", "basic_settings",
	}

	for _, r := range editorResources {
		if resource == r {
			return true
		}
	}
	return false
}

func isEditorAction(action string) bool {
	editorActions := []string{"read", "write", "create", "update"}

	for _, a := range editorActions {
		if action == a {
			return true
		}
	}
	return false
}

func isViewerResource(resource string) bool {
	// Viewers can read most resources except sensitive ones
	restrictedResources := []string{
		"billing", "subscription", "users", "settings",
		"integrations", "api", "webhooks",
	}

	for _, r := range restrictedResources {
		if resource == r {
			return false
		}
	}
	return true
}

func isMemberResource(resource string) bool {
	memberResources := []string{
		"dashboard", "profile", "notifications",
		"content", "basic_templates",
	}

	for _, r := range memberResources {
		if resource == r {
			return true
		}
	}
	return false
}

func isMemberAction(action string) bool {
	memberActions := []string{"read", "write"}

	for _, a := range memberActions {
		if action == a {
			return true
		}
	}
	return false
}

// HasRecentLogin checks if user has logged in recently
func (u *User) HasRecentLogin(hours int) bool {
	if u.lastLogin == nil {
		return false
	}

	threshold := time.Now().Add(-time.Duration(hours) * time.Hour)
	return u.lastLogin.Time().After(threshold)
}

// Events

// UserCreatedEvent represents a user creation event
type UserCreatedEvent struct {
	UserID    UserID
	TenantID  tenant.TenantID
	Email     string
	Name      string
	Role      string
	CreatedAt time.Time
}

// UserUpdatedEvent represents a user update event
type UserUpdatedEvent struct {
	UserID    UserID
	TenantID  tenant.TenantID
	Name      string
	UpdatedAt time.Time
	Version   int64
}

// UserRoleChangedEvent represents a user role change event
type UserRoleChangedEvent struct {
	UserID    UserID
	TenantID  tenant.TenantID
	OldRole   Role
	NewRole   Role
	ChangedAt time.Time
}

// UserLoginEvent represents a user login event
type UserLoginEvent struct {
	UserID    UserID
	TenantID  tenant.TenantID
	LoginAt   time.Time
	IPAddress string
	UserAgent string
}

// GenerateCreatedEvent generates a user created event
func (u *User) GenerateCreatedEvent() UserCreatedEvent {
	return UserCreatedEvent{
		UserID:    u.id,
		TenantID:  u.tenantID,
		Email:     u.email.String(),
		Name:      u.name,
		Role:      u.role.String(),
		CreatedAt: u.createdAt.Time(),
	}
}

// GenerateUpdatedEvent generates a user updated event
func (u *User) GenerateUpdatedEvent() UserUpdatedEvent {
	return UserUpdatedEvent{
		UserID:    u.id,
		TenantID:  u.tenantID,
		Name:      u.name,
		UpdatedAt: u.updatedAt.Time(),
		Version:   u.version,
	}
}

// GenerateRoleChangedEvent generates a role changed event
func (u *User) GenerateRoleChangedEvent(oldRole Role) UserRoleChangedEvent {
	return UserRoleChangedEvent{
		UserID:    u.id,
		TenantID:  u.tenantID,
		OldRole:   oldRole,
		NewRole:   u.role,
		ChangedAt: u.updatedAt.Time(),
	}
}

// GenerateLoginEvent generates a login event
func (u *User) GenerateLoginEvent(ipAddress, userAgent string) UserLoginEvent {
	return UserLoginEvent{
		UserID:    u.id,
		TenantID:  u.tenantID,
		LoginAt:   u.lastLogin.Time(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}
