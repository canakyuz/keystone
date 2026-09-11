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
| The service contract is defined | `buf lint` plus a `grpcurl` example | done |

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
| Verticals under `internal/domain` | 4 | The rest moved to examples/verticals |

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

## Done: Tracing

`pkg/tracing` installs OpenTelemetry with W3C propagation, off unless an endpoint is
configured. Server spans cover every request; the worker's span is linked to the request
that created the job, carried as a traceparent on the operation row (migration 034).
`request_id` travels the same way (migration 033), so the two processes join in a log
search as well as in a trace.

Two things came out of this phase that the original plan did not call for:

**Metrics are not labelled by `tenant_id`.** The plan said to label them that way. A
series exists per distinct combination of label values, so a tenant label makes the
series count grow with the customer count — silently, because the metric keeps working
while the storage bill grows. Per-tenant questions belong in traces and logs.
`pkg/metrics` carries the reasoning and a test asserts the property.

**The worker's span is a link, not a child.** The plan said the provisioning step should
"join the trace of the request that created it". Joining it would keep that trace open
until the job finally succeeds — a trace completes only when all its spans do — and every
retry would hang off a request that ended hours earlier. A link carries the causal
relationship without the lifetime problem.

Moving the provisioning routes behind the global middleware was part of this work and is
recorded in [SECURITY.md](../SECURITY.md): registered ahead of it, as they were, the
endpoint that creates tenants had no rate limit, no metrics and no request log.

**Still open:** spans for the repository and for SQL. Those need the database driver
wrapped, which is a different kind of change from the rest of this phase.

---

## Done: gRPC and protobuf contracts

`proto/keystone/v1` defines OperationService and UserService; `buf` lints them and
generates the Go code, and CI regenerates and diffs it so the committed output cannot
drift from the definitions. The server runs alongside Fiber on its own port, with an
interceptor chain in the same order as the HTTP middleware and for the same reasons:
recovery outermost, observability next so a rejected call is still counted, authentication
innermost so everything below it can assume a verified tenant.

The verification itself moved to `pkg/authn`, shared by the middleware and the
interceptor. The roadmap asked for logic shared rather than copied, and this is the part
where it matters: the escalation this repository already closed once lived exactly here.

The finishing condition was that the isolation tests pass over gRPC, not that the codegen
works. `test/e2e/grpc_isolation_test.go` asserts the same guarantees on this surface,
including that caller-supplied metadata cannot override the verified token.

**Not done:** `identity` and `entitlement` services. The original plan listed three
services; two exist. UserService is there because it is what the isolation tests drive —
a contract that only proves its own codegen works says nothing about whether the
guarantee survives a second transport. The other two would be surface without a
corresponding guarantee to test, so they wait until there is something behind them.

---

## Done: Focus, split out the vertical modules

The blog, booking, lesson, payment, project, service and website modules moved to
`examples/verticals`, taking 77 of the 165 Go files with them. What is left in `internal/`
is the control plane.

The composition root split with them. `internal/app` builds the control plane and exposes
an `Extension`; `examples/verticals` has its own `Build` and `Register`; `cmd/server` is
the only file that names both. Handing the middleware chain over rather than letting an
extension assemble its own is deliberate: a chain built independently could put
authentication after the tenant context and reintroduce the escalation this repository
already closed once.

The finishing condition was that the core comes up without the verticals. Rather than
demonstrate it once, `test/architecture` walks the import graph and fails if
`internal/...` ever reaches `examples/`, transitively. A second test asserts the arrow
points the other way, because otherwise the first would also pass if the two halves had
been separated by duplicating the control plane instead of depending on it.

---

## Phase 4: Remaining invariants

**Effort:** 1 week
**Why:** Rules 7 and 8 in [INVARIANTS.md](INVARIANTS.md) were marked "not yet",
and rule 1 "partial". A document that names its own gaps is good; leaving them open
forever is not.

**Work**

1. Audit table, written in the same transaction as the job status (rule 7). **Done.**
2. Transactional outbox and webhook delivery (rule 8). The event is written in
   the same transaction as the business data; delivery is a separate, retryable
   step with exponential backoff and full jitter, and a dead-letter state after
   the maximum number of attempts. **Done.**
3. A membership check, so that "is this subject a member of this tenant" becomes
   a real check (the gap in rule 1). **Done**, against the existing user record
   rather than a new table; see [ADR-0008](decisions/0008-membership-is-the-tenant-user-record.md).
4. A separate, narrowly privileged database role for the worker — the closing
   condition of ADR-0001. **Done**; see migration 039.

---

## Phase 5: Polish

**Effort:** 2-3 days

**Standing rule:** every new code comment, commit message and document is written
in English.

**Work**

1. A godoc comment on every exported symbol. **Done** for the control plane (`internal/`,
   `pkg/`, `cmd/`, `test/helpers`), and held there by `test/architecture/godoc_test.go`.
   The reference application under `examples/` is deliberately left out. Most of its 259
   undocumented symbols are handlers and repository methods whose names already say what
   they do, and a comment that repeats a name is noise rather than documentation.
2. An architecture diagram. **Done**; see the Architecture section of the README.
3. A CHANGELOG and SemVer tagging. **Done**; see [CHANGELOG.md](../CHANGELOG.md), starting
   at `v0.1.0`.
4. `interface{}` cleanup: turn the DTO usages into concrete types. **Done.** Registry
   payloads are `json.RawMessage` and statistics are `map[string]int64`. What remains is
   genuinely open-ended — JSONB metadata, SQL argument lists, logger fields — and is
   spelled `any`.

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
