# Seed data

Sample data for development and staging. It is not reference data and it never
runs in production.

Registry reference data — the module and tool catalogue — is a migration (023),
not a seed, because the application needs it in every environment.

## Running it

```bash
make seed-dev
```

Or directly:

```bash
psql -d keystone_dev -c "ALTER DATABASE keystone_dev SET app.environment = 'development';"
psql -d keystone_dev -f scripts/seed/dev_seed.sql
```

The accounts it creates use the password `DevPass123!` and live on
`dev.keystone.local` / `dev.keystone.dev` addresses. They exist only to have
something to log in as; do not reuse them anywhere real.

## The production guard

Every seed script refuses to run against production. Two independent checks,
because either one alone can be wrong:

```sql
-- 1. the environment setting
v_environment := current_setting('app.environment', TRUE);
IF v_environment = 'production' THEN
    RAISE EXCEPTION 'SEED GUARD: cannot seed production';
END IF;

-- 2. the database name
IF current_database() LIKE '%prod%' THEN
    RAISE EXCEPTION 'SEED GUARD: production database detected';
END IF;
```

The setting can be forgotten on a freshly restored database; the name check
catches that case. The name check can be defeated by a database that is not named
`*prod*`; the setting catches that one.

## Writing a seed script

**Idempotent.** Running it twice must be safe.

```sql
ON CONFLICT (email, tenant_id) DO NOTHING
```

**Fixed UUIDs.** Tests and manual checks need to be able to refer to a known row.

```sql
v_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000'::UUID;
```

**No real credentials.** No personal email addresses, and no password you use
anywhere else. This is a public repository.

## Seed versus migration

| | Migration | Seed |
|---|---|---|
| Purpose | Schema change | Sample data |
| Environments | All | Development and staging only |
| Idempotent | Required | Required |
| Rollback | A `.down.sql` is mandatory | Not needed |
