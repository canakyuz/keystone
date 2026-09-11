# 0008. Membership is the tenant's user record, read on every request

**Status:** Accepted, 2026-09-11

## Context

Authorization rested on the token alone. A token carries a tenant, a subject and a
role, all signed at login, and every route trusted them for the token's lifetime of
24 hours. Tests written against the production route table showed what that meant: a
suspended user kept access, a viewer could grant roles because no route checked one,
and the tenant endpoints acted on whatever tenant id the path named.

INVARIANTS rule 1 had described the gap as a missing membership table.

## Rules that must hold

- Losing a membership ends access on the next request, not when a token expires.
- The role a guard checks is the role the tenant holds for the subject now.
- The HTTP and gRPC surfaces reach the same verdict.

## Options considered

### A. A separate membership table

One identity row per person, and a `(tenant_id, user_id, role)` table joining it to
tenants. One person could belong to several tenants with one password.

Its cost is the whole authentication path. Login would return a choice of tenants,
registration would split identity from membership, and every existing user row would
be migrated into two tables.

### B. Short-lived tokens with a refresh step

Access ends when the token expires, in minutes rather than a day, and nothing is
looked up per request.

It narrows the window rather than closing it, and it leaves the role unanswered: a role
changed in the middle of a token's life is stale until the next refresh.

### C. The existing user row is the membership, read on every request

`users` already holds one row per subject per tenant, with a role and a status. A
primary-key lookup on `(id, tenant_id)` answers "is this an active member, and in which
role" directly.

## Decision

C.

## Rationale

The rule that failed was not "a person can belong to two tenants". It was "a token is
trusted after the membership behind it is gone". C closes that with the table that
already exists. A would close it too, and would spend most of its effort on a
capability nothing here uses yet.

B is a window, not a check. It does not conflict with C, and may be worth adding later
for its own reasons.

The lookup reads two columns and needs no tenant schema: `users` lives in `public`, and
only the RLS tenant context is set, transaction-local. It runs directly after
authentication and before the plan limiter, so a subject who is no longer a member
cannot spend its former tenant's quota.

The decision lives in `internal/authz`, apart from both transports, for the reason
`pkg/authn` does: two transports, one verdict.

## Consequences

- Every authenticated request costs one indexed lookup, O(log n). It is not cached. A
  cached membership is a stale membership, which is the problem being solved.
- A suspension, deletion or role change takes effect on the next request.
- The same person in two tenants is two rows with two passwords, and a login that names
  no tenant picks one of them.

## What this does not guarantee

- No platform permission exists. Routes that act across tenants are closed to everyone
  rather than granted to an operator.
- Creating a tenant requires only an active membership somewhere.

## When this decision becomes wrong

- When one person needs to act in several tenants under one identity. That is option A,
  and this record is where it starts.
- If the lookup becomes a measurable share of request latency. The answer then is a
  short-lived cache invalidated by the writes that change a membership, not trusting the
  token again.
