# 0004. Idempotency is enforced by a uniqueness constraint

**Status:** Accepted, 2026-09-07

## Context

The client sends `POST /v1/tenants`, the transaction commits, but the HTTP
response never reaches the client. The client sends the same request again. A
second tenant must not be created.

## Rules that must hold

- The same key in the same scope, arriving again with the same body, returns the
  same operation.
- The same key arriving with a different body returns an explicit conflict error.
- Two duplicate requests arriving at once cannot create two operations.

## Options considered

### A. Read first, write if absent

Check whether the key exists, insert the record if it does not.

Weakness: two requests can pass the check at the same time and both write. On its
own it does not prevent the race.

### B. Serializable isolation

Raise the transaction isolation level.

Weakness: it requires retries on conflict. The cost rises exactly when load
rises. It also pushes retry logic onto the caller.

### C. A uniqueness constraint plus catching the violation

A unique index on `(scope, idempotency_key)`. The write is attempted, and if the
constraint is violated the existing record is read.

## Decision

C, with a cheap read in front of it.

The read comes first because it handles the common repeat case in a single query.
If two simultaneous requests pass that check together, the second violates the
constraint during INSERT and lands on the same path.

## Rationale

Uniqueness is enforced by the database. The read is an optimisation, not the
guarantee. Separating where the guarantee lives matters: remove the read and the
system is still correct, just slower.

Silently returning the old result for a different body was rejected. The client
would believe a request it never sent had been processed.

## Consequences

- A normalised digest of the request body is stored. Normalisation is currently
  the SHA-256 of the raw body; a change in field order produces a different
  digest. That is a limit and must be stated in the API documentation.
- The `scope` field is mandatory. One customer's key must not match another
  customer's request.

## What this decision does NOT guarantee

Idempotency is not indefinite. Once the key's retention window (24 hours by
default) expires, the same key creates a new operation. The guarantee changes
silently at that point, and that has to be written into the API documentation.

## When this decision becomes wrong

- If clients retry with the same key hours later. The retention window has to
  grow, but table growth and a cleanup job then come into play.
- If body normalisation turns out to be insufficient. Clients whose JSON field
  order varies would receive spurious conflict errors. At that point canonical
  JSON serialisation is needed.
