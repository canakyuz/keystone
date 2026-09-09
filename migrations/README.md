# Migrations

This directory is the single source of truth for the database schema. Nothing
else holds a copy — the test helper runs these files directly, because this
repository once drifted unnoticed when a copy existed.

## Naming

```
XXX_description.up.sql    # forward
XXX_description.down.sql  # rollback
```

Every forward migration needs a rollback, and the rollback should be tested
before the pair is merged.

## Running them

```bash
make migrate-up        # apply everything pending
make migrate-down      # roll back the last one
make migrate-status    # list the migration files
make db-reset          # drop the dev database, recreate, migrate
```

The runner is stateful and tracks applied versions in `schema_migrations`.

## What is in here

| Range | Contents |
|---|---|
| 001-005 | Core entities: tenants, users, websites, projects, students |
| 006-010 | Business entities: lessons, assignments, appointments, services |
| 011-013 | Content: blog, sites |
| 014-016 | Payments: payments, refunds, events |
| 017-022 | Registry: module and tool catalogue, dependencies, activations |
| 023 | Registry seed data (modules and tools, all environments) |
| 025-026 | Tenant schema support, CMS tables |
| 027-030 | Tenant isolation hardening — see below |
| 031-032 | Durable operation model, extended tenant lifecycle states |

### The isolation hardening range, 027-030

These four are worth reading before the others. They are the fixes for the bugs
that surfaced once tests were written to actually verify isolation, and each file
carries the reasoning in a comment at the top.

- **027** adds `FORCE ROW LEVEL SECURITY` to 17 tables. Without it the table owner
  role — which the application was connecting as — is exempt from every policy.
- **028** scopes the `users` auth policy. It previously read `USING (TRUE)`, and
  because PostgreSQL combines permissive policies with OR, that removed read
  isolation from the `users` table entirely.
- **030** makes the remaining policies fail-closed. `sites` was fail-open through a
  `COALESCE`, and three payment policies called `current_setting` without the
  missing-ok argument, so a session with no context raised a hard error instead of
  returning nothing.

Full write-up: [SECURITY.md](../SECURITY.md).

## Rules for a new migration

**Backward compatible.** Stop writing to a column before you drop it, and drop it
in a separate release. A migration and the code that depends on it do not deploy
atomically.

**Tenant-scoped tables need all of this:**

- `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE`
- an index with `tenant_id` first in any composite index
- an RLS policy, and `ENABLE` *plus* `FORCE`
- the policy must be fail-closed:

```sql
-- correct: empty set when there is no context
USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)

-- wrong: falls back to TRUE when there is no context
USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID, tenant_id))
```

Adding a single `USING (TRUE)` policy to a table neutralises every other isolation
policy on it.

**No credentials.** Use `REPLACE_WITH_*` placeholders and document what the real
value should be in a comment.

## Seed data

Migration 023 seeds the registry catalogue and runs in every environment; it is
reference data, not sample data.

Development sample data lives outside the migration chain, in
`scripts/seed/dev_seed.sql`, and runs with `make seed-dev`. Migration 024 used to
seed a test tenant and was removed — the production migration chain no longer
inserts sample data.

Any seed that must stay out of production carries a guard:

```sql
IF COALESCE(current_setting('app.environment', TRUE), 'production') = 'production' THEN
    RAISE EXCEPTION 'BLOCKED: seed data cannot run in production';
END IF;
```

## Troubleshooting

A migration that failed part-way leaves `schema_migrations` behind the files.
Inspect it directly:

```bash
make db-shell
SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;
```

`make migrate-bootstrap` reconciles an existing database with the
`schema_migrations` table, for the case where the schema was created before the
runner existed.
