package subscription

import (
	"strings"
	"time"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/tenant"
	"nexspaces-api/internal/core/domain/user"
)

// SubscriptionID represents a unique subscription identifier
type SubscriptionID = shared.ID

// Subscription represents a tenant's subscription
type Subscription struct {
	id              SubscriptionID
	tenantID        tenant.TenantID
	planID          PlanID
	status          SubscriptionStatus
	billingCycle    BillingCycle
	currentPeriod   BillingPeriod
	nextBillingDate shared.Timestamp
	cancelAt        *shared.Timestamp
	canceledAt      *shared.Timestamp
	trialEndsAt     *shared.Timestamp
	usage           Usage
	metadata        map[string]interface{}
	createdAt       shared.Timestamp
	updatedAt       shared.Timestamp
	version         int64 // For optimistic locking
}

// PlanID represents a unique plan identifier
type PlanID = shared.ID

// Plan represents a subscription plan
type Plan struct {
	id          PlanID
	name        string
	slug        shared.Slug
	description string
	features    []Feature
	limits      Limits
	pricing     PlanPricing
	trialDays   int
	isActive    bool
	metadata    map[string]interface{}
	createdAt   shared.Timestamp
	updatedAt   shared.Timestamp
}

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	StatusActive            SubscriptionStatus = "active"
	StatusTrialing          SubscriptionStatus = "trialing"
	StatusPastDue           SubscriptionStatus = "past_due"
	StatusCanceled          SubscriptionStatus = "canceled"
	StatusUnpaid            SubscriptionStatus = "unpaid"
	StatusIncomplete        SubscriptionStatus = "incomplete"
	StatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	StatusPaused            SubscriptionStatus = "paused"
)

// IsValid checks if subscription status is valid
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusTrialing, StatusPastDue, StatusCanceled,
		StatusUnpaid, StatusIncomplete, StatusIncompleteExpired, StatusPaused:
		return true
	default:
		return false
	}
}

// BillingCycle represents billing cycle
type BillingCycle string

const (
	CycleMonthly BillingCycle = "monthly"
	CycleYearly  BillingCycle = "yearly"
)

// IsValid checks if billing cycle is valid
func (b BillingCycle) IsValid() bool {
	switch b {
	case CycleMonthly, CycleYearly:
		return true
	default:
		return false
	}
}

// BillingPeriod represents a billing period
type BillingPeriod struct {
	Start shared.Timestamp `json:"start"`
	End   shared.Timestamp `json:"end"`
}

// Feature represents a plan feature
type Feature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Included    bool   `json:"included"`
	Limit       *int   `json:"limit,omitempty"`        // nil means unlimited
	Unit        string `json:"unit,omitempty"`         // users, templates, MB, etc.
	MeteredRate *int   `json:"metered_rate,omitempty"` // cost per unit if metered
}

// Limits represents plan limits
type Limits struct {
	Users         int `json:"users"`          // Max users
	Templates     int `json:"templates"`      // Max templates
	Storage       int `json:"storage"`        // Storage in MB
	Bandwidth     int `json:"bandwidth"`      // Bandwidth in MB/month
	APICalls      int `json:"api_calls"`      // API calls per month
	Domains       int `json:"domains"`        // Custom domains
	Integrations  int `json:"integrations"`   // Third-party integrations
	AdminUsers    int `json:"admin_users"`    // Max admin users
	DeveloperSeat int `json:"developer_seat"` // Developer seats
}

// PlanPricing represents plan pricing
type PlanPricing struct {
	Monthly shared.Money  `json:"monthly"`
	Yearly  shared.Money  `json:"yearly"`
	Setup   *shared.Money `json:"setup,omitempty"`    // One-time setup fee
	PerUser *shared.Money `json:"per_user,omitempty"` // Per-user pricing
}

// Usage represents current usage
type Usage struct {
	Users        int              `json:"users"`
	Templates    int              `json:"templates"`
	Storage      int64            `json:"storage"`   // Bytes used
	Bandwidth    int64            `json:"bandwidth"` // Bytes used this period
	APICalls     int64            `json:"api_calls"` // API calls this period
	Domains      int              `json:"domains"`
	Integrations int              `json:"integrations"`
	LastUpdated  shared.Timestamp `json:"last_updated"`
}

// SubscriptionParams contains parameters for creating a subscription
type SubscriptionParams struct {
	TenantID     tenant.TenantID
	PlanID       PlanID
	BillingCycle BillingCycle
	StartTrial   bool
}

// NewSubscription creates a new subscription entity
func NewSubscription(params SubscriptionParams) (*Subscription, error) {
	// Validate required fields
	if params.TenantID.IsZero() {
		return nil, shared.NewValidationError("tenant ID is required")
	}

	if params.PlanID.IsZero() {
		return nil, shared.NewValidationError("plan ID is required")
	}

	if !params.BillingCycle.IsValid() {
		return nil, shared.NewValidationError("invalid billing cycle")
	}

	now := shared.Now()

	// Calculate billing period
	var nextBilling shared.Timestamp
	var currentPeriod BillingPeriod

	if params.BillingCycle == CycleMonthly {
		nextBilling = shared.NewTimestamp(now.Time().AddDate(0, 1, 0))
		currentPeriod = BillingPeriod{
			Start: now,
			End:   nextBilling,
		}
	} else { // Yearly
		nextBilling = shared.NewTimestamp(now.Time().AddDate(1, 0, 0))
		currentPeriod = BillingPeriod{
			Start: now,
			End:   nextBilling,
		}
	}

	status := StatusActive
	var trialEndsAt *shared.Timestamp

	// Set trial period if requested
	if params.StartTrial {
		status = StatusTrialing
		trialEnd := shared.NewTimestamp(now.Time().AddDate(0, 0, 14)) // 14 days trial
		trialEndsAt = &trialEnd
	}

	return &Subscription{
		id:              shared.NewID(),
		tenantID:        params.TenantID,
		planID:          params.PlanID,
		status:          status,
		billingCycle:    params.BillingCycle,
		currentPeriod:   currentPeriod,
		nextBillingDate: nextBilling,
		cancelAt:        nil,
		canceledAt:      nil,
		trialEndsAt:     trialEndsAt,
		usage:           Usage{LastUpdated: now},
		metadata:        make(map[string]interface{}),
		createdAt:       now,
		updatedAt:       now,
		version:         1,
	}, nil
}

// ReConstituteSubscription recreates a subscription from stored data
func ReConstituteSubscription(
	id SubscriptionID,
	tenantID tenant.TenantID,
	planID PlanID,
	status SubscriptionStatus,
	billingCycle BillingCycle,
	currentPeriod BillingPeriod,
	nextBillingDate shared.Timestamp,
	cancelAt *shared.Timestamp,
	canceledAt *shared.Timestamp,
	trialEndsAt *shared.Timestamp,
	usage Usage,
	metadata map[string]interface{},
	createdAt shared.Timestamp,
	updatedAt shared.Timestamp,
	version int64,
) *Subscription {
	return &Subscription{
		id:              id,
		tenantID:        tenantID,
		planID:          planID,
		status:          status,
		billingCycle:    billingCycle,
		currentPeriod:   currentPeriod,
		nextBillingDate: nextBillingDate,
		cancelAt:        cancelAt,
		canceledAt:      canceledAt,
		trialEndsAt:     trialEndsAt,
		usage:           usage,
		metadata:        metadata,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		version:         version,
	}
}

// Plan constructors
func NewPlan(name, slug, description string, features []Feature, limits Limits, pricing PlanPricing, trialDays int) (*Plan, error) {
	if strings.TrimSpace(name) == "" {
		return nil, shared.NewValidationError("plan name is required")
	}

	planSlug, err := shared.NewSlug(slug)
	if err != nil {
		return nil, shared.WrapDomainError(err, shared.ValidationError, "invalid plan slug")
	}

	now := shared.Now()

	return &Plan{
		id:          shared.NewID(),
		name:        strings.TrimSpace(name),
		slug:        planSlug,
		description: description,
		features:    features,
		limits:      limits,
		pricing:     pricing,
		trialDays:   trialDays,
		isActive:    true,
		metadata:    make(map[string]interface{}),
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// Getters - Subscription
func (s *Subscription) ID() SubscriptionID {
	return s.id
}

func (s *Subscription) TenantID() tenant.TenantID {
	return s.tenantID
}

func (s *Subscription) PlanID() PlanID {
	return s.planID
}

func (s *Subscription) Status() SubscriptionStatus {
	return s.status
}

func (s *Subscription) BillingCycle() BillingCycle {
	return s.billingCycle
}

func (s *Subscription) CurrentPeriod() BillingPeriod {
	return s.currentPeriod
}

func (s *Subscription) NextBillingDate() shared.Timestamp {
	return s.nextBillingDate
}

func (s *Subscription) CancelAt() *shared.Timestamp {
	return s.cancelAt
}

func (s *Subscription) CanceledAt() *shared.Timestamp {
	return s.canceledAt
}

func (s *Subscription) TrialEndsAt() *shared.Timestamp {
	return s.trialEndsAt
}

func (s *Subscription) Usage() Usage {
	return s.usage
}

func (s *Subscription) Metadata() map[string]interface{} {
	return s.metadata
}

func (s *Subscription) CreatedAt() shared.Timestamp {
	return s.createdAt
}

func (s *Subscription) UpdatedAt() shared.Timestamp {
	return s.updatedAt
}

func (s *Subscription) Version() int64 {
	return s.version
}

// Getters - Plan
func (p *Plan) ID() PlanID {
	return p.id
}

func (p *Plan) Name() string {
	return p.name
}

func (p *Plan) Slug() shared.Slug {
	return p.slug
}

func (p *Plan) Description() string {
	return p.description
}

func (p *Plan) Features() []Feature {
	return p.features
}

func (p *Plan) Limits() Limits {
	return p.limits
}

func (p *Plan) Pricing() PlanPricing {
	return p.pricing
}

func (p *Plan) TrialDays() int {
	return p.trialDays
}

func (p *Plan) IsActive() bool {
	return p.isActive
}

func (p *Plan) CreatedAt() shared.Timestamp {
	return p.createdAt
}

func (p *Plan) UpdatedAt() shared.Timestamp {
	return p.updatedAt
}

// Business Methods - Subscription

// ChangePlan changes subscription plan
func (s *Subscription) ChangePlan(newPlanID PlanID) error {
	if s.IsInactive() {
		return shared.NewBusinessRuleError("cannot change plan for inactive subscription")
	}

	if s.planID == newPlanID {
		return nil // Same plan
	}

	s.planID = newPlanID
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// ChangeBillingCycle changes billing cycle
func (s *Subscription) ChangeBillingCycle(cycle BillingCycle) error {
	if !cycle.IsValid() {
		return shared.NewValidationError("invalid billing cycle")
	}

	if s.IsInactive() {
		return shared.NewBusinessRuleError("cannot change billing cycle for inactive subscription")
	}

	s.billingCycle = cycle
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// Cancel schedules subscription for cancellation
func (s *Subscription) Cancel(cancelAt shared.Timestamp) error {
	if s.IsCanceled() {
		return nil // Already canceled
	}

	if s.IsInactive() {
		return shared.NewBusinessRuleError("cannot cancel inactive subscription")
	}

	s.cancelAt = &cancelAt
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// CancelImmediately cancels subscription immediately
func (s *Subscription) CancelImmediately() error {
	if s.IsCanceled() {
		return nil // Already canceled
	}

	now := shared.Now()
	s.status = StatusCanceled
	s.canceledAt = &now
	s.updatedAt = now
	s.version++

	return nil
}

// Reactivate reactivates canceled subscription
func (s *Subscription) Reactivate() error {
	if !s.IsCanceled() && s.cancelAt == nil {
		return nil // Already active
	}

	s.status = StatusActive
	s.cancelAt = nil
	s.canceledAt = nil
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// Pause pauses subscription
func (s *Subscription) Pause() error {
	if s.status == StatusPaused {
		return nil // Already paused
	}

	if s.IsInactive() {
		return shared.NewBusinessRuleError("cannot pause inactive subscription")
	}

	s.status = StatusPaused
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// Resume resumes paused subscription
func (s *Subscription) Resume() error {
	if s.status != StatusPaused {
		return shared.NewBusinessRuleError("only paused subscriptions can be resumed")
	}

	s.status = StatusActive
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// ProcessPayment processes successful payment
func (s *Subscription) ProcessPayment() error {
	now := shared.Now()

	// Update status if past due or unpaid
	if s.status == StatusPastDue || s.status == StatusUnpaid {
		s.status = StatusActive
	}

	// Update billing period
	if s.billingCycle == CycleMonthly {
		s.nextBillingDate = shared.NewTimestamp(s.nextBillingDate.Time().AddDate(0, 1, 0))
		s.currentPeriod = BillingPeriod{
			Start: s.currentPeriod.End,
			End:   s.nextBillingDate,
		}
	} else { // Yearly
		s.nextBillingDate = shared.NewTimestamp(s.nextBillingDate.Time().AddDate(1, 0, 0))
		s.currentPeriod = BillingPeriod{
			Start: s.currentPeriod.End,
			End:   s.nextBillingDate,
		}
	}

	s.updatedAt = now
	s.version++

	return nil
}

// ProcessFailedPayment processes failed payment
func (s *Subscription) ProcessFailedPayment() error {
	if s.IsInactive() {
		return nil // Already inactive
	}

	s.status = StatusPastDue
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// EndTrial ends trial period
func (s *Subscription) EndTrial() error {
	if s.status != StatusTrialing {
		return shared.NewBusinessRuleError("subscription is not in trial")
	}

	s.status = StatusActive
	s.trialEndsAt = nil
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// UpdateUsage updates usage metrics
func (s *Subscription) UpdateUsage(usage Usage) error {
	s.usage = usage
	s.usage.LastUpdated = shared.Now()
	s.updatedAt = shared.Now()
	s.version++

	return nil
}

// SetMetadata sets metadata key-value pair
func (s *Subscription) SetMetadata(key string, value interface{}) {
	if s.metadata == nil {
		s.metadata = make(map[string]interface{})
	}

	s.metadata[key] = value
	s.updatedAt = shared.Now()
	s.version++
}

// Business Logic Queries - Subscription

// IsActive checks if subscription is active
func (s *Subscription) IsActive() bool {
	return s.status == StatusActive
}

// IsTrialing checks if subscription is trialing
func (s *Subscription) IsTrialing() bool {
	return s.status == StatusTrialing
}

// IsCanceled checks if subscription is canceled
func (s *Subscription) IsCanceled() bool {
	return s.status == StatusCanceled
}

// IsInactive checks if subscription is inactive
func (s *Subscription) IsInactive() bool {
	return s.status == StatusCanceled || s.status == StatusUnpaid ||
		s.status == StatusIncompleteExpired
}

// IsPastDue checks if subscription is past due
func (s *Subscription) IsPastDue() bool {
	return s.status == StatusPastDue
}

// ShouldBeCanceled checks if subscription should be canceled now
func (s *Subscription) ShouldBeCanceled() bool {
	if s.cancelAt == nil {
		return false
	}

	return shared.Now().After(*s.cancelAt)
}

// IsTrialExpired checks if trial period has expired
func (s *Subscription) IsTrialExpired() bool {
	if s.trialEndsAt == nil {
		return false
	}

	return shared.Now().After(*s.trialEndsAt)
}

// CanUseFeature checks if subscription allows feature usage
func (s *Subscription) CanUseFeature(featureName string, plan *Plan) bool {
	if s.IsInactive() {
		return false
	}

	// Check if feature is included in plan
	for _, feature := range plan.Features() {
		if feature.Name == featureName {
			return feature.Included
		}
	}

	return false
}

// HasExceededLimit checks if usage has exceeded plan limit
func (s *Subscription) HasExceededLimit(resource string, plan *Plan) bool {
	limits := plan.Limits()
	usage := s.usage

	switch resource {
	case "users":
		return usage.Users > limits.Users
	case "templates":
		return usage.Templates > limits.Templates
	case "storage":
		return usage.Storage > int64(limits.Storage)*1024*1024 // Convert MB to bytes
	case "bandwidth":
		return usage.Bandwidth > int64(limits.Bandwidth)*1024*1024
	case "api_calls":
		return usage.APICalls > int64(limits.APICalls)
	case "domains":
		return usage.Domains > limits.Domains
	case "integrations":
		return usage.Integrations > limits.Integrations
	default:
		return false
	}
}

// GetRemainingLimit returns remaining limit for resource
func (s *Subscription) GetRemainingLimit(resource string, plan *Plan) int64 {
	limits := plan.Limits()
	usage := s.usage

	switch resource {
	case "users":
		return int64(limits.Users - usage.Users)
	case "templates":
		return int64(limits.Templates - usage.Templates)
	case "storage":
		return int64(limits.Storage)*1024*1024 - usage.Storage
	case "bandwidth":
		return int64(limits.Bandwidth)*1024*1024 - usage.Bandwidth
	case "api_calls":
		return int64(limits.APICalls) - usage.APICalls
	case "domains":
		return int64(limits.Domains - usage.Domains)
	case "integrations":
		return int64(limits.Integrations - usage.Integrations)
	default:
		return 0
	}
}

// Business Methods - Plan

// Activate activates plan
func (p *Plan) Activate() {
	p.isActive = true
	p.updatedAt = shared.Now()
}

// Deactivate deactivates plan
func (p *Plan) Deactivate() {
	p.isActive = false
	p.updatedAt = shared.Now()
}

// UpdatePricing updates plan pricing
func (p *Plan) UpdatePricing(pricing PlanPricing) error {
	p.pricing = pricing
	p.updatedAt = shared.Now()
	return nil
}

// UpdateLimits updates plan limits
func (p *Plan) UpdateLimits(limits Limits) error {
	p.limits = limits
	p.updatedAt = shared.Now()
	return nil
}

// GetPriceForCycle returns price for billing cycle
func (p *Plan) GetPriceForCycle(cycle BillingCycle) shared.Money {
	if cycle == CycleYearly {
		return p.pricing.Yearly
	}
	return p.pricing.Monthly
}

// Events

// SubscriptionCreatedEvent represents subscription creation
type SubscriptionCreatedEvent struct {
	SubscriptionID SubscriptionID
	TenantID       tenant.TenantID
	PlanID         PlanID
	BillingCycle   string
	CreatedAt      time.Time
}

// SubscriptionCanceledEvent represents subscription cancellation
type SubscriptionCanceledEvent struct {
	SubscriptionID SubscriptionID
	TenantID       tenant.TenantID
	CanceledAt     time.Time
	Reason         string
}

// PaymentProcessedEvent represents successful payment
type PaymentProcessedEvent struct {
	SubscriptionID SubscriptionID
	TenantID       tenant.TenantID
	Amount         shared.Money
	ProcessedAt    time.Time
}

// Generate events
func (s *Subscription) GenerateCreatedEvent() SubscriptionCreatedEvent {
	return SubscriptionCreatedEvent{
		SubscriptionID: s.id,
		TenantID:       s.tenantID,
		PlanID:         s.planID,
		BillingCycle:   string(s.billingCycle),
		CreatedAt:      s.createdAt.Time(),
	}
}

func (s *Subscription) GenerateCanceledEvent(reason string) SubscriptionCanceledEvent {
	return SubscriptionCanceledEvent{
		SubscriptionID: s.id,
		TenantID:       s.tenantID,
		CanceledAt:     s.updatedAt.Time(),
		Reason:         reason,
	}
}

func (s *Subscription) GeneratePaymentProcessedEvent(amount shared.Money) PaymentProcessedEvent {
	return PaymentProcessedEvent{
		SubscriptionID: s.id,
		TenantID:       s.tenantID,
		Amount:         amount,
		ProcessedAt:    s.updatedAt.Time(),
	}
}
