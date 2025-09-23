package events

import (
	"context"
	"time"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// EventBus defines the interface for event publishing and subscribing
type EventBus interface {
	// Publish publishes an event
	Publish(ctx context.Context, event DomainEvent) error

	// PublishBatch publishes multiple events in a batch
	PublishBatch(ctx context.Context, events []DomainEvent) error

	// Subscribe subscribes to events of a specific type
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error

	// Unsubscribe unsubscribes from events
	Unsubscribe(ctx context.Context, eventType string, handlerID string) error

	// Start starts the event bus
	Start(ctx context.Context) error

	// Stop stops the event bus
	Stop(ctx context.Context) error

	// GetEventHistory retrieves event history
	GetEventHistory(ctx context.Context, filter EventFilter) ([]*EventRecord, error)

	// ReplayEvents replays events from a specific point
	ReplayEvents(ctx context.Context, fromTimestamp shared.Timestamp, eventTypes []string) error
}

// EventStore defines the interface for event storage
type EventStore interface {
	// Store stores an event
	Store(ctx context.Context, event DomainEvent) error

	// StoreBatch stores multiple events
	StoreBatch(ctx context.Context, events []DomainEvent) error

	// Load loads events for an aggregate
	Load(ctx context.Context, aggregateID string, aggregateType string) ([]DomainEvent, error)

	// LoadFromVersion loads events from a specific version
	LoadFromVersion(ctx context.Context, aggregateID string, aggregateType string, fromVersion int64) ([]DomainEvent, error)

	// LoadFromTimestamp loads events from a specific timestamp
	LoadFromTimestamp(ctx context.Context, fromTimestamp shared.Timestamp, eventTypes []string) ([]DomainEvent, error)

	// GetLastVersion gets the last version for an aggregate
	GetLastVersion(ctx context.Context, aggregateID string, aggregateType string) (int64, error)

	// DeleteEvents deletes events (for GDPR compliance)
	DeleteEvents(ctx context.Context, aggregateID string, aggregateType string) error

	// CreateSnapshot creates a snapshot
	CreateSnapshot(ctx context.Context, snapshot *AggregateSnapshot) error

	// LoadSnapshot loads a snapshot
	LoadSnapshot(ctx context.Context, aggregateID string, aggregateType string) (*AggregateSnapshot, error)
}

// EventHandler defines the interface for event handlers
type EventHandler interface {
	// Handle handles an event
	Handle(ctx context.Context, event DomainEvent) error

	// GetID returns handler ID
	GetID() string

	// GetEventTypes returns event types this handler is interested in
	GetEventTypes() []string

	// GetConcurrency returns maximum concurrent handlers
	GetConcurrency() int

	// ShouldRetry determines if a failed event should be retried
	ShouldRetry(ctx context.Context, event DomainEvent, err error, attemptCount int) bool
}

// DomainEvent represents a domain event
type DomainEvent interface {
	// GetEventID returns unique event ID
	GetEventID() string

	// GetEventType returns event type
	GetEventType() string

	// GetEventVersion returns event version
	GetEventVersion() string

	// GetAggregateID returns aggregate ID
	GetAggregateID() string

	// GetAggregateType returns aggregate type
	GetAggregateType() string

	// GetAggregateVersion returns aggregate version when event occurred
	GetAggregateVersion() int64

	// GetOccurredOn returns when event occurred
	GetOccurredOn() time.Time

	// GetTenantID returns tenant ID (for multi-tenant events)
	GetTenantID() string

	// GetUserID returns user ID (for user-triggered events)
	GetUserID() string

	// GetData returns event data
	GetData() map[string]interface{}

	// GetMetadata returns event metadata
	GetMetadata() map[string]interface{}

	// IsReplayable returns if event can be replayed
	IsReplayable() bool
}

// BaseDomainEvent provides base implementation for domain events
type BaseDomainEvent struct {
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"`
	EventVersion     string                 `json:"event_version"`
	AggregateID      string                 `json:"aggregate_id"`
	AggregateType    string                 `json:"aggregate_type"`
	AggregateVersion int64                  `json:"aggregate_version"`
	OccurredOn       time.Time              `json:"occurred_on"`
	TenantID         string                 `json:"tenant_id"`
	UserID           string                 `json:"user_id"`
	Data             map[string]interface{} `json:"data"`
	Metadata         map[string]interface{} `json:"metadata"`
	Replayable       bool                   `json:"replayable"`
}

// Implement DomainEvent interface
func (e *BaseDomainEvent) GetEventID() string                  { return e.EventID }
func (e *BaseDomainEvent) GetEventType() string                { return e.EventType }
func (e *BaseDomainEvent) GetEventVersion() string             { return e.EventVersion }
func (e *BaseDomainEvent) GetAggregateID() string              { return e.AggregateID }
func (e *BaseDomainEvent) GetAggregateType() string            { return e.AggregateType }
func (e *BaseDomainEvent) GetAggregateVersion() int64          { return e.AggregateVersion }
func (e *BaseDomainEvent) GetOccurredOn() time.Time            { return e.OccurredOn }
func (e *BaseDomainEvent) GetTenantID() string                 { return e.TenantID }
func (e *BaseDomainEvent) GetUserID() string                   { return e.UserID }
func (e *BaseDomainEvent) GetData() map[string]interface{}     { return e.Data }
func (e *BaseDomainEvent) GetMetadata() map[string]interface{} { return e.Metadata }
func (e *BaseDomainEvent) IsReplayable() bool                  { return e.Replayable }

// EventRecord represents a stored event record
type EventRecord struct {
	ID               int64                  `json:"id"`
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"`
	EventVersion     string                 `json:"event_version"`
	AggregateID      string                 `json:"aggregate_id"`
	AggregateType    string                 `json:"aggregate_type"`
	AggregateVersion int64                  `json:"aggregate_version"`
	TenantID         string                 `json:"tenant_id"`
	UserID           string                 `json:"user_id"`
	Data             map[string]interface{} `json:"data"`
	Metadata         map[string]interface{} `json:"metadata"`
	OccurredOn       time.Time              `json:"occurred_on"`
	StoredOn         time.Time              `json:"stored_on"`
	ProcessedOn      *time.Time             `json:"processed_on"`
	ProcessingError  *string                `json:"processing_error"`
	RetryCount       int                    `json:"retry_count"`
	IsReplayable     bool                   `json:"is_replayable"`
}

// AggregateSnapshot represents an aggregate snapshot
type AggregateSnapshot struct {
	ID            string                 `json:"id"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int64                  `json:"version"`
	Data          map[string]interface{} `json:"data"`
	CreatedAt     time.Time              `json:"created_at"`
}

// EventFilter represents event filtering options
type EventFilter struct {
	// Event type filtering
	EventTypes []string `json:"event_types,omitempty"`

	// Aggregate filtering
	AggregateID   *string `json:"aggregate_id,omitempty"`
	AggregateType *string `json:"aggregate_type,omitempty"`

	// Tenant filtering
	TenantID *string `json:"tenant_id,omitempty"`

	// User filtering
	UserID *string `json:"user_id,omitempty"`

	// Time range filtering
	FromTimestamp *time.Time `json:"from_timestamp,omitempty"`
	ToTimestamp   *time.Time `json:"to_timestamp,omitempty"`

	// Version filtering
	MinVersion *int64 `json:"min_version,omitempty"`
	MaxVersion *int64 `json:"max_version,omitempty"`

	// Processing status filtering
	ProcessedOnly   bool `json:"processed_only"`
	UnprocessedOnly bool `json:"unprocessed_only"`
	FailedOnly      bool `json:"failed_only"`

	// Pagination
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	// Sorting
	SortBy    string `json:"sort_by"`    // occurred_on, stored_on, version
	SortOrder string `json:"sort_order"` // asc, desc
}

// EventProcessor defines the interface for event processing
type EventProcessor interface {
	// ProcessEvent processes a single event
	ProcessEvent(ctx context.Context, event DomainEvent) error

	// ProcessBatch processes multiple events
	ProcessBatch(ctx context.Context, events []DomainEvent) error

	// RetryFailedEvents retries failed events
	RetryFailedEvents(ctx context.Context, maxRetries int) error

	// GetProcessingStats returns processing statistics
	GetProcessingStats(ctx context.Context) (*ProcessingStats, error)

	// Start starts the processor
	Start(ctx context.Context) error

	// Stop stops the processor
	Stop(ctx context.Context) error
}

// ProcessingStats represents event processing statistics
type ProcessingStats struct {
	TotalEvents     int64      `json:"total_events"`
	ProcessedEvents int64      `json:"processed_events"`
	FailedEvents    int64      `json:"failed_events"`
	PendingEvents   int64      `json:"pending_events"`
	RetryingEvents  int64      `json:"retrying_events"`
	AverageLatency  int64      `json:"average_latency_ms"`
	EventsPerSecond int64      `json:"events_per_second"`
	LastProcessedAt *time.Time `json:"last_processed_at"`
}

// Specific Domain Events

// TenantEvents
type TenantCreatedEvent struct {
	*BaseDomainEvent
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	PlanID     string `json:"plan_id,omitempty"`
}

type TenantUpdatedEvent struct {
	*BaseDomainEvent
	TenantName string                 `json:"tenant_name"`
	Changes    map[string]interface{} `json:"changes"`
}

type TenantDeletedEvent struct {
	*BaseDomainEvent
	TenantName string `json:"tenant_name"`
	Reason     string `json:"reason,omitempty"`
}

type TenantStatusChangedEvent struct {
	*BaseDomainEvent
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Reason    string `json:"reason,omitempty"`
}

// UserEvents
type UserCreatedEvent struct {
	*BaseDomainEvent
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
	UserRole  string `json:"user_role"`
}

type UserUpdatedEvent struct {
	*BaseDomainEvent
	UserEmail string                 `json:"user_email"`
	Changes   map[string]interface{} `json:"changes"`
}

type UserDeletedEvent struct {
	*BaseDomainEvent
	UserEmail string `json:"user_email"`
	Reason    string `json:"reason,omitempty"`
}

type UserLoggedInEvent struct {
	*BaseDomainEvent
	UserEmail string `json:"user_email"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

type UserLoggedOutEvent struct {
	*BaseDomainEvent
	UserEmail string `json:"user_email"`
	SessionID string `json:"session_id"`
}

type UserRoleChangedEvent struct {
	*BaseDomainEvent
	UserEmail string `json:"user_email"`
	OldRole   string `json:"old_role"`
	NewRole   string `json:"new_role"`
	ChangedBy string `json:"changed_by"`
}

type UserInvitedEvent struct {
	*BaseDomainEvent
	InviteeEmail string    `json:"invitee_email"`
	InviterID    string    `json:"inviter_id"`
	Role         string    `json:"role"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TemplateEvents
type TemplateCreatedEvent struct {
	*BaseDomainEvent
	TemplateName     string `json:"template_name"`
	TemplateCategory string `json:"template_category"`
	AuthorID         string `json:"author_id"`
	IsPublic         bool   `json:"is_public"`
}

type TemplateUpdatedEvent struct {
	*BaseDomainEvent
	TemplateName string                 `json:"template_name"`
	Changes      map[string]interface{} `json:"changes"`
}

type TemplatePublishedEvent struct {
	*BaseDomainEvent
	TemplateName string `json:"template_name"`
	Version      string `json:"version"`
	AuthorID     string `json:"author_id"`
}

type TemplateInstalledEvent struct {
	*BaseDomainEvent
	TemplateName string  `json:"template_name"`
	InstalledBy  string  `json:"installed_by"`
	Version      string  `json:"version"`
	Price        *string `json:"price,omitempty"`
}

type TemplateUninstalledEvent struct {
	*BaseDomainEvent
	TemplateName  string `json:"template_name"`
	UninstalledBy string `json:"uninstalled_by"`
	Reason        string `json:"reason,omitempty"`
}

// SubscriptionEvents
type SubscriptionCreatedEvent struct {
	*BaseDomainEvent
	PlanID       string `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"`
	IsTrialing   bool   `json:"is_trialing"`
}

type SubscriptionUpdatedEvent struct {
	*BaseDomainEvent
	Changes map[string]interface{} `json:"changes"`
}

type SubscriptionCanceledEvent struct {
	*BaseDomainEvent
	Reason    string     `json:"reason,omitempty"`
	CancelAt  *time.Time `json:"cancel_at,omitempty"`
	Immediate bool       `json:"immediate"`
}

type SubscriptionRenewedEvent struct {
	*BaseDomainEvent
	PlanID      string    `json:"plan_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

type PaymentSucceededEvent struct {
	*BaseDomainEvent
	PaymentID     string `json:"payment_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
}

type PaymentFailedEvent struct {
	*BaseDomainEvent
	PaymentID     string `json:"payment_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	FailureReason string `json:"failure_reason"`
}

// SecurityEvents
type SecurityAlertEvent struct {
	*BaseDomainEvent
	AlertType   string `json:"alert_type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
}

type SuspiciousActivityEvent struct {
	*BaseDomainEvent
	ActivityType string                 `json:"activity_type"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Details      map[string]interface{} `json:"details"`
}

// System Events
type SystemMaintenanceEvent struct {
	*BaseDomainEvent
	MaintenanceType string    `json:"maintenance_type"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Description     string    `json:"description"`
}

type SystemErrorEvent struct {
	*BaseDomainEvent
	ErrorType    string `json:"error_type"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	StackTrace   string `json:"stack_trace,omitempty"`
}

// Event Factory Functions

// NewTenantCreatedEvent creates a new tenant created event
func NewTenantCreatedEvent(tenantID tenant.TenantID, name, slug string, planID *string, userID user.UserID) *TenantCreatedEvent {
	event := &TenantCreatedEvent{
		BaseDomainEvent: &BaseDomainEvent{
			EventID:          shared.NewID().String(),
			EventType:        "tenant.created",
			EventVersion:     "1.0",
			AggregateID:      tenantID.String(),
			AggregateType:    "tenant",
			AggregateVersion: 1,
			OccurredOn:       time.Now().UTC(),
			TenantID:         tenantID.String(),
			UserID:           userID.String(),
			Data:             make(map[string]interface{}),
			Metadata:         make(map[string]interface{}),
			Replayable:       true,
		},
		TenantName: name,
		TenantSlug: slug,
	}

	if planID != nil {
		event.PlanID = *planID
	}

	return event
}

// NewUserCreatedEvent creates a new user created event
func NewUserCreatedEvent(userID user.UserID, tenantID tenant.TenantID, email, name string, role user.Role) *UserCreatedEvent {
	return &UserCreatedEvent{
		BaseDomainEvent: &BaseDomainEvent{
			EventID:          shared.NewID().String(),
			EventType:        "user.created",
			EventVersion:     "1.0",
			AggregateID:      userID.String(),
			AggregateType:    "user",
			AggregateVersion: 1,
			OccurredOn:       time.Now().UTC(),
			TenantID:         tenantID.String(),
			UserID:           userID.String(),
			Data:             make(map[string]interface{}),
			Metadata:         make(map[string]interface{}),
			Replayable:       true,
		},
		UserEmail: email,
		UserName:  name,
		UserRole:  role.String(),
	}
}

// Event Type Constants
const (
	// Tenant Events
	EventTypeTenantCreated       = "tenant.created"
	EventTypeTenantUpdated       = "tenant.updated"
	EventTypeTenantDeleted       = "tenant.deleted"
	EventTypeTenantStatusChanged = "tenant.status_changed"

	// User Events
	EventTypeUserCreated     = "user.created"
	EventTypeUserUpdated     = "user.updated"
	EventTypeUserDeleted     = "user.deleted"
	EventTypeUserLoggedIn    = "user.logged_in"
	EventTypeUserLoggedOut   = "user.logged_out"
	EventTypeUserRoleChanged = "user.role_changed"
	EventTypeUserInvited     = "user.invited"

	// Template Events
	EventTypeTemplateCreated     = "template.created"
	EventTypeTemplateUpdated     = "template.updated"
	EventTypeTemplatePublished   = "template.published"
	EventTypeTemplateInstalled   = "template.installed"
	EventTypeTemplateUninstalled = "template.uninstalled"

	// Subscription Events
	EventTypeSubscriptionCreated  = "subscription.created"
	EventTypeSubscriptionUpdated  = "subscription.updated"
	EventTypeSubscriptionCanceled = "subscription.canceled"
	EventTypeSubscriptionRenewed  = "subscription.renewed"

	// Payment Events
	EventTypePaymentSucceeded = "payment.succeeded"
	EventTypePaymentFailed    = "payment.failed"

	// Security Events
	EventTypeSecurityAlert      = "security.alert"
	EventTypeSuspiciousActivity = "security.suspicious_activity"

	// System Events
	EventTypeSystemMaintenance = "system.maintenance"
	EventTypeSystemError       = "system.error"
)

// Helper methods
func (f *EventFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 10000 {
		return shared.NewValidationError("limit must be between 0 and 10000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	validSortFields := []string{"occurred_on", "stored_on", "version"}
	if f.SortBy != "" {
		valid := false
		for _, field := range validSortFields {
			if f.SortBy == field {
				valid = true
				break
			}
		}
		if !valid {
			return shared.NewValidationError("invalid sort field")
		}
	}

	if f.SortOrder != "" && f.SortOrder != "asc" && f.SortOrder != "desc" {
		return shared.NewValidationError("sort order must be 'asc' or 'desc'")
	}

	return nil
}

func (f *EventFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 100
	}

	if f.SortBy == "" {
		f.SortBy = "occurred_on"
	}

	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}
