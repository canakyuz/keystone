# Keystone Roadmap

Goal: make a Go engineer who spends ten minutes with this repository conclude
that the author is senior. This document splits that goal into measurable phases.

---

## Success criterion

A hiring engineer landing on the repository follows this sequence: the README,
then an attempt to run the tests, then a look at whichever package seems most
interesting. Every phase must improve one of those three steps.

The end state is that all of the following can be verified with a single command.

| Claim | Evidence |
|---|---|
| Tenant isolation works | `go test ./test/security/` |
| Isolation holds from HTTP down to the database | `go test ./test/e2e/` |
| Concurrency is written correctly | `go test -race ./internal/worker/` |
| The performance claim is measured | A P50/P95/P99 table in the README |
| The service contract is defined | `buf lint` plus a `grpcurl` example |

---

## Where things stand

Measured 2026-09-09.

| Measure | Value | Comment |
|---|---|---|
| Source files | 170 | Volume is sufficient |
| Test files | 17 | Coverage is narrow but the core is covered |
| Packages with tests | 15 | Worker, cache, ratelimit, RLS, operations |
| HTTP-layer end-to-end tests | 0 | Still the largest gap |
| Decision records | 7 | |
| Verticals under `internal/domain` | 13 | Core is 4 of them, the rest are noise |

Phases 1 through 4 of the original plan have shifted since it was written. What
actually got built is recorded below.

---

## Done

**Isolation hardening.** Five bugs surfaced by writing tests that actually
measure isolation, all fixed and documented in [SECURITY.md](../SECURITY.md). RLS
is now `FORCE`d and fail-closed, verified against real PostgreSQL with a
non-superuser role.

**Cache and rate limiting.** Two-tier cache with singleflight, negative caching
and TTL jitter. Redis token bucket in a single Lua script, keyed by tenant and
sized by plan.

**Durable provisioning.** This replaced the transactional outbox originally
planned as Phase 2. The same engineering ground is covered — worker pool, context
cancellation, graceful shutdown, `SKIP LOCKED` claiming — but on the pipeline the
product actually needs rather than a synthetic one. Idempotency enforced by a
uniqueness constraint, lease with a fencing token, per-tenant concurrency limits,
and tenant activation in the same transaction that closes the operation. Runs as
a separate process, `cmd/worker`.

**English documentation.** Originally the last phase. Moved to the front: the
depth is worth nothing to a reader who cannot read the prose describing it.

---

## Phase 1: End-to-end isolation proof

**Effort:** 2-3 days
**Why first:** It is the cheapest remaining phase and it finishes work already
done. Isolation is proven at the database layer today, but the chain from HTTP
down to the database is untested. The tenant middleware resolves the schema from
the request and all isolation rests on it; it has not got a single test.

**Work**

1. Open a `test/e2e/` package. Bring the real application up with Fiber's
   `app.Test()`, connected to real PostgreSQL.
2. Tenant middleware tests:
   - 401 when the tenant header is absent
   - 404 for an unknown tenant
   - 400 on an invalid schema name; a SQL injection attempt must be rejected
3. Cross-tenant test: an HTTP call using tenant A's token against tenant B's
   resource must return no data.
4. Verify that the middleware resets `search_path` at the end of the request. A
   dirty connection must not go back to the pool.

**Touches:** `internal/middleware/tenant_context.go`, `internal/app/routes.go`,
new `test/e2e/`

**Done when:** `go test ./test/e2e/` is green and the README has a section
proving isolation with a single command.

---

## Phase 2: Observability and measurement

**Effort:** 3-5 days
**Why:** "P95 under 200ms" is currently an unmeasured claim. A concrete number
beats an assertion. Without tracing it is also hard to show the worker's
behaviour.

**Work**

1. OpenTelemetry: spans for the HTTP request, the usecase, the repository and
   SQL. A provisioning step must join the trace of the request that created it.
2. Prometheus metrics: RED per endpoint (rate, errors, duration), labelled with
   `tenant_id`. Queue depth and claim latency for the worker.
3. zerolog is already in place; add trace id correlation.
4. A k6 scenario: a mix of tenant creation, login and listing.
5. Put the results in the README as a table. The hardware and scenario conditions
   must be written down, or the number means nothing.

**Done when:** the README carries a P50/P95/P99 table and the k6 file that
produced it is in the repository.

---

## Phase 3: gRPC and protobuf contracts

**Effort:** 1 week
**Why:** Job listings ask for REST and gRPC together. A type-safe contract also
naturally removes a share of the current `interface{}` usage.

**Work**

1. Tenant, identity and entitlement services under `proto/keystone/v1/`.
2. Codegen and lint with `buf`. `buf.yaml`, `buf.gen.yaml`, a `buf lint` step in
   CI.
3. The gRPC server alongside Fiber, on a separate port.
4. An interceptor chain: tenant context, auth, logging, panic recovery. These
   must share logic with the HTTP middleware rather than copying it.
5. Turn on server reflection and put a `grpcurl` example in the README.

**Touches:** new `proto/`, new `internal/grpc/`, `internal/app/app.go`

**Done when:** `buf lint` is clean, `grpcurl -plaintext localhost:9090 list`
prints the services, and the same isolation tests pass over gRPC too.

---

## Phase 4: Focus, split out the vertical modules

**Effort:** 3-4 days
**Why:** Of 170 source files, roughly 15 are the interesting ones. The blog,
booking, lesson, payment and website modules add volume without adding depth.
They pull a reviewer's attention away from the core.

**Why late:** This is a refactor and it touches 111 references inside
`internal/app/app.go` and `internal/app/routes.go`. Done before the core is
strong, not much would be left.

**Work**

1. Move the vertical modules under `examples/verticals/`.
2. Split the composition root in two: the core application and the example
   application. Today a single `app.go` wires everything.
3. Explain the boundary between core and example clearly in the README.

**Done when:** `go build ./...` is green and the core application comes up
without the vertical modules.

---

## Phase 5: Remaining invariants

**Effort:** 1 week
**Why:** Rules 7 and 8 in [INVARIANTS.md](INVARIANTS.md) are still marked "not
yet". A document that names its own gaps is good; leaving them open forever is
not.

**Work**

1. Audit table, written in the same transaction as the job status (rule 7).
2. Transactional outbox and webhook delivery (rule 8). The event is written in
   the same transaction as the business data; delivery is a separate, retryable
   step with exponential backoff and full jitter, and a dead-letter state after
   the maximum number of attempts.
3. A membership table, so that "is this subject a member of this tenant" becomes
   a real check (the gap in rule 1).
4. A separate, narrowly privileged database role for the worker — the closing
   condition of ADR-0001.

---

## Phase 6: Polish

**Effort:** 2-3 days

**Standing rule:** every new code comment, commit message and document is written
in English.

**Work**

1. A godoc comment on every exported symbol.
2. An architecture diagram.
3. A CHANGELOG and SemVer tagging.
4. `interface{}` cleanup: turn the DTO usages into concrete types.

---

## Out of scope

These are deliberately not being done.

- **Kubernetes and Helm.** They appear in job listings, but dropping a Helm chart
  into a repository is easy and copyable. A correctly written `SKIP LOCKED`
  worker is not. Low signal value.
- **Multi-region deployment.** Solving a scale problem that does not exist is
  YAGNI.
- **An admin interface.** This is a backend portfolio. Adding a frontend splits
  the focus.
- **New vertical modules.** There are already 13 and they are the problem itself.

---

## Why this order

Phase 1 comes first because it is the cheapest and it completes work already
done. The story of the five isolation bugs stays half-told without HTTP tests.

Phase 2 comes second because it converts the repository's remaining claims into
measurements, and because it is what makes the worker's behaviour visible.

Phase 4 sits late because it is a hard-to-reverse refactor. It should happen after
the core is strong.
