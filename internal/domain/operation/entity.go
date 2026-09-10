// Package operation models durable tracking of long-running work.
//
// Two concepts are kept apart:
//
//   - Operation: the user-facing record. "How is Acme's provisioning going?"
//   - Job: the executor-facing unit of work. "Who took this, and until when?"
//
// The reason for the split: a tenant's state and a single operation's state are not
// the same thing. After a failed provisioning, a new operation can be opened for the
// same tenant. Holding both in one field would make that distinction impossible.
package operation

import (
	"errors"
	"fmt"
	"time"
)

// Kind is the type of work.
type Kind string

const (
	// KindTenantProvision prepares the workspace for a new tenant.
	KindTenantProvision Kind = "tenant.provision"
)

// Status is the operation's state.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

// IsTerminal says the state will not change again.
func (s Status) IsTerminal() bool {
	return s == StatusSucceeded || s == StatusFailed
}

var (
	// ErrNotFound reports that the operation does not exist.
	ErrNotFound = errors.New("operation: not found")

	// ErrInvalidTransition reports a state transition that is not permitted.
	ErrInvalidTransition = errors.New("operation: invalid state transition")

	// ErrStaleFence rejects a late report from a worker.
	ErrStaleFence = errors.New("operation: stale fence, report rejected")

	// ErrIdempotencyConflict reports the same key being used with a different body.
	ErrIdempotencyConflict = errors.New("operation: idempotency key reused with a different request")
)

// Operation is the user-facing record of the work.
type Operation struct {
	ID       string
	TenantID string
	Kind     Kind
	Status   Status

	ErrorCode    string
	ErrorMessage string

	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// allowedTransitions is the table of permitted state transitions.
//
// Writing the table out explicitly beats reading the valid transitions out of ifs
// scattered through the code: why a transition is forbidden is visible in one place,
// and adding a new state cannot be forgotten.
var allowedTransitions = map[Status][]Status{
	StatusPending:   {StatusRunning, StatusFailed},
	StatusRunning:   {StatusSucceeded, StatusFailed},
	StatusSucceeded: {},
	StatusFailed:    {},
}

// CanTransitionTo says whether the transition is permitted.
// Complexity: O(k), k being the transitions leaving a state, at most 2.
func (o *Operation) CanTransitionTo(next Status) bool {
	for _, allowed := range allowedTransitions[o.Status] {
		if allowed == next {
			return true
		}
	}
	return false
}

// MarkRunning marks the operation as running.
func (o *Operation) MarkRunning() error {
	return o.transition(StatusRunning, nil)
}

// MarkSucceeded closes the operation successfully.
func (o *Operation) MarkSucceeded(now time.Time) error {
	return o.transition(StatusSucceeded, &now)
}

// MarkFailed closes the operation with an error.
//
// The error code is mandatory: clients branch on the code. Leaving only free text
// would push the caller into branching on the message content.
func (o *Operation) MarkFailed(code, message string, now time.Time) error {
	if code == "" {
		return fmt.Errorf("%w: an error code is required", ErrInvalidTransition)
	}

	if err := o.transition(StatusFailed, &now); err != nil {
		return err
	}

	o.ErrorCode = code
	o.ErrorMessage = message

	return nil
}

// transition validates and applies a state change.
func (o *Operation) transition(next Status, completedAt *time.Time) error {
	if !o.CanTransitionTo(next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, o.Status, next)
	}

	o.Status = next
	o.CompletedAt = completedAt

	return nil
}
