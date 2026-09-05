// Package ratelimit, replikalar arasinda paylasilan bir istek limitleyici sunar.
//
// Neden paylasimli: surec ici bir limitleyici, her replikada ayri sayar.
// Uc replika ve replika basina 100 istek/dakika ayari, gercekte 300 istek/dakika
// demektir. Load balancer arkasinda ayarlanan deger ile uygulanan deger arasinda
// replika sayisi kadar fark olusur.
//
// Algoritma token bucket'tir. Sabit pencere sayacinin aksine pencere sinirinda
// iki kat gecise izin vermez ve kapasite kadar patlamaya (burst) bilincli olarak
// izin verir.
package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// Result, tek bir limit sorgusunun sonucudur.
type Result struct {
	// Allowed, istegin gecirilip gecirilmeyecegini soyler.
	Allowed bool

	// Limit, kovanin kapasitesidir. X-RateLimit-Limit basligina yazilir.
	Limit int

	// Remaining, kalan token sayisidir (asagi yuvarlanmis).
	Remaining int

	// RetryAfter, reddedilen istegin ne kadar sonra tekrar denenebilecegidir.
	// Izin verilen isteklerde sifirdir.
	RetryAfter time.Duration
}

// Quota, bir anahtar icin uygulanacak limiti tanimlar.
type Quota struct {
	// Burst, kovanin kapasitesi. Ani yukte gecirilecek en fazla istek.
	Burst int

	// Rate, saniyede eklenen token sayisi (surekli hiz).
	Rate float64
}

// PerMinute, dakikalik istek sayisindan kota uretir.
// Burst, dakikalik degerin kendisidir: bir dakikalik hakkin tamami bir anda
// kullanilabilir, sonrasinda surekli hiza duser.
func PerMinute(requests int) Quota {
	return Quota{Burst: requests, Rate: float64(requests) / 60.0}
}

// Limiter, istek limitleyicinin sozlesmesidir.
type Limiter interface {
	// Allow, anahtar icin bir token tuketmeyi dener.
	Allow(ctx context.Context, key string, quota Quota) (Result, error)
}

// --- surec ici uygulama --------------------------------------------------

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// Memory, tek surecte calisan token bucket'tir.
//
// Yalnizca tek replikali kurulumlar ve testler icindir. Coklu replikada
// Redis destekli uygulama kullanilmalidir; aksi halde uygulanan limit
// replika sayisiyla carpilir.
type Memory struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

// NewMemory, surec ici limitleyici olusturur.
func NewMemory() *Memory {
	return &Memory{buckets: make(map[string]*bucket), now: time.Now}
}

// Allow, token tuketmeyi dener.
// Karmasiklik: O(1).
func (m *Memory) Allow(_ context.Context, key string, quota Quota) (Result, error) {
	if quota.Burst <= 0 {
		return Result{Allowed: false, Limit: 0}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	b, ok := m.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(quota.Burst), lastFill: now}
		m.buckets[key] = b
	}

	refill(b, quota, now)

	return consume(b, quota), nil
}

// refill, gecen sureye gore token ekler ve kapasiteyi asmaz.
func refill(b *bucket, quota Quota, now time.Time) {
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(float64(quota.Burst), b.tokens+elapsed*quota.Rate)
		b.lastFill = now
	}
}

// consume, bir token dusurur veya reddeder.
func consume(b *bucket, quota Quota) Result {
	if b.tokens < 1 {
		return Result{
			Allowed:    false,
			Limit:      quota.Burst,
			Remaining:  0,
			RetryAfter: retryAfter(b.tokens, quota.Rate),
		}
	}

	b.tokens--

	return Result{
		Allowed:   true,
		Limit:     quota.Burst,
		Remaining: int(b.tokens),
	}
}

// retryAfter, bir tokenin dolmasi icin gereken sureyi hesaplar.
func retryAfter(tokens, rate float64) time.Duration {
	if rate <= 0 {
		return time.Duration(math.MaxInt64)
	}

	seconds := (1 - tokens) / rate

	// Asagi yuvarlama istemciyi erken tekrar denemeye itecegi icin yukari alinir.
	return time.Duration(math.Ceil(seconds*1000)) * time.Millisecond
}
