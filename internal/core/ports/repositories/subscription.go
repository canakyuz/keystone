package repositories

import (
	"context"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/domain/subscription"
	"nexspaces-api/internal/core/domain/tenant"
)

// SubscriptionRepository defines the interface for subscription data access
type SubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, subscription *subscription.Subscription) error

	// GetByID retrieves a subscription by ID
	GetByID(ctx context.Context, id subscription.SubscriptionID) (*subscription.Subscription, error)

	// GetByTenant retrieves subscription for a tenant
	GetByTenant(ctx context.Context, tenantID tenant.TenantID) (*subscription.Subscription, error)

	// Update updates an existing subscription
	Update(ctx context.Context, subscription *subscription.Subscription) error

	// Delete deletes a subscription
	Delete(ctx context.Context, id subscription.SubscriptionID) error

	// List retrieves subscriptions with filtering and pagination
	List(ctx context.Context, filter SubscriptionFilter) (*SubscriptionList, error)

	// Count returns the total number of subscriptions matching the filter
	Count(ctx context.Context, filter SubscriptionFilter) (int64, error)

	// GetActiveSubscriptions retrieves all active subscriptions
	GetActiveSubscriptions(ctx context.Context) ([]*subscription.Subscription, error)

	// GetExpiringSubscriptions retrieves subscriptions expiring soon
	GetExpiringSubscriptions(ctx context.Context, days int) ([]*subscription.Subscription, error)

	// GetTrialingSubscriptions retrieves subscriptions in trial
	GetTrialingSubscriptions(ctx context.Context) ([]*subscription.Subscription, error)

	// GetExpiredTrials retrieves expired trial subscriptions
	GetExpiredTrials(ctx context.Context) ([]*subscription.Subscription, error)

	// GetPastDueSubscriptions retrieves past due subscriptions
	GetPastDueSubscriptions(ctx context.Context) ([]*subscription.Subscription, error)

	// GetCanceledSubscriptions retrieves canceled subscriptions
	GetCanceledSubscriptions(ctx context.Context, filter SubscriptionFilter) ([]*subscription.Subscription, error)

	// GetSubscriptionsPendingCancellation retrieves subscriptions scheduled for cancellation
	GetSubscriptionsPendingCancellation(ctx context.Context) ([]*subscription.Subscription, error)

	// UpdateUsage updates subscription usage
	UpdateUsage(ctx context.Context, subscriptionID subscription.SubscriptionID, usage subscription.Usage) error

	// GetUsage retrieves subscription usage
	GetUsage(ctx context.Context, subscriptionID subscription.SubscriptionID) (*subscription.Usage, error)

	// GetSubscriptionsByPlan retrieves subscriptions by plan
	GetSubscriptionsByPlan(ctx context.Context, planID subscription.PlanID) ([]*subscription.Subscription, error)

	// GetSubscriptionMetrics retrieves subscription metrics
	GetSubscriptionMetrics(ctx context.Context, subscriptionID subscription.SubscriptionID) (*SubscriptionMetrics, error)

	// GetRevenue retrieves revenue metrics
	GetRevenue(ctx context.Context, filter RevenueFilter) (*RevenueMetrics, error)
}

// PlanRepository defines the interface for plan data access
type PlanRepository interface {
	// Create creates a new plan
	Create(ctx context.Context, plan *subscription.Plan) error

	// GetByID retrieves a plan by ID
	GetByID(ctx context.Context, id subscription.PlanID) (*subscription.Plan, error)

	// GetBySlug retrieves a plan by slug
	GetBySlug(ctx context.Context, slug string) (*subscription.Plan, error)

	// Update updates an existing plan
	Update(ctx context.Context, plan *subscription.Plan) error

	// Delete deletes a plan
	Delete(ctx context.Context, id subscription.PlanID) error

	// List retrieves plans with filtering
	List(ctx context.Context, filter PlanFilter) ([]*subscription.Plan, error)

	// GetActive retrieves all active plans
	GetActive(ctx context.Context) ([]*subscription.Plan, error)

	// ExistsBySlug checks if a plan with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// Count returns the total number of plans matching the filter
	Count(ctx context.Context, filter PlanFilter) (int64, error)

	// GetFeatured retrieves featured plans
	GetFeatured(ctx context.Context) ([]*subscription.Plan, error)

	// GetDefaultPlan retrieves the default plan
	GetDefaultPlan(ctx context.Context) (*subscription.Plan, error)
}

// SubscriptionFilter represents filtering options for subscription queries
type SubscriptionFilter struct {
	// Status filtering
	Status *subscription.SubscriptionStatus

	// Plan filtering
	PlanID *subscription.PlanID

	// Billing cycle filtering
	BillingCycle *subscription.BillingCycle

	// Date range filtering
	CreatedAfter  *shared.Timestamp
	CreatedBefore *shared.Timestamp

	// Trial filtering
	IsTrialing   *bool
	TrialExpired *bool

	// Cancellation filtering
	IsCanceled          *bool
	CanceledAfter       *shared.Timestamp
	CanceledBefore      *shared.Timestamp
	PendingCancellation *bool

	// Revenue filtering
	MinRevenue *int64
	MaxRevenue *int64

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // created_at, updated_at, next_billing_date, revenue
	SortOrder string // asc, desc

	// Include related data
	IncludeUsage   bool
	IncludeTenant  bool
	IncludePlan    bool
	IncludeMetrics bool
}

// PlanFilter represents filtering options for plan queries
type PlanFilter struct {
	// Active status filtering
	IsActive *bool

	// Price range filtering
	MinMonthlyPrice *int64
	MaxMonthlyPrice *int64
	MinYearlyPrice  *int64
	MaxYearlyPrice  *int64

	// Features filtering
	HasFeature string

	// Trial filtering
	HasTrial *bool

	// Search term
	Search string

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // name, created_at, monthly_price, yearly_price
	SortOrder string // asc, desc

	// Include related data
	IncludeSubscriptionCount bool
}

// SubscriptionList represents a paginated list of subscriptions
type SubscriptionList struct {
	Items      []*subscription.Subscription `json:"items"`
	Total      int64                        `json:"total"`
	Limit      int                          `json:"limit"`
	Offset     int                          `json:"offset"`
	HasMore    bool                         `json:"has_more"`
	TotalPages int                          `json:"total_pages"`
}

// SubscriptionMetrics represents subscription metrics
type SubscriptionMetrics struct {
	SubscriptionID     subscription.SubscriptionID `json:"subscription_id"`
	TenantID           tenant.TenantID             `json:"tenant_id"`
	PlanID             subscription.PlanID         `json:"plan_id"`
	TotalRevenue       shared.Money                `json:"total_revenue"`
	MonthlyRevenue     shared.Money                `json:"monthly_revenue"`
	YearlyRevenue      shared.Money                `json:"yearly_revenue"`
	DaysActive         int                         `json:"days_active"`
	PaymentCount       int                         `json:"payment_count"`
	FailedPaymentCount int                         `json:"failed_payment_count"`
	LastPaymentAt      *shared.Timestamp           `json:"last_payment_at"`
	NextPaymentAt      *shared.Timestamp           `json:"next_payment_at"`
	ChurnRisk          ChurnRisk                   `json:"churn_risk"`
	UsagePercent       map[string]float64          `json:"usage_percent"` // usage as percentage of limits
	CreatedAt          shared.Timestamp            `json:"created_at"`
	UpdatedAt          shared.Timestamp            `json:"updated_at"`
}

// ChurnRisk represents churn risk level
type ChurnRisk string

const (
	ChurnRiskLow    ChurnRisk = "low"
	ChurnRiskMedium ChurnRisk = "medium"
	ChurnRiskHigh   ChurnRisk = "high"
)

// RevenueFilter represents filtering options for revenue queries
type RevenueFilter struct {
	// Date range
	StartDate shared.Timestamp
	EndDate   shared.Timestamp

	// Plan filtering
	PlanID *subscription.PlanID

	// Billing cycle filtering
	BillingCycle *subscription.BillingCycle

	// Aggregation period
	Period RevenuePeriod // daily, weekly, monthly, yearly

	// Currency filtering
	Currency *string

	// Include projections
	IncludeProjections bool
}

// RevenuePeriod represents revenue aggregation period
type RevenuePeriod string

const (
	RevenuePeriodDaily   RevenuePeriod = "daily"
	RevenuePeriodWeekly  RevenuePeriod = "weekly"
	RevenuePeriodMonthly RevenuePeriod = "monthly"
	RevenuePeriodYearly  RevenuePeriod = "yearly"
)

// RevenueMetrics represents revenue metrics
type RevenueMetrics struct {
	Period                RevenuePeriod           `json:"period"`
	StartDate             shared.Timestamp        `json:"start_date"`
	EndDate               shared.Timestamp        `json:"end_date"`
	TotalRevenue          shared.Money            `json:"total_revenue"`
	RecurringRevenue      shared.Money            `json:"recurring_revenue"`
	OneTimeRevenue        shared.Money            `json:"one_time_revenue"`
	ProjectedRevenue      *shared.Money           `json:"projected_revenue,omitempty"`
	RevenueByPlan         map[string]shared.Money `json:"revenue_by_plan"`
	SubscriptionCount     int64                   `json:"subscription_count"`
	NewSubscriptions      int64                   `json:"new_subscriptions"`
	CanceledSubscriptions int64                   `json:"canceled_subscriptions"`
	ChurnRate             float64                 `json:"churn_rate"`
	GrowthRate            float64                 `json:"growth_rate"`
	AverageRevenuePerUser shared.Money            `json:"average_revenue_per_user"`
	LifetimeValue         shared.Money            `json:"lifetime_value"`
	TimeSeriesData        []RevenueDataPoint      `json:"time_series_data,omitempty"`
}

// RevenueDataPoint represents a single data point in revenue time series
type RevenueDataPoint struct {
	Date                  shared.Timestamp `json:"date"`
	Revenue               shared.Money     `json:"revenue"`
	SubscriptionCount     int64            `json:"subscription_count"`
	NewSubscriptions      int64            `json:"new_subscriptions"`
	CanceledSubscriptions int64            `json:"canceled_subscriptions"`
}

// PaymentRecord represents a payment record
type PaymentRecord struct {
	ID             shared.ID                   `json:"id"`
	SubscriptionID subscription.SubscriptionID `json:"subscription_id"`
	TenantID       tenant.TenantID             `json:"tenant_id"`
	Amount         shared.Money                `json:"amount"`
	Currency       string                      `json:"currency"`
	Status         PaymentStatus               `json:"status"`
	PaymentMethod  string                      `json:"payment_method"`
	TransactionID  string                      `json:"transaction_id"`
	ProviderID     string                      `json:"provider_id"`  // Stripe, PayPal, etc.
	ProviderRef    string                      `json:"provider_ref"` // Provider's reference
	Description    string                      `json:"description"`
	Metadata       map[string]interface{}      `json:"metadata"`
	FailureReason  *string                     `json:"failure_reason"`
	ProcessedAt    *shared.Timestamp           `json:"processed_at"`
	RefundedAt     *shared.Timestamp           `json:"refunded_at"`
	RefundAmount   *shared.Money               `json:"refund_amount"`
	CreatedAt      shared.Timestamp            `json:"created_at"`
	UpdatedAt      shared.Timestamp            `json:"updated_at"`
}

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// PaymentRepository defines the interface for payment data access
type PaymentRepository interface {
	// Create creates a new payment record
	Create(ctx context.Context, payment *PaymentRecord) error

	// GetByID retrieves a payment by ID
	GetByID(ctx context.Context, id shared.ID) (*PaymentRecord, error)

	// GetBySubscription retrieves payments for a subscription
	GetBySubscription(ctx context.Context, subscriptionID subscription.SubscriptionID) ([]*PaymentRecord, error)

	// GetByTenant retrieves payments for a tenant
	GetByTenant(ctx context.Context, tenantID tenant.TenantID, filter PaymentFilter) ([]*PaymentRecord, error)

	// Update updates a payment record
	Update(ctx context.Context, payment *PaymentRecord) error

	// List retrieves payments with filtering and pagination
	List(ctx context.Context, filter PaymentFilter) (*PaymentList, error)

	// GetFailedPayments retrieves failed payments
	GetFailedPayments(ctx context.Context, hours int) ([]*PaymentRecord, error)

	// GetSuccessfulPayments retrieves successful payments in date range
	GetSuccessfulPayments(ctx context.Context, startDate, endDate shared.Timestamp) ([]*PaymentRecord, error)
}

// PaymentFilter represents filtering options for payment queries
type PaymentFilter struct {
	// Status filtering
	Status *PaymentStatus

	// Subscription filtering
	SubscriptionID *subscription.SubscriptionID

	// Tenant filtering
	TenantID *tenant.TenantID

	// Amount range filtering
	MinAmount *int64
	MaxAmount *int64

	// Date range filtering
	CreatedAfter  *shared.Timestamp
	CreatedBefore *shared.Timestamp

	// Payment method filtering
	PaymentMethod *string

	// Provider filtering
	Provider *string

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // created_at, amount, status
	SortOrder string // asc, desc
}

// PaymentList represents a paginated list of payments
type PaymentList struct {
	Items      []*PaymentRecord `json:"items"`
	Total      int64            `json:"total"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
	HasMore    bool             `json:"has_more"`
	TotalPages int              `json:"total_pages"`
}

// Validation methods
func (f *SubscriptionFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 1000 {
		return shared.NewValidationError("limit must be between 0 and 1000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	validSortFields := []string{"created_at", "updated_at", "next_billing_date"}
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

func (f *SubscriptionFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	if f.SortBy == "" {
		f.SortBy = "created_at"
	}

	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}

func (f *PlanFilter) Validate() error {
	if f.Limit < 0 || f.Limit > 1000 {
		return shared.NewValidationError("limit must be between 0 and 1000")
	}

	if f.Offset < 0 {
		return shared.NewValidationError("offset must be non-negative")
	}

	return nil
}

func (f *PlanFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	if f.SortBy == "" {
		f.SortBy = "name"
	}

	if f.SortOrder == "" {
		f.SortOrder = "asc"
	}
}

// Helper methods
func (p *PaymentRecord) IsSuccessful() bool {
	return p.Status == PaymentStatusSucceeded
}

func (p *PaymentRecord) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

func (p *PaymentRecord) IsRefunded() bool {
	return p.Status == PaymentStatusRefunded
}

func (s *SubscriptionMetrics) CalculateChurnRisk() ChurnRisk {
	// Simple churn risk calculation
	if s.FailedPaymentCount > 2 {
		return ChurnRiskHigh
	}
	if s.FailedPaymentCount > 0 {
		return ChurnRiskMedium
	}
	return ChurnRiskLow
}
