package middleware

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/pkg/cache"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/ratelimit"
)

// planQuotas maps a subscription plan to a per-minute request allowance.
//
// The values are a product decision, gathered here in one table rather than scattered
// through the code as magic numbers.
var planQuotas = map[string]ratelimit.Quota{
	"free":       ratelimit.PerMinute(60),
	"starter":    ratelimit.PerMinute(300),
	"pro":        ratelimit.PerMinute(1_200),
	"enterprise": ratelimit.PerMinute(6_000),
}

// anonymousQuota applies to unauthenticated requests. The tenant is unknown, so the
// key is the IP and the limit is deliberately tight.
var anonymousQuota = ratelimit.PerMinute(30)

// fallbackQuota applies when the plan cannot be resolved. The lowest plan is chosen:
// a resolution failure must not turn into a free upgrade.
var fallbackQuota = planQuotas["free"]

// RateLimitConfig configures the limiter middleware.
type RateLimitConfig struct {
	// Limiter is required.
	Limiter ratelimit.Limiter

	// Plans resolves tenant_id to a plan. It may be nil, in which case every
	// authenticated request is limited by fallbackQuota.
	Plans *TenantPlanCache

	// SkipPaths are the paths exempt from limiting.
	// The health endpoints must be exempt: if a load balancer probe is rejected for
	// hitting the limit, a healthy instance is pulled out of the pool.
	SkipPaths map[string]bool

	Logger *logger.Logger
}

// RateLimit builds the rate limiting middleware.
//
// Key selection:
//   - tenant_id when authenticated, so the limit applies per tenant and one tenant's
//     traffic cannot affect another's.
//   - The client IP otherwise.
//
// This replaces Fiber's built-in limiter. The built-in one keeps an in-process store
// by default: across three replicas, a setting of 100 requests per replica actually
// means 300. It also supports neither plan-based quotas nor a tenant-based key.
func RateLimit(cfg RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.SkipPaths[c.Path()] {
			return c.Next()
		}

		key, quota := resolveKeyAndQuota(c, cfg)

		result, err := cfg.Limiter.Allow(c.Context(), key, quota)
		if err != nil && cfg.Logger != nil {
			cfg.Logger.WithFields(logger.Fields{
				"key":   key,
				"error": err.Error(),
			}).Warn("rate limit query failed")
		}

		writeRateLimitHeaders(c, result)

		if !result.Allowed {
			return tooManyRequests(c, result)
		}

		return c.Next()
	}
}

// resolveKeyAndQuota decides which key and quota limit a request.
func resolveKeyAndQuota(c *fiber.Ctx, cfg RateLimitConfig) (string, ratelimit.Quota) {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "ip:" + c.IP(), anonymousQuota
	}

	return "tenant:" + tenantID, quotaForTenant(c.Context(), cfg, tenantID)
}

// quotaForTenant returns the quota matching the tenant's plan.
func quotaForTenant(ctx context.Context, cfg RateLimitConfig, tenantID string) ratelimit.Quota {
	if cfg.Plans == nil {
		return fallbackQuota
	}

	plan, err := cfg.Plans.GetPlan(ctx, tenantID)
	if err != nil {
		return fallbackQuota
	}

	if quota, ok := planQuotas[plan]; ok {
		return quota
	}

	return fallbackQuota
}

// writeRateLimitHeaders reports the state so clients can pace themselves. Without
// these headers a client only finds out by receiving a 429.
func writeRateLimitHeaders(c *fiber.Ctx, result ratelimit.Result) {
	c.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
}

// tooManyRequests returns the 429 response together with Retry-After.
func tooManyRequests(c *fiber.Ctx, result ratelimit.Result) error {
	seconds := int(result.RetryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	c.Set("Retry-After", strconv.Itoa(seconds))

	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"type":   "https://keystone.dev/problems/rate-limit-exceeded",
		"title":  "Rate limit exceeded",
		"status": fiber.StatusTooManyRequests,
		"detail": "Too many requests. See the Retry-After header.",
	})
}

// --- plan cache -----------------------------------------------------------

const (
	tenantPlanKeyPrefix = "tenant:plan:"
	planTTL             = 5 * time.Minute
	planLocalTTL        = 30 * time.Second
	planNegativeTTL     = 15 * time.Second
)

// TenantPlanCache caches a tenant's subscription plan.
//
// Going to the database for the plan on every request would turn the limiter itself
// into a source of load. pkg/cache is reused: the same stampede protection and
// negative caching apply here too.
type TenantPlanCache struct {
	db    *sql.DB
	cache *cache.TwoTier
}

// NewTenantPlanCache builds the plan cache. rdb may be nil.
func NewTenantPlanCache(rdb *redis.Client, db *sql.DB) *TenantPlanCache {
	c := &TenantPlanCache{db: db}

	var redisClient cache.RedisClient
	if rdb != nil {
		redisClient = rdb
	}

	c.cache = cache.New(cache.Config{
		Redis:       redisClient,
		Loader:      c.loadPlanFromDB,
		KeyPrefix:   tenantPlanKeyPrefix,
		TTL:         planTTL,
		L1TTL:       planLocalTTL,
		NegativeTTL: planNegativeTTL,
		Jitter:      defaultJitter,
	})

	return c
}

// GetPlan returns the tenant's plan.
func (c *TenantPlanCache) GetPlan(ctx context.Context, tenantID string) (string, error) {
	plan, err := c.cache.Get(ctx, tenantID)
	if errors.Is(err, cache.ErrNotFound) {
		return "", ErrTenantNotFound
	}

	return plan, err
}

// Invalidate must be called when a plan changes.
func (c *TenantPlanCache) Invalidate(ctx context.Context, tenantID string) error {
	return c.cache.Invalidate(ctx, tenantID)
}

// loadPlanFromDB fetches the plan from the source of truth.
func (c *TenantPlanCache) loadPlanFromDB(ctx context.Context, tenantID string) (string, error) {
	const query = `SELECT plan FROM tenants WHERE id = $1 AND deleted_at IS NULL`

	var plan string
	err := c.db.QueryRowContext(ctx, query, tenantID).Scan(&plan)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", cache.ErrNotFound
	case err != nil:
		return "", err
	}

	return plan, nil
}
