package booking

import "time"

// Availability DTOs
type CreateAvailabilityRequest struct {
	UserID        string     `json:"user_id,omitempty"`
	Title         string     `json:"title" validate:"required"`
	Description   string     `json:"description,omitempty"`
	StartTime     time.Time  `json:"start_time" validate:"required"`
	EndTime       time.Time  `json:"end_time" validate:"required"`
	Recurrence    string     `json:"recurrence" validate:"required,oneof=none daily weekly monthly"`
	RecurrenceEnd *time.Time `json:"recurrence_end,omitempty"`
	DaysOfWeek    []int      `json:"days_of_week,omitempty"`
}

type UpdateAvailabilityRequest struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Status      string     `json:"status,omitempty"`
}

type AvailabilityResponse struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	UserID        string                 `json:"user_id,omitempty"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Status        string                 `json:"status"`
	Recurrence    string                 `json:"recurrence"`
	RecurrenceEnd *time.Time             `json:"recurrence_end,omitempty"`
	DaysOfWeek    []int                  `json:"days_of_week,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type AvailabilityListResponse struct {
	Availabilities []AvailabilityResponse `json:"availabilities"`
	Total          int64                  `json:"total"`
	Limit          int                    `json:"limit"`
	Offset         int                    `json:"offset"`
}

// Appointment DTOs
type CreateAppointmentRequest struct {
	UserID      string    `json:"user_id,omitempty"`
	ClientName  string    `json:"client_name" validate:"required"`
	ClientEmail string    `json:"client_email" validate:"required,email"`
	ClientPhone string    `json:"client_phone,omitempty"`
	Title       string    `json:"title" validate:"required"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type" validate:"required,oneof=consultation meeting lesson interview other"`
	StartTime   time.Time `json:"start_time" validate:"required"`
	EndTime     time.Time `json:"end_time" validate:"required"`
	Duration    int       `json:"duration" validate:"required"`
	Location    string    `json:"location,omitempty"`
	MeetingURL  string    `json:"meeting_url,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

type UpdateAppointmentRequest struct {
	Title              string `json:"title,omitempty"`
	Description        string `json:"description,omitempty"`
	Status             string `json:"status,omitempty"`
	Notes              string `json:"notes,omitempty"`
	CancellationReason string `json:"cancellation_reason,omitempty"`
}

type AppointmentResponse struct {
	ID                 string                 `json:"id"`
	TenantID           string                 `json:"tenant_id"`
	UserID             string                 `json:"user_id,omitempty"`
	ClientName         string                 `json:"client_name"`
	ClientEmail        string                 `json:"client_email"`
	ClientPhone        string                 `json:"client_phone,omitempty"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description,omitempty"`
	Type               string                 `json:"type"`
	Status             string                 `json:"status"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	Duration           int                    `json:"duration"`
	Location           string                 `json:"location,omitempty"`
	MeetingURL         string                 `json:"meeting_url,omitempty"`
	Notes              string                 `json:"notes,omitempty"`
	CancellationReason string                 `json:"cancellation_reason,omitempty"`
	CancelledAt        *time.Time             `json:"cancelled_at,omitempty"`
	ConfirmedAt        *time.Time             `json:"confirmed_at,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	ReminderSent       bool                   `json:"reminder_sent"`
	ReminderSentAt     *time.Time             `json:"reminder_sent_at,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type AppointmentListResponse struct {
	Appointments []AppointmentResponse `json:"appointments"`
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
}
