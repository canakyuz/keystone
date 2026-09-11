# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the version is below 1.0.0, a minor release may change the HTTP API, the gRPC
contract or the database schema in ways that are not backward compatible. When it does,
the entry says so under **Changed**.

## [Unreleased]

### Added

- `console/`, a web console over the API: sign-in, the tenant's record, member management,
  and tenant provisioning with a live view of the operation. It keeps the token in an
  httpOnly cookie on its server and holds no authorization rules of its own.

### Fixed

- `GET /api/v1/auth/me` answered 404 to every caller: it read the user without the tenant
  context the lookup needs.
- Two API validation messages were in Turkish (`POST /api/v1/tenants`, the tool catalogue
  search). The language check now reads Go string literals as well as comments, and found
  25 more in test assertions.

## [0.1.0] - 2026-09-11

The first tagged release: the control plane extracted from an earlier multi-tenant API,
with its isolation claims tested against a non-superuser database role.

### Added

- Durable tenant provisioning. `POST /api/v1/tenants` answers 202 with an operation
  resource, and the tenant, operation, job and idempotency key are written in one
  transaction.
- A worker process that claims jobs with `FOR UPDATE SKIP LOCKED`, holds them under a
  lease with a fencing token, and shuts down gracefully.
- Idempotent creation, enforced by a uniqueness constraint.
- A transactional outbox with signed, retried webhook delivery and a dead-letter state.
- An append-only audit log, written in the transaction that makes the change.
- A two-tier cache with singleflight and negative caching for tenant schema resolution.
- A Redis token bucket rate limiter with plan-aware quotas.
- Prometheus metrics with bounded labels; OpenTelemetry tracing, with the provisioning
  span linked to the request that caused it; a request id carried into the worker.
- A gRPC surface that shares authentication and repositories with the REST API.
- Membership and role checks against the tenant's own record on every request, on both
  transports (`internal/authz`).
- `keystone_worker`, a database role holding only what the worker does.
- Liveness and readiness probes.
- End-to-end isolation tests from the HTTP request to the database, a test of the import
  graph between the control plane and the examples, a test that every exported symbol in
  the control plane is documented, and a CI check of comment language.
- Eight architecture decision records, an invariants document and a roadmap.

### Changed

- The vertical business modules moved to `examples/verticals` as a reference
  application. The control plane does not import them.
- Registry payload fields are `json.RawMessage`. An empty payload is now left out of the
  response instead of being written as `null`.
- Operations that act across tenants — listing and counting tenants, looking one up by
  slug, suspending, reactivating and changing a plan — answer 403 until a platform
  permission model exists.
- The worker requires `WORKER_DB_USER` in production.

### Fixed

- Tenant isolation. Row level security is forced on every tenant table and fails closed;
  the `users` read policy that admitted every row is gone; the tenant is taken only from
  the verified token; `app.current_tenant` is set where tenant queries run.
- The tenant endpoints acted on whatever tenant id the path named.
- No route checked a role, and a token outlived the membership it was issued for.
- Provisioning and the operation lookup failed under the non-superuser role.
- The worker found no work under any role but a superuser.
- Webhook delivery could reach internal addresses.
- The plan rate limit never applied; every tenant was held to the anonymous quota.
- The provisioning routes were registered ahead of the global middleware.
- Login logged request details and returned raw errors to the caller.

### Removed

- The unused `pkg/errors` package.
- Planning documents that described an abandoned architecture.

[Unreleased]: https://github.com/canakyuz/keystone/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/canakyuz/keystone/releases/tag/v0.1.0
