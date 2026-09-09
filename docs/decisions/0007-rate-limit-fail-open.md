# 0007. The rate limit is shared and stays open when Redis is down

**Status:** Accepted, 2026-09-07

## Context

Fiber's built-in limiter counted in-process by default. Across three replicas, a
setting of a hundred requests per replica actually meant three hundred. The gap
between the configured value and the enforced one scaled with the replica count.

The key was also the IP, so the limit applied per connection rather than per
tenant.

## Rules that must hold

- The configured limit must be enforced regardless of replica count.
- One tenant's traffic must not consume another's limit.
- The counter update must be atomic.

## Options considered

### A. Fixed window counter

Simple. Weakness: it allows twice the limit across a window boundary. A full
limit's worth of requests can pass at the end of one window and the start of the
next.

### B. Sliding window log

Correct, but it stores a timestamp per request. Memory cost grows with request
rate.

### C. Token bucket, in Redis with Lua

Deliberately allows bursting up to capacity, then settles to a steady rate. The
read, compute and write sequence runs in a single script.

## Decision

C.

## Rationale

The Lua script provides the atomicity. Done with three separate Redis commands,
two replicas could spend the same token. Optimistic locking with `WATCH`/`MULTI`
was also possible, but it requires retries on conflict and the cost rises exactly
when the limit starts biting.

A test verifies that out of two hundred concurrent requests exactly the capacity
passes.

The limit is derived from the tenant's plan. The schema already had a `plan`
column.

## When Redis is down: it stays open

This is a deliberate decision, and it is written out separately because it is
debatable.

The default behaviour is to let the request through. The reasoning: the limiter is
an abuse brake, not an availability tool. Rejecting all traffic when Redis goes
down manufactures the very outage it is meant to prevent.

The counter-argument is valid: on endpoints where abuse is expensive,
authentication for example, staying open opens the door to brute force. That is
why the behaviour is configurable (`FailOpen`) and can be turned off per
endpoint. It has not been turned off anywhere yet.

## Consequences

- Every request adds one Redis round trip. Health endpoints are exempt.
- Plan resolution is cached separately; otherwise the limiter itself would become
  a source of load.

## When this decision becomes wrong

- If Redis latency takes a noticeable share of request latency. At that point a
  local pre-filter plus periodic synchronisation is needed.
- If the cost of abuse exceeds the cost of unavailability. The fail-open default
  should then be inverted.
