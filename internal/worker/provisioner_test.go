package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
)

// fakeStore is an in-memory store that behaves like a real queue.
//
// Real PostgreSQL is not used here, because what is under test is the worker's
// capacity and shutdown behaviour, not the database guarantees. Those are verified
// against real Postgres in the internal/repository/operation tests.
type fakeStore struct {
	mu   sync.Mutex
	jobs []*domain.Job

	claimed   atomic.Int64
	succeeded atomic.Int64
	failed    atomic.Int64
	renewals  atomic.Int64
}

func newFakeStore(jobs ...*domain.Job) *fakeStore {
	return &fakeStore{jobs: jobs}
}

func (s *fakeStore) Claim(_ context.Context, workerID string, _ time.Duration) (*domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.jobs) == 0 {
		return nil, oprepo.ErrNoJob
	}

	job := s.jobs[0]
	s.jobs = s.jobs[1:]
	job.LeaseOwner = workerID
	s.claimed.Add(1)

	return job, nil
}

func (s *fakeStore) RenewLease(_ context.Context, _, _ string, _ int64, _ time.Duration) error {
	s.renewals.Add(1)
	return nil
}

func (s *fakeStore) CompleteSuccess(_ context.Context, _, _ string, _ int64, _ bool) error {
	s.succeeded.Add(1)
	return nil
}

func (s *fakeStore) CompleteFailure(_ context.Context, _, _ string, _ int64, _, _ string, _ time.Duration) error {
	s.failed.Add(1)
	return nil
}

func job(id, tenantID string) *domain.Job {
	return &domain.Job{
		ID: id, OperationID: "op-" + id, TenantID: tenantID,
		Status: domain.JobRunning, Attempts: 1, MaxAttempts: 3, Fence: 1,
	}
}

func testConfig() Config {
	cfg := DefaultConfig("worker-test")
	cfg.PollInterval = 5 * time.Millisecond
	cfg.LeaseRenewInterval = 5 * time.Millisecond
	cfg.ShutdownGrace = 2 * time.Second
	cfg.JobTimeout = 2 * time.Second
	return cfg
}

// runFor runs the worker for a set period, then stops it.
func runFor(t *testing.T, p *Provisioner, d time.Duration) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	return p.Run(ctx)
}

// TestRun_RespectsMaxConcurrent verifies the total concurrency limit is not exceeded.
//
// Spawning an unbounded goroutine per job is easy; making progress while protecting
// the system's resources takes design.
func TestRun_RespectsMaxConcurrent(t *testing.T) {
	const maxConcurrent = 3

	jobs := make([]*domain.Job, 0, 30)
	for i := 0; i < 30; i++ {
		jobs = append(jobs, job(string(rune('a'+i)), "tenant-"+string(rune('a'+i))))
	}

	store := newFakeStore(jobs...)

	var running atomic.Int64
	var peak atomic.Int64

	handler := func(ctx context.Context, j *domain.Job) error {
		current := running.Add(1)
		defer running.Add(-1)

		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}

		time.Sleep(20 * time.Millisecond)
		return nil
	}

	cfg := testConfig()
	cfg.MaxConcurrent = maxConcurrent
	cfg.MaxPerTenant = maxConcurrent

	p := New(cfg, store, handler, nil)
	require.NoError(t, runFor(t, p, 700*time.Millisecond))

	assert.LessOrEqual(t, peak.Load(), int64(maxConcurrent),
		"the total concurrency limit was exceeded")
	assert.Greater(t, store.succeeded.Load(), int64(0), "hic is tamamlanmadi")
}

// TestRun_RespectsPerTenantLimit verifies a single tenant cannot consume the whole
// capacity.
//
// With only a total limit, a tenant with a lot of work could fill every slot and
// leave the others waiting.
func TestRun_RespectsPerTenantLimit(t *testing.T) {
	const perTenant = 2

	jobs := make([]*domain.Job, 0, 20)
	for i := 0; i < 20; i++ {
		jobs = append(jobs, job(string(rune('a'+i)), "tek-tenant"))
	}

	store := newFakeStore(jobs...)

	var running atomic.Int64
	var peak atomic.Int64

	handler := func(ctx context.Context, j *domain.Job) error {
		current := running.Add(1)
		defer running.Add(-1)

		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}

		time.Sleep(20 * time.Millisecond)
		return nil
	}

	cfg := testConfig()
	cfg.MaxConcurrent = 8
	cfg.MaxPerTenant = perTenant

	p := New(cfg, store, handler, nil)
	require.NoError(t, runFor(t, p, 700*time.Millisecond))

	assert.LessOrEqual(t, peak.Load(), int64(perTenant),
		"the per-tenant concurrency limit was exceeded")
}

// TestRun_HandlerFailureIsReported verifies a failed job is reported.
func TestRun_HandlerFailureIsReported(t *testing.T) {
	store := newFakeStore(job("a", "tenant-1"))

	handler := func(ctx context.Context, j *domain.Job) error {
		return errors.New("could not create schema")
	}

	p := New(testConfig(), store, handler, nil)
	require.NoError(t, runFor(t, p, 300*time.Millisecond))

	assert.Equal(t, int64(1), store.failed.Load())
	assert.Zero(t, store.succeeded.Load())
}

// TestRun_PanicDoesNotKillWorker verifies that a panicking job does not take the
// worker down, and that it counts as a failed job.
func TestRun_PanicDoesNotKillWorker(t *testing.T) {
	store := newFakeStore(job("a", "tenant-1"), job("b", "tenant-2"))

	var handled atomic.Int64
	handler := func(ctx context.Context, j *domain.Job) error {
		if handled.Add(1) == 1 {
			panic("beklenmeyen durum")
		}
		return nil
	}

	p := New(testConfig(), store, handler, nil)
	require.NoError(t, runFor(t, p, 400*time.Millisecond))

	assert.Equal(t, int64(1), store.failed.Load(), "a panicking job was not counted as failed")
	assert.Equal(t, int64(1), store.succeeded.Load(), "the worker stopped after a panic")
}

// TestRun_GracefulShutdownWaitsForRunningJobs verifies a running job is waited for
// during shutdown.
func TestRun_GracefulShutdownWaitsForRunningJobs(t *testing.T) {
	store := newFakeStore(job("a", "tenant-1"))

	var completed atomic.Bool
	handler := func(ctx context.Context, j *domain.Job) error {
		time.Sleep(200 * time.Millisecond)
		completed.Store(true)
		return nil
	}

	cfg := testConfig()
	cfg.ShutdownGrace = time.Second

	p := New(cfg, store, handler, nil)

	// Send the shutdown signal right after the job starts.
	require.NoError(t, runFor(t, p, 50*time.Millisecond))

	assert.True(t, completed.Load(), "shutdown cut a running job short")
	assert.Equal(t, int64(1), store.succeeded.Load())
}

// TestRun_ShutdownGraceExpiryCancelsJobs verifies jobs are cancelled once the grace
// period expires.
//
// Cancelled jobs do not count as finished; they are taken over by another worker once
// their leases expire.
func TestRun_ShutdownGraceExpiryCancelsJobs(t *testing.T) {
	store := newFakeStore(job("a", "tenant-1"))

	started := make(chan struct{})
	var cancelled atomic.Bool

	handler := func(ctx context.Context, j *domain.Job) error {
		close(started)
		<-ctx.Done()
		cancelled.Store(true)
		return ctx.Err()
	}

	cfg := testConfig()
	cfg.ShutdownGrace = 100 * time.Millisecond

	p := New(cfg, store, handler, nil)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- p.Run(ctx) }()

	<-started
	cancel()

	select {
	case err := <-errCh:
		assert.Error(t, err, "no error was reported although the grace period expired")
	case <-time.After(3 * time.Second):
		t.Fatal("worker kapanmadi")
	}

	assert.True(t, cancelled.Load(), "the job was not cancelled although the grace period expired")
}

// TestRun_RenewsLeaseWhileWorking verifies the lease is renewed during a
// long-running job.
//
// Without renewal, a job still executing would be handed to another worker because
// its lease expired.
func TestRun_RenewsLeaseWhileWorking(t *testing.T) {
	store := newFakeStore(job("a", "tenant-1"))

	handler := func(ctx context.Context, j *domain.Job) error {
		time.Sleep(80 * time.Millisecond)
		return nil
	}

	cfg := testConfig()
	cfg.LeaseRenewInterval = 10 * time.Millisecond

	p := New(cfg, store, handler, nil)
	require.NoError(t, runFor(t, p, 400*time.Millisecond))

	assert.Greater(t, store.renewals.Load(), int64(2), "lease yenilenmedi")
}

// TestRun_StopsClaimingAfterCancel verifies no new job is claimed after cancellation.
//
// The jobs are spread across separate tenants so the per-tenant quota does not kick in
// and release them quickly; what we want to measure here is the slot limit and the
// cancellation.
func TestRun_StopsClaimingAfterCancel(t *testing.T) {
	jobs := make([]*domain.Job, 0, 100)
	for i := 0; i < 100; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		jobs = append(jobs, job(id, "tenant-"+id))
	}

	store := newFakeStore(jobs...)

	handler := func(ctx context.Context, j *domain.Job) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return nil
		}
	}

	cfg := testConfig()
	cfg.MaxConcurrent = 2
	cfg.MaxPerTenant = 2

	p := New(cfg, store, handler, nil)
	require.NoError(t, runFor(t, p, 150*time.Millisecond))

	before := store.claimed.Load()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, before, store.claimed.Load(), "jobs were still claimed after cancellation")
	assert.Less(t, before, int64(100), "the slot limit did not bound claiming")
}
