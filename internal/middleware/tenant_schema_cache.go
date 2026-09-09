package middleware

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/pkg/cache"
	"github.com/canakyuz/keystone/pkg/logger"
)

// ErrTenantNotFound, tenant'in bulunamadigini bildirir.
// Cagiran bunu 404'e cevirmelidir.
var ErrTenantNotFound = errors.New("tenant not found")

const (
	// tenantSchemaKeyPrefix, Redis anahtarlarini isimlendirir.
	// Prefix, toplu gecersiz kilmayi (tenant:schema:*) mumkun kilar.
	tenantSchemaKeyPrefix = "tenant:schema:"

	// defaultSchemaTTL, Redis katmaninin yasam suresidir.
	// Tenant schema'si yalnizca onboarding sirasinda degisir, uzun TTL uygundur.
	defaultSchemaTTL = 10 * time.Minute

	// defaultLocalTTL, surec ici katmanin yasam suresidir.
	// Redis TTL'inden cok daha kisa tutulur: bir schema degisikliginin tum
	// replikalara yayilma gecikmesini bu deger belirler.
	defaultLocalTTL = 30 * time.Second

	// defaultNegativeTTL, var olmayan tenant'lar icin tutulan suredir.
	// Kisadir: yeni olusturulan bir tenant'in kisa surede gorunmesi gerekir.
	defaultNegativeTTL = 15 * time.Second

	// defaultJitter, TTL'e eklenen rastgelelik oranidir.
	defaultJitter = 0.2
)

// TenantSchemaCache, tenant_id -> schema_name cozumlemesini onbellekler.
//
// Bu tip artik yalnizca alan bilgisini (sorgu ve anahtar bicimi) tasir;
// onbellek mekanigi pkg/cache icindeki iki katmanli uygulamaya devredilmistir.
//
// Onceki uygulamada kapatilan sorunlar:
//
//   - Onbellek yigilmasi. Soguk bir anahtara ayni anda gelen N istek N adet
//     veritabani sorgusuna donusuyordu. Artik singleflight ile tek sorguya
//     indirgeniyor.
//   - Negatif onbellekleme yoktu. Var olmayan rastgele tenant kimlikleriyle
//     yapilan istek seli her seferinde veritabanina iniyordu.
//   - Onbellek doldurma her istekte yeni bir goroutine aciyordu; panic
//     recovery yoktu ve goroutine sayisi istek sayisiyla birlikte buyuyordu.
//   - Redis tek hata noktasiydi. Artik surec ici bir L1 katmani var ve Redis
//     erisilemezse servis calismaya devam eder.
//   - Isabet orani olculmuyordu. Artik Stats() ile olculebiliyor.
type TenantSchemaCache struct {
	db     *sql.DB
	cache  *cache.TwoTier
	logger *logger.Logger
}

// NewTenantSchemaCache, onbellegi kurar.
// rdb nil olabilir; o durumda yalnizca surec ici katman ve veritabani kullanilir.
func NewTenantSchemaCache(rdb *redis.Client, db *sql.DB, log *logger.Logger) *TenantSchemaCache {
	c := &TenantSchemaCache{db: db, logger: log}

	// Tipli nil'in arayuze non-nil olarak sarilmasini engelle.
	var redisClient cache.RedisClient
	if rdb != nil {
		redisClient = rdb
	}

	c.cache = cache.New(cache.Config{
		Redis:       redisClient,
		Loader:      c.loadSchemaFromDB,
		KeyPrefix:   tenantSchemaKeyPrefix,
		TTL:         defaultSchemaTTL,
		L1TTL:       defaultLocalTTL,
		NegativeTTL: defaultNegativeTTL,
		Jitter:      defaultJitter,
	})

	return c
}

// GetTenantSchema, tenant'in schema adini dondurur.
// Tenant yoksa ErrTenantNotFound doner.
//
// Karmasiklik: L1 isabetinde O(1); iskada bir indeksli tekil satir sorgusu.
func (c *TenantSchemaCache) GetTenantSchema(ctx context.Context, tenantID string) (string, error) {
	schema, err := c.cache.Get(ctx, tenantID)
	if errors.Is(err, cache.ErrNotFound) {
		return "", ErrTenantNotFound
	}

	return schema, err
}

// InvalidateTenantSchema, tenant'in onbellek kaydini duserur.
//
// Uyari: yalnizca bu surecin L1 katmani anlik temizlenir. Diger replikalar
// kendi L1 TTL'leri (defaultLocalTTL) dolana kadar eski degeri gorebilir.
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

// Stats, onbellek sayaclarini dondurur.
// "%99 isabet" gibi iddialar ancak olculerek dogrulanabilir.
func (c *TenantSchemaCache) Stats() cache.Stats {
	return c.cache.Stats()
}

// loadSchemaFromDB, asil kaynaktan schema adini getirir.
//
// Yalnizca aktif ve silinmemis tenant'lar cozumlenir: askiya alinmis bir
// tenant'in istekleri schema cozumleme asamasinda durur.
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
		// Negatif onbelleklenebilmesi icin cache'in anladigi hataya cevrilir.
		return "", cache.ErrNotFound
	case err != nil:
		return "", err
	}

	return schemaName, nil
}
