package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
)

// ErrNoJob reports that no claimable job exists.
var ErrNoJob = errors.New("no claimable job")

// Claim takes a runnable job for a bounded period.
//
// QUERY SHAPE
// FOR UPDATE SKIP LOCKED is used. With two workers running at once, one locks the
// row and the other skips it and moves to the next. The alternative was managing a
// lock in the application; with the database already able to do this, a second lock
// layer would be needless complexity and a new source of failure.
//
// Without SKIP LOCKED the second worker would wait for the first's transaction to
// finish and the queue would effectively collapse to a single processor.
//
// LEASE
// A job is marked with a time-bounded claim rather than a persistent "processing"
// flag. If the worker dies mid-provisioning, the lease expires and the job can be
// claimed again. With a persistent flag the job would stay stuck behind it forever.
//
// FENCE
// The fence increments by one on every claim. A result must be reported with the
// current fence value, so a report from an old worker whose lease has expired is
// rejected when it comes back.
//
// Complexity: thanks to the partial index only runnable jobs are scanned; completed
// jobs never enter the plan.
func (r *Repository) Claim(
	ctx context.Context, workerID string, leaseDuration time.Duration,
) (*domain.Job, error) {
	if workerID == "" {
		return nil, errors.New("worker identity is required")
	}

	job := &domain.Job{}
	var leaseOwner sql.NullString
	var leaseExpires sql.NullTime
	var lastError sql.NullString

	err := r.db.QueryRowContext(ctx, `
		WITH claimable AS (
			SELECT id
			FROM provisioning_jobs
			WHERE status IN ('pending', 'running')
			  AND next_attempt_at <= NOW()
			  AND (lease_expires_at IS NULL OR lease_expires_at <= NOW())
			ORDER BY next_attempt_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE provisioning_jobs j
		SET status           = 'running',
		    lease_owner      = $1,
		    lease_expires_at = NOW() + make_interval(secs => $2),
		    fence            = j.fence + 1,
		    attempts         = j.attempts + 1,
		    updated_at       = NOW()
		FROM claimable c
		WHERE j.id = c.id
		RETURNING j.id, j.operation_id, j.tenant_id, j.status,
		          j.attempts, j.max_attempts, j.next_attempt_at,
		          j.lease_owner, j.lease_expires_at, j.fence,
		          j.last_error, j.created_at, j.updated_at`,
		workerID, int(leaseDuration.Seconds()),
	).Scan(&job.ID, &job.OperationID, &job.TenantID, &job.Status,
		&job.Attempts, &job.MaxAttempts, &job.NextAttempt,
		&leaseOwner, &leaseExpires, &job.Fence,
		&lastError, &job.CreatedAt, &job.UpdatedAt)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNoJob
	case err != nil:
		return nil, fmt.Errorf("could not claim job: %w", err)
	}

	job.LeaseOwner = leaseOwner.String
	job.LastError = lastError.String
	if leaseExpires.Valid {
		job.LeaseExpires = &leaseExpires.Time
	}

	return job, nil
}

// RenewLease extends the claim on a long-running job.
//
// The fence is verified here too: an old worker must not be able to disrupt the
// current holder by extending the lease on a job that was handed over.
//
// The fence does NOT increment. A renewal is not a new claim; incrementing would
// invalidate the fence value the worker holds.
func (r *Repository) RenewLease(
	ctx context.Context, jobID, workerID string, fence int64, leaseDuration time.Duration,
) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET lease_expires_at = NOW() + make_interval(secs => $1),
		    updated_at       = NOW()
		WHERE id = $2 AND lease_owner = $3 AND fence = $4 AND status = 'running'`,
		int(leaseDuration.Seconds()), jobID, workerID, fence,
	)
	if err != nil {
		return fmt.Errorf("lease yenilenemedi: %w", err)
	}

	return requireOneRow(result, domain.ErrStaleFence)
}

// CompleteSuccess closes the job and the operation successfully in ONE transaction.
//
// WHY one transaction: written separately, a crash in between could leave a record
// that looks "completed" to the user while its tenant is still not active. The
// promise made to the user would itself be inconsistent.
//
// When activateTenant is true, the tenant is activated in the same transaction.
func (r *Repository) CompleteSuccess(
	ctx context.Context, jobID, workerID string, fence int64, activateTenant bool,
) error {
	return r.completeInTx(ctx, jobID, workerID, fence, func(ctx context.Context, tx *sql.Tx, tenantID string) error {
		if err := markJobSucceeded(ctx, tx, jobID); err != nil {
			return err
		}
		if err := markOperationSucceeded(ctx, tx, jobID); err != nil {
			return err
		}
		if !activateTenant {
			return nil
		}

		_, err := tx.ExecContext(ctx, `
			UPDATE tenants SET status = 'active', updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL`, tenantID)
		if err != nil {
			return fmt.Errorf("could not activate tenant: %w", err)
		}

		return nil
	})
}

// CompleteFailure records a failed attempt.
//
// If attempts remain the job is scheduled for a retry; if not it is marked 'dead' and
// the operation is closed with an error.
func (r *Repository) CompleteFailure(
	ctx context.Context, jobID, workerID string, fence int64,
	errCode, errMessage string, retryAfter time.Duration,
) error {
	return r.completeInTx(ctx, jobID, workerID, fence, func(ctx context.Context, tx *sql.Tx, _ string) error {
		var attempts, maxAttempts int
		err := tx.QueryRowContext(ctx,
			`SELECT attempts, max_attempts FROM provisioning_jobs WHERE id = $1`, jobID,
		).Scan(&attempts, &maxAttempts)
		if err != nil {
			return fmt.Errorf("could not read attempt count: %w", err)
		}

		if attempts >= maxAttempts {
			return exhaustJob(ctx, tx, jobID, errCode, errMessage)
		}

		return rescheduleJob(ctx, tx, jobID, errMessage, retryAfter)
	})
}

// completeInTx verifies the fence and runs the given operation inside a
// transaction.
//
// The fence check uses SELECT ... FOR UPDATE: because the row is locked, no other
// worker can take the job over between the check and the update.
func (r *Repository) completeInTx(
	ctx context.Context, jobID, workerID string, fence int64,
	fn func(context.Context, *sql.Tx, string) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentFence int64
	var currentOwner sql.NullString
	var tenantID string

	err = tx.QueryRowContext(ctx, `
		SELECT fence, lease_owner, tenant_id
		FROM provisioning_jobs
		WHERE id = $1
		FOR UPDATE`, jobID,
	).Scan(&currentFence, &currentOwner, &tenantID)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.ErrNotFound
	case err != nil:
		return fmt.Errorf("could not read job: %w", err)
	}

	// Both the fence and the owner are verified. The fence alone is sufficient, but the
	// owner check catches a mistaken call earlier and more legibly.
	if currentFence != fence || currentOwner.String != workerID {
		return fmt.Errorf("%w: job fence=%d owner=%s, report fence=%d owner=%s",
			domain.ErrStaleFence, currentFence, currentOwner.String, fence, workerID)
	}

	if err := fn(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

func markJobSucceeded(ctx context.Context, tx *sql.Tx, jobID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET status = 'succeeded', lease_owner = NULL, lease_expires_at = NULL,
		    last_error = NULL, updated_at = NOW()
		WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("could not complete job: %w", err)
	}

	return nil
}

func markOperationSucceeded(ctx context.Context, tx *sql.Tx, jobID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE operations
		SET status = 'succeeded', completed_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT operation_id FROM provisioning_jobs WHERE id = $1)`, jobID)
	if err != nil {
		return fmt.Errorf("could not complete operation: %w", err)
	}

	return nil
}

// exhaustJob marks a job out of attempts as dead and closes the operation.
func exhaustJob(ctx context.Context, tx *sql.Tx, jobID, errCode, errMessage string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET status = 'dead', lease_owner = NULL, lease_expires_at = NULL,
		    last_error = $2, updated_at = NOW()
		WHERE id = $1`, jobID, errMessage)
	if err != nil {
		return fmt.Errorf("could not mark job dead: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE operations
		SET status = 'failed', error_code = $2, error_message = $3,
		    completed_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT operation_id FROM provisioning_jobs WHERE id = $1)`,
		jobID, errCode, errMessage)
	if err != nil {
		return fmt.Errorf("could not mark operation failed: %w", err)
	}

	return nil
}

// rescheduleJob schedules the job for a retry.
//
// The lease is released: the job becomes claimable at next_attempt_at, not at once.
func rescheduleJob(ctx context.Context, tx *sql.Tx, jobID, errMessage string, retryAfter time.Duration) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET status           = 'pending',
		    lease_owner      = NULL,
		    lease_expires_at = NULL,
		    next_attempt_at  = NOW() + make_interval(secs => $2),
		    last_error       = $3,
		    updated_at       = NOW()
		WHERE id = $1`, jobID, int(retryAfter.Seconds()), errMessage)
	if err != nil {
		return fmt.Errorf("could not reschedule job: %w", err)
	}

	return nil
}

// requireOneRow insists that the update affected exactly one row.
func requireOneRow(result sql.Result, onMismatch error) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not read affected row count: %w", err)
	}
	if affected != 1 {
		return onMismatch
	}

	return nil
}

// QueueDepth counts jobs by status.
//
// This is the number an on-call engineer actually looks at. The counters answer "how
// much work happened"; only a depth answers "are the workers keeping up", which is the
// question during an incident.
//
// Complexity: an aggregate over the partial index, so completed jobs are not scanned
// and the cost stays flat as the table grows.
func (r *Repository) QueueDepth(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, count(*)
		FROM provisioning_jobs
		WHERE status IN ('pending', 'running')
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("could not read queue depth: %w", err)
	}
	defer rows.Close()

	// Both statuses are reported even when empty. A gauge that simply stops being
	// published is indistinguishable from a scrape failure, and an alert on a missing
	// series is far noisier than one on a zero.
	depth := map[string]int{"pending": 0, "running": 0}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("could not scan queue depth: %w", err)
		}
		depth[status] = count
	}

	return depth, rows.Err()
}
