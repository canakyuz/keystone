package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/pkg/logger"
)

// 🎓 BACKEND KONSEPT: Cache-Aside Pattern (Lazy Loading)
//
// Cache-Aside Nedir?
// 1. Önce cache'e bak (fast path)
// 2. Cache'de yoksa DB'den al (slow path)
// 3. Sonucu cache'e kaydet (populate)
//
// Neden Cache-Aside?
// - Simple implementation
// - Cache failure'da fallback to DB (dayanıklı)
// - Control bizde (invalidation, TTL strategy)
//
// Alternatifler:
// - Read-Through: Cache otomatik DB'den fetch eder
// - Write-Through: Write sırasında cache güncellenir
// - Write-Behind: Async cache update

// TenantSchemaCache, tenant schema bilgilerini cache'leyen yapı.
//
// 🎓 GO KONSEPT: Struct Composition
// Struct içinde Redis client ve DB handler'ı birlikte tutar
type TenantSchemaCache struct {
	redis  *redis.Client
	db     *sql.DB
	logger *logger.Logger
	ttl    time.Duration // Cache Time-To-Live
}

// NewTenantSchemaCache, yeni bir cache instance oluşturur.
//
// 🎓 GO KONSEPT: Constructor Pattern
// Go'da constructor yoktur, New* prefix'li fonksiyon convention'dır
func NewTenantSchemaCache(rdb *redis.Client, db *sql.DB, log *logger.Logger) *TenantSchemaCache {
	return &TenantSchemaCache{
		redis:  rdb,
		db:     db,
		logger: log,
		ttl:    10 * time.Minute, // 🎓 Default TTL: 10 dakika
	}
}

// GetTenantSchema, tenant_id ile schema_name'i cache-aware şekilde getirir.
//
// 🎓 BACKEND KONSEPT: Cache-Aside Implementation
//
// Flow:
// 1. Redis cache'e bak (GET tenant:schema:{tenant_id})
// 2. Hit -> return cached value (fast)
// 3. Miss -> Query DB (slow)
// 4. Store in cache (SET with TTL)
// 5. Return value
func (c *TenantSchemaCache) GetTenantSchema(ctx context.Context, tenantID string) (string, error) {
	// 🎓 CACHE KEY STRATEGY
	// Pattern: tenant:schema:{id}
	// Neden bu format?
	// - Namespace isolation (tenant: prefix)
	// - Descriptive (ne için kullanıldığı açık)
	// - Pattern-based invalidation kolaylığı (tenant:* ile toplu silme)
	cacheKey := fmt.Sprintf("tenant:schema:%s", tenantID)

	// 🎓 STEP 1: Try cache first (FAST PATH)
	// Redis GET operation: O(1) complexity, ~1ms latency
	cachedSchema, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit! 🎉
		if c.logger != nil {
			c.logger.WithFields(logger.Fields{
				"tenant_id": tenantID,
				"source":    "cache",
			}).Debug("Tenant schema cache'den alındı")
		}
		return cachedSchema, nil
	}

	// 🎓 GO KONSEPT: Error Handling
	// redis.Nil: Key does not exist (expected, cache miss)
	// Other errors: Connection problem (unexpected, log and continue)
	if err != redis.Nil {
		// Redis connection error (non-fatal, fallback to DB)
		if c.logger != nil {
			c.logger.WithFields(logger.Fields{
				"tenant_id": tenantID,
				"error":     err.Error(),
			}).Warn("Redis cache hatası, DB'ye fallback yapılıyor")
		}
	}

	// 🎓 STEP 2: Cache miss, query database (SLOW PATH)
	// PostgreSQL query: ~5-20ms latency (depends on load)
	schema, err := c.queryDatabaseForSchema(ctx, tenantID)
	if err != nil {
		return "", err
	}

	// 🎓 STEP 3: Populate cache for next request
	// Background operation (fire and forget)
	// Neden background? Main flow'u bloklamak istemiyoruz
	go func() {
		// 🎓 GO KONSEPT: Goroutine (Concurrency)
		// Async operation, ana flow'u bloklamaz
		// Dikkat: Panic recovery gerekebilir (production'da)

		// Background context (parent context iptal olabilir)
		bgCtx := context.Background()

		// 🎓 REDIS OPERATION: SET with Expiration
		// TTL (Time To Live): Automatic expiration
		err := c.redis.Set(bgCtx, cacheKey, schema, c.ttl).Err()
		if err != nil && c.logger != nil {
			c.logger.WithFields(logger.Fields{
				"tenant_id": tenantID,
				"error":     err.Error(),
			}).Warn("Schema cache'e kaydedilemedi")
		}
	}()

	if c.logger != nil {
		c.logger.WithFields(logger.Fields{
			"tenant_id": tenantID,
			"source":    "database",
		}).Debug("Tenant schema DB'den alındı")
	}

	return schema, nil
}

// queryDatabaseForSchema, DB'den tenant schema'sını getirir.
//
// 🎓 SQL BEST PRACTICES:
// - Prepared statement (SQL injection prevention)
// - Index usage (tenant_id PRIMARY KEY)
// - Null check (deleted_at IS NULL)
// - Status filter (sadece aktif tenant'lar)
func (c *TenantSchemaCache) queryDatabaseForSchema(ctx context.Context, tenantID string) (string, error) {
	var schemaName string

	// 🎓 QUERY OPTIMIZATION
	// Index: tenants(id) PRIMARY KEY -> O(log n) lookup
	// Filter: deleted_at IS NULL AND status = 'active'
	query := `
		SELECT schema_name
		FROM tenants
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status = 'active'
	`

	// 🎓 GO KONSEPT: QueryRowContext
	// Context-aware (timeout, cancellation support)
	// Single row result beklenir
	err := c.db.QueryRowContext(ctx, query, tenantID).Scan(&schemaName)
	if err != nil {
		return "", err
	}

	// 🎓 VALIDATION
	// Empty string check (data integrity)
	if schemaName == "" {
		return "", sql.ErrNoRows
	}

	return schemaName, nil
}

// InvalidateTenantSchema, tenant schema cache'ini temizler.
//
// 🎓 CACHE INVALIDATION
// Ne zaman kullanılır?
// - Tenant schema değiştiğinde (provisioning)
// - Tenant status değiştiğinde (active -> suspended)
// - Tenant silindiğinde (soft delete)
//
// 🎓 BACKEND KONSEPT: Cache Invalidation Strategies
// 1. Time-based (TTL) - Automatic, simple
// 2. Event-based (invalidate on update) - Accurate, complex
// 3. Hybrid (TTL + manual invalidation) - Bu projede bunu kullanıyoruz
func (c *TenantSchemaCache) InvalidateTenantSchema(ctx context.Context, tenantID string) error {
	cacheKey := fmt.Sprintf("tenant:schema:%s", tenantID)

	// 🎓 REDIS OPERATION: DELETE
	// Atomic operation (single round-trip)
	err := c.redis.Del(ctx, cacheKey).Err()
	if err != nil {
		if c.logger != nil {
			c.logger.WithFields(logger.Fields{
				"tenant_id": tenantID,
				"error":     err.Error(),
			}).Error("Cache invalidation başarısız")
		}
		return err
	}

	if c.logger != nil {
		c.logger.WithFields(logger.Fields{
			"tenant_id": tenantID,
		}).Debug("Tenant schema cache'i temizlendi")
	}

	return nil
}

// 🎓 PERFORMANCE METRICS (Estimated)
//
// Cache Hit Scenario:
// - Redis GET: ~1ms
// - Total: ~1ms
// - DB Load: 0 queries
//
// Cache Miss Scenario:
// - Redis GET (miss): ~1ms
// - DB Query: ~5-20ms
// - Redis SET (async): ~1ms (non-blocking)
// - Total: ~6-21ms (first request)
//
// Cache Hit Rate Target: >95%
// With 10 min TTL, expected hit rate: 98-99%
//
// 🎓 TTL STRATEGY
//
// Current: 10 minutes (good for most cases)
//
// Considerations:
// - Tenant schema nadiren değişir (onboarding time)
// - Çok uzun TTL -> Schema değişikliği propagation gecikmesi
// - Çok kısa TTL -> Cache hit rate düşer, DB yükü artar
//
// Production Tuning:
// - Active tenants (high request rate): 10-15 min
// - Inactive tenants (low request rate): 5 min or no cache
// - VIP tenants (strict SLA): 15-30 min with priority invalidation
