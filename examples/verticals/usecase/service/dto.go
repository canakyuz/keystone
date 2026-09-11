package service

import "time"

type CreateServiceRequest struct {
	Name         string   `json:"name" validate:"required"`
	Slug         string   `json:"slug" validate:"required"`
	Description  string   `json:"description,omitempty"`
	Category     string   `json:"category" validate:"required,oneof=consulting training support other"`
	Features     []string `json:"features,omitempty"`
	BasePrice    float64  `json:"base_price" validate:"required,gte=0"`
	Currency     string   `json:"currency" validate:"required"`
	PricingModel string   `json:"pricing_model" validate:"required,oneof=one-time recurring subscription usage-based"`
	BillingCycle string   `json:"billing_cycle,omitempty"`
	Duration     int      `json:"duration,omitempty"`
	MaxClients   int      `json:"max_clients,omitempty"`
	IsPublic     bool     `json:"is_public"`
	Image        string   `json:"image,omitempty"`
}

type UpdateServiceRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Features    []string `json:"features,omitempty"`
	BasePrice   float64  `json:"base_price,omitempty"`
	Status      string   `json:"status,omitempty"`
	Featured    bool     `json:"featured,omitempty"`
	Image       string   `json:"image,omitempty"`
}

type ServiceResponse struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Description  string         `json:"description,omitempty"`
	Category     string         `json:"category"`
	Status       string         `json:"status"`
	Features     []string       `json:"features,omitempty"`
	BasePrice    float64        `json:"base_price"`
	Currency     string         `json:"currency"`
	PricingModel string         `json:"pricing_model"`
	BillingCycle string         `json:"billing_cycle,omitempty"`
	Duration     int            `json:"duration,omitempty"`
	MaxClients   int            `json:"max_clients,omitempty"`
	IsPublic     bool           `json:"is_public"`
	Featured     bool           `json:"featured"`
	Image        string         `json:"image,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ServiceListResponse struct {
	Services []ServiceResponse `json:"services"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}
