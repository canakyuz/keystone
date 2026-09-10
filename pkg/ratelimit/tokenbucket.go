// Package ratelimit provides a request limiter shared across replicas.
//
// Why shared: an in-process limiter counts separately on every replica. Three
// replicas configured at 100 requests/minute each actually enforce 300
// requests/minute. Behind a load balancer, the gap between the configured value and
// the enforced one scales with the replica count.
//
// The algorithm is a token bucket. Unlike a fixed window counter it does not allow
// twice the limit across a window boundary, and it deliberately permits bursting up
// to capacity.
package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// Result is the outcome of a single limit check.
type Result struct {
	// Allowed says whether the request may proceed.
	Allowed bool

	// Limit is the bucket's capacity. It is written to the X-RateLimit-Limit header.
	Limit int

	// Remaining is the number of tokens left, rounded down.
	Remaining int

	// RetryAfter is how long a rejected request should wait before retrying. It is
	// zero for allowed requests.
	RetryAfter time.Duration
}

// Quota defines the limit applied to one key.
type Quota struct {
	// Burst is the bucket's capacity: the most requests let through in a spike.
	Burst int

	// Rate is the tokens added per second, that is, the steady rate.
	Rate float64
}

// PerMinute builds a quota from a per-minute request count. Burst equals that
// per-minute value: a full minute's allowance can be spent at once, after which the
// caller drops to the steady rate.
func PerMinute(requests int) Quota {
	return Quota{Burst: requests, Rate: float64(requests) / 60.0}
}

// Limiter is the rate limiter contract.
type Limiter interface {
	// Allow tries to consume one token for the key.
	Allow(ctx context.Context, key string, quota Quota) (Result, error)
}

// --- in-process implementation -------------------------------------------

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// Memory is a token bucket confined to a single process.
//
// It is for single-replica deployments and tests only. Across several replicas the
// Redis-backed implementation must be used; otherwise the enforced limit is
// multiplied by the replica count.
type Memory struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

// NewMemory creates the in-process limiter.
func NewMemory() *Memory {
	return &Memory{buckets: make(map[string]*bucket), now: time.Now}
}

// Allow tries to consume a token.
// Complexity: O(1).
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

// refill adds tokens for the elapsed time, never exceeding capacity.
func refill(b *bucket, quota Quota, now time.Time) {
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(float64(quota.Burst), b.tokens+elapsed*quota.Rate)
		b.lastFill = now
	}
}

// consume takes one token, or rejects.
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

// retryAfter computes how long it takes for one token to become available.
func retryAfter(tokens, rate float64) time.Duration {
	if rate <= 0 {
		return time.Duration(math.MaxInt64)
	}

	seconds := (1 - tokens) / rate

	// Rounded up: rounding down would invite the client to retry too early.
	return time.Duration(math.Ceil(seconds*1000)) * time.Millisecond
}
