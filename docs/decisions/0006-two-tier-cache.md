# 0006. The cache is two-tier and singleflight-protected

**Status:** Accepted, 2026-09-07

## Context

Resolving a tenant id to a schema name runs on every request. It is the hottest
path in the system.

## Rules that must hold

- The service must keep working when Redis is unreachable.
- Non-existent tenant ids must not put load on the database.
- Concurrent misses on the same key must collapse into a single load.

## Options considered

### A. Redis only (the previous state)

Measuring it surfaced four weaknesses at once:

- N concurrent requests for a cold key produced N database queries. The cache
  failed to protect at exactly the moment it was most needed.
- Non-existent ids were not cached. A flood of requests with random ids reached
  the database every time.
- Cache fill spawned a new goroutine per request, with no panic recovery.
- Redis was a single point of failure.

### B. In-process cache only

Simple, but nothing is shared across replicas. Each replica warms up separately
and database load is multiplied by the replica count.

### C. Two tiers: in-process plus Redis

## Decision

C, with singleflight and negative caching.

## Rationale

The L1 tier removes Redis as a single point of failure. When Redis is down the
service keeps working with L1 and the database.

`singleflight` solves the stampede. Requests arriving at once for the same key
collapse into a single load; a test verifies that a hundred concurrent requests
produce one query.

Negative caching closes a cheap load-amplification vector.

TTLs carry jitter. Keys created at the same moment would otherwise expire at the
same moment and produce a synchronized wave of misses.

## Consequences

- There is a cross-process inconsistency window. Invalidating an entry takes
  immediate effect only in that process; other replicas may see the stale value
  until the L1 TTL (30 seconds) expires.
- This is acceptable for the tenant schema because the schema only changes during
  onboarding. It would not be acceptable for frequently changing data.

## When this decision becomes wrong

- If the cached data starts changing often. The 30-second inconsistency window
  becomes unacceptable, and invalidation broadcast over pub/sub is needed.
- If memory pressure appears. The L1 ceiling is 10,000 entries; once the tenant
  count exceeds that, eviction becomes frequent and the L1 benefit shrinks.

## Measurement note

A comment in the previous implementation claimed a "98-99% hit rate"; it had
never been measured. `Stats()` was added but is not wired to metrics yet.
