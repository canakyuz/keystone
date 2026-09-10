package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript updates the token bucket in a single atomic step.
//
// WHY LUA: if the read, compute and write sequence is done with three separate Redis
// commands, two replicas can read at the same time and spend the same token. Redis
// runs a Lua script as one operation, so the race cannot occur by construction.
// Optimistic locking with WATCH/MULTI was also possible, but it requires retries on
// conflict, and under load the cost rises exactly when the limit starts biting.
//
// KEYS[1] : bucket key
// ARGV[1] : capacity (burst)
// ARGV[2] : refill rate per second
// ARGV[3] : now, in milliseconds
// ARGV[4] : key lifetime, in seconds
//
// Returns: {allowed(0/1), remaining_tokens, retry_after_ms}
const tokenBucketScript = `
local key      = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate     = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local ttl      = tonumber(ARGV[4])

local state     = redis.call('HMGET', key, 'tokens', 'ts')
local tokens    = tonumber(state[1])
local last      = tonumber(state[2])

if tokens == nil then
  tokens = capacity
  last   = now
end

-- Refill for the elapsed time. Ignore a negative interval if the clock went back.
local elapsed = math.max(0, now - last) / 1000.0
tokens = math.min(capacity, tokens + elapsed * rate)

local allowed = 0
local retry   = 0

if tokens >= 1 then
  allowed = 1
  tokens  = tokens - 1
else
  if rate > 0 then
    retry = math.ceil(((1 - tokens) / rate) * 1000)
  else
    retry = ttl * 1000
  end
end

redis.call('HSET', key, 'tokens', tokens, 'ts', now)
redis.call('EXPIRE', key, ttl)

return {allowed, math.floor(tokens), retry}
`

// Redis is the token bucket limiter shared across replicas.
type Redis struct {
	client *redis.Client
	script *redis.Script
	prefix string

	// FailOpen decides whether requests pass when Redis is unreachable.
	//
	// It defaults to open, that is, requests pass. The reasoning: the limiter is an
	// abuse brake, not an availability tool. Rejecting all traffic when Redis goes down
	// manufactures the very outage it is meant to prevent. On endpoints where abuse is
	// expensive, authentication for example, this can be deliberately turned off.
	FailOpen bool
}

// NewRedis creates the shared limiter.
func NewRedis(client *redis.Client, keyPrefix string) *Redis {
	return &Redis{
		client:   client,
		script:   redis.NewScript(tokenBucketScript),
		prefix:   keyPrefix,
		FailOpen: true,
	}
}

// Allow tries to consume one token for the key.
//
// Complexity: one Redis round trip, O(1) work server-side.
func (r *Redis) Allow(ctx context.Context, key string, quota Quota) (Result, error) {
	if quota.Burst <= 0 {
		return Result{Allowed: false}, nil
	}

	ttl := bucketTTL(quota)

	raw, err := r.script.Run(ctx, r.client,
		[]string{r.prefix + key},
		quota.Burst, quota.Rate, time.Now().UnixMilli(), int(ttl.Seconds()),
	).Slice()
	if err != nil {
		return r.onFailure(quota), fmt.Errorf("rate limit query failed: %w", err)
	}

	return parseResult(raw, quota)
}

// bucketTTL computes how long the bucket key lives.
//
// It must not be shorter than the time an empty bucket needs to refill completely;
// otherwise the key is deleted early and the client wins back its full quota.
func bucketTTL(quota Quota) time.Duration {
	if quota.Rate <= 0 {
		return time.Hour
	}

	fill := time.Duration(float64(quota.Burst)/quota.Rate) * time.Second

	return max(2*fill, time.Minute)
}

// onFailure produces the behaviour applied when Redis is unreachable.
func (r *Redis) onFailure(quota Quota) Result {
	return Result{
		Allowed:    r.FailOpen,
		Limit:      quota.Burst,
		Remaining:  0,
		RetryAfter: time.Second,
	}
}

// parseResult converts the Lua script's return value into a Result.
func parseResult(raw []any, quota Quota) (Result, error) {
	if len(raw) != 3 {
		return Result{}, fmt.Errorf("unexpected rate limit response: %d fields", len(raw))
	}

	allowed, _ := raw[0].(int64)
	remaining, _ := raw[1].(int64)
	retryMS, _ := raw[2].(int64)

	return Result{
		Allowed:    allowed == 1,
		Limit:      quota.Burst,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(retryMS) * time.Millisecond,
	}, nil
}
