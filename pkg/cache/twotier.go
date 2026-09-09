// Package cache provides a two-tier cache for read-heavy lookups.
//
// The tiers are an in-process map (L1), Redis (L2), and the caller's loader
// underneath. Lookup goes L1, L2, loader, and a value found lower down is filled
// back upwards.
//
// Redis is optional. Passed nil, the cache runs on L1 and the loader alone, which
// is what keeps the service up when Redis is unreachable.
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

// ErrNotFound reports that the key does not exist in the source. When the loader
// returns this error, the result is cached negatively.
var ErrNotFound = errors.New("cache: record not found")

// negativeSentinel represents "this key does not exist" inside the cache. It is
// deliberately an unusable string so it cannot be confused with a real value.
const negativeSentinel = "\x00__absent__"

// Loader fetches a key that missed the cache from the underlying source. It must
// return ErrNotFound when the record does not exist.
type Loader func(ctx context.Context, key string) (string, error)

// RedisClient narrows the Redis operations actually used. go-redis's *redis.Client
// satisfies this interface.
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, ttl time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// Stats is a measurable summary of cache behaviour. A claim like "99% hit rate" in
// a comment can only be checked this way.
type Stats struct {
	L1Hits       uint64
	L2Hits       uint64
	Misses       uint64
	NegativeHits uint64
	LoaderErrors uint64
}

// Config determines how TwoTier behaves.
type Config struct {
	// Redis may be nil. When it is, only L1 and the loader are used.
	Redis RedisClient

	// Loader zorunludur.
	Loader Loader

	// KeyPrefix is prepended to Redis keys. It prevents namespace collisions in a
	// multi-tenant deployment.
	KeyPrefix string

	// TTL is the base lifetime for the L2 (Redis) tier.
	TTL time.Duration

	// L1TTL is the lifetime of the in-process tier. It must be shorter than TTL:
	// this value is what bounds the cross-process inconsistency window.
	L1TTL time.Duration

	// NegativeTTL is how long a missing key stays cached. Zero disables negative
	// caching.
	//
	// Neden gerekli: bu alan olmadan, var olmayan rastgele anahtarlarla
	// a flood of requests carrying random ids reaches the source every time. That is
	// a cheap load-amplification vector.
	NegativeTTL time.Duration

	// Jitter is the randomness ratio added to the TTL (0.0 - 1.0). It stops keys
	// created at the same moment from expiring at the same moment.
	Jitter float64

	// MaxL1Entries caps the in-process tier. Zero means 10,000.
	MaxL1Entries int
}

// TwoTier is the two-tier cache. It is safe for concurrent use.
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

// New builds a cache from the given configuration.
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

// Get returns the value for a key.
//
// A missing record yields ErrNotFound, and that outcome is cached for NegativeTTL.
//
// Concurrency: N requests arriving at once for the same key collapse into a single
// loader call. Without singleflight, a burst on a cold key would turn into N source
// queries.
//
// Complexity: O(1) on an L1 hit. On a miss, O(1) plus the loader's cost.
func (c *TwoTier) Get(ctx context.Context, key string) (string, error) {
	if value, ok := c.l1.get(key); ok {
		c.l1Hits.Add(1)
		c.countIfNegative(value)
		return c.interpret(value)
	}

	// singleflight guarantees a single load per key. The returned value is shared, so
	// no extra copy or lock management is needed.
	value, err, _ := c.group.Do(key, func() (any, error) {
		return c.loadThroughL2(ctx, key)
	})
	if err != nil {
		return "", err
	}

	return c.interpret(value.(string))
}

// loadThroughL2 tries L2, falls through to the loader, and fills both tiers.
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

// interpret converts the negative sentinel into ErrNotFound.
func (c *TwoTier) interpret(value string) (string, error) {
	if value == negativeSentinel {
		return "", ErrNotFound
	}
	return value, nil
}

// countIfNegative counts a negative result served from cache.
//
// It is only called on L1 and L2 hits. The first resolution, the one that reaches
// the loader, is not a "hit"; counting it would overstate how much work the
// negative cache is actually doing.
func (c *TwoTier) countIfNegative(value string) {
	if value == negativeSentinel {
		c.negativeHits.Add(1)
	}
}

// storeNegative caches a missing key for a short while.
func (c *TwoTier) storeNegative(ctx context.Context, key string) {
	if c.cfg.NegativeTTL <= 0 {
		return
	}
	c.store(ctx, key, negativeSentinel, c.cfg.NegativeTTL)
}

// store writes the value into both tiers.
//
// The L2 write is synchronous. The previous implementation did it in a fresh
// goroutine per request: there was no panic recovery, and the goroutine count grew
// without bound alongside requests. Thanks to singleflight this path already runs
// once per key, so a synchronous write costs little and behaves predictably.
func (c *TwoTier) store(ctx context.Context, key, value string, ttl time.Duration) {
	c.l1.set(key, value, min(ttl, c.cfg.L1TTL))

	if c.cfg.Redis == nil || ttl <= 0 {
		return
	}

	// The error is swallowed: a failed cache write must not fail the request.
	_ = c.cfg.Redis.Set(ctx, c.cfg.KeyPrefix+key, value, c.withJitter(ttl)).Err()
}

// readL2 reads from Redis. A missing or erroring Redis counts as a miss.
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

// withJitter adds randomness in [0, ttl*Jitter) to the TTL.
//
// Why: if keys created at the same moment expire at the same moment, expiry
// produces a synchronized wave of misses and the source takes a spike.
func (c *TwoTier) withJitter(ttl time.Duration) time.Duration {
	if c.cfg.Jitter <= 0 {
		return ttl
	}

	spread := float64(ttl) * min(c.cfg.Jitter, 1.0)
	return ttl + time.Duration(rand.Float64()*spread)
}

// Invalidate drops the key from both tiers.
//
// Note: L1 is only cleared in this process. Other replicas may see the stale value
// until their own L1TTL expires. That is the deliberate trade-off of the two-tier
// design; the inconsistency window is bounded by L1TTL.
func (c *TwoTier) Invalidate(ctx context.Context, key string) error {
	c.l1.delete(key)

	if c.cfg.Redis == nil {
		return nil
	}

	return c.cfg.Redis.Del(ctx, c.cfg.KeyPrefix+key).Err()
}

// Stats returns the counters accumulated so far.
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

// memo is an in-process map with TTLs and a size ceiling.
type memo struct {
	mu      sync.RWMutex
	items   map[string]entry
	maxSize int
}

func newMemo(maxSize int) *memo {
	return &memo{items: make(map[string]entry, maxSize/4+1), maxSize: maxSize}
}

// get returns the value if it has not expired.
// Complexity: O(1).
func (m *memo) get(key string) (string, bool) {
	m.mu.RLock()
	it, ok := m.items[key]
	m.mu.RUnlock()

	if !ok || time.Now().After(it.expiresAt) {
		return "", false
	}

	return it.value, true
}

// set writes the value, making room first if needed.
// Complexity: O(1) normally; O(n) eviction when the ceiling is reached.
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

// evictLocked discards expired entries first, and if there are none, drops the one
// expiring soonest. The caller must be holding the lock.
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
