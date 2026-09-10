package worker

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opdomain "github.com/canakyuz/keystone/internal/domain/operation"
	domain "github.com/canakyuz/keystone/internal/domain/outbox"
)

// fakeOutboxStore is an in-memory queue that behaves like the real one.
//
// No database here: what is under test is the worker's delivery behaviour — signing,
// retries, shutdown — not the storage guarantees, which are verified against real
// PostgreSQL in internal/repository/outbox.
type fakeOutboxStore struct {
	mu     sync.Mutex
	events []*domain.Event

	delivered atomic.Int64
	failed    atomic.Int64
	lastRetry atomic.Int64
}

func (s *fakeOutboxStore) add(events ...*domain.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
}

func (s *fakeOutboxStore) Claim(_ context.Context, workerID string, _ time.Duration) (*domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.events {
		if e.Status == domain.StatusPending {
			e.Status = domain.StatusDelivering
			e.LeaseOwner = workerID
			e.Fence++
			e.Attempts++

			claimed := *e

			return &claimed, nil
		}
	}

	return nil, domain.ErrNoEvent
}

func (s *fakeOutboxStore) MarkDelivered(_ context.Context, eventID, _ string, _ int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.events {
		if e.ID == eventID {
			e.Status = domain.StatusDelivered
		}
	}
	s.delivered.Add(1)

	return nil
}

func (s *fakeOutboxStore) MarkFailed(
	_ context.Context, eventID, _ string, _ int64, _ string, retryAfter time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.events {
		if e.ID == eventID {
			e.Status = domain.StatusDead
		}
	}
	s.failed.Add(1)
	s.lastRetry.Store(int64(retryAfter))

	return nil
}

// event builds a pending event pointing at the given receiver.
func event(id, url string, payload string) *domain.Event {
	return &domain.Event{
		ID:          id,
		TenantID:    "11111111-1111-4111-8111-111111111111",
		EventType:   "tenant.provisioned",
		Payload:     []byte(payload),
		EndpointURL: url,
		Secret:      "shhh",
		Status:      domain.StatusPending,
		MaxAttempts: 3,
	}
}

// runOutboxFor drives the worker for a bounded period.
func runOutboxFor(t *testing.T, o *Outbox, d time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	_ = o.Run(ctx)
}

// newOutbox builds a worker with test-sized timings.
func newOutbox(store OutboxStore) *Outbox {
	cfg := DefaultOutboxConfig("worker-1")
	cfg.PollInterval = 10 * time.Millisecond
	cfg.RequestTimeout = 2 * time.Second
	cfg.ShutdownGrace = time.Second
	cfg.Backoff = opdomain.BackoffConfig{Base: time.Second, Max: time.Minute}

	return NewOutbox(cfg, store, nil, nil)
}

// TestOutbox_DeliversToTheEndpoint is the baseline.
func TestOutbox_DeliversToTheEndpoint(t *testing.T) {
	var received atomic.Int64

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	store := &fakeOutboxStore{}
	store.add(event("e1", receiver.URL, `{"tenant_id":"t1"}`))

	runOutboxFor(t, newOutbox(store), 300*time.Millisecond)

	assert.Equal(t, int64(1), received.Load())
	assert.Equal(t, int64(1), store.delivered.Load())
}

// TestOutbox_SignsTheExactBytesItSends verifies the receiver can verify the delivery.
//
// The signature is computed over the delivered body, not over a re-encoding of the
// payload. A signature over anything but what was sent is a signature of something else,
// and the receiver's check would fail for reasons nobody could diagnose.
func TestOutbox_SignsTheExactBytesItSends(t *testing.T) {
	const payload = `{"tenant_id":"t1","event":"tenant.provisioned"}`

	var gotSignature, gotKey string
	var gotBody []byte

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature = r.Header.Get("X-Keystone-Signature")
		gotKey = r.Header.Get("Idempotency-Key")
		gotBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	store := &fakeOutboxStore{}
	store.add(event("e1", receiver.URL, payload))

	runOutboxFor(t, newOutbox(store), 300*time.Millisecond)

	mac := hmac.New(sha256.New, []byte("shhh"))
	mac.Write(gotBody)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, want, gotSignature, "the signature does not cover the bytes that were sent")
	assert.Equal(t, "e1", gotKey, "the idempotency key is not the event id")
}

// TestOutbox_IdempotencyKeyIsStableAcrossAttempts is what makes at-least-once usable.
//
// The receiver is told the same key every time so it can recognise a repeat. A key that
// changed per attempt would carry no information, which is the same as sending none.
func TestOutbox_IdempotencyKeyIsStableAcrossAttempts(t *testing.T) {
	var keys sync.Map
	var seen atomic.Int64

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys.Store(r.Header.Get("Idempotency-Key"), true)
		seen.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer receiver.Close()

	store := &fakeOutboxStore{}
	e := event("e1", receiver.URL, `{}`)
	store.add(e)

	// Two attempts of the same event.
	o := newOutbox(store)
	runOutboxFor(t, o, 200*time.Millisecond)

	store.mu.Lock()
	e.Status = domain.StatusPending
	store.mu.Unlock()

	runOutboxFor(t, o, 200*time.Millisecond)

	distinct := 0
	keys.Range(func(any, any) bool { distinct++; return true })

	require.GreaterOrEqual(t, seen.Load(), int64(2), "the receiver was not tried twice")
	assert.Equal(t, 1, distinct, "the idempotency key changed between attempts")
}

// TestOutbox_RetriesOnRejection verifies a non-2xx is a failure with a backoff.
//
// 4xx is retried too. A receiver answering 404 today may be deployed tomorrow, and
// deciding otherwise means deciding on the receiver's behalf that the event is worthless.
func TestOutbox_RetriesOnRejection(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer receiver.Close()

			store := &fakeOutboxStore{}
			store.add(event("e1", receiver.URL, `{}`))

			runOutboxFor(t, newOutbox(store), 300*time.Millisecond)

			assert.Zero(t, store.delivered.Load(), "a rejected delivery was recorded as delivered")
			assert.Positive(t, store.failed.Load())
			assert.Positive(t, store.lastRetry.Load(), "the retry was scheduled with no backoff")
		})
	}
}

// TestOutbox_UnreachableEndpointIsRetried covers the receiver being down entirely.
func TestOutbox_UnreachableEndpointIsRetried(t *testing.T) {
	store := &fakeOutboxStore{}
	// A port nobody is listening on.
	store.add(event("e1", "http://127.0.0.1:1/hook", `{}`))

	runOutboxFor(t, newOutbox(store), 500*time.Millisecond)

	assert.Zero(t, store.delivered.Load())
	assert.Positive(t, store.failed.Load())
}

// TestOutbox_RespectsMaxConcurrent verifies deliveries are bounded.
//
// Every in-flight delivery is an open connection to somebody else's server. An unbounded
// worker turns a backlog into an accidental denial of service against the receivers it
// is trying to notify.
func TestOutbox_RespectsMaxConcurrent(t *testing.T) {
	var inFlight, peak atomic.Int64

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := inFlight.Add(1)
		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}

		time.Sleep(40 * time.Millisecond)
		inFlight.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	store := &fakeOutboxStore{}
	for i := 0; i < 30; i++ {
		store.add(event("e"+string(rune('a'+i)), receiver.URL, `{}`))
	}

	cfg := DefaultOutboxConfig("worker-1")
	cfg.MaxConcurrent = 3
	cfg.PollInterval = 5 * time.Millisecond
	cfg.ShutdownGrace = time.Second

	runOutboxFor(t, NewOutbox(cfg, store, nil, nil), 400*time.Millisecond)

	assert.LessOrEqual(t, peak.Load(), int64(3), "the concurrency limit was exceeded")
}

// TestOutbox_GracefulShutdownWaitsForInFlightDeliveries verifies a delivery in progress
// is allowed to finish.
//
// Cutting it off would leave an event whose receiver may well have accepted it, and the
// worker unable to say so.
func TestOutbox_GracefulShutdownWaitsForInFlightDeliveries(t *testing.T) {
	var completed atomic.Bool

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		completed.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	store := &fakeOutboxStore{}
	store.add(event("e1", receiver.URL, `{}`))

	cfg := DefaultOutboxConfig("worker-1")
	cfg.PollInterval = 5 * time.Millisecond
	cfg.ShutdownGrace = time.Second

	ctx, cancel := context.WithCancel(context.Background())
	o := NewOutbox(cfg, store, nil, nil)

	done := make(chan error, 1)
	go func() { done <- o.Run(ctx) }()

	// Cancel while the delivery is still in flight.
	time.Sleep(50 * time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.True(t, completed.Load(), "an in-flight delivery was cut off by shutdown")
	assert.Equal(t, int64(1), store.delivered.Load())
}
