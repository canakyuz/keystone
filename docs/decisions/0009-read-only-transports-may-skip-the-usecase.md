# 0009. A read with no rule of its own may go from transport to repository

**Status:** Accepted, 2026-09-26

## Context

The layer rule reads handler, usecase, domain, repository. Six transports skipped the
usecase: the audit, operation, upload and webhook handlers, and both gRPC services. Four of
them carried rules, and two of those rules had already drifted:

- Tenant creation checked the platform permission in a REST route middleware. The gRPC
  port has no route table, so it did not check it at all, and any member of any tenant
  could create tenants over gRPC.
- REST trimmed the required provisioning fields and capped the idempotency key; gRPC did
  neither.

Those four now go through usecases (`operation`, `webhook`, `upload`). Two do not: the
audit handler and the gRPC `ListUsers`.

## Rules that must hold

- A rule that decides who may do what, or what input is acceptable, exists once, and every
  transport reaches it.
- A layer is added for a rule, not for symmetry.

## Options considered

### A. Every transport goes through a usecase

A pass-through usecase for the audit trail and for the gRPC user list: one method each,
forwarding its arguments.

The layer would hold nothing. The next reader has to open it to learn that, and the next
change to the repository signature has to be made twice.

### B. Reads with no rule of their own call the repository directly

The tenant scope and the role guard these reads depend on are not theirs: the tenant comes
from the verified claim, set by the middleware or the interceptor, and the repository runs
inside it. Pagination bounds are input parsing, which is the transport's job.

## Decision

B. A transport may call a repository directly when the operation is a read and the
transport would otherwise forward its arguments unchanged. Today that is
`internal/handler/audit` and `internal/grpc/user_service.go`.

## Consequences

- `test/architecture` does not forbid handler-to-repository imports, because two are
  allowed. Review has to catch a new one.
- The consumer-side interface in the transport names exactly the repository methods it
  uses, so the exception stays visible in one place per transport.

## What this does not guarantee

That the two excepted reads stay rule-free. Nothing mechanical notices when a filter,
a redaction or a permission is added to one of them in the transport.

## When this decision becomes wrong

When either excepted read gains a rule: a field hidden from some roles, a filter that
depends on who is asking, or a second transport serving the same read. At that point the
rule moves to a usecase, both transports call it, and the exception list above shrinks.
