# 0002. Job claiming is done with FOR UPDATE SKIP LOCKED

**Status:** Accepted, 2026-09-07

## Context

Several workers take jobs from the same queue. Two workers running the same job
means steps like schema creation run twice.

## Rules that must hold

- Two workers cannot claim the same job at the same time.
- One worker being busy must not block another.
- Claiming must not be open to a race between the check and the write.

## Options considered

### A. An application-side lock

A lock is taken in Redis or in memory, then the job is updated.

Weakness: the lock and the database write live in two different systems. If the
write fails after the lock is taken, the lock is orphaned. The lock itself also
adds an availability dependency.

### B. SELECT then UPDATE

Read a pending job, then write the claim.

Weakness: another worker can read the same job between the two steps. The classic
check-then-write race. Serializable isolation solves it but requires retries on
conflict, and the cost rises exactly when load rises.

### C. FOR UPDATE SKIP LOCKED

The row is locked, locked rows are skipped, and the claim is written, all in one
statement.

## Decision

C.

## Rationale

Claim atomicity falls to the database's own locking. A second locking layer means
a second source of failure.

The `SKIP LOCKED` part matters. Without it, the second worker waits for the
first's transaction to finish and the queue effectively collapses to a single
processor. Skipping ahead lets N workers take N different jobs in parallel.

Thanks to the partial index (`WHERE status IN ('pending','running')`), completed
jobs are not scanned, so claim cost stays flat as the table grows.

## Consequences

- The queue is tied to PostgreSQL. Moving to a separate queue system would mean
  rewriting this query from scratch.
- Ordering is by `next_attempt_at`. Adding priority or fair scheduling would
  require changing both the query and the index.

## When this decision becomes wrong

- If job volume exceeds what the database can carry. The claim query runs on every
  worker loop; at a high worker count, lock contention can become the bottleneck.
- If jobs take milliseconds rather than seconds. At that scale the database round
  trip costs more than the work itself.
- If fair scheduling becomes necessary. If one tenant filling the queue delays
  small tenants, simple ordering is not enough.

## Measurement note

The cost of this query was assumed, not measured. The query plan, index usage and
lock contention each need to be looked at. Not done yet.
