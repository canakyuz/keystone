# Architecture Decision Records

Every record answers one question: why is this design the way it is, what was the
alternative, and under which condition does it become wrong.

That last question is deliberate. Writing down when a decision will stop holding
is the proof that you actually understood it. A decision record claiming to be
right forever is not a decision record, it is a defence brief.

| No | Decision | Note |
|---|---|---|
| [0001](0001-worker-accesses-the-database-directly.md) | The worker accesses the database directly | A deliberate deviation from the target architecture |
| [0002](0002-job-claiming-with-skip-locked.md) | Job claiming via `FOR UPDATE SKIP LOCKED` | |
| [0003](0003-lease-and-fencing.md) | Claims are protected by a lease and a fencing token | |
| [0004](0004-idempotency-uniqueness-constraint.md) | Idempotency is enforced by a uniqueness constraint | |
| [0005](0005-schema-per-tenant.md) | Schema-per-tenant, together with RLS | |
| [0006](0006-two-tier-cache.md) | The cache is two-tier and singleflight-protected | |
| [0007](0007-rate-limit-fail-open.md) | The rate limit is shared and stays open when Redis is down | Debatable; the reasoning is written out |

## The order to read these in

For someone looking at the system for the first time: 0005, 0003, 0002, then the
rest.

0005 draws what tenant isolation means here and what it does not cover. 0003 sets
up the basis for recovering from failure. 0002 explains the concurrency mechanism
underneath it.

## A theme that repeats across the records

Every record has a section on what it does **not** guarantee. Those are not
filler.

- Fencing protects metadata, it does not prevent external side effects.
- Schema separation gives logical separation, not resource isolation.
- The idempotency guarantee is not indefinite.
- In a two-tier cache, cross-process consistency is not immediate.

Knowing what a system does not guarantee is harder, and more telling, than
knowing what it does.

## See also

For the promises the system makes, along with the code and tests behind them, see
[../INVARIANTS.md](../INVARIANTS.md).
