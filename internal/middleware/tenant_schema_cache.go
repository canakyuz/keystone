package middleware

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/pkg/cache"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
)

// ErrTenantNotFound reports that the tenant does not exist.
// The caller should turn it into a 404.
var ErrTenantNotFound = errors.New("tenant not found")

const (
	// tenantSchemaKeyPrefix names the Redis keys.
	// The prefix makes bulk invalidation (tenant:schema:*) possible.
	tenantSchemaKeyPrefix = "tenant:schema:"

	// defaultSchemaTTL is the lifetime of the Redis tier.
	// A tenant's schema only changes during onboarding, so a long TTL is appropriate.
	defaultSchemaTTL = 10 * time.Minute

	// defaultLocalTTL is the lifetime of the in-process tier.
	// It is kept far below the Redis TTL: this value bounds how long a schema change
	// takes to reach every replica.
	defaultLocalTTL = 30 * time.Second

	// defaultNegativeTTL is how long a non-existent tenant stays cached.
	// It is short: a newly created tenant has to become visible quickly.
	defaultNegativeTTL = 15 * time.Second

	// defaultJitter is the randomness ratio added to the TTLs.
	defaultJitter = 0.2
)

// TenantSchemaCache caches the tenant_id -> schema_name resolution.
//
// This type now carries only the domain knowledge (the query and the key format); the
// caching mechanics are delegated to the two-tier implementation in pkg/cache.
//
// Problems closed since the previous implementation:
//
//   - Cache stampede. N requests arriving at once for a cold key turned into N
//     database queries. singleflight now collapses them into one.
//   - No negative caching. A flood of requests carrying random non-existent tenant ids
//     reached the database every time.
//   - Cache fill spawned a new goroutine per request; there was no panic recovery and
//     the goroutine count grew along with the request count.
//   - Redis was a single point of failure. There is now an in-process L1 tier, and the
//     service keeps serving when Redis is unreachable.
//   - The hit rate was not measured. Stats() now exposes it.
type TenantSchemaCache struct {
	db     *sql.DB
	cache  *cache.TwoTier
	logger *logger.Logger
}

// NewTenantSchemaCache builds the cache.
// rdb may be nil, in which case only the in-process tier and the database are used.
func NewTenantSchemaCache(rdb *redis.Client, db *sql.DB, log *logger.Logger, reg *metrics.Registry) *TenantSchemaCache {
	c := &TenantSchemaCache{db: db, logger: log}

	// Keep a typed nil from being wrapped as a non-nil interface.
	var redisClient cache.RedisClient
	if rdb != nil {
		redisClient = rdb
	}

	// The cache reports each outcome here rather than importing a metrics registry
	// itself; see cache.Config.OnEvent. This is what ADR-0006 was missing: Stats()
	// already counted these, but until they were exported the hit rate could be read in
	// a debugger and nowhere else.
	var onEvent func(string)
	if reg != nil {
		onEvent = func(event string) { reg.CacheEvent("tenant_schema", event) }
	}

	c.cache = cache.New(cache.Config{
		Redis:       redisClient,
		Loader:      c.loadSchemaFromDB,
		OnEvent:     onEvent,
		KeyPrefix:   tenantSchemaKeyPrefix,
		TTL:         defaultSchemaTTL,
		L1TTL:       defaultLocalTTL,
		NegativeTTL: defaultNegativeTTL,
		Jitter:      defaultJitter,
	})

	return c
}

// GetTenantSchema returns the tenant's schema name.
// It returns ErrTenantNotFound when the tenant does not exist.
//
// Complexity: O(1) on an L1 hit; on a miss, one indexed single-row query.
func (c *TenantSchemaCache) GetTenantSchema(ctx context.Context, tenantID string) (string, error) {
	schema, err := c.cache.Get(ctx, tenantID)
	if errors.Is(err, cache.ErrNotFound) {
		return "", ErrTenantNotFound
	}

	return schema, err
}

// InvalidateTenantSchema drops the tenant's cache entry.
//
// Warning: only this process's L1 tier is cleared immediately. Other replicas may see
// the stale value until their own L1 TTL (defaultLocalTTL) expires.
func (c *TenantSchemaCache) InvalidateTenantSchema(ctx context.Context, tenantID string) error {
	if err := c.cache.Invalidate(ctx, tenantID); err != nil {
		if c.logger != nil {
			c.logger.WithFields(logger.Fields{
				"tenant_id": tenantID,
				"error":     err.Error(),
			}).Error("cache invalidation failed")
		}
		return err
	}

	return nil
}

// Stats returns the cache counters.
// A claim like "99% hit rate" can only be verified by measuring it.
func (c *TenantSchemaCache) Stats() cache.Stats {
	return c.cache.Stats()
}

// loadSchemaFromDB fetches the schema name from the source of truth.
//
// Only active, non-deleted tenants resolve: a suspended tenant's requests stop at the
// schema resolution step.
func (c *TenantSchemaCache) loadSchemaFromDB(ctx context.Context, tenantID string) (string, error) {
	const query = `
		SELECT schema_name
		FROM tenants
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status = 'active'
	`

	var schemaName string
	err := c.db.QueryRowContext(ctx, query, tenantID).Scan(&schemaName)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Converted to the error the cache understands, so it can be cached negatively.
		return "", cache.ErrNotFound
	case err != nil:
		return "", err
	}

	return schemaName, nil
}
