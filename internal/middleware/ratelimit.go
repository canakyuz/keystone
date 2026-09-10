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
	"github.com/canakyuz/keystone/pkg/metrics"
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

// anonymousQuota applies to requests that carry no credentials at all. The tenant is
// unknown, so the key is the IP and the limit is deliberately tight.
var anonymousQuota = ratelimit.PerMinute(30)

// addressQuota is the coarse per-address brake applied to credential-bearing traffic
// ahead of authentication.
//
// It sits above the largest plan on purpose. The plan quota is the product policy and
// has to be the binding limit; this one exists so that a flood of requests carrying
// junk credentials cannot be served for free just because it will be rejected by the
// authenticator. Set below the top plan it would shadow it, and an enterprise tenant
// would silently receive less than it pays for.
//
// It is deliberately generous rather than exact: several tenants behind one address is
// normal, and a shared office or NAT gateway must not throttle them collectively.
var addressQuota = ratelimit.PerMinute(12_000)

// fallbackQuota applies when the plan cannot be resolved. The lowest plan is chosen:
// a resolution failure must not turn into a free upgrade.
var fallbackQuota = planQuotas["free"]

// Scope selects which limit a middleware instance enforces.
//
// Two instances are needed because the plan cannot be resolved before the request is
// authenticated, and unauthenticated traffic still has to be limited. Registering one
// global instance and hoping it sees a tenant is what the previous version did: the
// limiter ran before the authenticator, c.Locals("tenant_id") was always empty, and
// every authenticated tenant was silently held to the anonymous quota of 30 requests a
// minute regardless of the plan it paid for. The plan table existed and never applied.
type Scope int

const (
	// ScopeIP limits by client address. Registered globally, ahead of authentication, as
	// an abuse brake.
	ScopeIP Scope = iota

	// ScopeTenant limits by tenant, using the plan's quota. Registered inside the
	// authenticated groups, where the tenant is known.
	ScopeTenant
)

// RateLimitConfig configures the limiter middleware.
type RateLimitConfig struct {
	// Limiter is required.
	Limiter ratelimit.Limiter

	// Plans resolves tenant_id to a plan. It may be nil, in which case every
	// authenticated request is limited by fallbackQuota.
	Plans *TenantPlanCache

	// Scope selects which limit this instance enforces.
	Scope Scope

	// Metrics records each decision. It may be nil.
	Metrics *metrics.Registry

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

		key, quota, plan, ok := resolveKeyAndQuota(c, cfg)
		if !ok {
			return c.Next()
		}

		result, err := cfg.Limiter.Allow(c.Context(), key, quota)
		if err != nil && cfg.Logger != nil {
			cfg.Logger.WithFields(logger.Fields{
				"key":   key,
				"error": err.Error(),
			}).Warn("rate limit query failed")
		}

		writeRateLimitHeaders(c, result)

		if cfg.Metrics != nil {
			cfg.Metrics.RateLimitDecision(plan, result.Allowed)
		}

		if !result.Allowed {
			return tooManyRequests(c, result)
		}

		return c.Next()
	}
}

// resolveKeyAndQuota decides which key and quota limit a request, and names the plan it
// resolved. The final return says whether this instance applies at all.
//
// The plan name is returned for the metric label. It is bounded — there are five of
// them plus two fallbacks — so it answers "which tier is being throttled?" without the
// unbounded cardinality of a tenant id.
func resolveKeyAndQuota(c *fiber.Ctx, cfg RateLimitConfig) (string, ratelimit.Quota, string, bool) {
	if cfg.Scope == ScopeTenant {
		tenantID, _ := c.Locals("tenant_id").(string)

		// Registered after authentication, so a missing tenant means the request never
		// identified itself. The address limiter already covered it.
		if tenantID == "" {
			return "", ratelimit.Quota{}, "", false
		}

		quota, plan := quotaForTenant(c.Context(), cfg, tenantID)

		return "tenant:" + tenantID, quota, plan, true
	}

	// ScopeIP runs before authentication, so the tenant is not knowable here. What is
	// knowable is whether the caller presented credentials at all.
	//
	// The earlier version tried to read c.Locals("tenant_id") and skip authenticated
	// requests. That value is written by the authenticator, which has not run yet, so
	// the branch was never taken: every authenticated request was charged against the
	// anonymous budget of 30 a minute, and an enterprise tenant paying for 6000 was cut
	// off after 30. The load test is what surfaced it.
	if c.Get("Authorization") != "" {
		return "ip:" + c.IP(), addressQuota, "address", true
	}

	return "ip:" + c.IP(), anonymousQuota, "anonymous", true
}

// quotaForTenant returns the quota matching the tenant's plan, and the plan name.
//
// An unresolvable plan is reported as "unknown" rather than folded into the real plan
// names. A rise in that series means plan resolution is failing, and silently serving
// those requests the fallback quota would hide it.
func quotaForTenant(ctx context.Context, cfg RateLimitConfig, tenantID string) (ratelimit.Quota, string) {
	if cfg.Plans == nil {
		return fallbackQuota, "unknown"
	}

	plan, err := cfg.Plans.GetPlan(ctx, tenantID)
	if err != nil {
		return fallbackQuota, "unknown"
	}

	if quota, ok := planQuotas[plan]; ok {
		return quota, plan
	}

	return fallbackQuota, "unknown"
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
