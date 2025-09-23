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
type PlanID uuid.UUID

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

// This is a placeholder for a proper JSON import.
