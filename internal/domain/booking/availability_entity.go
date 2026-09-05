package booking

import (
	"time"
)

// AvailabilityStatus represents availability status
type AvailabilityStatus string

const (
	AvailabilityStatusAvailable   AvailabilityStatus = "available"
	AvailabilityStatusBooked      AvailabilityStatus = "booked"
	AvailabilityStatusUnavailable AvailabilityStatus = "unavailable"
)

// RecurrenceType represents recurrence pattern
type RecurrenceType string

const (
	RecurrenceTypeNone    RecurrenceType = "none"
	RecurrenceTypeDaily   RecurrenceType = "daily"
	RecurrenceTypeWeekly  RecurrenceType = "weekly"
	RecurrenceTypeMonthly RecurrenceType = "monthly"
)

// Availability represents a time slot availability
type Availability struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"` // ⚠️ Multi-tenant isolation
	UserID        string                 `json:"user_id,omitempty"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Status        AvailabilityStatus     `json:"status"`
	Recurrence    RecurrenceType         `json:"recurrence"`
	RecurrenceEnd *time.Time             `json:"recurrence_end,omitempty"`
	DaysOfWeek    []int                  `json:"days_of_week,omitempty"` // 0=Sunday, 1=Monday, etc.
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// IsActive checks if availability is currently active
func (a *Availability) IsActive() bool {
	now := time.Now()
	return a.StartTime.Before(now) && a.EndTime.After(now)
}

// MarkAsBooked marks the availability as booked
func (a *Availability) MarkAsBooked() {
	a.Status = AvailabilityStatusBooked
	a.UpdatedAt = time.Now()
}

// MarkAsUnavailable marks the availability as unavailable
func (a *Availability) MarkAsUnavailable() {
	a.Status = AvailabilityStatusUnavailable
	a.UpdatedAt = time.Now()
}

// MarkAsAvailable marks the availability as available
func (a *Availability) MarkAsAvailable() {
	a.Status = AvailabilityStatusAvailable
	a.UpdatedAt = time.Now()
}
