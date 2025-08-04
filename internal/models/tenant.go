package models

import (
	"time"
)

type Tenant struct {
	ID           string    `json:"id" db:"id"`
	Slug         string    `json:"slug" db:"slug"`
	Name         string    `json:"name" db:"name"`
	CustomDomain *string   `json:"custom_domain" db:"custom_domain"`
	PlanID       *string   `json:"plan_id" db:"plan_id"`
	Status       string    `json:"status" db:"status"`
	Settings     string    `json:"settings" db:"settings"` // JSON string
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type TenantUser struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	Email      string    `json:"email" db:"email"`
	Password   string    `json:"-" db:"password_hash"`
	Name       string    `json:"name" db:"name"`
	Role       string    `json:"role" db:"role"`
	LastLogin  *time.Time `json:"last_login" db:"last_login"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Plan struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	Modules   string    `json:"modules" db:"modules"` // JSON array
	Limits    string    `json:"limits" db:"limits"`   // JSON object
	Pricing   string    `json:"pricing" db:"pricing"` // JSON object
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}