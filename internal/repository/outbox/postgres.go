// Package outbox stores events durably alongside the work that produced them.
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/canakyuz/keystone/internal/domain/outbox"
)

// Repository reads and writes outbox events.
type Repository struct {
	db *sql.DB
}

// New creates the repository.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// AppendRequest describes an event to emit.
type AppendRequest struct {
	TenantID      string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       any
	MaxAttempts   int
}

// AppendTx writes one event per active endpoint, inside the caller's transaction.
//
// The transaction is a parameter and not an implementation detail, because that is the
// entire pattern: the caller has just written the business data and has not committed
// yet. Opening a transaction here would put the event in a different one, and the moment
// the two can commit independently the guarantee is gone — a provisioning could succeed
// with nobody ever told, or a notification could go out for work that rolled back.
//
// A tenant with no active endpoint produces no rows and no error. Nobody asked to be
// notified, so there is nothing owed; recording an undeliverable event would fill the
// table with rows that can only ever go dead.
//
// Complexity: one indexed read of the tenant's endpoints, then one insert per endpoint.
func (r *Repository) AppendTx(ctx context.Context, tx *sql.Tx, req AppendRequest) (int, error) {
	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return 0, fmt.Errorf("could not encode the event payload: %w", err)
	}

	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 8
	}

	rows, err := tx.QueryContext(ctx, `
		INSERT INTO outbox_events (
			tenant_id, aggregate_type, aggregate_id, event_type, payload, endpoint_id, max_attempts
		)
		SELECT $1, $2, $3, $4, $5::jsonb, e.id, $6
		FROM tenant_webhook_endpoints e
		WHERE e.tenant_id = $1
		  AND e.active
		RETURNING id`,
		req.TenantID, req.AggregateType, req.AggregateID, req.EventType, string(payload), maxAttempts,
	)
	if err != nil {
		return 0, fmt.Errorf("could not append the outbox event: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("could not read the appended event: %w", err)
		}
		count++
	}

	return count, rows.Err()
}

// Claim takes one deliverable event for a bounded period.
//
// FOR UPDATE SKIP LOCKED, as in the provisioning queue and for the same reason: two
// workers must not take the same row, and the second must not wait for the first. See
// docs/decisions/0002-job-claiming-with-skip-locked.md.
//
// The join to the endpoint is deliberate. Reading the URL separately would mean the
// worker could deliver to an endpoint that was deactivated between the claim and the
// read, and the row is small enough that fetching it here costs nothing.
func (r *Repository) Claim(
	ctx context.Context, workerID string, lease time.Duration,
) (*domain.Event, error) {
	if workerID == "" {
		return nil, fmt.Errorf("worker identity is required")
	}

	event := &domain.Event{}
	var leaseOwner sql.NullString
	var leaseExpires sql.NullTime
	var lastError sql.NullString
	var deliveredAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		WITH claimable AS (
			SELECT o.id
			FROM outbox_events o
			JOIN tenant_webhook_endpoints e ON e.id = o.endpoint_id
			WHERE o.status IN ('pending', 'delivering')
			  AND o.next_attempt_at <= NOW()
			  AND (o.lease_expires_at IS NULL OR o.lease_expires_at <= NOW())
			  AND e.active
			ORDER BY o.next_attempt_at
			FOR UPDATE OF o SKIP LOCKED
			LIMIT 1
		)
		UPDATE outbox_events o
		SET status           = 'delivering',
		    lease_owner      = $1,
		    lease_expires_at = NOW() + make_interval(secs => $2),
		    fence            = o.fence + 1,
		    attempts         = o.attempts + 1,
		    updated_at       = NOW()
		FROM claimable c, tenant_webhook_endpoints e
		WHERE o.id = c.id
		  AND e.id = o.endpoint_id
		RETURNING o.id, o.tenant_id, o.aggregate_type, o.aggregate_id, o.event_type,
		          o.payload, o.endpoint_id, e.url, e.secret,
		          o.status, o.attempts, o.max_attempts, o.next_attempt_at,
		          o.lease_owner, o.lease_expires_at, o.fence,
		          o.last_error, o.created_at, o.updated_at, o.delivered_at`,
		workerID, int(lease.Seconds()),
	).Scan(&event.ID, &event.TenantID, &event.AggregateType, &event.AggregateID, &event.EventType,
		&event.Payload, &event.EndpointID, &event.EndpointURL, &event.Secret,
		&event.Status, &event.Attempts, &event.MaxAttempts, &event.NextAttempt,
		&leaseOwner, &leaseExpires, &event.Fence,
		&lastError, &event.CreatedAt, &event.UpdatedAt, &deliveredAt)

	switch {
	case err == sql.ErrNoRows:
		return nil, domain.ErrNoEvent
	case err != nil:
		return nil, fmt.Errorf("could not claim an outbox event: %w", err)
	}

	event.LeaseOwner = leaseOwner.String
	event.LastError = lastError.String
	if leaseExpires.Valid {
		event.LeaseExpires = &leaseExpires.Time
	}
	if deliveredAt.Valid {
		event.DeliveredAt = &deliveredAt.Time
	}

	return event, nil
}

// MarkDelivered records a successful delivery.
func (r *Repository) MarkDelivered(ctx context.Context, eventID, workerID string, fence int64) error {
	return r.completeWithFence(ctx, eventID, workerID, fence, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE outbox_events
			SET status = 'delivered', delivered_at = NOW(), last_error = NULL, updated_at = NOW(),
			    lease_owner = NULL, lease_expires_at = NULL
			WHERE id = $1`, eventID)

		return err
	})
}

// MarkFailed records a failed attempt and schedules the retry, or gives up.
func (r *Repository) MarkFailed(
	ctx context.Context, eventID, workerID string, fence int64, cause string, retryAfter time.Duration,
) error {
	return r.completeWithFence(ctx, eventID, workerID, fence, func(tx *sql.Tx) error {
		var attempts, maxAttempts int
		if err := tx.QueryRowContext(ctx,
			`SELECT attempts, max_attempts FROM outbox_events WHERE id = $1`, eventID,
		).Scan(&attempts, &maxAttempts); err != nil {
			return fmt.Errorf("could not read the attempt count: %w", err)
		}

		if attempts >= maxAttempts {
			// Dead, not retried forever. An endpoint that has refused eight times with
			// increasing backoff is not going to accept the ninth, and a queue that keeps
			// retrying it burns capacity that working endpoints need.
			_, err := tx.ExecContext(ctx, `
				UPDATE outbox_events
				SET status = 'dead', last_error = $2, updated_at = NOW(),
				    lease_owner = NULL, lease_expires_at = NULL
				WHERE id = $1`, eventID, cause)

			return err
		}

		// The lease is released so the event becomes claimable again, but only at
		// next_attempt_at: releasing it without moving that forward would spin.
		_, err := tx.ExecContext(ctx, `
			UPDATE outbox_events
			SET status = 'pending', last_error = $2, next_attempt_at = NOW() + make_interval(secs => $3),
			    updated_at = NOW(), lease_owner = NULL, lease_expires_at = NULL
			WHERE id = $1`, eventID, cause, retryAfter.Seconds())

		return err
	})
}

// completeWithFence verifies the claim is still current, then applies the change.
//
// SELECT ... FOR UPDATE rather than a plain read: the row is locked for the whole
// transaction, so no other worker can take the event over between the check and the
// write. Without the lock this would be the same check-then-write race the fence exists
// to close.
func (r *Repository) completeWithFence(
	ctx context.Context, eventID, workerID string, fence int64, apply func(*sql.Tx) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentFence int64
	var currentOwner sql.NullString

	if err := tx.QueryRowContext(ctx,
		`SELECT fence, lease_owner FROM outbox_events WHERE id = $1 FOR UPDATE`, eventID,
	).Scan(&currentFence, &currentOwner); err != nil {
		return fmt.Errorf("could not read the event: %w", err)
	}

	if currentFence != fence || currentOwner.String != workerID {
		return fmt.Errorf("%w: event fence=%d owner=%s, report fence=%d owner=%s",
			domain.ErrStaleFence, currentFence, currentOwner.String, fence, workerID)
	}

	if err := apply(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// Depth counts events by status.
//
// The number an on-call engineer looks at: counters say how much was delivered, only a
// depth says whether anything is falling behind.
func (r *Repository) Depth(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, count(*)
		FROM outbox_events
		WHERE status IN ('pending', 'delivering', 'dead')
		GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("could not read the outbox depth: %w", err)
	}
	defer rows.Close()

	// Reported even when empty: a gauge that stops being published looks identical to a
	// scrape failure.
	depth := map[string]int{"pending": 0, "delivering": 0, "dead": 0}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("could not scan the outbox depth: %w", err)
		}
		depth[status] = count
	}

	return depth, rows.Err()
}
