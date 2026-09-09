# Keystone

A multi-tenant SaaS control plane written in Go. It brings tenant provisioning,
schema-per-tenant isolation and Row Level Security together in one place.

[![CI](https://github.com/canakyuz/keystone/actions/workflows/ci.yml/badge.svg)](https://github.com/canakyuz/keystone/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## What it does

A SaaS product has to open a separate data space for every customer, isolate
that space, and prove the isolation actually holds. Keystone does those three
things.

- **Tenant provisioning.** Creates a PostgreSQL schema for a new tenant, applies
  the migrations, and writes it into the registry.
- **Two layers of isolation.** A per-tenant schema, plus RLS on the shared tables.
- **Verified isolation.** The claims live under `test/security/` and are exercised
  against real PostgreSQL using a non-superuser role.

## Why it is interesting

This repository does not merely claim tenant isolation, it tests it. Writing the
tests that actually measure isolation surfaced five separate bugs, all of them on
the record.

The most striking one: the `users` table carried two permissive policies.

```sql
CREATE POLICY tenant_isolation_policy ON users FOR ALL
  USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

CREATE POLICY auth_policy ON users FOR SELECT
  USING (TRUE);
```

PostgreSQL combines permissive policies with OR. Because the second one was a
constant `TRUE`, the combined condition collapsed to true on every `SELECT`.
There was no tenant isolation at all on reads from the `users` table.

Full list and fixes: [SECURITY.md](SECURITY.md).

## Provisioning is durable, not a request handler

Creating a tenant means creating a schema and running migrations. That work does
not belong inside an HTTP request: it takes too long, and a process that dies
halfway through leaves a half-built tenant behind. So the API accepts the intent
and a separate worker process carries it out.

```
POST /api/v1/tenants          202 Accepted + Location: /api/v1/operations/{id}
  Idempotency-Key: <key>
        |
        |  one transaction: tenant (pending) + operation + job + idempotency key
        v
   operations table
        |
        |  SELECT ... FOR UPDATE SKIP LOCKED    <- two workers never take one job
        v
  cmd/worker  (N goroutines, per-tenant limit, graceful drain)
        |
        |  create schema -> run migrations -> report result with current fence
        v
  one transaction: mark job done + flip tenant to active

GET /api/v1/operations/{id}   caller polls for the outcome
```

Three properties are worth naming.

**The same idempotency key cannot be reused with a different body.** Same body
replays the existing operation; a different body is a conflict. Uniqueness is
enforced by the database, because read-then-write does not survive two requests
arriving at once.

**A worker that lost its lease cannot report a result.** Every claim increments a
fence counter, and a completion must carry the current fence. Checking the lease
owner alone would not work — a stale worker knows its own name and would pass
that check.

**A tenant becomes `active` in the same transaction that closes the operation.**
Written separately, a crash in between would leave a tenant the user sees as
"done" but cannot actually use.

The guarantees and their limits are written down in
[docs/INVARIANTS.md](docs/INVARIANTS.md), including the ones not implemented yet.

## Cache

Tenant resolution runs on every request, which makes it the hottest path.
`pkg/cache` offers a two-tier cache: L1 in-process, L2 Redis, the database
underneath.

Problems closed:

- **Cache stampede.** N concurrent requests for a cold key turned into N database
  queries. `singleflight` collapses them into one. A test verifies that a hundred
  concurrent requests produce a single load.
- **No negative caching.** A flood of requests carrying random non-existent tenant
  IDs reached the database every time. A cheap load-amplification vector.
- **Unbounded goroutines.** Cache fill spawned a new goroutine per request, with no
  panic recovery.
- **Redis was a single point of failure.** Thanks to the L1 tier the service keeps
  serving when Redis is down.
- **Hit rate was not measured.** `Stats()` exposes it.

TTLs carry jitter. Keys created at the same moment would otherwise expire at the
same moment and produce a synchronized wave of misses.

## Rate limiting

`pkg/ratelimit` implements a token bucket shared over Redis. The read, compute
and write sequence runs inside a single Lua script, so two replicas cannot spend
the same token.

Fiber's built-in limiter is not used. It counts in-process by default: across
three replicas, a setting of a hundred requests per replica actually means three
hundred. The gap between the configured value and the enforced one scales with
the replica count.

The limit follows the tenant's plan, and the key is the tenant rather than the IP.

| Plan | Requests per minute |
|---|---|
| free | 60 |
| starter | 300 |
| pro | 1,200 |
| enterprise | 6,000 |
| unauthenticated | 30 |

When Redis is unreachable the default behaviour is to let the request through.
The limiter is an abuse brake, not an availability tool. Rejecting all traffic
when Redis goes down would manufacture the very outage it is meant to prevent.

A concurrency test verifies that out of two hundred simultaneous requests exactly
the capacity passes.

## Health endpoints

Two endpoints, two different questions.

| Endpoint | Question | On failure | Checks dependencies |
|---|---|---|---|
| `/health` | Is the process alive | Container restarts | No |
| `/ready` | Can it serve requests | Load balancer drains traffic | Yes |

`/health` deliberately does not touch the database. Restarting healthy processes
because the database blipped creates a connection storm during recovery.

Redis is not required for `/ready`. The cache and the limiter fall back to their
in-process paths without it.

## Design decisions

Why the system is built this way, what the alternatives were, and the condition
under which each decision becomes wrong is written under
[docs/decisions/](docs/decisions/). There are seven decision records.

For the promises the system makes, along with the code and tests backing them,
see [docs/INVARIANTS.md](docs/INVARIANTS.md). A rule with nothing behind it is
marked "not yet" there rather than being listed as a slogan.

## Architecture

```
HTTP (Fiber)                     cmd/worker
    |                                 |
    v                                 |
Handler  --->  Usecase  --->  Repository  --->  PostgreSQL
                                  |
                                  +-- TenantConnectionManager
                                      checks out one connection from the pool,
                                      sets search_path to the tenant schema,
                                      hands THAT SAME connection to the callback
```

Dependencies always point inward. `internal/domain` knows nothing about any outer
layer. Tenant context keys live in `pkg/tenantctx`, a leaf package, so the
repository layer never has to import HTTP middleware.

## Quick start

```bash
git clone https://github.com/canakyuz/keystone.git
cd keystone
cp .env.example .env

docker compose up -d postgres
go run ./cmd/server      # API
go run ./cmd/worker      # provisioning worker, separate process
```

Health check:

```bash
curl localhost:8080/health
```

## Tests

```bash
go test ./...                 # everything
go test ./test/security/      # isolation claims only
go test -race ./internal/worker/   # concurrency, shutdown, per-tenant limits
```

Tests use real PostgreSQL. Each test opens its own isolated database and drops it
afterwards, so running them in parallel is safe. When Postgres is unreachable the
tests that need it are skipped.

The single source of truth for the schema is the `migrations/` directory. The test
helper runs those files directly and keeps no copy.

## Operating requirement

The application must connect to the database with a **non-superuser** role. In
PostgreSQL, superusers bypass RLS under all circumstances. Details:
[SECURITY.md](SECURITY.md).

## Status

The core works: tenant provisioning, durable operations with lease and fencing,
user management, RLS enforcement, migration runner. The vertical modules (blog,
booking, lesson, payment) sit in the repository as a reference application,
showing how features are built on top of the control plane.

Roadmap: gRPC contracts, transactional outbox, webhook delivery, measured load
test results. See [docs/ROADMAP.md](docs/ROADMAP.md).

## License

MIT. See [LICENSE](LICENSE).
