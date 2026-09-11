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
- **Verified isolation.** The claims are exercised against real PostgreSQL using a
  non-superuser role, at the database layer (`test/security/`) and across the whole
  request path (`test/e2e/`).

## Why it is interesting

This repository does not merely claim tenant isolation, it tests it. Writing the
tests that actually measure isolation surfaced six separate bugs, all of them on the
record.

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

The most recent one is the inverse failure, and it is the reason `test/e2e` exists.
`ExecuteInTenantContext` set `search_path` but never `app.current_tenant`, so under
the non-superuser role the operating requirements mandate, every RLS-protected read
returned the empty set. Every other test passed, because they all connect as the
superuser and superusers bypass RLS. Writing one test that entered through HTTP and
connected as the application role surfaced it on the first run.

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

## Measured behaviour

The claims above are measured rather than asserted. `scripts/loadtest.sh` reproduces
this: it brings up the dependencies, applies the migrations, seeds two tenants on
different plans, mints their tokens, starts the server and runs `loadtest/tenant_read.js`
against it.

### What the load test established

| Property | Result |
|---|---|
| Tenant-scoped read, median latency | 4-10 ms across every run |
| Liveness probe, p95 | under 2 ms |
| Readiness probe, p95 | 10-50 ms, and it touches the database and Redis |
| Enterprise tenant at its 100/s ceiling | 6001 requests, 0 rejected |
| Pro tenant under a 300/s burst | ~50% rejected, the rest served |
| Tenant schema cache | 7396 L1 hits, 3 L2 hits, 1 miss |

The cache line is the one ADR-0006 was missing. A comment in the previous implementation
claimed a "98-99% hit rate" and had never been checked; the measured figure on this
profile is 99.96%, and it is now a metric rather than a comment.

The rate limiter lines matter more than they look. The plan quotas are enforced exactly:
an enterprise tenant runs at its ceiling without a single rejection, while a pro tenant's
smaller bucket turns away half of a burst. The token bucket lets the first 1200 through
on purpose — bursting up to capacity is what ADR-0007 chose over a fixed window.

### What it did not establish

**The tail latency is not a property of this service on this hardware.** The load
generator, the database, Redis and the server all share one laptop, and p95 for the
tenant read moved between 5 ms and 2.39 s across runs while the median stayed in its
4-10 ms band. Plotted against the system load average — 10 to 17 on a 15-core machine,
with nine unrelated containers running — the tail tracks the machine, not the code.

So there is no P95 figure here. Publishing the best run would be exactly the kind of
unverified number the rest of this repository exists to avoid. What is published is the
part that held steady regardless of load, and the script that lets anyone produce the
rest on hardware where it would mean something.

Conditions for the numbers above: Apple M5 Pro, 15 cores, 24 GB, macOS 26.6.1,
PostgreSQL 16 and Redis 7 in Docker, k6 2.2.0, everything on one host, `RATE=100` for
60 s with a `BURST_RATE=300` for 10 s.

## Metrics

`/metrics` serves the Prometheus exposition; the worker publishes its own on
`WORKER_METRICS_ADDR`, because it is a separate process and scaling the two together
would be an accident rather than a decision.

RED signals per endpoint, worker claim and outcome counts with queue depth, cache
outcomes, and rate limit decisions by plan.

Nothing is labelled by tenant id. A Prometheus series exists per distinct combination of
label values, so a tenant label makes the series count grow with the customer count and
the monitoring bill grows with it — silently, because the metric keeps working. The
labels here are bounded by construction: route templates rather than paths, status
classes rather than codes, plan names rather than tenant ids. `pkg/metrics` carries the
reasoning, and a test asserts the property rather than trusting review to catch it.

## Tracing

Metrics answer "how often, how fast, how many". They cannot answer "what happened to
this one request", and provisioning is asynchronous, so the failures worth investigating
are the ones that span two processes and several minutes.

Tracing is off unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set. Off means a no-op tracer
rather than a branch at every call site, so the traced and untraced builds run the same
code. The propagator is installed either way: an untraced deployment still forwards a
caller's trace context instead of breaking someone else's trace.

### The interesting part: the hop with no headers

The HTTP hop carries trace context in a header. The provisioning hop does not — the API
writes a database row and a worker reads it back, in another process, minutes later and
possibly several times. So the W3C traceparent travels on that row (migration 034).

The worker turns it into a **link**, not a parent:

```
request trace                    worker trace
┌────────────────────┐           ┌────────────────────┐
│ POST /v1/tenants   │◀─ link ───│ provision <tenant> │
│ (ends in ~5ms)     │           │ (runs for seconds, │
└────────────────────┘           │  maybe retried)    │
                                 └────────────────────┘
```

Parenting would be the obvious choice and the wrong one. A trace only completes when all
its spans do, so a job that keeps failing would leave the request's trace open forever,
and every retry would hang off a request that ended hours earlier. A link says "this was
caused by that" without claiming the two are one operation, which is what the
OpenTelemetry conventions call for when producer and consumer are decoupled in time.

`X-Trace-ID` comes back on every response, so "this request was slow" turns into a trace
lookup rather than a guess.

## gRPC

A typed contract runs alongside the REST API, on its own port and in the same process.
Both surfaces call the same repositories, so idempotency, tenant isolation and the
operation state machine have one implementation and two ways in. A second implementation
would be a second set of bugs.

```bash
GRPC_ADDR=127.0.0.1:9099 GRPC_REFLECTION=true go run ./cmd/server

grpcurl -plaintext 127.0.0.1:9099 list
# keystone.v1.OperationService
# keystone.v1.UserService

grpcurl -plaintext -H "authorization: Bearer $TOKEN" \
  -d '{"limit": 3}' 127.0.0.1:9099 keystone.v1.UserService/ListUsers
```

Reflection is off by default. It hands anyone who can reach the port a complete list of
methods and message shapes, which is useful in development and is a map of the attack
surface anywhere else.

### The verification is shared, not duplicated

`pkg/authn` decides what a token says; the HTTP middleware and the gRPC interceptor are
the two halves that pull it out of a header or out of metadata and translate a failure
into a status code. That is not tidiness. The escalation this repository already closed
once lived exactly here — a tenant read from an untrusted header instead of a verified
claim — and two copies of that logic would be two chances to make the mistake and one
place it gets fixed.

`ListUsersRequest` has no tenant field, and that is the design. The tenant comes from the
verified token; a field the caller controls is a field the caller can change.

### It is held to the same tests

The roadmap's finishing condition for this phase was that the isolation tests pass over
gRPC, not that the codegen works. `test/e2e/grpc_isolation_test.go` drives the real
interceptor chain against real PostgreSQL over the non-superuser role, and asserts the
same things the HTTP tests do: an unauthenticated call is refused, a forged token is
refused, an unknown tenant is refused, a cross-tenant read returns nothing, and
caller-supplied metadata cannot override the token.

```bash
buf lint
go test ./test/e2e/ -run TestGRPC
```

CI regenerates the code and diffs it, so the committed output cannot drift from the
`.proto` files it came from.

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

## What is the control plane and what is an example

```
internal/          the control plane          88 Go files
  domain/          tenant, operation, user, registry
  repository/      the same, plus template
  usecase/         tenant provisioning, users
  handler/         auth, tenant, operation, user, registry, upload
  middleware/      auth, tenant context, rate limit, metrics, tracing
  worker/          job claiming, leases, graceful shutdown
  grpc/            the typed surface
pkg/               cache, ratelimit, database, metrics, tracing, authn, tenantctx
examples/verticals/ a reference application  77 Go files
  blog, booking, lessons, payments, projects, services, websites
```

The verticals are a reference application. They are here to show what building on top of
tenant provisioning and isolation looks like, and they are out of `internal/` so that the
part of this repository worth reading is not buried under them.

**The dependency runs one way, and the compiler enforces it.** `examples/verticals`
imports the control plane; `internal/app` does not name a single one of those packages.
`cmd/server` is the only place that knows about both, and dropping one line there leaves a
control plane that builds and runs without them.

That is asserted rather than described. `test/architecture` walks the real import graph —
transitively, because an import three packages deep couples the two just as firmly as a
direct one — and fails the build if the arrow ever points the wrong way.

```bash
go test ./test/architecture/
```

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
go test ./...                       # everything
go test ./test/security/            # the RLS configuration, driven directly with SQL
go test ./test/e2e/                 # the whole chain, from HTTP down to the database
go test -race ./internal/worker/    # concurrency, shutdown, per-tenant limits
```

The two isolation suites answer different questions. `test/security` asks whether the
policies are right. `test/e2e` asks whether a real request actually ends up inside
them: it drives the assembled application over the non-superuser role production uses,
covering unauthenticated and forged tokens, unknown and malformed tenant ids, a
cross-tenant read, both untrusted tenant sources, and the state of a connection handed
back to the pool.

Tests use real PostgreSQL. Each test opens its own isolated database and drops it
afterwards, so running them in parallel is safe. When Postgres is unreachable the
tests that need it are skipped.

The single source of truth for the schema is the `migrations/` directory. The test
helper runs those files directly and keeps no copy.

## Operating requirement

The application must connect to the database with a **non-superuser** role. In
PostgreSQL, superusers bypass RLS under all circumstances. Details:
[SECURITY.md](SECURITY.md).

The worker connects as a role of its own: create a login role, grant it
`keystone_worker` (defined by migration 039), and set `WORKER_DB_USER` and
`WORKER_DB_PASSWORD`. In production the worker refuses to start without one.

## Status

The control plane works: tenant provisioning with durable operations, leases and fencing,
a worker running under its own narrowly privileged database role, RLS enforcement verified end to end, membership and roles checked against the tenant's own
record on every request, an audit trail and a transactional outbox written in the same
transaction as the change they describe, a REST and a gRPC surface over the same
repositories, metrics, tracing, and a migration runner. The vertical modules sit in
`examples/verticals` as a reference application.

Open: a platform permission model for the routes that act across tenants, audit entries
for changes other than provisioning, and a load measurement taken somewhere the load
generator is not sharing a machine with the service. See [docs/ROADMAP.md](docs/ROADMAP.md).

## License

MIT. See [LICENSE](LICENSE).
