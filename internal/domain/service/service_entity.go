package service

import "time"

// ServiceStatus represents service status
type ServiceStatus string

const (
	ServiceStatusActive   ServiceStatus = "active"
	ServiceStatusInactive ServiceStatus = "inactive"
	ServiceStatusArchived ServiceStatus = "archived"
)

// ServiceCategory represents service category
type ServiceCategory string

const (
	CategoryConsulting ServiceCategory = "consulting"
	CategoryTraining   ServiceCategory = "training"
	CategorySupport    ServiceCategory = "support"
	CategoryOther      ServiceCategory = "other"
)

// PricingModel represents pricing model
type PricingModel string

const (
	PricingOneTime      PricingModel = "one-time"
	PricingRecurring    PricingModel = "recurring"
	PricingSubscription PricingModel = "subscription"
	PricingUsageBased   PricingModel = "usage-based"
)

// Service represents a service offering
type Service struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"` // Multi-tenant isolation
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	Category     ServiceCategory        `json:"category"`
	Status       ServiceStatus          `json:"status"`
	Features     []string               `json:"features"`
	BasePrice    float64                `json:"base_price"`
	Currency     string                 `json:"currency"`
	PricingModel PricingModel           `json:"pricing_model"`
	BillingCycle string                 `json:"billing_cycle,omitempty"` // monthly, yearly
	Duration     int                    `json:"duration,omitempty"`      // in minutes/hours
	MaxClients   int                    `json:"max_clients,omitempty"`
	IsPublic     bool                   `json:"is_public"`
	Featured     bool                   `json:"featured"`
	Image        string                 `json:"image,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedBy    string                 `json:"created_by"`
	UpdatedBy    string                 `json:"updated_by"`
}

// Activate activates the service
func (s *Service) Activate() {
	s.Status = ServiceStatusActive
	s.UpdatedAt = time.Now()
}

// Deactivate deactivates the service
func (s *Service) Deactivate() {
	s.Status = ServiceStatusInactive
	s.UpdatedAt = time.Now()
}

// Archive archives the service
func (s *Service) Archive() {
	s.Status = ServiceStatusArchived
	s.UpdatedAt = time.Now()
}

// SetFeatured sets featured status
func (s *Service) SetFeatured(featured bool) {
	s.Featured = featured
	s.UpdatedAt = time.Now()
}
