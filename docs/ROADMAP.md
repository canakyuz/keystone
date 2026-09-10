# Keystone Roadmap

Goal: make a Go engineer who spends ten minutes with this repository conclude
that the author is senior. This document splits that goal into measurable phases.

---

## Success criterion

A hiring engineer landing on the repository follows this sequence: the README,
then an attempt to run the tests, then a look at whichever package seems most
interesting. Every phase must improve one of those three steps.

The end state is that all of the following can be verified with a single command.

| Claim | Evidence | Status |
|---|---|---|
| Tenant isolation works | `go test ./test/security/` | done |
| Isolation holds from HTTP down to the database | `go test ./test/e2e/` | done |
| Concurrency is written correctly | `go test -race ./internal/worker/` | done |
| The performance claim is measured | The measured behaviour section of the README | partial |
| The service contract is defined | `buf lint` plus a `grpcurl` example | open |

---

## Where things stand

Measured 2026-09-10.

| Measure | Value | Comment |
|---|---|---|
| Source files | 172 | Volume is sufficient |
| Test files | 22 | Coverage is narrow but the core is covered |
| Packages with tests | 17 | Worker, cache, ratelimit, metrics, RLS, operations, e2e |
| HTTP-layer end-to-end tests | 10 | Found five defects no other test reached |
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

**English documentation.** Originally the last phase. Moved to the front: the depth
is worth nothing to a reader who cannot read the prose describing it.

**Metrics and a reproducible load profile.** Covered in phase 1 below, which is where
the four defects it uncovered are listed.

**End-to-end isolation proof.** `test/e2e` drives the assembled application, from an
HTTP request through the real JWT and tenant middleware down to real PostgreSQL, over
the non-superuser role production uses. It covers the unauthenticated and forged-token
paths, an unknown tenant, malformed tenant ids, a cross-tenant read, both untrusted
tenant sources, and the state of a connection returned to the pool.

It paid for itself on the first run: `ExecuteInTenantContext` set `search_path` but
never `app.current_tenant`, so under the application role every RLS-protected read
returned the empty set. Every other test passed because they connect as the superuser,
and superusers bypass RLS. See [SECURITY.md](../SECURITY.md).

---

## Phase 1: Finish the measurement

**Effort:** 2-3 days
**State:** the instrumentation is done and the profile runs; the tail is not measurable
on the machine that has been available.

`pkg/metrics` publishes RED signals, worker counts with queue depth, cache outcomes and
rate limit decisions by plan, all with bounded labels. `loadtest/` and
`scripts/loadtest.sh` reproduce a run end to end. That work closed four defects that no
test had reached: the plan quotas were never applied, trial tenants were locked out, the
user endpoints returned 500, and a metric label was being read out of a pooled buffer.

What is left is a machine. The load generator, database, Redis and server currently share
one laptop, and p95 tracks its load average rather than the code. The remaining work is
to run the existing profile somewhere the two are separated and record the numbers.

**Still to do**

1. Run `scripts/loadtest.sh` with the load generator on a separate host.
2. Record P50/P95/P99 for the tenant read in the README, with the conditions.
3. Put the claim query under load and settle the measurement note in ADR-0002.

---

## Phase 1b: Tracing

**Effort:** 3-5 days
**Why:** "P95 under 200ms" is currently an unmeasured claim. A concrete number
beats an assertion. Without tracing it is also hard to show the worker's
behaviour.

**Work**

1. OpenTelemetry: spans for the HTTP request, the usecase, the repository and SQL. A
   provisioning step must join the trace of the request that created it.
2. Carry `request_id` from the HTTP layer into the worker, closing the gap noted in
   `internal/worker/provision_handler.go`.

Note on the original plan for this phase: it said to label the metrics with `tenant_id`.
That was not done, on purpose. A series exists per distinct combination of label values,
so a tenant label makes the series count grow with the customer count. Per-tenant
questions belong in traces and logs, which are sampled and indexed for exactly that.
`pkg/metrics` carries the full reasoning.

**Done when:** a provisioning job's spans appear under the trace of the request that
created it.

---

## Phase 2: gRPC and protobuf contracts

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

## Phase 3: Focus, split out the vertical modules

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

## Phase 4: Remaining invariants

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

## Phase 5: Polish

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

Measurement comes first among what is left, because it converts the repository's
remaining unsupported claims into numbers. Two decision records currently say in as
many words that a cost was assumed rather than measured; that is the one place a
careful reader can push back.

gRPC comes second. Job listings ask for it, but a correctly written `SKIP LOCKED`
worker is the harder thing to demonstrate, and that already exists. Generating
protobuf is comparatively easy to copy.

Splitting out the verticals sits late because it is a hard-to-reverse refactor
touching 111 references in the composition root. It should happen after the core is
strong.
