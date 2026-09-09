# 0003. Claims are protected by a lease and a fencing token

**Status:** Accepted, 2026-09-07

## Context

A worker's process can die after it has taken a job. The death is not announced:
the server crashes, the container is killed, the network partitions.

If the job looks claimed but the worker is gone, the job is left in a state where
nobody is running it.

## Rules that must hold

- If a process dies mid-job, the job must be reclaimable.
- Only the current holder can record a result.
- The previous holder of a reclaimed job must not be able to corrupt the result.

## Options considered

### A. A persistent "processing" flag

`status = 'processing'` is written when the job is taken, and changed when it
finishes.

Weakness: if the worker dies, the flag stays forever. The job is never reclaimed.
Recovery needs manual intervention, or a cleanup job along the lines of "reset
anything that has been processing for longer than X". The latter is just a badly
implemented lease.

### B. A lease alone

The claim is written with an expiry. Once it expires, the job can be reclaimed.

Weakness: the old worker can come back and report a result. It knows its own name,
so it passes the `lease_owner` check. It corrupts the current holder's work.

### C. A lease plus a fencing token

The claim has an expiry, and a counter is incremented on every claim. A result
must be reported with the current counter value.

## Decision

C.

## Rationale

B alone is not enough, and the reason is subtle. The old worker's identity does
not change; an identity check does not stop it. What stops it is knowing how many
times the job has been claimed. The old worker arrives with the number it holds,
the current number is larger, and the report is rejected.

The fence does not increment on renewal, only on claiming. If it incremented on
renewal, the worker's own renewal would invalidate the value it holds.

## Consequences

- Every result report performs a fence check, which is an extra read. It is done
  with `SELECT ... FOR UPDATE`, so no takeover can slip between the check and the
  write.
- Long-running jobs have to renew the lease. Without renewal, the job is taken
  away while it is still running. The renewal interval is kept below a third of
  the lease duration, so that a missed renewal still leaves room for a second
  attempt.

## What this decision does NOT guarantee

Fencing protects the metadata update in the management table. It does not prevent
the old worker from producing a side effect in an external system.

A concrete example: the old worker lost its lease but is still running and
executing `CREATE SCHEMA` in the tenant schema. Fencing does not stop that.
Fencing only stops it from writing "completed".

For that, the following are also needed, and do not exist yet:

- Provisioning steps that are safe to repeat (`IF NOT EXISTS` and the like).
- A per-tenant lock, or fencing support in the target system itself.

This is why the system does not offer an "runs at most once" guarantee. A job can
run again, and the handlers have to tolerate it.

## When this decision becomes wrong

- If job duration regularly exceeds the lease duration. Renewal covers that, but
  renewal can fail too; at that point the lease duration has to grow or the job
  has to be split into pieces.
- If clock skew becomes serious. The lease expiry is written using the database
  clock, so right now it does not depend on worker clocks. Moving the lease check
  into the application would bring that dependency back.
