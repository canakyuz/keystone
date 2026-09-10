package user

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusPending   UserStatus = "pending" // Email not verified
)

// UserRole represents user roles within a tenant
type UserRole string

const (
	RoleOwner  UserRole = "owner"  // Tenant owner (full access)
	RoleAdmin  UserRole = "admin"  // Administrator
	RoleEditor UserRole = "editor" // Can edit content
	RoleViewer UserRole = "viewer" // Read-only access
)

// User represents a user in the system
type User struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"` // CRITICAL: Multi-tenant isolation
	Email     string     `json:"email"`
	Password  string     `json:"-"` // Never serialize password
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Role      UserRole   `json:"role"`
	Status    UserStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

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

	// Metadata
	Preferences map[string]interface{} `json:"preferences,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// New creates a new user with hashed password
func New(tenantID, email, password, firstName, lastName string, role UserRole) (*User, error) {
	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		Email:            email,
		Password:         hashedPassword,
		FirstName:        firstName,
		LastName:         lastName,
		Role:             role,
		Status:           UserStatusPending, // Requires email verification
		CreatedAt:        now,
		UpdatedAt:        now,
		EmailVerified:    false,
		TwoFactorEnabled: false,
		Preferences:      make(map[string]interface{}),
		Metadata:         make(map[string]interface{}),
	}

	// Validate user
	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate validates user data
func (u *User) Validate() error {
	if u.ID == "" {
		return ErrInvalidUserID
	}

	if u.TenantID == "" {
		return ErrTenantIDRequired
	}

	if u.Email == "" {
		return ErrEmailRequired
	}

	if len(u.Email) < 3 || len(u.Email) > 255 {
		return ErrInvalidEmail
	}

	if u.Password == "" {
		return ErrPasswordRequired
	}

	if u.FirstName == "" {
		return ErrFirstNameRequired
	}

	if u.LastName == "" {
		return ErrLastNameRequired
	}

	if !u.Role.IsValid() {
		return ErrInvalidRole
	}

	if !u.Status.IsValid() {
		return ErrInvalidStatus
	}

	return nil
}

// HashPassword hashes a plain text password using bcrypt
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}

	if len(password) > 72 {
		return "", ErrPasswordTooLong
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// CheckPassword verifies if the provided password matches the hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// UpdatePassword updates the user's password
func (u *User) UpdatePassword(newPassword string) error {
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	u.Password = hashedPassword
	now := time.Now()
	u.PasswordChangedAt = &now
	u.UpdatedAt = now

	return nil
}

// IsActive checks if the user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsSuspended checks if the user is suspended
func (u *User) IsSuspended() bool {
	return u.Status == UserStatusSuspended
}

// IsPending checks if the user is pending email verification
func (u *User) IsPending() bool {
	return u.Status == UserStatusPending
}

// IsEmailVerified checks if email is verified
func (u *User) IsEmailVerified() bool {
	return u.EmailVerified
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role UserRole) bool {
	return u.Role == role
}

// IsOwner checks if user is the tenant owner
func (u *User) IsOwner() bool {
	return u.Role == RoleOwner
}

// IsAdmin checks if user is an administrator
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleOwner
}

// CanEdit checks if user can edit content
func (u *User) CanEdit() bool {
	return u.Role == RoleEditor || u.IsAdmin()
}

// CanView checks if user can view content
func (u *User) CanView() bool {
	return true // All authenticated users can view
}

// VerifyEmail marks the user's email as verified
func (u *User) VerifyEmail() error {
	if u.IsEmailVerified() {
		return ErrEmailAlreadyVerified
	}

	now := time.Now()
	u.EmailVerified = true
	u.EmailVerifiedAt = &now
	u.Status = UserStatusActive // Activate user after email verification
	u.UpdatedAt = now

	return nil
}

// Suspend suspends the user account
func (u *User) Suspend(reason string) error {
	if u.IsSuspended() {
		return ErrUserAlreadySuspended
	}

	u.Status = UserStatusSuspended
	u.UpdatedAt = time.Now()

	if u.Metadata == nil {
		u.Metadata = make(map[string]interface{})
	}
	u.Metadata["suspension_reason"] = reason
	u.Metadata["suspended_at"] = time.Now()

	return nil
}

// Activate activates the user account
func (u *User) Activate() error {
	if u.IsActive() {
		return ErrUserAlreadyActive
	}

	u.Status = UserStatusActive
	u.UpdatedAt = time.Now()

	if u.Metadata != nil {
		delete(u.Metadata, "suspension_reason")
		delete(u.Metadata, "suspended_at")
	}

	return nil
}

// UpdateLastLogin updates the user's last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// UpdateProfile updates user profile information
func (u *User) UpdateProfile(firstName, lastName, phone, timezone, locale string) {
	if firstName != "" {
		u.FirstName = firstName
	}
	if lastName != "" {
		u.LastName = lastName
	}
	if phone != "" {
		u.Phone = phone
	}
	if timezone != "" {
		u.Timezone = timezone
	}
	if locale != "" {
		u.Locale = locale
	}
	u.UpdatedAt = time.Now()
}

// UpdateAvatar updates user avatar
func (u *User) UpdateAvatar(avatarURL string) {
	u.Avatar = avatarURL
	u.UpdatedAt = time.Now()
}

// UpdateRole updates user role
func (u *User) UpdateRole(newRole UserRole) error {
	if !newRole.IsValid() {
		return ErrInvalidRole
	}

	if u.Role == newRole {
		return ErrSameRole
	}

	u.Role = newRole
	u.UpdatedAt = time.Now()

	return nil
}

// RecordLogin records user login activity
func (u *User) RecordLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// EnableTwoFactor enables two-factor authentication
func (u *User) EnableTwoFactor() {
	u.TwoFactorEnabled = true
	u.UpdatedAt = time.Now()
}

// DisableTwoFactor disables two-factor authentication
func (u *User) DisableTwoFactor() {
	u.TwoFactorEnabled = false
	u.UpdatedAt = time.Now()
}

// UpdatePreference updates a user preference
func (u *User) UpdatePreference(key string, value interface{}) {
	if u.Preferences == nil {
		u.Preferences = make(map[string]interface{})
	}
	u.Preferences[key] = value
	u.UpdatedAt = time.Now()
}

// GetPreference retrieves a user preference
func (u *User) GetPreference(key string) (interface{}, bool) {
	if u.Preferences == nil {
		return nil, false
	}
	value, exists := u.Preferences[key]
	return value, exists
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// IsValid checks if user status is valid
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusSuspended, UserStatusPending:
		return true
	default:
		return false
	}
}

// IsValid checks if user role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

// GetPermissions returns permissions for the role
func (r UserRole) GetPermissions() []string {
	switch r {
	case RoleOwner:
		return []string{
			"tenant:manage",
			"users:manage",
			"websites:manage",
			"templates:manage",
			"billing:manage",
			"settings:manage",
			"content:write",
			"content:read",
		}
	case RoleAdmin:
		return []string{
			"users:manage",
			"websites:manage",
			"templates:manage",
			"settings:manage",
			"content:write",
			"content:read",
		}
	case RoleEditor:
		return []string{
			"websites:write",
			"templates:write",
			"content:write",
			"content:read",
		}
	case RoleViewer:
		return []string{
			"content:read",
		}
	default:
		return []string{}
	}
}

// HasPermission checks if role has a specific permission
func (r UserRole) HasPermission(permission string) bool {
	permissions := r.GetPermissions()
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
