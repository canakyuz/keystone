# 0001. The worker accesses the database directly

**Status:** Accepted, 2026-09-07
**Note:** This decision is a deliberate deviation from the target architecture.

## Context

Provisioning jobs take a long time, need different database privileges, and
require a concurrency limit independent of user requests. For those three reasons
the worker has to be a separate process from the Control API.

Being a separate process raises the question of how the worker reaches the job
queue.

The target architecture suggests the worker should not touch the management
tables directly, and should instead talk to the Control API over gRPC to claim
work and report results.

## Rules that must hold

- Only the current lease holder can record a result for a job.
- Two workers cannot run the same job at the same time.
- If a process dies mid-job, the job must be reclaimable.

## Options considered

### A. The worker accesses the database directly

Claiming a job, renewing the lease and reporting a result are each a single SQL
statement.

Strength: claim atomicity falls to PostgreSQL's own locking. `FOR UPDATE SKIP
LOCKED` solves it in one step. The queue keeps draining even if the Control API
is down.

Weakness: the worker becomes coupled to the shape of the management schema. A
schema change affects two components at once. The worker's database privileges
approach the API's, so the benefit of a separate privilege boundary weakens.

### B. The worker talks to the Control API over gRPC

Claiming, renewing and reporting each become an RPC.

Strength: access to the management tables is concentrated in one place. The
worker's database privileges can be limited to tenant schemas only. The contract
becomes explicit and versionable.

Weakness: the worker now depends on the API's availability. If the API is down,
the queue stops. Claim atomicity also moves behind a network call; when an RPC
response is lost, the worker cannot tell whether it took the job, and that
ambiguity has to be solved separately.

## Decision

A, for now. The worker accesses the database directly.

## Rationale

Claiming is the most critical atomicity point in this system. Putting it behind a
network call reopens a solved problem: if the RPC response is lost the worker
cannot tell whether it claimed the job, and solving that requires a second
idempotency mechanism at the gRPC layer.

B's real gain is privilege separation. But that gain can also be had by defining
a separate database role for the worker, which is cheaper. That has not been done
yet.

## Consequences

- The worker and the Control API are bound to the same management schema. A
  schema change concerns both.
- If they are split into separate repositories, those two repositories have to
  share a schema version. Not sharing business logic through a common package is
  not enough; the schema version is a dependency too.
- Privilege separation does not exist yet. The worker and the API currently
  connect with the same role.

## When this decision becomes wrong

- If the worker is to be run by third parties. Database access cannot be handed
  out, and B becomes mandatory.
- If the management schema starts changing often. The cost of breaking two
  components at once overtakes the cost of maintaining an RPC contract.
- If the number of workers strains the database connection budget. Pooling
  through the API becomes the only way out.

## Closing condition

Until a separate, narrowly privileged database role is defined for the worker,
this decision counts as only partly implemented. Privilege separation was the
precondition for it.
