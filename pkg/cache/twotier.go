// Package cache, okuma ağırlıklı lookup'lar için iki katmanlı bir önbellek sunar.
//
// Katmanlar: L1 süreç içi map, L2 Redis, en altta çağıranın verdiği loader.
// Sıra L1, L2, loader şeklindedir ve bulunan değer yukarı doğru doldurulur.
//
// Redis zorunlu değildir. nil verilirse önbellek L1 ve loader ile çalışır;
// bu, Redis'in erişilemediği durumlarda servisin ayakta kalmasını sağlar.
package cache

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// ErrNotFound, anahtarın kaynakta bulunmadığını bildirir.
// Loader bu hatayı döndürdüğünde sonuç negatif olarak önbelleğe alınır.
var ErrNotFound = errors.New("cache: kayıt bulunamadı")

// negativeSentinel, "bu anahtar yok" bilgisini önbellekte temsil eder.
// Geçerli bir değerle karışmaması için kasıtlı olarak kullanılamaz bir dizedir.
const negativeSentinel = "\x00__absent__"

// Loader, önbellekte bulunamayan anahtarı asıl kaynaktan getirir.
// Kayıt yoksa ErrNotFound döndürmelidir.
type Loader func(ctx context.Context, key string) (string, error)

// RedisClient, kullanılan Redis işlemlerini daraltır.
// go-redis'in *redis.Client'ı bu arayüzü karşılar.
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, ttl time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// Stats, önbellek davranışının ölçülebilir özetidir.
// Yorum satırındaki "%99 hit rate" gibi iddialar ancak böyle doğrulanabilir.
type Stats struct {
	L1Hits       uint64
	L2Hits       uint64
	Misses       uint64
	NegativeHits uint64
	LoaderErrors uint64
}

// Config, TwoTier'ın davranışını belirler.
type Config struct {
	// Redis nil olabilir. Nil ise yalnızca L1 ve loader kullanılır.
	Redis RedisClient

	// Loader zorunludur.
	Loader Loader

	// KeyPrefix, Redis anahtarlarının önüne eklenir.
	// Çok kiracılı kurulumlarda namespace çakışmasını engeller.
	KeyPrefix string

	// TTL, L2 (Redis) için taban yaşam süresidir.
	TTL time.Duration

	// L1TTL, süreç içi katmanın yaşam süresidir. TTL'den kısa olmalıdır:
	// süreçler arası tutarsızlık penceresini bu değer belirler.
	L1TTL time.Duration

	// NegativeTTL, bulunamayan anahtarların önbellekte tutulma süresidir.
	// Sıfır verilirse negatif önbellekleme kapalıdır.
	//
	// Neden gerekli: bu alan olmadan, var olmayan rastgele anahtarlarla
	// yapılan istek seli her seferinde asıl kaynağa iner. Ucuz bir
	// yük yükseltme vektörüdür.
	NegativeTTL time.Duration

	// Jitter, TTL'e eklenecek rastgelelik oranıdır (0.0 - 1.0).
	// Aynı anda oluşturulan anahtarların aynı anda düşmesini engeller.
	Jitter float64

	// MaxL1Entries, süreç içi katmanın üst sınırıdır. Sıfır ise 10.000.
	MaxL1Entries int
}

// TwoTier, iki katmanlı önbellektir. Eşzamanlı kullanıma uygundur.
type TwoTier struct {
	cfg   Config
	l1    *memo
	group singleflight.Group

	l1Hits       atomic.Uint64
	l2Hits       atomic.Uint64
	misses       atomic.Uint64
	negativeHits atomic.Uint64
	loaderErrors atomic.Uint64
}

// New, verilen yapılandırmayla önbellek oluşturur.
func New(cfg Config) *TwoTier {
	if cfg.MaxL1Entries <= 0 {
		cfg.MaxL1Entries = 10_000
	}
	if cfg.L1TTL <= 0 {
		cfg.L1TTL = 30 * time.Second
	}

	return &TwoTier{
		cfg: cfg,
		l1:  newMemo(cfg.MaxL1Entries),
	}
}

// Get, anahtarın değerini döndürür.
//
// Kayıt yoksa ErrNotFound döner ve bu sonuç NegativeTTL boyunca önbelleklenir.
//
// Eşzamanlılık: aynı anahtar için aynı anda gelen N istek tek bir loader
// çağrısına indirgenir. singleflight olmadan soğuk bir anahtar üzerindeki
// ani yük, N adet kaynak sorgusuna dönüşürdü.
//
// Karmaşıklık: L1 isabetinde O(1). Iskada O(1) artı loader maliyeti.
func (c *TwoTier) Get(ctx context.Context, key string) (string, error) {
	if value, ok := c.l1.get(key); ok {
		c.l1Hits.Add(1)
		c.countIfNegative(value)
		return c.interpret(value)
	}

	// singleflight, aynı anahtar için tek bir yükleme yapılmasını sağlar.
	// Dönen değer paylaşılır; ek kopya veya kilit yönetimi gerekmez.
	value, err, _ := c.group.Do(key, func() (any, error) {
		return c.loadThroughL2(ctx, key)
	})
	if err != nil {
		return "", err
	}

	return c.interpret(value.(string))
}

// loadThroughL2, L2'yi dener, olmazsa loader'a iner ve iki katmanı da doldurur.
func (c *TwoTier) loadThroughL2(ctx context.Context, key string) (string, error) {
	if value, ok := c.readL2(ctx, key); ok {
		c.l2Hits.Add(1)
		c.countIfNegative(value)
		c.l1.set(key, value, c.cfg.L1TTL)
		return value, nil
	}

	c.misses.Add(1)

	value, err := c.cfg.Loader(ctx, key)
	switch {
	case errors.Is(err, ErrNotFound):
		c.storeNegative(ctx, key)
		return negativeSentinel, nil
	case err != nil:
		c.loaderErrors.Add(1)
		return "", err
	}

	c.store(ctx, key, value, c.cfg.TTL)
	return value, nil
}

// interpret, negatif sentinel'i ErrNotFound'a çevirir.
func (c *TwoTier) interpret(value string) (string, error) {
	if value == negativeSentinel {
		return "", ErrNotFound
	}
	return value, nil
}

// countIfNegative, önbellekten servis edilen bir negatif sonucu sayar.
//
// Yalnızca L1 ve L2 isabetlerinde çağrılır. Loader'a inen ilk çözümleme bir
// "isabet" değildir; onu da saymak, negatif önbelleğin ne kadar iş yaptığını
// olduğundan büyük gösterirdi.
func (c *TwoTier) countIfNegative(value string) {
	if value == negativeSentinel {
		c.negativeHits.Add(1)
	}
}

// storeNegative, bulunamayan anahtarı kısa süreli önbelleğe alır.
func (c *TwoTier) storeNegative(ctx context.Context, key string) {
	if c.cfg.NegativeTTL <= 0 {
		return
	}
	c.store(ctx, key, negativeSentinel, c.cfg.NegativeTTL)
}

// store, değeri her iki katmana da yazar.
//
// L2 yazımı senkron yapılır. Önceki uygulama bunu her istekte yeni bir
// goroutine ile yapıyordu: panic recovery yoktu ve goroutine sayısı
// isteklerle birlikte sınırsız büyüyordu. singleflight sayesinde bu yol
// anahtar başına zaten tek sefer çalışır, dolayısıyla senkron yazmanın
// maliyeti sınırlıdır ve davranış öngörülebilirdir.
func (c *TwoTier) store(ctx context.Context, key, value string, ttl time.Duration) {
	c.l1.set(key, value, min(ttl, c.cfg.L1TTL))

	if c.cfg.Redis == nil || ttl <= 0 {
		return
	}

	// Hata yutulur: önbellek yazımı başarısız olsa da istek servis edilmelidir.
	_ = c.cfg.Redis.Set(ctx, c.cfg.KeyPrefix+key, value, c.withJitter(ttl)).Err()
}

// readL2, Redis'ten okur. Redis yoksa veya hata verirse ıska sayılır.
func (c *TwoTier) readL2(ctx context.Context, key string) (string, bool) {
	if c.cfg.Redis == nil {
		return "", false
	}

	value, err := c.cfg.Redis.Get(ctx, c.cfg.KeyPrefix+key).Result()
	if err != nil {
		return "", false
	}

	return value, true
}

// withJitter, TTL'e [0, ttl*Jitter) aralığında rastgelelik ekler.
//
// Neden: aynı anda oluşturulan anahtarlar aynı anda düşerse, sona erme
// anında toplu bir ıska dalgası oluşur ve kaynak ani yük görür.
func (c *TwoTier) withJitter(ttl time.Duration) time.Duration {
	if c.cfg.Jitter <= 0 {
		return ttl
	}

	spread := float64(ttl) * min(c.cfg.Jitter, 1.0)
	return ttl + time.Duration(rand.Float64()*spread)
}

// Invalidate, anahtarı her iki katmandan da düşürür.
//
// Not: L1 yalnızca bu süreçte temizlenir. Diğer replikalar kendi L1TTL'leri
// dolana kadar eski değeri görebilir. Bu, iki katmanlı tasarımın bilinçli
// takasıdır; tutarsızlık penceresi L1TTL ile sınırlıdır.
func (c *TwoTier) Invalidate(ctx context.Context, key string) error {
	c.l1.delete(key)

	if c.cfg.Redis == nil {
		return nil
	}

	return c.cfg.Redis.Del(ctx, c.cfg.KeyPrefix+key).Err()
}

// Stats, o ana kadarki sayaçları döndürür.
func (c *TwoTier) Stats() Stats {
	return Stats{
		L1Hits:       c.l1Hits.Load(),
		L2Hits:       c.l2Hits.Load(),
		Misses:       c.misses.Load(),
		NegativeHits: c.negativeHits.Load(),
		LoaderErrors: c.loaderErrors.Load(),
	}
}

// --- L1 ------------------------------------------------------------------

type entry struct {
	value     string
	expiresAt time.Time
}

// memo, TTL'li ve üst sınırlı süreç içi haritadır.
type memo struct {
	mu      sync.RWMutex
	items   map[string]entry
	maxSize int
}

func newMemo(maxSize int) *memo {
	return &memo{items: make(map[string]entry, maxSize/4+1), maxSize: maxSize}
}

// get, süresi dolmamış değeri döndürür.
// Karmaşıklık: O(1).
func (m *memo) get(key string) (string, bool) {
	m.mu.RLock()
	it, ok := m.items[key]
	m.mu.RUnlock()

	if !ok || time.Now().After(it.expiresAt) {
		return "", false
	}

	return it.value, true
}

// set, değeri yazar ve gerekirse yer açar.
// Karmaşıklık: normalde O(1); sınır dolduğunda O(n) tahliye.
func (m *memo) set(key, value string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.items) >= m.maxSize {
		m.evictLocked()
	}

	m.items[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (m *memo) delete(key string) {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
}

// evictLocked, önce süresi dolanları atar; hiçbiri yoksa en erken sona erecek
// olanı düşürür. Çağıranın kilidi tutuyor olması gerekir.
func (m *memo) evictLocked() {
	now := time.Now()
	for key, it := range m.items {
		if now.After(it.expiresAt) {
			delete(m.items, key)
		}
	}

	if len(m.items) < m.maxSize {
		return
	}

	var oldestKey string
	var oldest time.Time
	for key, it := range m.items {
		if oldest.IsZero() || it.expiresAt.Before(oldest) {
			oldestKey, oldest = key, it.expiresAt
		}
	}
	delete(m.items, oldestKey)
}
