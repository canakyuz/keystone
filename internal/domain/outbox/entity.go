// Package outbox models events that are written with the work they describe and
// delivered afterwards.
//
// The pattern exists to close a gap that has no other answer: a database transaction and
// an HTTP call cannot commit together. Calling the endpoint inside the transaction holds
// it open for as long as the receiver takes and rolls back real work when the receiver is
// down. Calling it after the commit loses the notification if the process dies in
// between. Writing a row inside the transaction and delivering it later moves the problem
// somewhere it can be solved, because a row can be retried and a lost call cannot.
package outbox

import (
	"errors"
	"fmt"
	"time"
)

// Status is the delivery state of an event.
type Status string

const (
	// StatusPending is waiting to be delivered, or waiting out a backoff.
	StatusPending Status = "pending"

	// StatusDelivering is claimed by a worker.
	StatusDelivering Status = "delivering"

	// StatusDelivered was accepted by the receiver.
	StatusDelivered Status = "delivered"

	// StatusDead exhausted its attempts. It is never retried automatically; getting it
	// out requires someone to look at why, which is the point of a separate state rather
	// than silently retrying forever.
	StatusDead Status = "dead"
)

var (
	// ErrNoEvent reports that nothing is ready to deliver.
	ErrNoEvent = errors.New("outbox: no deliverable event")

	// ErrStaleFence rejects a report from a worker whose lease was taken over.
	ErrStaleFence = errors.New("outbox: stale fence, report rejected")
)

// Event is one notification owed to one endpoint.
//
// One row per endpoint rather than one per event with a list of destinations: an endpoint
// that is down would otherwise hold up delivery to every other endpoint of the same
// tenant, and a single retry counter cannot describe two receivers failing at different
// rates.
type Event struct {
	ID       string
	TenantID string

	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte

	EndpointID  string
	EndpointURL string
	Secret      string

	Status      Status
	Attempts    int
	MaxAttempts int
	NextAttempt time.Time

	LeaseOwner   string
	LeaseExpires *time.Time
	Fence        int64

	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeliveredAt *time.Time
}

// VerifyFence confirms a report comes from the current claim.
//
// Same reasoning as the provisioning queue: an old worker knows its own name, so checking
// the owner alone would let it mark an event delivered after somebody else took it over
// and is still trying.
func (e *Event) VerifyFence(fence int64) error {
	if fence != e.Fence {
		return fmt.Errorf("%w: expected %d, got %d", ErrStaleFence, e.Fence, fence)
	}

	return nil
}

// ExhaustedAfterThisAttempt says no attempts remain after this one.
func (e *Event) ExhaustedAfterThisAttempt() bool {
	return e.Attempts >= e.MaxAttempts
}

// IdempotencyKey is what the receiver uses to recognise a repeated delivery.
//
// The event id, not a fresh value per attempt. At-least-once delivery means the receiver
// will see the same event more than once; a key that changed per attempt would tell it
// nothing, which is the same as having no key at all.
func (e *Event) IdempotencyKey() string {
	return e.ID
}
