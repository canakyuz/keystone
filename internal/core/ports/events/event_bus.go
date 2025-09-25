package events

import (
	"context"
	"time"
)

// Event represents a domain event
type Event interface {
	GetID() string
	GetType() string
	GetAggregateID() string
	GetTenantID() string
	GetOccurredAt() time.Time
	GetPayload() map[string]interface{}
}

// EventHandler represents a handler for domain events
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	GetEventTypes() []string
}

// EventBus represents the event bus interface
type EventBus interface {
	// Publish publishes an event to the bus
	Publish(ctx context.Context, event Event) error

	// Subscribe subscribes a handler to specific event types
	Subscribe(handler EventHandler) error

	// Unsubscribe removes a handler from the bus
	Unsubscribe(handler EventHandler) error

	// Start starts the event bus
	Start(ctx context.Context) error

	// Stop stops the event bus
	Stop(ctx context.Context) error
}

// BaseEvent provides a basic implementation of the Event interface
type BaseEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	AggregateID string                 `json:"aggregate_id"`
	TenantID    string                 `json:"tenant_id"`
	OccurredAt  time.Time              `json:"occurred_at"`
	Payload     map[string]interface{} `json:"payload"`
}

// GetID returns the event ID
func (e *BaseEvent) GetID() string {
	return e.ID
}

// GetType returns the event type
func (e *BaseEvent) GetType() string {
	return e.Type
}

// GetAggregateID returns the aggregate ID
func (e *BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

// GetTenantID returns the tenant ID
func (e *BaseEvent) GetTenantID() string {
	return e.TenantID
}

// GetOccurredAt returns when the event occurred
func (e *BaseEvent) GetOccurredAt() time.Time {
	return e.OccurredAt
}

// GetPayload returns the event payload
func (e *BaseEvent) GetPayload() map[string]interface{} {
	return e.Payload
}

// Domain Events

// UserCreatedEvent represents a user creation event
type UserCreatedEvent struct {
	BaseEvent
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	UserRole  string `json:"user_role"`
}

// UserUpdatedEvent represents a user update event
type UserUpdatedEvent struct {
	BaseEvent
	UserID    string                 `json:"user_id"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy string                 `json:"updated_by"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// UserDeletedEvent represents a user deletion event
type UserDeletedEvent struct {
	BaseEvent
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
}

// TenantCreatedEvent represents a tenant creation event
type TenantCreatedEvent struct {
	BaseEvent
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
}

// TenantUpdatedEvent represents a tenant update event
type TenantUpdatedEvent struct {
	BaseEvent
	Changes map[string]interface{} `json:"changes"`
}

// TemplateCreatedEvent represents a template creation event
type TemplateCreatedEvent struct {
	BaseEvent
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	Category     string `json:"category"`
}

// TemplateInstalledEvent represents a template installation event
type TemplateInstalledEvent struct {
	BaseEvent
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	InstalledBy  string `json:"installed_by"`
}

// SubscriptionCreatedEvent represents a subscription creation event
type SubscriptionCreatedEvent struct {
	BaseEvent
	SubscriptionID string `json:"subscription_id"`
	PlanID         string `json:"plan_id"`
	BillingCycle   string `json:"billing_cycle"`
}

// SubscriptionUpdatedEvent represents a subscription update event
type SubscriptionUpdatedEvent struct {
	BaseEvent
	SubscriptionID string                 `json:"subscription_id"`
	Changes        map[string]interface{} `json:"changes"`
}

// SubscriptionCancelledEvent represents a subscription cancellation event
type SubscriptionCancelledEvent struct {
	BaseEvent
	SubscriptionID string `json:"subscription_id"`
	Reason         string `json:"reason"`
}

// TemplatePublishedEvent represents a template publication event
type TemplatePublishedEvent struct {
	BaseEvent
	TemplateID   string `json:"template_id"`
	Version      string `json:"version"`
	ReleaseNotes string `json:"release_notes"`
}

// UserDeactivatedEvent represents a user deactivation event
type UserDeactivatedEvent struct {
	BaseEvent
	UserID        string    `json:"user_id"`
	DeactivatedBy string    `json:"deactivated_by"`
	Reason        string    `json:"reason"`
	DeactivatedAt time.Time `json:"deactivated_at"`
}

// Event Types Constants
const (
	EventTypeUserCreated           = "user.created"
	EventTypeUserUpdated           = "user.updated"
	EventTypeUserDeleted           = "user.deleted"
	EventTypeUserDeactivated       = "user.deactivated"
	EventTypeTenantCreated         = "tenant.created"
	EventTypeTenantUpdated         = "tenant.updated"
	EventTypeTemplateCreated       = "template.created"
	EventTypeTemplatePublished     = "template.published"
	EventTypeTemplateInstalled     = "template.installed"
	EventTypeSubscriptionCreated   = "subscription.created"
	EventTypeSubscriptionUpdated   = "subscription.updated"
	EventTypeSubscriptionCancelled = "subscription.cancelled"
)
