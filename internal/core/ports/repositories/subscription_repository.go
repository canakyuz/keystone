package repositories

import (
	"context"
	"github.com/google/uuid"
	"nexspaces-api/internal/core/domain/subscription"
)

// SubscriptionRepository defines the interface for subscription data operations
type SubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, subscription *subscription.Subscription) error

	// GetByID retrieves a subscription by ID
	GetByID(ctx context.Context, subscriptionID uuid.UUID) (*subscription.Subscription, error)

	// GetByTenantID retrieves the active subscription for a tenant
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*subscription.Subscription, error)

	// Update updates an existing subscription
	Update(ctx context.Context, subscription *subscription.Subscription) error

	// Delete soft deletes a subscription
	Delete(ctx context.Context, subscriptionID uuid.UUID) error

	// List retrieves subscriptions with pagination
	List(ctx context.Context, limit, offset int) ([]*subscription.Subscription, error)

	// Count returns the total number of subscriptions
	Count(ctx context.Context) (int, error)

	// GetByStatus retrieves subscriptions by status
	GetByStatus(ctx context.Context, status subscription.Status) ([]*subscription.Subscription, error)

	// GetExpiring retrieves subscriptions expiring within a specified number of days
	GetExpiring(ctx context.Context, days int) ([]*subscription.Subscription, error)

	// UpdateStatus updates subscription status
	UpdateStatus(ctx context.Context, subscriptionID uuid.UUID, status subscription.Status) error

	// UpdateUsage updates current usage for a subscription
	UpdateUsage(ctx context.Context, subscriptionID uuid.UUID, usage subscription.CurrentUsage) error

	// GetByStripeID retrieves a subscription by Stripe subscription ID
	GetByStripeID(ctx context.Context, stripeID string) (*subscription.Subscription, error)

	// GetActiveByTenant retrieves the active subscription for a tenant
	GetActiveByTenant(ctx context.Context, tenantID uuid.UUID) (*subscription.Subscription, error)

	// GetActiveSubscriptionsCount returns the number of active subscriptions
	GetActiveSubscriptionsCount(ctx context.Context) (int, error)

	// GetSubscriptionsByPlan retrieves subscriptions by plan ID
	GetSubscriptionsByPlan(ctx context.Context, planID string) ([]*subscription.Subscription, error)

	// GetTrialSubscriptions retrieves subscriptions in trial period
	GetTrialSubscriptions(ctx context.Context) ([]*subscription.Subscription, error)
}
