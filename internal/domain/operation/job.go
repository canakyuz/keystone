package operation

import (
	"fmt"
	"math"
	"time"
)

// JobStatus is the state of a unit of work.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"

	// JobDead is a job that has exhausted its attempts. It is never picked up again
	// automatically; it needs intervention.
	JobDead JobStatus = "dead"
)

// Job is the unit of work a worker claims.
type Job struct {
	ID          string
	OperationID string
	TenantID    string
	Status      JobStatus

	Attempts    int
	MaxAttempts int
	NextAttempt time.Time

	// LeaseOwner is the identity of the worker currently holding the job.
	LeaseOwner string

	// LeaseExpires is when the claim ends. After that moment another worker may take
	// the job over.
	LeaseExpires *time.Time

	// Fence is a counter incremented on every claim.
	//
	// A result must be reported with the current fence value. Once the lease has
	// expired and the job has been handed over, an old worker coming back cannot report
	// with the stale fence it holds. Checking LeaseOwner alone is not enough: the old
	// worker knows its own name and would pass that check.
	Fence int64

	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LeaseValidAt says whether the claim is still valid at the given moment.
func (j *Job) LeaseValidAt(now time.Time) bool {
	return j.LeaseExpires != nil && now.Before(*j.LeaseExpires)
}

// Claimable says whether the job can be taken over at the given moment.
//
// To be claimable it must either never have been claimed, or the previous claim must
// have expired. The retry time must also have arrived.
func (j *Job) Claimable(now time.Time) bool {
	if j.Status != JobPending && j.Status != JobRunning {
		return false
	}
	if now.Before(j.NextAttempt) {
		return false
	}

	return !j.LeaseValidAt(now)
}

// VerifyFence confirms the report comes from the current claim.
func (j *Job) VerifyFence(fence int64) error {
	if fence != j.Fence {
		return fmt.Errorf("%w: expected %d, got %d", ErrStaleFence, j.Fence, fence)
	}
	return nil
}

// ExhaustedAfterThisAttempt says no attempts remain after this one.
func (j *Job) ExhaustedAfterThisAttempt() bool {
	return j.Attempts >= j.MaxAttempts
}

// BackoffConfig sets the retry intervals.
type BackoffConfig struct {
	// Base is the first wait.
	Base time.Duration

	// Max caps any single wait.
	Max time.Duration

	// Jitter is the randomness ratio, in [0,1].
	//
	// Why it is needed: without jitter, N jobs that failed at the same moment retry at
	// the same moment, sending a synchronized wave at a dependency that is already in
	// trouble. That is the thundering herd.
	Jitter float64
}

// DefaultBackoff returns sensible defaults.
func DefaultBackoff() BackoffConfig {
	return BackoffConfig{Base: time.Second, Max: 5 * time.Minute, Jitter: 0.3}
}

// BackoffFor computes the wait for a given attempt number.
//
// The exponential base is two: 1s, 2s, 4s, 8s, capped by Max. randFraction must be a
// value in [0,1); passing it in from outside is what makes the test deterministic.
//
// Complexity: O(1).
func BackoffFor(cfg BackoffConfig, attempt int, randFraction float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	// A bounded shift rather than raw math.Pow, so a very large attempt count cannot
	// overflow.
	shift := math.Min(float64(attempt-1), 30)
	wait := time.Duration(float64(cfg.Base) * math.Pow(2, shift))

	if wait > cfg.Max || wait <= 0 {
		wait = cfg.Max
	}

	if cfg.Jitter > 0 {
		spread := float64(wait) * math.Min(cfg.Jitter, 1)
		wait += time.Duration(randFraction * spread)
	}

	return wait
}
