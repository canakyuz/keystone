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

**Status:** Partial

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

**Gap:** There is no membership table yet. The tenant identity comes from the
JWT, but the question "is this subject a member of this tenant" is not verified
against a separate record. The same person being an administrator in one tenant
and a viewer in another cannot be represented until that table exists.

**Gap:** `POST /api/v1/tenants` is protected by authentication alone. Creating a
tenant should be a platform-level permission, but no such permission model
exists yet. This endpoint does not look up tenant membership, and cannot: the
tenant does not exist yet.

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
- Tests: `test/security/rls_test.go`

This rule used to be violated in three separate ways. The `USING (TRUE)` policy
on the `users` table removed read isolation entirely. Seventeen tables were
missing `FORCE`, so the table owner role was exempt from the policies. The
`sites` policy was fail-open: with no context set, every row was visible.

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

**Status:** Not yet

The audit table has not been created. Who made which change is currently held
only in the `operations.created_by` field; there is no separate audit trail.

---

## 8. An external webhook failure does not roll back a completed provisioning

**Status:** Not yet

Webhook delivery is not implemented. This rule will be met by a transactional
outbox once delivery is added: the event is written in the same transaction as
the business data, and delivery becomes a separate, retryable step.

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
