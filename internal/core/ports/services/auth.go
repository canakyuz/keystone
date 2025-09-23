package services

import (
	"context"
	"time"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// ValidateToken validates JWT token and returns user context
	ValidateToken(ctx context.Context, token string) (*AuthContext, error)

	// RefreshToken refreshes JWT token
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

	// GenerateTokens generates access and refresh tokens for user
	GenerateTokens(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) (*TokenPair, error)

	// RevokeToken revokes token (logout)
	RevokeToken(ctx context.Context, token string) error

	// RevokeAllTokens revokes all tokens for user (logout from all devices)
	RevokeAllTokens(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) error

	// ValidatePermission validates user permission for resource and action
	ValidatePermission(ctx context.Context, authCtx *AuthContext, resource, action string) error

	// GetUserSessions returns active sessions for user
	GetUserSessions(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) ([]*UserSession, error)

	// RevokeSession revokes specific session
	RevokeSession(ctx context.Context, sessionID string) error

	// GetSessionInfo returns session information
	GetSessionInfo(ctx context.Context, sessionID string) (*UserSession, error)

	// GenerateAPIKey generates API key for user
	GenerateAPIKey(ctx context.Context, userID user.UserID, tenantID tenant.TenantID, name string, scopes []string) (*APIKey, error)

	// ValidateAPIKey validates API key
	ValidateAPIKey(ctx context.Context, apiKey string) (*AuthContext, error)

	// RevokeAPIKey revokes API key
	RevokeAPIKey(ctx context.Context, keyID string) error

	// ListAPIKeys lists user's API keys
	ListAPIKeys(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) ([]*APIKey, error)
}

// TwoFactorService defines the interface for 2FA operations
type TwoFactorService interface {
	// GenerateSecret generates 2FA secret for user
	GenerateSecret(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) (*TwoFactorSecret, error)

	// EnableTwoFactor enables 2FA for user
	EnableTwoFactor(ctx context.Context, userID user.UserID, tenantID tenant.TenantID, code string) error

	// DisableTwoFactor disables 2FA for user
	DisableTwoFactor(ctx context.Context, userID user.UserID, tenantID tenant.TenantID, code string) error

	// ValidateCode validates 2FA code
	ValidateCode(ctx context.Context, userID user.UserID, tenantID tenant.TenantID, code string) error

	// GenerateBackupCodes generates backup codes
	GenerateBackupCodes(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) ([]string, error)

	// ValidateBackupCode validates backup code
	ValidateBackupCode(ctx context.Context, userID user.UserID, tenantID tenant.TenantID, code string) error

	// GetTwoFactorStatus returns 2FA status for user
	GetTwoFactorStatus(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) (*TwoFactorStatus, error)
}

// PasswordService defines the interface for password operations
type PasswordService interface {
	// HashPassword hashes password
	HashPassword(password string) (string, error)

	// VerifyPassword verifies password against hash
	VerifyPassword(password, hash string) error

	// GenerateResetToken generates password reset token
	GenerateResetToken(ctx context.Context, email string) (*PasswordResetToken, error)

	// ValidateResetToken validates password reset token
	ValidateResetToken(ctx context.Context, token string) (*PasswordResetToken, error)

	// ResetPassword resets password using token
	ResetPassword(ctx context.Context, token, newPassword string) error

	// ChangePassword changes password for authenticated user
	ChangePassword(ctx context.Context, userID user.UserID, tenantID tenant.TenantID, oldPassword, newPassword string) error

	// ValidatePasswordStrength validates password strength
	ValidatePasswordStrength(password string) error

	// GetPasswordHistory returns password history (for preventing reuse)
	GetPasswordHistory(ctx context.Context, userID user.UserID, tenantID tenant.TenantID) ([]*PasswordHistory, error)
}

// AuthContext represents authenticated user context
type AuthContext struct {
	UserID      user.UserID            `json:"user_id"`
	TenantID    tenant.TenantID        `json:"tenant_id"`
	Email       string                 `json:"email"`
	Name        string                 `json:"name"`
	Role        user.Role              `json:"role"`
	Permissions []Permission           `json:"permissions"`
	SessionID   string                 `json:"session_id"`
	IssuedAt    shared.Timestamp       `json:"issued_at"`
	ExpiresAt   shared.Timestamp       `json:"expires_at"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int64            `json:"expires_in"` // seconds
	ExpiresAt    shared.Timestamp `json:"expires_at"`
	Scope        []string         `json:"scope"`
}

// Permission represents a user permission
type Permission struct {
	Resource  string `json:"resource"`
	Action    string `json:"action"`
	Condition string `json:"condition,omitempty"` // ABAC condition
}

// UserSession represents an active user session
type UserSession struct {
	ID           string                 `json:"id"`
	UserID       user.UserID            `json:"user_id"`
	TenantID     tenant.TenantID        `json:"tenant_id"`
	RefreshToken string                 `json:"-"` // Never expose in JSON
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	DeviceInfo   *DeviceInfo            `json:"device_info"`
	IsActive     bool                   `json:"is_active"`
	LastUsedAt   shared.Timestamp       `json:"last_used_at"`
	CreatedAt    shared.Timestamp       `json:"created_at"`
	ExpiresAt    shared.Timestamp       `json:"expires_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// DeviceInfo represents device information
type DeviceInfo struct {
	OS       string `json:"os"`
	Browser  string `json:"browser"`
	Device   string `json:"device"`
	Location string `json:"location,omitempty"`
}

// APIKey represents an API key
type APIKey struct {
	ID         string                 `json:"id"`
	UserID     user.UserID            `json:"user_id"`
	TenantID   tenant.TenantID        `json:"tenant_id"`
	Name       string                 `json:"name"`
	Key        string                 `json:"key,omitempty"` // Only show during creation
	KeyHash    string                 `json:"-"`
	Scopes     []string               `json:"scopes"`
	IsActive   bool                   `json:"is_active"`
	LastUsedAt *shared.Timestamp      `json:"last_used_at"`
	ExpiresAt  *shared.Timestamp      `json:"expires_at"`
	CreatedAt  shared.Timestamp       `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// TwoFactorSecret represents 2FA secret
type TwoFactorSecret struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"` // Base64 encoded QR code image
	URL    string `json:"url"`     // TOTP URL
}

// TwoFactorStatus represents 2FA status
type TwoFactorStatus struct {
	Enabled          bool              `json:"enabled"`
	EnabledAt        *shared.Timestamp `json:"enabled_at"`
	BackupCodesCount int               `json:"backup_codes_count"`
	LastUsedAt       *shared.Timestamp `json:"last_used_at"`
}

// PasswordResetToken represents password reset token
type PasswordResetToken struct {
	Token     string           `json:"token"`
	UserID    user.UserID      `json:"user_id"`
	TenantID  tenant.TenantID  `json:"tenant_id"`
	Email     string           `json:"email"`
	ExpiresAt shared.Timestamp `json:"expires_at"`
	CreatedAt shared.Timestamp `json:"created_at"`
	Used      bool             `json:"used"`
}

// PasswordHistory represents password history entry
type PasswordHistory struct {
	ID           shared.ID        `json:"id"`
	UserID       user.UserID      `json:"user_id"`
	TenantID     tenant.TenantID  `json:"tenant_id"`
	PasswordHash string           `json:"-"`
	CreatedAt    shared.Timestamp `json:"created_at"`
}

// Helper methods for AuthContext
func (ctx *AuthContext) HasPermission(resource, action string) bool {
	for _, perm := range ctx.Permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
		// Check for wildcard permissions
		if perm.Resource == "*" || perm.Action == "*" {
			return true
		}
	}
	return false
}

func (ctx *AuthContext) IsOwner() bool {
	return ctx.Role == user.RoleOwner
}

func (ctx *AuthContext) IsAdmin() bool {
	return ctx.Role == user.RoleAdmin || ctx.Role == user.RoleOwner
}

func (ctx *AuthContext) IsExpired() bool {
	return shared.Now().After(ctx.ExpiresAt)
}

// Helper methods for TokenPair
func (tp *TokenPair) IsExpired() bool {
	return shared.Now().After(tp.ExpiresAt)
}

// Helper methods for UserSession
func (s *UserSession) IsExpired() bool {
	return shared.Now().After(s.ExpiresAt)
}

func (s *UserSession) UpdateLastUsed() {
	s.LastUsedAt = shared.Now()
}

// Helper methods for APIKey
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return shared.Now().After(*k.ExpiresAt)
}

func (k *APIKey) IsValid() bool {
	return k.IsActive && !k.IsExpired()
}

func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// Helper methods for PasswordResetToken
func (t *PasswordResetToken) IsExpired() bool {
	return shared.Now().After(t.ExpiresAt)
}

func (t *PasswordResetToken) IsValid() bool {
	return !t.Used && !t.IsExpired()
}

// Constants for permissions
const (
	// Resources
	ResourceTenants      = "tenants"
	ResourceUsers        = "users"
	ResourceTemplates    = "templates"
	ResourceSubscription = "subscription"
	ResourceBilling      = "billing"
	ResourceAnalytics    = "analytics"
	ResourceSettings     = "settings"
	ResourceInvitations  = "invitations"
	ResourceAPIKeys      = "api_keys"

	// Actions
	ActionRead    = "read"
	ActionWrite   = "write"
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionAdmin   = "admin"
	ActionPublish = "publish"
	ActionInstall = "install"

	// Scopes for API keys
	ScopeRead      = "read"
	ScopeWrite     = "write"
	ScopeAdmin     = "admin"
	ScopeTemplates = "templates"
	ScopeUsers     = "users"
	ScopeBilling   = "billing"
)

// AuthError represents authentication errors
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common auth errors
var (
	ErrInvalidToken            = &AuthError{Code: "invalid_token", Message: "Invalid or malformed token"}
	ErrExpiredToken            = &AuthError{Code: "expired_token", Message: "Token has expired"}
	ErrInvalidCredentials      = &AuthError{Code: "invalid_credentials", Message: "Invalid email or password"}
	ErrAccountLocked           = &AuthError{Code: "account_locked", Message: "Account is locked"}
	ErrAccountDisabled         = &AuthError{Code: "account_disabled", Message: "Account is disabled"}
	ErrTwoFactorRequired       = &AuthError{Code: "two_factor_required", Message: "Two-factor authentication required"}
	ErrInvalidTwoFactor        = &AuthError{Code: "invalid_two_factor", Message: "Invalid two-factor code"}
	ErrInsufficientPermissions = &AuthError{Code: "insufficient_permissions", Message: "Insufficient permissions"}
	ErrSessionExpired          = &AuthError{Code: "session_expired", Message: "Session has expired"}
	ErrInvalidAPIKey           = &AuthError{Code: "invalid_api_key", Message: "Invalid API key"}
	ErrAPIKeyExpired           = &AuthError{Code: "api_key_expired", Message: "API key has expired"}
	ErrAPIKeyRevoked           = &AuthError{Code: "api_key_revoked", Message: "API key has been revoked"}
)
