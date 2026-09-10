// Package worker runs provisioning jobs.
//
// Design decisions and their reasoning:
//
//   - No unbounded goroutine per job. Making progress without protecting resources is
//     easy; making progress without collapsing under load takes design.
//   - Concurrency is limited both in total and per tenant. Without the per-tenant
//     limit, one customer's load could consume the entire capacity.
//   - On shutdown, new claims stop first, running jobs get a bounded grace period,
//     and are cancelled when it expires. Jobs that cannot finish are taken over by
//     another worker once their lease expires.
//   - Context cancellation does NOT undo side effects that were already committed.
//     That distinction is preserved: cancelling means "try to stop", not "erase what
//     was done".
package worker

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
	"github.com/canakyuz/keystone/pkg/tracing"
)

// Handler performs the actual work of a job.
//
// Contract: it must be safe to run again. A job whose lease expires can move to
// another worker and the same steps run from the start. There is no "runs at most
// once" guarantee.
type Handler func(ctx context.Context, job *domain.Job) error

// Config sets the worker's limits.
type Config struct {
	// ID identifies this worker process. Lease ownership is recorded with it.
	ID string

	// MaxConcurrent is the total number of jobs run at once.
	MaxConcurrent int

	// MaxPerTenant is the number of jobs run at once for a single tenant.
	//
	// Why a separate limit: with only a total limit, a tenant with a lot of work could
	// fill every slot and leave the others waiting.
	MaxPerTenant int

	// LeaseDuration is how long a claim lasts. It must be meaningfully longer than a
	// typical job, or the lease expires before the job finishes.
	LeaseDuration time.Duration

	// LeaseRenewInterval is how often the lease is renewed. It must be below a third
	// of LeaseDuration, so that a missed renewal still leaves room for a second attempt
	// before the lease lapses.
	LeaseRenewInterval time.Duration

	// PollInterval is how long to wait when no job was found.
	PollInterval time.Duration

	// JobTimeout caps a single job.
	JobTimeout time.Duration

	// Metrics records claim, outcome and duration. It may be nil.
	Metrics *metrics.Registry

	// ShutdownGrace is the time running jobs get during shutdown.
	ShutdownGrace time.Duration

	Backoff domain.BackoffConfig
}

// DefaultConfig returns sensible defaults.
func DefaultConfig(id string) Config {
	return Config{
		ID:                 id,
		MaxConcurrent:      8,
		MaxPerTenant:       2,
		LeaseDuration:      60 * time.Second,
		LeaseRenewInterval: 15 * time.Second,
		PollInterval:       time.Second,
		JobTimeout:         5 * time.Minute,
		ShutdownGrace:      30 * time.Second,
		Backoff:            domain.DefaultBackoff(),
	}
}

// Store defines the storage behaviour the worker needs.
//
// The interface is declared here, on the consumer side: the worker knows only the
// four methods it uses, not the whole repository.
type Store interface {
	Claim(ctx context.Context, workerID string, lease time.Duration) (*domain.Job, error)
	RenewLease(ctx context.Context, jobID, workerID string, fence int64, lease time.Duration) error
	CompleteSuccess(ctx context.Context, jobID, workerID string, fence int64, activateTenant bool) error
	CompleteFailure(ctx context.Context, jobID, workerID string, fence int64,
		errCode, errMessage string, retryAfter time.Duration) error
}

// DepthReporter is an optional capability of a Store.
//
// It is a separate interface rather than a fifth method on Store because it is not
// needed to run jobs: a Store that cannot report depth still works, it just publishes
// no gauge. Widening Store would force every implementation, including the fake in the
// tests, to carry a method it has no use for.
type DepthReporter interface {
	QueueDepth(ctx context.Context) (map[string]int, error)
}

// Provisioner is the worker loop for provisioning jobs.
type Provisioner struct {
	cfg     Config
	store   Store
	handler Handler
	log     *logger.Logger

	// slots enforces the total concurrency limit.
	slots chan struct{}

	// tenantGate enforces the per-tenant concurrency limit.
	tenantMu sync.Mutex
	tenant   map[string]int

	wg sync.WaitGroup
}

// New creates a worker.
func New(cfg Config, store Store, handler Handler, log *logger.Logger) *Provisioner {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if cfg.MaxPerTenant <= 0 {
		cfg.MaxPerTenant = 1
	}

	return &Provisioner{
		cfg:     cfg,
		store:   store,
		handler: handler,
		log:     log,
		slots:   make(chan struct{}, cfg.MaxConcurrent),
		tenant:  make(map[string]int),
	}
}

// Run drives the worker loop until ctx is cancelled.
//
// Shutdown order:
//  1. Claiming stops (the loop exits on ctx.Done).
//  2. Running jobs get ShutdownGrace.
//  3. If that expires, the job contexts are cancelled.
//  4. Jobs that could not finish are taken over once their lease expires.
func (p *Provisioner) Run(ctx context.Context) error {
	// jobCtx is the parent of the job contexts. It is kept SEPARATE from Run's ctx, so
	// that when the shutdown signal arrives we can grant the grace period first instead
	// of cancelling running jobs immediately.
	jobCtx, cancelJobs := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelJobs()

	stopDepth := p.publishQueueDepth(ctx)
	defer stopDepth()

	p.loop(ctx, jobCtx)

	return p.drain(cancelJobs)
}

// publishQueueDepth polls the queue depth in the background and returns the function
// that stops it.
//
// WHY a poll rather than counting in the worker: the depth is a property of the queue,
// not of this process. A worker can only count what it claimed, and with several
// workers running, none of them knows the total. Reading it back from the database is
// the only answer that stays correct as workers are added or removed.
//
// The interval is deliberately slower than the poll interval. Depth is a trend, not an
// event, and a count query per claim cycle would put load on the table the claim query
// needs to stay fast.
func (p *Provisioner) publishQueueDepth(ctx context.Context) func() {
	reporter, ok := p.store.(DepthReporter)
	if !ok || p.cfg.Metrics == nil {
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
				depth, err := reporter.QueueDepth(ctx)
				if err != nil {
					// A failed gauge read is not worth failing the worker over, and not worth
					// a log line per tick either. The scrape shows a stale value, which the
					// staleness of the series itself makes visible.
					continue
				}

				for status, count := range depth {
					p.cfg.Metrics.SetQueueDepth(status, count)
				}
			}
		}
	}()

	return func() { once.Do(func() { close(done) }) }
}

// queueDepthInterval is how often the queue depth gauge is refreshed.
const queueDepthInterval = 15 * time.Second

// loop repeatedly finds and runs jobs.
func (p *Provisioner) loop(ctx, jobCtx context.Context) {
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for a slot. With no slot we do not claim: holding a job we cannot start
		// would block another worker that could have run it.
		select {
		case <-ctx.Done():
			return
		case p.slots <- struct{}{}:
		}

		job, err := p.store.Claim(ctx, p.cfg.ID, p.cfg.LeaseDuration)
		if err != nil {
			<-p.slots
			if p.waitBeforeRetry(ctx, ticker, err) {
				return
			}
			continue
		}

		if !p.reserveTenant(job.TenantID) {
			// This tenant is at its quota. We release the job immediately; another worker can
			// take it once the lease expires. Holding it would lock a job in the queue behind a
			// quota that is not free.
			p.releaseJob(jobCtx, job)
			<-p.slots
			continue
		}

		p.wg.Add(1)
		go p.execute(jobCtx, job)
	}
}

// waitBeforeRetry waits after finding no job or hitting an error.
// It returns true if it was cancelled.
func (p *Provisioner) waitBeforeRetry(ctx context.Context, ticker *time.Ticker, err error) bool {
	if !errors.Is(err, oprepo.ErrNoJob) && p.log != nil {
		p.log.WithFields(logger.Fields{"error": err.Error()}).Warn("could not claim job")
	}

	select {
	case <-ctx.Done():
		return true
	case <-ticker.C:
		return false
	}
}

// execute runs a single job and reports the result.
func (p *Provisioner) execute(parent context.Context, job *domain.Job) {
	defer p.wg.Done()
	defer func() { <-p.slots }()
	defer p.releaseTenant(job.TenantID)

	// The duration measured here is claim to outcome, which includes the retry backoff
	// of nothing and the handler of everything. It is the number that answers "how long
	// does provisioning take?", the question the product actually has.
	started := time.Now()
	finish := p.observeStart()

	ctx, cancel := context.WithTimeout(parent, p.cfg.JobTimeout)
	defer cancel()

	// The job's span is LINKED to the request that created it, not parented to it.
	//
	// The request finished long ago. Parenting would keep its trace open until this job
	// finally succeeds, and a trace only completes when all its spans do, so a job that
	// keeps failing would leave a trace that never closes. Every retry would also appear
	// as another child of a request that ended hours earlier.
	//
	// A link says "this was caused by that" without claiming the two are one operation.
	ctx, span := tracing.Tracer("keystone-worker").Start(ctx,
		"provision "+job.TenantID,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(tracing.LinkFrom(job.TraceContext)...),
		trace.WithAttributes(
			attribute.String("keystone.operation_id", job.OperationID),
			attribute.String("keystone.job_id", job.ID),
			attribute.String("keystone.tenant_id", job.TenantID),
			attribute.Int("keystone.attempt", job.Attempts),
			attribute.String("keystone.request_id", job.RequestID),
		),
	)
	defer span.End()

	// Lease renewal runs in the background while the job runs. Without it, a
	// long-running job would be handed to another worker while still executing.
	stopRenew := p.startLeaseRenewal(ctx, job)
	defer stopRenew()

	// A panic must not take the worker process down. A job that panics counts as a
	// failed job and is retried.
	err := p.runHandler(ctx, job)

	finish(outcomeOf(err), time.Since(started))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	p.report(job, err)
}

// observeStart records a claim and returns the function that records the outcome.
//
// It returns a no-op when metrics are not configured, so the call sites stay free of
// nil checks.
func (p *Provisioner) observeStart() func(outcome string, elapsed time.Duration) {
	if p.cfg.Metrics == nil {
		return func(string, time.Duration) {}
	}

	return p.cfg.Metrics.JobClaimed(string(domain.KindTenantProvision))
}

// outcomeOf reduces an error to a bounded label.
//
// The error text is deliberately not used: it can contain a tenant name, a schema name
// or a database message, and any of those as a label would make the series count grow
// without limit.
func outcomeOf(err error) string {
	if err == nil {
		return "succeeded"
	}

	return "failed"
}

// runHandler runs the handler with panic protection.
func (p *Provisioner) runHandler(ctx context.Context, job *domain.Job) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("job panicked: %v", recovered)
		}
	}()

	return p.handler(ctx, job)
}

// report records the result with the current fence.
//
// A separate, non-cancellable context is used for the report: the job's context may
// have timed out, but the result still has to be recorded. Otherwise the job would
// have run and produced its side effects, yet be run again from the start because its
// status was never updated.
func (p *Provisioner) report(job *domain.Job, runErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if runErr == nil {
		if err := p.store.CompleteSuccess(ctx, job.ID, p.cfg.ID, job.Fence, true); err != nil {
			p.logReportFailure(job, err)
		}
		return
	}

	retryAfter := domain.BackoffFor(p.cfg.Backoff, job.Attempts, rand.Float64())

	err := p.store.CompleteFailure(ctx, job.ID, p.cfg.ID, job.Fence,
		"provisioning_failed", runErr.Error(), retryAfter)
	if err != nil {
		p.logReportFailure(job, err)
	}
}

func (p *Provisioner) logReportFailure(job *domain.Job, err error) {
	if p.log == nil {
		return
	}

	// A stale fence is expected: the job moved to another worker. It is logged as
	// information, not as an error.
	level := p.log.WithFields(logger.Fields{
		"job_id":       job.ID,
		"operation_id": job.OperationID,
		"fence":        job.Fence,
		"error":        err.Error(),
	})

	if errors.Is(err, domain.ErrStaleFence) {
		level.Info("could not report result: job was handed over")
		return
	}

	level.Error("could not report result")
}

// startLeaseRenewal begins renewing the lease in the background and returns the
// function that stops it.
func (p *Provisioner) startLeaseRenewal(ctx context.Context, job *domain.Job) func() {
	renewCtx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(p.cfg.LeaseRenewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				err := p.store.RenewLease(renewCtx, job.ID, p.cfg.ID, job.Fence, p.cfg.LeaseDuration)
				if err != nil && p.log != nil {
					p.log.WithFields(logger.Fields{
						"job_id": job.ID,
						"error":  err.Error(),
					}).Warn("Lease yenilenemedi")
				}
			}
		}
	}()

	return cancel
}

// releaseJob puts back a job that was claimed but could not be started.
func (p *Provisioner) releaseJob(ctx context.Context, job *domain.Job) {
	err := p.store.CompleteFailure(ctx, job.ID, p.cfg.ID, job.Fence,
		"tenant_capacity", "per-tenant concurrency limit reached", p.cfg.PollInterval)
	if err != nil && p.log != nil {
		p.log.WithFields(logger.Fields{
			"job_id": job.ID,
			"error":  err.Error(),
		}).Warn("could not release job")
	}
}

// reserveTenant takes a slot from the tenant's quota.
// Complexity: O(1).
func (p *Provisioner) reserveTenant(tenantID string) bool {
	p.tenantMu.Lock()
	defer p.tenantMu.Unlock()

	if p.tenant[tenantID] >= p.cfg.MaxPerTenant {
		return false
	}

	p.tenant[tenantID]++
	return true
}

// releaseTenant returns the slot to the tenant's quota.
func (p *Provisioner) releaseTenant(tenantID string) {
	p.tenantMu.Lock()
	defer p.tenantMu.Unlock()

	p.tenant[tenantID]--
	if p.tenant[tenantID] <= 0 {
		// Keep the map from growing without bound.
		delete(p.tenant, tenantID)
	}
}

// drain waits for running jobs to finish, cancelling them when the grace expires.
func (p *Provisioner) drain(cancelJobs context.CancelFunc) error {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(p.cfg.ShutdownGrace):
		cancelJobs()
	}

	<-done

	// Cancelled jobs do not count as finished. They are taken over by another worker
	// once their leases expire.
	return fmt.Errorf("shutdown grace period expired, running jobs were cancelled")
}
