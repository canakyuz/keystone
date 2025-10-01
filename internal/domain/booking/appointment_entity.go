package booking

import (
	"time"
)

// AppointmentStatus represents appointment status
type AppointmentStatus string

const (
	AppointmentStatusPending   AppointmentStatus = "pending"
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
	AppointmentStatusCompleted AppointmentStatus = "completed"
	AppointmentStatusNoShow    AppointmentStatus = "no-show"
)

// AppointmentType represents appointment type
type AppointmentType string

const (
	AppointmentTypeConsultation AppointmentType = "consultation"
	AppointmentTypeMeeting      AppointmentType = "meeting"
	AppointmentTypeLesson       AppointmentType = "lesson"
	AppointmentTypeInterview    AppointmentType = "interview"
	AppointmentTypeOther        AppointmentType = "other"
)

// Appointment represents a scheduled appointment
type Appointment struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"` // ⚠️ Multi-tenant isolation
	UserID          string                 `json:"user_id,omitempty"`
	ClientName      string                 `json:"client_name"`
	ClientEmail     string                 `json:"client_email"`
	ClientPhone     string                 `json:"client_phone,omitempty"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description,omitempty"`
	Type            AppointmentType        `json:"type"`
	Status          AppointmentStatus      `json:"status"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	Duration        int                    `json:"duration"` // minutes
	Location        string                 `json:"location,omitempty"`
	MeetingURL      string                 `json:"meeting_url,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	CancellationReason string              `json:"cancellation_reason,omitempty"`
	CancelledAt     *time.Time             `json:"cancelled_at,omitempty"`
	ConfirmedAt     *time.Time             `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	ReminderSent    bool                   `json:"reminder_sent"`
	ReminderSentAt  *time.Time             `json:"reminder_sent_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Confirm confirms the appointment
func (a *Appointment) Confirm() error {
	if a.Status == AppointmentStatusCancelled {
		return ErrAppointmentCancelled
	}
	now := time.Now()
	a.Status = AppointmentStatusConfirmed
	a.ConfirmedAt = &now
	a.UpdatedAt = now
	return nil
}

// Cancel cancels the appointment
func (a *Appointment) Cancel(reason string) error {
	if a.Status == AppointmentStatusCompleted {
		return ErrAppointmentCompleted
	}
	now := time.Now()
	a.Status = AppointmentStatusCancelled
	a.CancellationReason = reason
	a.CancelledAt = &now
	a.UpdatedAt = now
	return nil
}

// Complete marks the appointment as completed
func (a *Appointment) Complete() error {
	if a.Status == AppointmentStatusCancelled {
		return ErrAppointmentCancelled
	}
	now := time.Now()
	a.Status = AppointmentStatusCompleted
	a.CompletedAt = &now
	a.UpdatedAt = now
	return nil
}

// MarkAsNoShow marks the appointment as no-show
func (a *Appointment) MarkAsNoShow() error {
	if a.Status == AppointmentStatusCancelled {
		return ErrAppointmentCancelled
	}
	a.Status = AppointmentStatusNoShow
	a.UpdatedAt = time.Now()
	return nil
}

// SendReminder marks reminder as sent
func (a *Appointment) SendReminder() {
	now := time.Now()
	a.ReminderSent = true
	a.ReminderSentAt = &now
	a.UpdatedAt = now
}

// IsUpcoming checks if appointment is in the future
func (a *Appointment) IsUpcoming() bool {
	return a.StartTime.After(time.Now()) && a.Status != AppointmentStatusCancelled
}

// IsPast checks if appointment is in the past
func (a *Appointment) IsPast() bool {
	return a.EndTime.Before(time.Now())
}
