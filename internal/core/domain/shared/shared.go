package shared

import (
	"database/sql/driver"
	"errors"
	"fmt"
	_ "strings"
	"time"

	"encoding/json"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
)

// General purpose errors
var (
	ErrTemplateAlreadyExists        = errors.New("template with this name already exists for the tenant")
	ErrTemplateNotFound             = errors.New("template not found")
	ErrTemplateAlreadyInstalled     = errors.New("template is already installed for this tenant")
	ErrInstallationNotFound         = errors.New("installation not found")
	ErrTenantNotFound               = errors.New("tenant not found")
	ErrTenantSlugAlreadyExists      = errors.New("tenant slug already exists")
	ErrTenantNotActive              = errors.New("tenant not active")
	ErrCustomDomainAlreadyExists    = errors.New("custom domain already exists")
	ErrUserNotFound                 = errors.New("user not found")
	ErrUserAlreadyExists            = errors.New("user with this email already exists in the tenant")
	ErrInvalidCredentials           = errors.New("invalid credentials")
	ErrUserNotActive                = errors.New("user not active")
	ErrEmailNotVerified             = errors.New("email not verified")
	ErrCannotDeactivateSelf         = errors.New("cannot deactivate self")
	ErrCrossTenantAccess            = errors.New("cross-tenant access is not allowed")
	ErrInsufficientPermissions      = errors.New("user does not have sufficient permissions")
	ErrTemplateNotAccessible        = errors.New("template not accessible")
	ErrSubscriptionNotFound         = errors.New("subscription not found")
	ErrTenantAlreadyHasSubscription = errors.New("tenant already has a subscription")
	ErrSubscriptionNotActive        = errors.New("subscription not active")
	ErrUsageLimitExceeded           = errors.New("usage limit exceeded")
	ErrPaymentRequired              = errors.New("payment required")
	ErrResourceLocked               = errors.New("resource locked")
	ErrRateLimitExceeded            = errors.New("rate limit exceeded")
)

// ID types
type TemplateID uuid.UUID
type TenantID uuid.UUID
type UserID uuid.UUID
type InstallationID uuid.UUID
type SubscriptionID uuid.UUID
type PlanID string

// String methods for ID types
func (t TemplateID) String() string {
	return uuid.UUID(t).String()
}

func (t TenantID) String() string {
	return uuid.UUID(t).String()
}

func (u UserID) String() string {
	return uuid.UUID(u).String()
}

func (i InstallationID) String() string {
	return uuid.UUID(i).String()
}

func (s SubscriptionID) String() string {
	return uuid.UUID(s).String()
}

func (p PlanID) String() string {
	return string(p)
}

// Email represents a validated email address.
type Email struct {
	value string
}

// NewEmail creates a new Email value object.
func NewEmail(email string) (*Email, error) {
	// In a real application, you would have more sophisticated email validation.
	if len(email) == 0 {
		return nil, errors.New("email cannot be empty")
	}
	return &Email{value: email}, nil
}

// String returns the string representation of the email.
func (e *Email) String() string {
	if e == nil {
		return ""
	}
	return e.value
}

// TenantSlug represents a tenant's unique slug.
type TenantSlug struct {
	value string
}

// NewTenantSlug creates a new TenantSlug value object.
func NewTenantSlug(slug string) (*TenantSlug, error) {
	if len(slug) == 0 {
		return nil, errors.New("slug cannot be empty")
	}
	return &TenantSlug{value: slug}, nil
}

// String returns the string representation of the slug.
func (s *TenantSlug) String() string {
	if s == nil {
		return ""
	}
	return s.value
}

// Domain represents a custom domain.
type Domain struct {
	value string
}

// NewDomain creates a new Domain value object.
func NewDomain(domain string) (*Domain, error) {
	if len(domain) == 0 {
		return nil, errors.New("domain cannot be empty")
	}
	return &Domain{value: domain}, nil
}

// String returns the string representation of the domain.
func (d *Domain) String() string {
	if d == nil {
		return ""
	}
	return d.value
}

// Timestamp represents a wrapper around time.Time for consistent handling.
type Timestamp struct {
	time time.Time
}

func NewTimestamp() *Timestamp {
	return &Timestamp{time: time.Now().UTC()}
}

func NewTimestampFromTime(t time.Time) *Timestamp {
	return &Timestamp{time: t}
}

func (t *Timestamp) Time() time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.time
}

// Money represents a monetary value.
type Money struct {
	amount   int64
	currency string
}

func NewMoney(amount int64, currency string) (*Money, error) {
	if currency == "" {
		return nil, errors.New("currency cannot be empty")
	}
	return &Money{amount: amount, currency: currency}, nil
}

func (m *Money) Amount() int64 {
	if m == nil {
		return 0
	}
	return m.amount
}

func (m *Money) Currency() string {
	if m == nil {
		return ""
	}
	return m.currency
}

// Version represents a semantic version.
type Version struct {
	*semver.Version
}

func NewVersion(v string) (*Version, error) {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return nil, err
	}
	return &Version{sv}, nil
}

// Value implements the driver.Valuer interface for database serialization.
func (v Version) Value() (driver.Value, error) {
	return v.String(), nil
}

// Scan implements the sql.Scanner interface for database deserialization.
func (v *Version) Scan(value interface{}) error {
	if value == nil {
		v.Version = nil
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return errors.New("Scan source for Version must be a string")
	}
	sv, err := semver.NewVersion(s)
	if err != nil {
		return err
	}
	v.Version = sv
	return nil
}

// JSON-compatible types for map[string]interface{}
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	// A simple implementation might just return the map,
	// but proper JSON marshaling is better.
	// For now, we assume the driver handles map[string]interface{}
	return j, nil
}

func (j *JSONB) Scan(src interface{}) error {
	// Type assertion and assignment
	if src == nil {
		*j = nil
		return nil
	}
	// The driver will likely return []byte
	source, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	// Unmarshal from JSON
	var i map[string]interface{}
	if err := json.Unmarshal(source, &i); err != nil {
		return err
	}
	*j = i
	return nil
}

// Parse functions for ID types
func ParseUserID(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID(uuid.Nil), fmt.Errorf("invalid user ID: %w", err)
	}
	return UserID(id), nil
}

func ParseTenantID(s string) (TenantID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return TenantID(uuid.Nil), fmt.Errorf("invalid tenant ID: %w", err)
	}
	return TenantID(id), nil
}

func ParseTemplateID(s string) (TemplateID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return TemplateID(uuid.Nil), fmt.Errorf("invalid template ID: %w", err)
	}
	return TemplateID(id), nil
}

func ParseSubscriptionID(s string) (SubscriptionID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return SubscriptionID(uuid.Nil), fmt.Errorf("invalid subscription ID: %w", err)
	}
	return SubscriptionID(id), nil
}

// Role type and constants
type Role string

const (
	RoleOwner        Role = "owner"
	RoleAdmin        Role = "admin"
	RoleEditor       Role = "editor"
	RoleViewer       Role = "viewer"
	RoleBillingAdmin Role = "billing_admin"
)

func (r Role) String() string {
	return string(r)
}

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleOwner, RoleAdmin, RoleEditor, RoleViewer, RoleBillingAdmin:
		return Role(s), nil
	default:
		return "", fmt.Errorf("invalid role: %s", s)
	}
}

// UUID conversion methods for ID types
func (t TenantID) UUID() uuid.UUID {
	return uuid.UUID(t)
}

func (u UserID) UUID() uuid.UUID {
	return uuid.UUID(u)
}

func (t TemplateID) UUID() uuid.UUID {
	return uuid.UUID(t)
}

func (i InstallationID) UUID() uuid.UUID {
	return uuid.UUID(i)
}

func (s SubscriptionID) UUID() uuid.UUID {
	return uuid.UUID(s)
}

// New ID creation from UUID
func NewTenantID(id uuid.UUID) TenantID {
	return TenantID(id)
}

func NewUserID(id uuid.UUID) UserID {
	return UserID(id)
}

func NewTemplateID(id uuid.UUID) TemplateID {
	return TemplateID(id)
}

func NewInstallationID(id uuid.UUID) InstallationID {
	return InstallationID(id)
}

func NewSubscriptionID(id uuid.UUID) SubscriptionID {
	return SubscriptionID(id)
}
