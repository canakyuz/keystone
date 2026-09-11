# Invariants

This document lists the promises the system makes. Each rule is followed by the
code that enforces it and the test that verifies it. A rule with nothing behind
it does not sit here as a slogan — it is marked "not yet".

The status column takes three values. **Enforced**: code and verification exist.
**Partial**: part of it exists, and the gap is stated explicitly. **Not yet**:
not implemented.

For the reasoning behind the decisions and their alternatives, see the decision
records under [decisions/](decisions/).

---

## 1. Authorization rests on a verified subject and tenant membership

**Status:** Enforced

Whichever tenant a request is made on behalf of, that fact is read only from a
verified JWT claim. The `X-Tenant-ID` header and the `tenant_id` query parameter
are accepted only while `ENVIRONMENT=development`, and only behind an explicit
switch.

- Code: `internal/middleware/tenant_context.go`, `extractTenantID`
- Test: `internal/middleware/tenant_source_test.go`

This rule used to be violated. `extractTenantID` read the `c.Locals("user")`
key; `AuthMiddleware` never writes such a key. The JWT path never ran, and every
request silently fell through to the header. Any user carrying a valid token
could read another tenant's data.

A verified token is not enough on its own. It records what the subject was when it
logged in, and it stays valid for a day. On every authenticated request, on both
transports, the subject is looked up in the tenant's own record: it must exist in
that tenant, must not be deleted, and must be active. The role every guard checks is
the one that record holds, not the token's claim.

- Code: `internal/authz`, `VerifyMember`; `internal/middleware/authorization.go`;
  `internal/grpc/interceptors.go`, `authInterceptor`
- Test: `internal/app/authorization_test.go`, which drives the production route
  table over the non-superuser role; `test/e2e/grpc_isolation_test.go`,
  `TestGRPC_NonMember_IsRefused`

This part was violated in four ways at once. Tests written before the fix showed the
first three:

- The tenant endpoints took the tenant id from the path and never compared it with
  the caller's. The owner of one tenant could read another, and suspend it.
- No route checked a role. A viewer made another user an administrator.
- A suspended user kept full access until the token expired.
- Listing every tenant, counting them, and changing any tenant's plan or status had
  no guard at all in the route table.

**What this does not cover:** there is no platform permission model. The routes that
act across tenants are closed to everyone (`middleware.PlatformOnly`) rather than
granted to an operator, and `POST /api/v1/tenants` requires only an active membership
in some tenant. A membership is one user row per tenant, so the same person in two
tenants is two rows with two passwords. See
[decisions/0008-membership-is-the-tenant-user-record.md](decisions/0008-membership-is-the-tenant-user-record.md).

---

## 2. The same idempotency key cannot be reused with different content

**Status:** Enforced

A request arriving with the same scope and key returns the existing operation if
the body is identical. If the body differs, it returns a conflict. Silently
returning the old result would make the client believe a request it never sent
had been processed.

Uniqueness is enforced by the database. Reading first and then writing does not
prevent the race on its own; two requests arriving at once can pass the check
together.

- Code: `internal/repository/operation/postgres.go`, `Create`
- Schema: `migrations/031_create_operation_model.up.sql`, `idx_idempotency_scope_key`
- Tests: `TestCreate_SameKeySameBodyReplays`,
  `TestCreate_SameKeyDifferentBodyConflicts`,
  `TestCreate_ConcurrentSameKeyProducesOneOperation`

**Limit:** The guarantee is not indefinite. Once the key's retention window
expires, the same key creates a new operation. The default is 24 hours.

---

## 3. Only the current lease holder can record a result for a job

**Status:** Enforced

Every claim increments `fence` by one. A completion must be reported with the
current fence value. Once a lease expires and the job is handed over, a report
from the returning old worker is rejected.

Checking `lease_owner` alone is not enough: the old worker knows its own name and
would pass that check.

- Code: `internal/repository/operation/claim.go`, `completeInTx`
- Tests: `TestComplete_StaleWorkerIsRejected`, `TestRenewLease_RejectsStaleFence`

**Limit:** Fencing protects the metadata update. It does not by itself stop an
old worker from producing a side effect in an external system. The setup steps
must also be safe to repeat.

---

## 4. A tenant cannot become `active` before provisioning completes

**Status:** Enforced

The tenant record is written in the `pending` state. Becoming `active` depends on
the worker completing the provisioning steps. A tenant whose schema is not ready
cannot accept requests.

Activating the tenant and closing the operation happen in the same transaction.
Written separately, a crash in between could leave a record that looks
"completed" to the user while the tenant is unusable.

Making a tenant `active` is not the worker handler's job; activation lives in the
same transaction that marks the job done.

The intermediate states the worker writes (`provisioning`, `failed`) fall outside
fencing protection. For that reason the transitions are conditional on the source
state: an active tenant cannot be pulled back to `provisioning`. A late call from
a worker that lost its lease quietly has no effect.

- Code: `internal/repository/operation/claim.go`, `CompleteSuccess`
- Code: `internal/repository/operation/provision.go`, `insertPendingTenant`
- Code: `internal/repository/tenant/lifecycle.go`, guarded transitions
- Code: `internal/worker/provision_handler.go`, `Handle`
- Schema: `migrations/032_extend_tenant_lifecycle_states.up.sql`
- Test: `TestCompleteSuccess_ActivatesTenantInSameTransaction`

---

## 5. Without tenant context, access to tenant data is denied

**Status:** Enforced

Row Level Security policies are fail-closed. If the tenant context is not set,
the result is the empty set: neither an error nor every row.

- Schema: `migrations/030_harden_tenant_isolation_policies.up.sql`
- Schema: `migrations/027_force_row_level_security.up.sql`
- Code: `internal/repository/user/postgres.go`, reading the schema from context
- Code: `pkg/database/tenant_connection_manager.go`, `ExecuteInTenantContext`
- Tests: `test/security/rls_test.go`, `test/e2e/tenant_isolation_test.go`

This rule used to be violated in four separate ways. The `USING (TRUE)` policy on
the `users` table removed read isolation entirely. Seventeen tables were missing
`FORCE`, so the table owner role was exempt from the policies. The `sites` policy was
fail-open: with no context set, every row was visible.

The fourth was the inverse failure, and it hid behind the test setup.
`ExecuteInTenantContext` set `search_path` but never `app.current_tenant`, so under
the non-superuser role this rule requires, the policies matched nothing and every
read returned the empty set. It went unnoticed because the repository tests connect
as the superuser, and superusers bypass RLS. `test/e2e`, which connects as the
application role, surfaced it immediately.

**Operating requirement:** The application must connect with a non-superuser
role. Superusers bypass RLS under all circumstances.

---

## 6. One tenant's load cannot consume the entire worker capacity

**Status:** Enforced

The worker applies a concurrency limit both in total and per tenant. With only a
total limit, a tenant with a lot of work could fill every slot and leave the
others waiting.

- Code: `internal/worker/provisioner.go`, `reserveTenant`
- Tests: `TestRun_RespectsMaxConcurrent`, `TestRun_RespectsPerTenantLimit`

**Limit:** Right now this is a simple per-tenant counter. If a single tenant
filling the queue starts delaying small tenants, fair scheduling is needed; that
does not exist yet.

---

## 7. A successful job status and its audit record are kept in the same transaction

**Status:** Enforced

Completing a provisioning writes the audit entry into the same transaction that marks the
job done and activates the tenant. If the entry cannot be written, none of it stands.

A trail that can disagree with the data is worse than no trail, because somebody will
eventually trust it. Written after the change, a crash in between leaves an activation
nobody can account for; written before, a rollback leaves a record of an activation that
never happened.

The actor is recorded as the system rather than as the user who originally requested the
provisioning. A worker completing a job hours later, possibly after several retries, is
not that user acting, and saying so would be a lie about who performed the transition.

The table is append-only, and that is enforced by revoking UPDATE and DELETE rather than
by nobody writing the code. A trail the audited system can edit answers no question worth
asking.

- Schema: `migrations/036_create_audit_log.up.sql`
- Code: `internal/repository/audit/postgres.go`, `AppendTx`
- Code: `internal/repository/operation/claim.go`, `appendAudit`
- Tests: `TestCompleteSuccess_WritesTheAuditEntryInTheSameTransaction`,
  `TestCompleteSuccess_AuditFailureRollsBackTheActivation`

**Limit:** only the provisioning path writes to it so far. Tenant suspension, plan
changes and user role changes all belong in the trail and are not there yet.

---

## 8. An external webhook failure does not roll back a completed provisioning

**Status:** Enforced

Completing a provisioning writes the notification into the same transaction that
activates the tenant, and delivers nothing. Either both commit or neither does. Delivery
is a separate loop reading committed rows, so a receiver that is slow, down or angry
cannot touch the transaction that produced the event.

The obvious alternative fails in both directions. Calling the endpoint inside the
transaction holds it open for as long as the receiver takes and rolls back completed work
when the receiver is down. Calling it after the commit loses the notification whenever the
process dies in between.

- Schema: `migrations/035_create_outbox.up.sql`
- Code: `internal/repository/operation/claim.go`, `CompleteSuccess`
- Code: `internal/repository/outbox/postgres.go`, `AppendTx`
- Code: `internal/worker/outbox.go`
- Tests: `TestCompleteSuccess_EmitsTheEventInTheSameTransaction`,
  `TestAppendTx_IsRolledBackWithItsTransaction`,
  `TestClaim_ConcurrentWorkersNeverShareAnEvent`,
  `TestMarkFailed_RetriesUntilExhaustedThenDies`

**Limit:** delivery is at-least-once, not exactly-once, which is not available over HTTP.
A receiver that accepts a request and fails before answering will be sent it again. Each
delivery carries a stable `Idempotency-Key` — the event id, unchanged across attempts —
so the receiver can recognise the repeat. Making use of it is the receiver's half of the
contract.

**Limit:** an endpoint that refuses every attempt goes to a dead-letter state rather than
being retried forever. Getting it out requires somebody to look at why.

---

## Diagnosability

Not a rule either, but a promise the design makes: a provisioning can be followed from
the request that asked for it to the migration step that failed.

The chain is `request_id -> operation_id -> job_id -> attempt -> worker -> step`, and
every link is written. `request_id` is the correlation id the HTTP middleware assigns,
stored on the operation and read back when a worker claims the job, so the two processes
can be joined in a log search rather than by timestamp.

- Schema: `migrations/033_add_operation_request_id.up.sql`
- Code: `internal/handler/operation/handler.go`, `requestID`
- Code: `internal/worker/provision_handler.go`, `logStep`
- Tests: `TestClaim_CarriesTheRequestID`, `TestClaim_ToleratesAMissingRequestID`

**Limit:** operations created before the column existed have no request id, and the
field is omitted rather than emitted empty.

---

## Shutdown behaviour

Not a rule, but a promise of the same class: while shutting down, the worker
first stops taking new work, gives running jobs a bounded amount of time, and
cancels them when that time runs out. Jobs that cannot finish are taken over by
another worker once the lease expires.

Context cancellation does not undo side effects that have already been committed.
Cancellation means "try to stop", not "erase what was done".

- Code: `internal/worker/provisioner.go`, `drain`
- Tests: `TestRun_GracefulShutdownWaitsForRunningJobs`,
  `TestRun_ShutdownGraceExpiryCancelsJobs`

---

## Promises deliberately not made

These are intentionally not guaranteed.

**There is no "runs at most once" guarantee.** A job can run again. A job whose
lease expires passes to another worker and the steps run from the start. Handlers
must be safe to repeat.

**Cross-process cache consistency is not immediate.** The L1 tier is
process-local. Invalidating an entry takes immediate effect only in that process;
other replicas may see the stale value until their own L1 TTL expires.

**The rate limit is not enforced when Redis is down.** The default behaviour is to
let the request through. The limiter is an abuse brake, not an availability tool.
