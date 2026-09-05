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

// limiters, ayni davranis sozlesmesini iki uygulamada da sinamak icin kullanilir.
// Surec ici ve Redis destekli uygulamalarin ayni sekilde davranmasi gerekir;
// aksi halde yerelde gecen bir davranis uretimde farkli calisir.
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
		t.Logf("Redis erişilemiyor (%v), yalnızca süreç içi uygulama sınanıyor", err)
		return nil
	}

	t.Cleanup(func() { client.Close() })
	return client
}

var prefixCounter atomic.Uint64

// uniquePrefix, testlerin birbirinin kovasini gormesini engeller.
func uniquePrefix(t *testing.T) string {
	t.Helper()
	return "test:rl:" + t.Name() + ":" + time.Now().Format("150405.000") + ":" +
		string(rune('a'+prefixCounter.Add(1)%26)) + ":"
}

// TestAllow_BurstThenDeny, kapasite kadar istegin gectigini ve sonrakinin
// reddedildigini dogrular.
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

// TestAllow_KeysAreIsolated, bir anahtarin limitinin digerini etkilemedigini
// dogrular. Cok kiracili bir sistemde bu, bir tenant'in digerini
// aclikta birakmasini engeller.
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

// TestAllow_RefillsOverTime, zaman gectikce hakkin geri geldigini dogrular.
func TestAllow_RefillsOverTime(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			// Saniyede 20 token: bir token ~50ms'de dolar.
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
			assert.True(t, refilled.Allowed, "bekleme sonrasi hak geri gelmedi")
		})
	}
}

// TestAllow_ConcurrentDoesNotExceedBurst, esazamanli isteklerin kapasiteyi
// asmadigini dogrular.
//
// Bu, "oku, hesapla, yaz" dizisinin atomik olmasini gerektirir. Redis
// uygulamasinda bunu Lua betigi saglar; ayri komutlarla yazilsaydi iki
// replika ayni tokeni harcayabilirdi.
func TestAllow_ConcurrentDoesNotExceedBurst(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			const burst = 20
			const attempts = 200

			// Rate sifir: test suresince dolum olmasin, sayim kesin olsun.
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

// TestPerMinute, dakikalik kotanin beklenen hiza cevrildigini dogrular.
func TestPerMinute(t *testing.T) {
	quota := PerMinute(120)

	assert.Equal(t, 120, quota.Burst)
	assert.InDelta(t, 2.0, quota.Rate, 0.0001)
}

// TestQuota_ZeroBurstDeniesEverything, sifir kotanin her istegi reddettigini
// dogrular. Askiya alinmis bir tenant icin kullanilabilir.
func TestQuota_ZeroBurstDeniesEverything(t *testing.T) {
	for name, limiter := range limiters(t) {
		t.Run(name, func(t *testing.T) {
			res, err := limiter.Allow(context.Background(), "askida", Quota{Burst: 0})
			require.NoError(t, err)
			assert.False(t, res.Allowed)
		})
	}
}

// TestRedis_FailOpenWhenUnreachable, Redis erisilemedigi zaman varsayilan
// davranisin istegi gecirmek oldugunu dogrular.
//
// Gerekce: limitleyici bir kullanilabilirlik araci degil, kotuye kullanim
// frenidir. Redis dustugunde tum trafigi reddetmek, onlemeye calistigi
// kesintiyi kendi eliyle yaratir.
func TestRedis_FailOpenWhenUnreachable(t *testing.T) {
	// Kimsenin dinlemedigi bir port.
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	limiter := NewRedis(client, "test:unreachable:")
	quota := PerMinute(10)

	res, err := limiter.Allow(context.Background(), "anahtar", quota)

	assert.Error(t, err, "erisim hatasi bildirilmedi")
	assert.True(t, res.Allowed, "fail-open acikken istek reddedildi")

	limiter.FailOpen = false
	res, err = limiter.Allow(context.Background(), "anahtar", quota)

	assert.Error(t, err)
	assert.False(t, res.Allowed, "fail-open kapaliyken istek gecti")
}
