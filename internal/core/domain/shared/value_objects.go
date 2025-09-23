package shared

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrInvalidID    = errors.New("invalid ID")
	ErrInvalidEmail = errors.New("invalid email address")
	ErrInvalidSlug  = errors.New("invalid slug format")
	ErrEmptyValue   = errors.New("value cannot be empty")
)

// ID represents a unique identifier
type ID struct {
	value uuid.UUID
}

// NewID creates a new unique ID
func NewID() ID {
	return ID{value: uuid.New()}
}

// ParseID parses string to ID
func ParseID(s string) (ID, error) {
	if s == "" {
		return ID{}, ErrEmptyValue
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return ID{}, fmt.Errorf("%w: %s", ErrInvalidID, err.Error())
	}

	return ID{value: id}, nil
}

// String returns string representation
func (id ID) String() string {
	return id.value.String()
}

// UUID returns the underlying UUID
func (id ID) UUID() uuid.UUID {
	return id.value
}

// IsZero checks if ID is zero value
func (id ID) IsZero() bool {
	return id.value == uuid.Nil
}

// Email represents a validated email address
type Email struct {
	value string
}

// NewEmail creates a new Email value object
func NewEmail(email string) (Email, error) {
	if email == "" {
		return Email{}, ErrEmptyValue
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return Email{}, fmt.Errorf("%w: %s", ErrInvalidEmail, err.Error())
	}

	return Email{value: strings.ToLower(email)}, nil
}

// String returns the email string
func (e Email) String() string {
	return e.value
}

// Domain returns the domain part of email
func (e Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// Slug represents a URL-safe identifier
type Slug struct {
	value string
}

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NewSlug creates a new Slug value object
func NewSlug(slug string) (Slug, error) {
	if slug == "" {
		return Slug{}, ErrEmptyValue
	}

	slug = strings.ToLower(strings.TrimSpace(slug))

	if !slugRegex.MatchString(slug) {
		return Slug{}, fmt.Errorf("%w: must contain only lowercase letters, numbers, and hyphens", ErrInvalidSlug)
	}

	if len(slug) < 2 || len(slug) > 63 {
		return Slug{}, fmt.Errorf("%w: must be between 2 and 63 characters", ErrInvalidSlug)
	}

	return Slug{value: slug}, nil
}

// String returns the slug string
func (s Slug) String() string {
	return s.value
}

// Money represents monetary value with currency
type Money struct {
	Amount   int64  `json:"amount"`   // Amount in smallest currency unit (cents)
	Currency string `json:"currency"` // ISO 4217 currency code
}

// NewMoney creates new Money value object
func NewMoney(amount int64, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: strings.ToUpper(currency),
	}
}

// Dollars returns amount in dollars (for USD)
func (m Money) Dollars() float64 {
	return float64(m.Amount) / 100.0
}

// IsZero checks if money amount is zero
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// Add adds two Money values (must be same currency)
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, errors.New("cannot add different currencies")
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: m.Currency,
	}, nil
}

// Timestamp represents a moment in time
type Timestamp struct {
	value time.Time
}

// Now creates timestamp for current time
func Now() Timestamp {
	return Timestamp{value: time.Now().UTC()}
}

// NewTimestamp creates timestamp from time.Time
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{value: t.UTC()}
}

// Time returns the underlying time.Time
func (ts Timestamp) Time() time.Time {
	return ts.value
}

// IsZero checks if timestamp is zero
func (ts Timestamp) IsZero() bool {
	return ts.value.IsZero()
}

// Before checks if this timestamp is before another
func (ts Timestamp) Before(other Timestamp) bool {
	return ts.value.Before(other.value)
}

// After checks if this timestamp is after another
func (ts Timestamp) After(other Timestamp) bool {
	return ts.value.After(other.value)
}

// Status represents entity status
type Status string

const (
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
	StatusPending   Status = "pending"
)

// IsValid checks if status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusSuspended, StatusDeleted, StatusPending:
		return true
	default:
		return false
	}
}

// String returns string representation
func (s Status) String() string {
	return string(s)
}

// Settings represents JSON configuration
type Settings map[string]interface{}

// Get retrieves a setting value
func (s Settings) Get(key string) (interface{}, bool) {
	value, exists := s[key]
	return value, exists
}

// Set updates a setting value
func (s Settings) Set(key string, value interface{}) {
	s[key] = value
}

// GetString retrieves string setting
func (s Settings) GetString(key string) string {
	if value, exists := s[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetBool retrieves boolean setting
func (s Settings) GetBool(key string) bool {
	if value, exists := s[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// GetInt retrieves integer setting
func (s Settings) GetInt(key string) int {
	if value, exists := s[key]; exists {
		if i, ok := value.(float64); ok {
			return int(i)
		}
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// Merge combines settings with another Settings object
func (s Settings) Merge(other Settings) Settings {
	result := make(Settings)

	// Copy current settings
	for k, v := range s {
		result[k] = v
	}

	// Override with other settings
	for k, v := range other {
		result[k] = v
	}

	return result
}

// Validate performs basic validation on settings
func (s Settings) Validate() error {
	// Add common validation rules here
	return nil
}
