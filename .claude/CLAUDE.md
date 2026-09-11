# Keystone - working rules

> Multi-tenant SaaS control plane (Go + Fiber + PostgreSQL)
> Architecture: clean architecture + schema-per-tenant

---

## Non-negotiables

- **Real PostgreSQL, not mocks.** Most bugs in this repository (RLS bypass, schema
  drift, `search_path` leaking) only appear against a real database. A test that
  passes against a mock proves nothing about isolation.
- **Test what is load-bearing.** Isolation, concurrency and the state machine get
  tests. Getters do not.
- **Write the code, then the test.** Not the other way around.
- **Commit every meaningful change.** Conventional commits, one line.

---

## Priorities, in order

1. **Tenant isolation.** Cross-tenant leakage is zero, and that claim is tested.
2. **Clean architecture.** The layer boundary is absolute.
3. **Production ready.** Every commit could ship.
4. **Performance.** P95 under 200ms — currently an unmeasured claim, see the roadmap.
5. **Security.** SOC 2 and GDPR aligned.

---

## Layers

```
Handlers (HTTP)  ->  Usecases (business logic)  ->  Domain  ->  Repository (database)
```

Dependencies point inward only. An inner layer must never import an outer one. If
the repository layer needs something from HTTP middleware, the shared piece belongs
in a leaf package — see `pkg/tenantctx`.

Two processes: `cmd/server` serves HTTP, `cmd/worker` drains the provisioning queue.
They share the management schema; see ADR-0001 for why, and what that costs.

---

## Multi-tenant rules

Every request carries a tenant identity, read only from a verified JWT claim.

```go
type RequestContext struct {
    TenantID   string `validate:"required"`
    SchemaName string `validate:"required"`
    UserID     string `validate:"required"`
}
```

**Database.** Queries inside a tenant context must run on the connection
`ExecuteInTenantContext` hands to the callback. Going through the pool (`*sql.DB`)
silently lands on a different connection whose `search_path` is not set.

```go
// Wrong: no tenant context
db.Query("SELECT * FROM users WHERE email = $1", email)

// Right: the callback's connection, with search_path already set
mgr.ExecuteInTenantContext(ctx, schema, func(conn *sql.Conn) error {
    return conn.QueryRowContext(ctx, "SELECT ... WHERE email = $1", email).Scan(&u)
})
```

**RLS policies are fail-closed.** With no context set the result is the empty set,
never every row. Adding one `USING (TRUE)` policy neutralises every other isolation
policy on that table — this repository lost read isolation on `users` exactly that
way.

**Cache keys carry the tenant prefix:** `tenant:{tenant_id}:{resource}:{id}`.

---

## Code rules

- **DRY:** factor out on the third repetition, not the second.
- **KISS:** the simplest thing that works. No speculative abstraction.
- **YAGNI:** do not build for a requirement that does not exist today.
- **Single responsibility:** one job per function.

---

## Security

**Validate input at the boundary.**

```go
type CreateTenantRequest struct {
    Name  string `validate:"required,min=3,max=100"`
    Email string `validate:"required,email"`
}
```

**Never interpolate into SQL.** Use `$1`, `$2`. Where a placeholder is impossible —
a schema name in `SET search_path` — validate the identifier by hand and quote it
with `pq.QuoteIdentifier`.

**Do not leak internals in errors.** Log the detail, return the generic message.

```go
logger.Error("db_error", zap.Error(err))
return errors.New("failed to process request")
```

---

## Performance

**No N+1.** Batch instead.

```go
// Wrong: a query per iteration
for _, user := range users {
    roles := repo.GetUserRoles(user.ID)
}

// Right: one query for all of them
rolesByUserID := repo.GetRolesByUserIDs(extractIDs(users))
```

**Indexes lead with `tenant_id`.**

```sql
-- Wrong
CREATE INDEX idx_users_email ON users(email);

-- Right
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
```

**Cache tiers:** L1 in-process, L2 Redis, database underneath. TTLs carry jitter.

---

## Commits

Conventional commits, single line, English.

```
feat(tenant): add schema provisioning
fix(rls): force row level security on tenant tables
refactor(user): extract validation logic
perf(db): optimize tenant queries
```

Before committing: tests pass, `gofmt -l .` is empty, `go vet ./...` is clean, no
credentials in the diff, tenant isolation intact.

---

## Comments

Explain WHY, not WHAT. The code already says what it does.

```go
// Useless
counter++ // increment counter

// Useful
// Exponential backoff, because the payment API rate limits under load.
// Three attempts, 8s ceiling.
```

When a comment records a bug that was fixed, describe the bug concretely. Those
comments are the most valuable ones in this repository.

---

## Anti-patterns

```go
// Hardcoded credentials
const dbPassword = "secret"

// Panicking in a library
func ProcessData(data string) {
    if data == "" {
        panic("empty data")
    }
}

// Ignored error
db.Exec("UPDATE users SET active = true")

// Global mutable state, a race waiting to happen
var currentTenantID string

// Database access in a handler
func (h *Handler) GetUser(c *fiber.Ctx) error {
    row := h.db.QueryRow("SELECT ...")
}
```

---

## Layout

```
keystone/
├── cmd/server/          # HTTP entry point
├── cmd/worker/          # provisioning worker
├── internal/
│   ├── domain/          # entities, pure
│   ├── usecase/         # business logic
│   ├── repository/      # data access
│   ├── handler/         # HTTP handlers
│   ├── middleware/      # auth, tenant context, rate limit
│   └── worker/          # job claiming, leases, shutdown
├── pkg/                 # cache, ratelimit, database, tenantctx
├── console/             # web console (Next.js); reads only through the API
├── migrations/          # the single source of truth for the schema
├── docs/decisions/      # architecture decision records
└── scripts/seed/        # development seed data only
```

---

## Definition of done

**Quality:** layer boundaries respected, no speculative abstraction, `gofmt` and
`go vet` clean.

**Security:** input validated, tenant isolation verified by a test, no sensitive
data in logs, no string-built SQL.

**Performance:** no N+1, indexes lead with `tenant_id`, caching strategy stated.

**Production ready:** errors handled, logging structured, migration backward
compatible with a tested rollback.

**Documented:** if the change alters a guarantee, `docs/INVARIANTS.md` is updated.
If it settles a design question, a decision record is added — including the
condition under which the decision becomes wrong.
