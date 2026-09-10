package operation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOperation_RejectsInvalidTransitions verifies invalid state transitions are
// rejected.
//
// If the domain layer does not reject them, an inconsistent state is only caught by a
// database constraint and the error message means nothing to the caller.
func TestOperation_RejectsInvalidTransitions(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name    string
		from    Status
		apply   func(*Operation) error
		allowed bool
	}{
		{"pending -> running", StatusPending, func(o *Operation) error { return o.MarkRunning() }, true},
		{"pending -> failed", StatusPending, func(o *Operation) error { return o.MarkFailed("x", "y", now) }, true},
		{"pending -> succeeded", StatusPending, func(o *Operation) error { return o.MarkSucceeded(now) }, false},
		{"running -> succeeded", StatusRunning, func(o *Operation) error { return o.MarkSucceeded(now) }, true},
		{"running -> running", StatusRunning, func(o *Operation) error { return o.MarkRunning() }, false},
		{"succeeded -> failed", StatusSucceeded, func(o *Operation) error { return o.MarkFailed("x", "y", now) }, false},
		{"failed -> running", StatusFailed, func(o *Operation) error { return o.MarkRunning() }, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			op := &Operation{Status: tc.from}
			err := tc.apply(op)

			if tc.allowed {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrInvalidTransition)
			assert.Equal(t, tc.from, op.Status, "a rejected transition changed the status")
		})
	}
}

// TestOperation_MarkFailedRequiresCode verifies the error code is mandatory.
// Clients branch on the code; leaving only free text would push the caller into
// branching on the message content.
func TestOperation_MarkFailedRequiresCode(t *testing.T) {
	op := &Operation{Status: StatusRunning}

	err := op.MarkFailed("", "something went wrong", time.Now())

	assert.ErrorIs(t, err, ErrInvalidTransition)
	assert.Equal(t, StatusRunning, op.Status)
}

// TestOperation_TerminalStatesSetCompletedAt verifies a closed operation gets its
// completion time stamped.
func TestOperation_TerminalStatesSetCompletedAt(t *testing.T) {
	now := time.Now()

	succeeded := &Operation{Status: StatusRunning}
	require.NoError(t, succeeded.MarkSucceeded(now))
	assert.NotNil(t, succeeded.CompletedAt)
	assert.True(t, succeeded.Status.IsTerminal())

	failed := &Operation{Status: StatusRunning}
	require.NoError(t, failed.MarkFailed("schema_error", "no schema", now))
	assert.NotNil(t, failed.CompletedAt)
	assert.Equal(t, "schema_error", failed.ErrorCode)
}

// TestJob_Claimable verifies when a job becomes claimable.
func TestJob_Claimable(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Minute)
	past := now.Add(-time.Minute)

	cases := []struct {
		name string
		job  Job
		want bool
	}{
		{"hic sahiplenilmemis", Job{Status: JobPending, NextAttempt: past}, true},
		{"lease suresi dolmus", Job{Status: JobRunning, NextAttempt: past, LeaseExpires: &past}, true},
		{"lease gecerli", Job{Status: JobRunning, NextAttempt: past, LeaseExpires: &future}, false},
		{"deneme zamani gelmemis", Job{Status: JobPending, NextAttempt: future}, false},
		{"tamamlanmis", Job{Status: JobSucceeded, NextAttempt: past}, false},
		{"olu", Job{Status: JobDead, NextAttempt: past}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.job.Claimable(now))
		})
	}
}

// TestJob_VerifyFence verifies a late report is rejected.
func TestJob_VerifyFence(t *testing.T) {
	job := Job{Fence: 5}

	assert.NoError(t, job.VerifyFence(5))
	assert.ErrorIs(t, job.VerifyFence(4), ErrStaleFence)
	assert.ErrorIs(t, job.VerifyFence(6), ErrStaleFence)
}

// TestBackoffFor verifies the retry interval grows exponentially and is capped.
func TestBackoffFor(t *testing.T) {
	cfg := BackoffConfig{Base: time.Second, Max: time.Minute}

	// Jitter is zero and randFraction is fixed, so the result is deterministic.
	assert.Equal(t, 1*time.Second, BackoffFor(cfg, 1, 0))
	assert.Equal(t, 2*time.Second, BackoffFor(cfg, 2, 0))
	assert.Equal(t, 4*time.Second, BackoffFor(cfg, 3, 0))
	assert.Equal(t, 8*time.Second, BackoffFor(cfg, 4, 0))

	assert.Equal(t, time.Minute, BackoffFor(cfg, 20, 0), "ust sinira takilmadi")
	assert.Equal(t, time.Minute, BackoffFor(cfg, 1000, 0), "cok buyuk deneme sayisi tasti")
}

// TestBackoffFor_Jitter verifies jitter stays within the expected range.
//
// Without jitter, N jobs that failed at the same moment retry at the same moment and
// send a synchronized wave at a dependency that is already in trouble.
func TestBackoffFor_Jitter(t *testing.T) {
	cfg := BackoffConfig{Base: time.Second, Max: time.Minute, Jitter: 0.5}

	lowest := BackoffFor(cfg, 1, 0)
	highest := BackoffFor(cfg, 1, 0.999)

	assert.Equal(t, time.Second, lowest, "jitter tabani kisaltti")
	assert.Greater(t, highest, lowest)
	assert.LessOrEqual(t, highest, time.Second+500*time.Millisecond)
}
