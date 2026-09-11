# Security

## Reporting a vulnerability

If you find a security issue, please do not open a public issue.
Contact: canakyuz@wesan.co

## How tenant isolation is enforced

Keystone uses two layers of isolation.

1. **Schema-per-tenant.** Every tenant has its own PostgreSQL schema.
   `TenantConnectionManager` checks out a single connection from the pool, sets
   `search_path` to the tenant schema on that connection, and hands **that same
   connection** to the callback.
2. **Row Level Security.** Tenant-owned tables in the shared `public` schema are
   protected by RLS policies. The policies read the `app.current_tenant` session
   variable.

## Operating requirements

Isolation is not guaranteed unless these hold.

**The application connection must not be a superuser.** In PostgreSQL,
superusers bypass RLS policies under all circumstances. `FORCE ROW LEVEL
SECURITY` does not change that. Create a separate, non-superuser role for the
application.

**The worker must connect as its own role.** Grant `keystone_worker` (migration 039)
to a login role and set `WORKER_DB_USER`. That role can run jobs, create tenant schemas
and deliver notifications, and it cannot read tenant data or change the audit trail. In
production the worker will not start without it.

**`ENVIRONMENT` must be production.** While it is `development`, the tenant can
be selected with the `X-Tenant-ID` header or the `tenant_id` query parameter.
That is a local development convenience only. Left on in production, any
authenticated user can reach another tenant's data.

**Every query inside a tenant context must use the connection the callback
provides.** Running a query through the pool (`*sql.DB`) inside
`ExecuteInTenantContext` silently falls onto a different connection, and that
connection's `search_path` is not set to this tenant.

## Outbound requests

Webhook delivery sends a request to a URL a tenant chose, from inside a network the
tenant is not in. That is the shape of every SSRF, and the request is the damage even
when the response never comes back.

Three defences, because each one alone has a hole.

| Where | What it catches | Why it is not enough alone |
|---|---|---|
| Registration (`outbound.ValidateURL`) | Plain http, credentials in the URL, a literal internal address | A hostname's meaning can change after it is registered |
| Dial (`outbound.NewClient`) | The address the connection actually resolves to, every answer, not just the first | — |
| Redirects | A public URL answering 302 with an internal `Location` | — |

The dial-time check is the one that holds, because DNS rebinding defeats anything that
inspects the URL. A name that resolves to a public address when it is registered can
resolve to `169.254.169.254` by the time a delivery goes out, and no amount of parsing
sees that. The dialler resolves, checks every answer, and then connects **by address**, so
the connection cannot be re-resolved to somewhere else between the check and the socket.

The blocklist covers loopback, RFC 1918, link-local including the metadata range,
carrier-grade NAT, multicast, the IPv6 equivalents, and IPv4 addresses written in IPv6
form — `::ffff:127.0.0.1` walks straight past an IPv4-only check.

Migration 037 also constrains stored destinations to https at the database level. That is
a last line rather than the defence: it catches a row inserted by something that forgot to
call the validator, and it cannot see anything about where the name points.

## Known and deliberate design decisions

The following are intentional behaviour, not holes.

- `websites.public_websites_policy` makes published sites readable across the
  tenant boundary. This is the intended behaviour for public CMS content.
- `projects.public_projects_policy` opens completed and featured projects the
  same way.
- `users.auth_lookup_policy` allows the login lookup that happens before the
  tenant is known. It applies only while the `app.auth_lookup` session flag is
  set, and the repository confines that flag to a single transaction with
  `SET LOCAL`.

## Vulnerabilities closed

This repository keeps a record of the bugs that surfaced once tests were written
to verify the isolation claim. The details live in the comments at the top of the
relevant migration files.

| Problem | Impact | Fix |
|---|---|---|
| `USING (TRUE)` policy on `users` | Read isolation was entirely absent; every tenant could read all users, password hashes included | `028` |
| `ENABLE` without `FORCE` on 17 tables | The application, connecting as the table owner role, was exempt from the policies | `027` |
| `sites` policy made fail-open by `COALESCE` | All rows were visible when the context was not set | `030` |
| `current_setting` without the missing-ok argument in `payments`, `refunds`, `payment_events` policies | Hard error on a session with no context set | `030` |
| `ExecuteInTenantContext` did not pass the connection to the callback | `search_path` was not applied to the connection the queries actually ran on | `pkg/database` |
| `extractTenantID` read the wrong context key | The JWT tenant claim was never used; anyone holding a valid token could switch tenants with the `X-Tenant-ID` header | `internal/middleware` |
| `ExecuteInTenantContext` never set `app.current_tenant` | Under the non-superuser role this file requires, every RLS-protected read returned the empty set. The application worked in development only because it connected as a superuser, which bypasses RLS | `pkg/database` |
| The rate limiter ran before authentication | It read the tenant from a value the authenticator had not written yet, so the branch was never taken. Every authenticated tenant was held to the anonymous quota of 30 requests a minute regardless of the plan it paid for, and the plan table was dead code | `internal/middleware` |
| Login logged the request email and returned the raw service error | The address went to stderr on every attempt, in an unstructured stream nothing rotates or redacts, and a repository failure would have described the schema to the caller | `internal/handler/auth` |
| The provisioning endpoints were registered ahead of the global middleware | In Fiber that means the middleware never runs for them, so the endpoint that creates tenants had no rate limit, no metrics and no request log | `internal/app` |
| Webhook delivery connected to any address a tenant supplied | Server-side request forgery: a registered destination of `http://169.254.169.254/` reaches the cloud metadata credentials, and any internal service is reachable from inside the perimeter. Not exploitable at the time it was found — there is no endpoint for registering a destination yet — but the delivery code that will serve one was unguarded | `pkg/outbound` |
| Provisioning wrote to `operations` without tenant context, and the status lookup read it without any | Under the non-superuser role this file requires, the tenant policy refused the insert, so `POST /tenants` failed on every request, and every status poll and idempotent retry answered 404. Development connected as a superuser and saw neither. The lookup is now visible to the subject that created the operation and to no one else | `038` |
| The tenant endpoints acted on the tenant id in the path | The owner of any tenant could read, change or delete any other tenant, and suspend it, by putting its id in the URL | `internal/middleware`, `SameTenant` |
| No route checked a role | A viewer could grant roles, including owner, and create or delete users in its tenant | `internal/app/routes.go` |
| The token was trusted after the membership behind it ended | A suspended or deleted user kept access for the rest of the token's lifetime, up to 24 hours, and a demoted administrator kept administering | `internal/authz` |
| Routes that act across tenants had no guard | Any authenticated caller could list every tenant, read counts across all of them, and change any tenant's plan or status | `internal/middleware`, `PlatformOnly` |
| The worker had no role of its own and worked only as a superuser | Its claims read queue tables that carry the tenant policy, so under any other role it found no work. The process that delivers webhooks to addresses tenants supply therefore held a credential that ignores every policy in the database | `039` |

## Testing it

The isolation claims are verified at two levels, both against real PostgreSQL and
both using a non-superuser role.

```
go test ./test/security/   # the RLS configuration, driven directly with SQL
go test ./test/e2e/        # the whole chain, from an HTTP request to the database
```

`test/security` answers "are the policies right?". `test/e2e` answers "does a real
request actually end up inside them?" — the step where the middleware turns a request
into a tenant schema, which the other tests do not cover.

That distinction is not academic. The `app.current_tenant` bug in the table above sat
in the code while every other test passed, because they all connected as the
superuser. Writing `test/e2e` against the non-superuser role surfaced it on the first
run.
