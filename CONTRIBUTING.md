# Contributing

## Development environment

Requirements: Go 1.24+, PostgreSQL 16+, Docker (optional).

```
git clone https://github.com/canakyuz/keystone.git
cd keystone
cp .env.example .env
go build ./...
```

## Tests

Tests run against a real PostgreSQL. No mock database is used; most of the bugs
in this repository only show up against a real one.

```
go test ./...
```

If Postgres is unreachable the tests that need a database are skipped rather than
failing. Connection settings can be overridden with environment variables:
`TEST_DB_HOST`, `TEST_DB_PORT`, `TEST_DB_USER`, `TEST_DB_PASSWORD`,
`TEST_DB_NAME`.

Each test creates its own isolated database and drops it afterwards, so running
them in parallel is safe.

## Schema changes

The single source of truth for the schema is the `migrations/` directory. The
test helper runs those files directly. Do not copy the schema anywhere else; this
repository once drifted unnoticed for exactly that reason.

Migrations must be backward compatible. Stop writing to a column before you drop
it, and drop it in a separate release.

## Writing RLS policies

Policies must be fail-closed. With no tenant context set, the result must be the
empty set — neither an error nor every row.

```sql
-- correct
USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)

-- wrong: falls back to TRUE when there is no context
USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID, tenant_id))
```

Permissive policies combine with OR. Adding `USING (TRUE)` to a table neutralises
every other isolation policy on that table.

## Commit format

Conventional commits, single line.

```
feat(tenant): add schema provisioning
fix(rls): force row level security on tenant tables
```

## Changelog

A change that someone using the API, the worker or the schema would notice goes under
**Unreleased** in [CHANGELOG.md](CHANGELOG.md), in the section that fits: Added, Changed,
Fixed or Removed. A release moves those entries under a version number and tags that commit
`vMAJOR.MINOR.PATCH`. Below 1.0.0 a minor release may break compatibility, and when it does
the entry says so.

## Before submitting

```
gofmt -l .
go vet ./...
go test ./...
```
