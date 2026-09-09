package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingLoader builds a loader that counts how many times it was called.
func countingLoader(calls *atomic.Int64, value string, err error) Loader {
	return func(ctx context.Context, key string) (string, error) {
		calls.Add(1)
		return value, err
	}
}

// TestGet_CollapsesConcurrentMisses verifies that the cache stampede is prevented.
//
// Without singleflight, N requests arriving at once for a cold key turn into N
// source queries. That means the cache fails to protect at exactly the moment it is
// most needed, which is under a burst.
func TestGet_CollapsesConcurrentMisses(t *testing.T) {
	var calls atomic.Int64

	// The loader is deliberately slow, so the concurrent requests overlap.
	slowLoader := func(ctx context.Context, key string) (string, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "tenant_acme", nil
	}

	c := New(Config{Loader: slowLoader, TTL: time.Minute, L1TTL: time.Minute})

	const goroutines = 100
	var wg sync.WaitGroup
	results := make([]string, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			value, err := c.Get(context.Background(), "tenant-1")
			require.NoError(t, err)
			results[idx] = value
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int64(1), calls.Load(), "eszamanli iskalar tek yuklemeye indirgenmedi")

	for _, got := range results {
		assert.Equal(t, "tenant_acme", got)
	}
}

// TestGet_NegativeCaching verifies that non-existent keys are cached too.
//
// Without it, a flood of requests carrying random non-existent keys reaches the
// source every time. That is a cheap load-amplification vector.
func TestGet_NegativeCaching(t *testing.T) {
	var calls atomic.Int64
	c := New(Config{
		Loader:      countingLoader(&calls, "", ErrNotFound),
		TTL:         time.Minute,
		L1TTL:       time.Minute,
		NegativeTTL: time.Minute,
	})

	for i := 0; i < 10; i++ {
		_, err := c.Get(context.Background(), "yok-boyle-tenant")
		assert.ErrorIs(t, err, ErrNotFound)
	}

	assert.Equal(t, int64(1), calls.Load(), "bulunamayan anahtar her seferinde kaynaga indi")
	assert.Equal(t, uint64(9), c.Stats().NegativeHits)
}

// TestGet_NegativeCachingDisabled verifies that negative caching is off when
// NegativeTTL is zero, so the default cannot change silently.
func TestGet_NegativeCachingDisabled(t *testing.T) {
	var calls atomic.Int64
	c := New(Config{
		Loader: countingLoader(&calls, "", ErrNotFound),
		TTL:    time.Minute,
		L1TTL:  time.Minute,
	})

	for i := 0; i < 3; i++ {
		_, err := c.Get(context.Background(), "yok")
		assert.ErrorIs(t, err, ErrNotFound)
	}

	assert.Equal(t, int64(3), calls.Load(), "negatif onbellek kapaliyken bile atlandi")
}

// TestGet_ServesFromL1 verifies that a second read does not reach the loader.
func TestGet_ServesFromL1(t *testing.T) {
	var calls atomic.Int64
	c := New(Config{
		Loader: countingLoader(&calls, "tenant_acme", nil),
		TTL:    time.Minute,
		L1TTL:  time.Minute,
	})

	ctx := context.Background()
	first, err := c.Get(ctx, "tenant-1")
	require.NoError(t, err)
	second, err := c.Get(ctx, "tenant-1")
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, uint64(1), c.Stats().L1Hits)
	assert.Equal(t, uint64(1), c.Stats().Misses)
}

// TestGet_LoaderErrorIsNotCached verifies that transient errors are not cached. A
// connection error must not turn into a permanent "not found".
func TestGet_LoaderErrorIsNotCached(t *testing.T) {
	var calls atomic.Int64
	boom := errors.New("gecici baglanti hatasi")
	c := New(Config{
		Loader:      countingLoader(&calls, "", boom),
		TTL:         time.Minute,
		L1TTL:       time.Minute,
		NegativeTTL: time.Minute,
	})

	for i := 0; i < 3; i++ {
		_, err := c.Get(context.Background(), "tenant-1")
		assert.ErrorIs(t, err, boom)
	}

	assert.Equal(t, int64(3), calls.Load(), "gecici hata onbelleklendi")
	assert.Equal(t, uint64(3), c.Stats().LoaderErrors)
}

// TestGet_L1Expiry verifies that the source is consulted again once L1 expires.
func TestGet_L1Expiry(t *testing.T) {
	var calls atomic.Int64
	c := New(Config{
		Loader: countingLoader(&calls, "tenant_acme", nil),
		TTL:    50 * time.Millisecond,
		L1TTL:  50 * time.Millisecond,
	})

	ctx := context.Background()
	_, err := c.Get(ctx, "tenant-1")
	require.NoError(t, err)

	time.Sleep(80 * time.Millisecond)

	_, err = c.Get(ctx, "tenant-1")
	require.NoError(t, err)

	assert.Equal(t, int64(2), calls.Load(), "TTL dolmasina ragmen eski deger servis edildi")
}

// TestInvalidate verifies that an invalidated key is loaded again.
func TestInvalidate(t *testing.T) {
	var calls atomic.Int64
	c := New(Config{
		Loader: countingLoader(&calls, "tenant_acme", nil),
		TTL:    time.Minute,
		L1TTL:  time.Minute,
	})

	ctx := context.Background()
	_, err := c.Get(ctx, "tenant-1")
	require.NoError(t, err)

	require.NoError(t, c.Invalidate(ctx, "tenant-1"))

	_, err = c.Get(ctx, "tenant-1")
	require.NoError(t, err)

	assert.Equal(t, int64(2), calls.Load(), "invalidate sonrasi eski deger servis edildi")
}

// TestWithJitter verifies that jitter keeps the TTL inside the expected range.
//
// Without jitter, keys created at the same moment expire at the same moment and
// produce a synchronized wave of misses.
func TestWithJitter(t *testing.T) {
	base := time.Minute
	c := New(Config{TTL: base, Jitter: 0.2})

	seen := make(map[time.Duration]bool)
	for i := 0; i < 200; i++ {
		got := c.withJitter(base)
		assert.GreaterOrEqual(t, got, base, "jitter TTL'i kisaltti")
		assert.Less(t, got, base+time.Duration(float64(base)*0.2)+time.Millisecond)
		seen[got] = true
	}

	assert.Greater(t, len(seen), 100, "jitter yeterince dagilmiyor")
}

// TestWithJitter_Disabled verifies that the TTL is constant when jitter is off.
func TestWithJitter_Disabled(t *testing.T) {
	base := time.Minute
	c := New(Config{TTL: base})

	assert.Equal(t, base, c.withJitter(base))
}

// TestL1_RespectsMaxEntries verifies that the in-process tier does not grow without bound.
func TestL1_RespectsMaxEntries(t *testing.T) {
	const maxEntries = 50
	var calls atomic.Int64
	c := New(Config{
		Loader:       countingLoader(&calls, "value", nil),
		TTL:          time.Minute,
		L1TTL:        time.Minute,
		MaxL1Entries: maxEntries,
	})

	ctx := context.Background()
	for i := 0; i < maxEntries*4; i++ {
		_, err := c.Get(ctx, string(rune('a'+i%26))+string(rune('a'+i/26)))
		require.NoError(t, err)
	}

	c.l1.mu.RLock()
	size := len(c.l1.items)
	c.l1.mu.RUnlock()

	assert.LessOrEqual(t, size, maxEntries, "L1 ust sinirini asti")
}
