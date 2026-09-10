package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	opdomain "github.com/canakyuz/keystone/internal/domain/operation"
	domain "github.com/canakyuz/keystone/internal/domain/outbox"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
)

// OutboxStore is the storage behaviour the delivery worker needs.
//
// Declared on the consumer side, as elsewhere: the worker knows the four methods it
// calls, not the whole repository.
type OutboxStore interface {
	Claim(ctx context.Context, workerID string, lease time.Duration) (*domain.Event, error)
	MarkDelivered(ctx context.Context, eventID, workerID string, fence int64) error
	MarkFailed(ctx context.Context, eventID, workerID string, fence int64, cause string, retryAfter time.Duration) error
}

// OutboxDepthReporter is optional, exactly as it is for the provisioning queue.
type OutboxDepthReporter interface {
	Depth(ctx context.Context) (map[string]int, error)
}

// OutboxConfig bounds the delivery worker.
type OutboxConfig struct {
	// ID identifies this worker. Lease ownership is recorded with it.
	ID string

	// MaxConcurrent is how many deliveries run at once.
	//
	// Bounded for a reason that is not about this process: every in-flight delivery is an
	// open connection to somebody else's server. An unbounded worker turns a backlog into
	// an accidental denial of service against the receivers it is trying to notify.
	MaxConcurrent int

	// LeaseDuration must exceed a typical delivery by a clear margin, or an event is
	// taken away from a worker that is still waiting on the receiver.
	LeaseDuration time.Duration

	// PollInterval is how long to wait when there is nothing to deliver.
	PollInterval time.Duration

	// RequestTimeout caps a single delivery.
	//
	// Without it a receiver that accepts a connection and never answers holds a slot until
	// the lease expires, and a handful of those stall the queue.
	RequestTimeout time.Duration

	// ShutdownGrace is what in-flight deliveries get when the worker is stopping.
	ShutdownGrace time.Duration

	// Backoff schedules the retries.
	Backoff opdomain.BackoffConfig

	// Metrics may be nil.
	Metrics *metrics.Registry
}

// DefaultOutboxConfig returns sensible defaults.
func DefaultOutboxConfig(workerID string) OutboxConfig {
	return OutboxConfig{
		ID:             workerID,
		MaxConcurrent:  8,
		LeaseDuration:  time.Minute,
		PollInterval:   2 * time.Second,
		RequestTimeout: 10 * time.Second,
		ShutdownGrace:  20 * time.Second,
		Backoff:        opdomain.DefaultBackoff(),
	}
}

// Outbox delivers events to tenant endpoints.
type Outbox struct {
	cfg    OutboxConfig
	store  OutboxStore
	client *http.Client
	log    *logger.Logger

	slots chan struct{}
	wg    sync.WaitGroup
}

// NewOutbox creates the delivery worker.
func NewOutbox(cfg OutboxConfig, store OutboxStore, client *http.Client, log *logger.Logger) *Outbox {
	if client == nil {
		client = &http.Client{Timeout: cfg.RequestTimeout}
	}

	return &Outbox{
		cfg:    cfg,
		store:  store,
		client: client,
		log:    log,
		slots:  make(chan struct{}, cfg.MaxConcurrent),
	}
}

// Run drives the delivery loop until ctx is cancelled.
//
// Shutdown follows the same order as the provisioning worker: stop claiming, give
// in-flight deliveries a bounded grace period, cancel what is left. An event whose
// delivery is cancelled is not marked failed — its lease simply expires and another
// worker picks it up, which is correct because the receiver may well have accepted it.
func (o *Outbox) Run(ctx context.Context) error {
	deliveryCtx, cancelDeliveries := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelDeliveries()

	stopDepth := o.publishDepth(ctx)
	defer stopDepth()

	o.loop(ctx, deliveryCtx)

	return o.drain(cancelDeliveries)
}

// loop claims and delivers.
func (o *Outbox) loop(ctx, deliveryCtx context.Context) {
	ticker := time.NewTicker(o.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case o.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}

		event, err := o.store.Claim(ctx, o.cfg.ID, o.cfg.LeaseDuration)
		if err != nil {
			<-o.slots

			if o.waitBeforeRetry(ctx, ticker, err) {
				return
			}

			continue
		}

		o.wg.Add(1)
		go o.deliver(deliveryCtx, event)
	}
}

// waitBeforeRetry pauses after an empty claim or an error.
func (o *Outbox) waitBeforeRetry(ctx context.Context, ticker *time.Ticker, err error) bool {
	if !errors.Is(err, domain.ErrNoEvent) && o.log != nil {
		o.log.WithFields(logger.Fields{"error": err.Error()}).Warn("could not claim an outbox event")
	}

	select {
	case <-ctx.Done():
		return true
	case <-ticker.C:
		return false
	}
}

// deliver sends one event and records the outcome.
func (o *Outbox) deliver(parent context.Context, event *domain.Event) {
	defer o.wg.Done()
	defer func() { <-o.slots }()

	started := time.Now()
	finish := o.observeStart()

	ctx, cancel := context.WithTimeout(parent, o.cfg.RequestTimeout)
	defer cancel()

	err := o.post(ctx, event)

	finish(outcomeOf(err), time.Since(started))

	o.report(event, err)
}

// post sends the event to its endpoint.
func (o *Outbox) post(ctx context.Context, event *domain.Event) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, event.EndpointURL, bytes.NewReader(event.Payload))
	if err != nil {
		return fmt.Errorf("could not build the delivery request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// The receiver's half of the at-least-once contract. Delivery over HTTP cannot be
	// exactly-once — a receiver that accepts a request and dies before answering will be
	// sent it again — so it is given a stable key to recognise the repeat by.
	req.Header.Set("Idempotency-Key", event.IdempotencyKey())
	req.Header.Set("X-Keystone-Event", event.EventType)
	req.Header.Set("X-Keystone-Attempt", strconv.Itoa(event.Attempts))

	// Signed so the receiver can tell a real notification from anyone who guessed the URL.
	// HMAC over the exact bytes sent, not over a re-encoding of the payload: a signature
	// computed over anything but the delivered body is a signature of something else.
	req.Header.Set("X-Keystone-Signature", sign(event.Secret, event.Payload))

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("delivery failed: %w", err)
	}
	defer resp.Body.Close()

	// 2xx is accepted. Everything else is a failure to be retried, including 4xx: a
	// receiver answering 404 today may be deployed tomorrow, and the alternative is
	// deciding on the receiver's behalf that the event does not matter.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("receiver answered %d", resp.StatusCode)
	}

	return nil
}

// sign computes the delivery signature.
func sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)

	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// report records the outcome under the current fence.
//
// A separate, non-cancellable context, for the same reason the provisioning worker uses
// one: the delivery context may have timed out, but the result still has to be written or
// the event is delivered again for no reason.
func (o *Outbox) report(event *domain.Event, deliveryErr error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 10*time.Second)
	defer cancel()

	var err error
	if deliveryErr == nil {
		err = o.store.MarkDelivered(ctx, event.ID, o.cfg.ID, event.Fence)
	} else {
		err = o.store.MarkFailed(ctx, event.ID, o.cfg.ID, event.Fence,
			deliveryErr.Error(), o.retryAfter(event))
	}

	if err == nil || o.log == nil {
		return
	}

	fields := logger.Fields{
		"event_id":  event.ID,
		"tenant_id": event.TenantID,
		"attempt":   event.Attempts,
	}

	// A stale fence is expected rather than exceptional: the event moved to another
	// worker. Logged as information so it does not page anybody.
	if errors.Is(err, domain.ErrStaleFence) {
		o.log.WithFields(fields).Info("could not report delivery: the event was handed over")

		return
	}

	o.log.WithFields(fields).Error("could not report delivery")
}

// retryAfter computes the wait before the next attempt.
func (o *Outbox) retryAfter(event *domain.Event) time.Duration {
	return opdomain.BackoffFor(o.cfg.Backoff, event.Attempts, rand.Float64())
}

// observeStart records a delivery and returns the function that records its outcome.
func (o *Outbox) observeStart() func(outcome string, elapsed time.Duration) {
	if o.cfg.Metrics == nil {
		return func(string, time.Duration) {}
	}

	return o.cfg.Metrics.JobClaimed("outbox.deliver")
}

// publishDepth polls the outbox depth in the background.
func (o *Outbox) publishDepth(ctx context.Context) func() {
	reporter, ok := o.store.(OutboxDepthReporter)
	if !ok || o.cfg.Metrics == nil {
		return func() {}
	}

	done := make(chan struct{})
	var once sync.Once

	go func() {
		ticker := time.NewTicker(queueDepthInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				depth, err := reporter.Depth(ctx)
				if err != nil {
					continue
				}

				for status, count := range depth {
					o.cfg.Metrics.SetQueueDepth("outbox_"+status, count)
				}
			}
		}
	}()

	return func() { once.Do(func() { close(done) }) }
}

// drain waits for in-flight deliveries, cancelling them when the grace expires.
func (o *Outbox) drain(cancelDeliveries context.CancelFunc) error {
	done := make(chan struct{})

	go func() {
		o.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(o.cfg.ShutdownGrace):
		cancelDeliveries()
		o.wg.Wait()

		return fmt.Errorf("shutdown grace period expired, deliveries were cancelled")
	}
}
