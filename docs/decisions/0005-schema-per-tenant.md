# 0005. Schema-per-tenant, together with RLS

**Status:** Accepted, 2026-09-07

## Context

Every customer's data has to be kept apart from the others'. There are three
common approaches, and all three guarantee different things.

## Rules that must hold

- One tenant cannot read another tenant's data.
- If the tenant context is missing, access is denied rather than opened.
- Migrations must be applicable to every tenant.

## Options considered

| Approach | Strength | Operational cost |
|---|---|---|
| Shared table + `tenant_id` + RLS | One schema to migrate, cross-tenant querying is easy | Policy, role and query correctness are critical |
| Schema per tenant | Logical separation is clear | Schema count and migration management grow |
| Database per tenant | Separate backup and resource management | Connection and infrastructure cost rises |

## Decision

Schema per tenant, plus RLS on the shared tables.

Management data (tenants, operations, provisioning_jobs) stays in the `public`
schema and is protected by RLS. Tenant business data goes into its own schema.

## Rationale

The two solve different problems and do not substitute for each other.

Schema separation splits the migration and backup unit for tenant business data.
RLS draws a row-level boundary on the shared management tables; those tables
cannot be split by schema because the control plane queries all of them at once.

## What this decision does NOT guarantee

These matter because they draw the boundary of the word "isolated".

**There is no resource isolation.** Separate schemas in the same PostgreSQL
instance provide no CPU or I/O isolation. One tenant's heavy query slows the
others down.

**Schema separation alone is not a security boundary.** If the same application
role can reach every schema, the separation is only a namespace separation.
Security then rests on the application pointing at the right schema.

**RLS does not stop a superuser.** The application must connect with a
non-superuser role. `FORCE ROW LEVEL SECURITY` covers the table owner, not the
superuser.

## Points that still need verifying

The proof of this approach is not two successful API calls made in sequence.

| Question | Status |
|---|---|
| Is the tenant id supplied by the client checked against membership | Partial, no membership table |
| Do tenant queries run in the same connection context | Yes, `ExecuteInTenantContext` hands the connection to the callback |
| Can `search_path` leak to another request through the pool | No, this is tested |
| Does a missing table cause a fallback to the `public` schema | Not tested |
| Is the schema name generated safely | Yes, `pq.QuoteIdentifier` plus name validation |
| Are the runtime role and the schema-creating role separate | No, not yet separated |
| How does a migration propagate to every tenant | Unsolved |

## When this decision becomes wrong

- If the tenant count passes the thousands. Per-schema migration cost becomes
  T×M steps; raising concurrency does not reduce the total work, it only trades
  completion time against database load.
- If a customer demands resource isolation. At that point a database per tenant,
  or a separate instance, is required.
- If cross-tenant reporting becomes necessary. Writing queries across schemas is
  markedly more awkward than with the shared-table approach.
