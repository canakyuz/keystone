package ratelimit

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// limiters exercises the same behavioural contract against both implementations. The
// in-process and Redis-backed limiters must behave identically; otherwise behaviour
// that passes locally runs differently in production.
func limiters(t *testing.T) map[string]Limiter {
	t.Helper()

	out := map[string]Limiter{"memory": NewMemory()}

	if client := dialRedis(t); client != nil {
		out["redis"] = NewRedis(client, uniquePrefix(t))
	}

	return out
}

func dialRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		t.Logf("redis unreachable (%v), exercising the in-process implementation only", err)
		return nil
	}

	t.Cleanup(func() { client.Close() })
	return client
}

var prefixCounter atomic.Uint64

// uniquePrefix keeps tests from seeing each other's buckets.
func uniquePrefix(t *testing.T) string {
	t.Helper()
	return "test:rl:" + t.Name() + ":" + time.Now().Format("150405.000") + ":" +
		string(rune('a'+prefixCounter.Add(1)%26)) + ":"
}

// TestAllow_BurstThenDeny verifies that capacity-many requests pass and the next one
// is rejected.
func TestAllow_BurstThenDeny(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			quota := Quota{Burst: 5, Rate: 1}
			key := "kullanici-1"

			for i := 0; i < quota.Burst; i++ {
				res, err := limiter.Allow(ctx, key, quota)
				require.NoError(t, err)
				assert.True(t, res.Allowed, "kapasite icindeki %d. istek reddedildi", i+1)
				assert.Equal(t, quota.Burst, res.Limit)
			}

			res, err := limiter.Allow(ctx, key, quota)
			require.NoError(t, err)
			assert.False(t, res.Allowed, "kapasite asildigi halde istek gecti")
			assert.Zero(t, res.Remaining)
			assert.Greater(t, res.RetryAfter, time.Duration(0), "Retry-After hesaplanmadi")
		})
	}
}

// TestAllow_KeysAreIsolated verifies one key's limit does not affect another's. In a
// multi-tenant system this is what keeps one tenant from starving another.
func TestAllow_KeysAreIsolated(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			quota := Quota{Burst: 2, Rate: 1}

			for i := 0; i < quota.Burst; i++ {
				res, err := limiter.Allow(ctx, "tenant-a", quota)
				require.NoError(t, err)
				require.True(t, res.Allowed)
			}

			exhausted, err := limiter.Allow(ctx, "tenant-a", quota)
			require.NoError(t, err)
			require.False(t, exhausted.Allowed)

			fresh, err := limiter.Allow(ctx, "tenant-b", quota)
			require.NoError(t, err)
			assert.True(t, fresh.Allowed, "bir tenant'in limiti digerini etkiledi")
		})
	}
}

// TestAllow_RefillsOverTime verifies the allowance comes back as time passes.
func TestAllow_RefillsOverTime(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			// 20 tokens per second, so one token refills in about 50ms.
			quota := Quota{Burst: 1, Rate: 20}
			key := "yenilenen"

			first, err := limiter.Allow(ctx, key, quota)
			require.NoError(t, err)
			require.True(t, first.Allowed)

			denied, err := limiter.Allow(ctx, key, quota)
			require.NoError(t, err)
			require.False(t, denied.Allowed)

			time.Sleep(120 * time.Millisecond)

			refilled, err := limiter.Allow(ctx, key, quota)
			require.NoError(t, err)
			assert.True(t, refilled.Allowed, "the allowance did not come back after waiting")
		})
	}
}

// TestAllow_ConcurrentDoesNotExceedBurst verifies concurrent requests do not exceed
// capacity.
//
// This requires the "read, compute, write" sequence to be atomic. In the Redis
// implementation the Lua script provides that; written as separate commands, two
// replicas could spend the same token.
func TestAllow_ConcurrentDoesNotExceedBurst(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			const burst = 20
			const attempts = 200

			// Rate is zero so nothing refills during the test and the count is exact.
			quota := Quota{Burst: burst, Rate: 0}
			key := "eszamanli"

			var allowed atomic.Int64
			var wg sync.WaitGroup

			wg.Add(attempts)
			for i := 0; i < attempts; i++ {
				go func() {
					defer wg.Done()
					res, err := limiter.Allow(ctx, key, quota)
					if err == nil && res.Allowed {
						allowed.Add(1)
					}
				}()
			}
			wg.Wait()

			assert.Equal(t, int64(burst), allowed.Load(),
				"eszamanli yukte gecen istek sayisi kapasiteden farkli")
		})
	}
}

// TestPerMinute verifies a per-minute quota converts to the expected rate.
func TestPerMinute(t *testing.T) {
	quota := PerMinute(120)

	assert.Equal(t, 120, quota.Burst)
	assert.InDelta(t, 2.0, quota.Rate, 0.0001)
}

// TestQuota_ZeroBurstDeniesEverything verifies a zero quota rejects everything. It is
// usable for a suspended tenant.
func TestQuota_ZeroBurstDeniesEverything(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			res, err := limiter.Allow(context.Background(), "suspended", Quota{Burst: 0})
			require.NoError(t, err)
			assert.False(t, res.Allowed)
		})
	}
}

// TestRedis_FailOpenWhenUnreachable verifies the default behaviour is to let the
// request through when Redis is unreachable.
//
// The reasoning: the limiter is an abuse brake, not an availability tool. Rejecting
// all traffic when Redis goes down manufactures the very outage it is meant to
// prevent.
func TestRedis_FailOpenWhenUnreachable(t *testing.T) {
	// A port nobody is listening on.
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	limiter := NewRedis(client, "test:unreachable:")
	quota := PerMinute(10)

	res, err := limiter.Allow(context.Background(), "anahtar", quota)

	assert.Error(t, err, "the connection error was not reported")
	assert.True(t, res.Allowed, "fail-open acikken istek reddedildi")

	limiter.FailOpen = false
	res, err = limiter.Allow(context.Background(), "anahtar", quota)

	assert.Error(t, err)
	assert.False(t, res.Allowed, "fail-open kapaliyken istek gecti")
}
