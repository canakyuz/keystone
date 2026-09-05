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

// planQuotas, abonelik planini dakikalik istek hakkina cevirir.
//
// Degerler urun karari olarak burada toplanmistir; koda dagilmis sihirli
// sayilar yerine tek bir tabloda durur.
var planQuotas = map[string]ratelimit.Quota{
	"free":       ratelimit.PerMinute(60),
	"starter":    ratelimit.PerMinute(300),
	"pro":        ratelimit.PerMinute(1_200),
	"enterprise": ratelimit.PerMinute(6_000),
}

// anonymousQuota, kimlik dogrulamasi yapilmamis istekler icin uygulanir.
// Tenant bilinmedigi icin anahtar IP'dir ve limit bilincli olarak dardir.
var anonymousQuota = ratelimit.PerMinute(30)

// fallbackQuota, plan cozumlenemediginde uygulanir.
// En dusuk plan secilir: cozumleme hatasi ucretsiz bir yukseltmeye donusmemeli.
var fallbackQuota = planQuotas["free"]

// RateLimitConfig, limitleyici middleware'ini yapilandirir.
type RateLimitConfig struct {
	// Limiter zorunludur.
	Limiter ratelimit.Limiter

	// Plans, tenant_id -> plan cozumlemesi yapar. Nil olabilir; o durumda
	// tum kimlik dogrulanmis istekler fallbackQuota ile sinirlanir.
	Plans *TenantPlanCache

	// SkipPaths, limitlemeden muaf yollardir.
	// Saglik ucu muaf olmalidir: load balancer'in yoklamasi, limit dolduğu
	// icin basarisiz olursa saglikli instance havuzdan cikarilir.
	SkipPaths map[string]bool

	Logger *logger.Logger
}

// RateLimit, istek limitleyici middleware'ini olusturur.
//
// Anahtar secimi:
//   - Kimlik dogrulanmissa tenant_id. Boylece limit kiraci basina uygulanir
//     ve bir kiracinin trafigi digerini etkilemez.
//   - Degilse istemci IP'si.
//
// Fiber'in yerlesik limiter'i yerine bu kullanilir. Yerlesik olan varsayilan
// olarak surec ici bir store tutar: uc replikada, replika basina 100 istek
// ayari gercekte 300 istek demektir. Ayrica plan bazli kota veya kiraci
// bazli anahtar desteklemez.
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
			}).Warn("Rate limit sorgusu başarısız")
		}

		writeRateLimitHeaders(c, result)

		if !result.Allowed {
			return tooManyRequests(c, result)
		}

		return c.Next()
	}
}

// resolveKeyAndQuota, istegin hangi anahtar ve kota ile sinirlanacagini belirler.
func resolveKeyAndQuota(c *fiber.Ctx, cfg RateLimitConfig) (string, ratelimit.Quota) {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "ip:" + c.IP(), anonymousQuota
	}

	return "tenant:" + tenantID, quotaForTenant(c.Context(), cfg, tenantID)
}

// quotaForTenant, tenant'in planina karsilik gelen kotayi dondurur.
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

// writeRateLimitHeaders, istemcinin kendini ayarlayabilmesi icin durum bildirir.
// Bu basliklar olmadan istemciler ancak 429 gorerek ogrenir.
func writeRateLimitHeaders(c *fiber.Ctx, result ratelimit.Result) {
	c.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
}

// tooManyRequests, 429 yanitini Retry-After ile birlikte dondurur.
func tooManyRequests(c *fiber.Ctx, result ratelimit.Result) error {
	seconds := int(result.RetryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	c.Set("Retry-After", strconv.Itoa(seconds))

	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"type":   "https://keystone.dev/problems/rate-limit-exceeded",
		"title":  "İstek limiti aşıldı",
		"status": fiber.StatusTooManyRequests,
		"detail": "Çok fazla istek gönderildi. Retry-After başlığına bakın.",
	})
}

// --- plan onbellegi -------------------------------------------------------

const (
	tenantPlanKeyPrefix = "tenant:plan:"
	planTTL             = 5 * time.Minute
	planLocalTTL        = 30 * time.Second
	planNegativeTTL     = 15 * time.Second
)

// TenantPlanCache, tenant'in abonelik planini onbellekler.
//
// Her istekte plan icin veritabanina inmek, limitleyicinin kendisini bir
// yuk kaynagina cevirirdi. pkg/cache yeniden kullanilir: ayni yigilma
// korumasi ve negatif onbellekleme burada da gecerlidir.
type TenantPlanCache struct {
	db    *sql.DB
	cache *cache.TwoTier
}

// NewTenantPlanCache, plan onbellegini kurar. rdb nil olabilir.
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

// GetPlan, tenant'in planini dondurur.
func (c *TenantPlanCache) GetPlan(ctx context.Context, tenantID string) (string, error) {
	plan, err := c.cache.Get(ctx, tenantID)
	if errors.Is(err, cache.ErrNotFound) {
		return "", ErrTenantNotFound
	}

	return plan, err
}

// Invalidate, plan degistiginde cagrilmalidir.
func (c *TenantPlanCache) Invalidate(ctx context.Context, tenantID string) error {
	return c.cache.Invalidate(ctx, tenantID)
}

// loadPlanFromDB, plani asil kaynaktan getirir.
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
