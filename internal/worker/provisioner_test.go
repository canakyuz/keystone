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

// fakeStore, gercek bir kuyruk gibi davranan bellek ici depodur.
//
// Burada gercek PostgreSQL kullanilmaz cunku sinanan sey veritabani
// garantileri degil, worker'in kapasite ve kapanma davranisidir. Veritabani
// garantileri internal/repository/operation testlerinde gercek Postgres'e
// karsi dogrulanir.
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

// runFor, worker'i belirli sure calistirip durdurur.
func runFor(t *testing.T, p *Provisioner, d time.Duration) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	return p.Run(ctx)
}

// TestRun_RespectsMaxConcurrent, toplam eszamanlilik sinirinin asilmadigini
// dogrular.
//
// Her is icin sinirsiz goroutine acmak kolaydir; sistemin kaynaklarini
// koruyarak ilerlemesi tasarim gerektirir.
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
		"toplam eszamanlilik siniri asildi")
	assert.Greater(t, store.succeeded.Load(), int64(0), "hic is tamamlanmadi")
}

// TestRun_RespectsPerTenantLimit, tek bir tenant'in butun kapasiteyi
// tuketemedigini dogrular.
//
// Yalnizca toplam sinir olsaydi, cok isi olan bir tenant butun yuvalari
// doldurup digerlerini bekletebilirdi.
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
		"tek tenant icin eszamanlilik siniri asildi")
}

// TestRun_HandlerFailureIsReported, basarisiz isin bildirildigini dogrular.
func TestRun_HandlerFailureIsReported(t *testing.T) {
	store := newFakeStore(job("a", "tenant-1"))

	handler := func(ctx context.Context, j *domain.Job) error {
		return errors.New("şema oluşturulamadı")
	}

	p := New(testConfig(), store, handler, nil)
	require.NoError(t, runFor(t, p, 300*time.Millisecond))

	assert.Equal(t, int64(1), store.failed.Load())
	assert.Zero(t, store.succeeded.Load())
}

// TestRun_PanicDoesNotKillWorker, panige duşen isin worker'i dusurmedigini
// ve basarisiz sayildigini dogrular.
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

	assert.Equal(t, int64(1), store.failed.Load(), "panikleyen is basarisiz sayilmadi")
	assert.Equal(t, int64(1), store.succeeded.Load(), "worker paniktan sonra durdu")
}

// TestRun_GracefulShutdownWaitsForRunningJobs, kapanirken calisan isin
// tamamlanmasinin beklendigini dogrular.
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

	// Is basladiktan hemen sonra kapatma sinyali gonder.
	require.NoError(t, runFor(t, p, 50*time.Millisecond))

	assert.True(t, completed.Load(), "kapanma calisan isi yarida kesti")
	assert.Equal(t, int64(1), store.succeeded.Load())
}

// TestRun_ShutdownGraceExpiryCancelsJobs, nazik surenin dolmasi durumunda
// islerin iptal edildigini dogrular.
//
// Iptal edilen isler tamamlanmis sayilmaz; lease sureleri dolunca baska bir
// worker tarafindan devralinirlar.
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
		assert.Error(t, err, "nazik sure dolmasina ragmen hata bildirilmedi")
	case <-time.After(3 * time.Second):
		t.Fatal("worker kapanmadi")
	}

	assert.True(t, cancelled.Load(), "sure dolmasina ragmen is iptal edilmedi")
}

// TestRun_RenewsLeaseWhileWorking, uzun suren is sirasinda lease'in
// yenilendigini dogrular.
//
// Yenilenmezse, henuz calisan bir is lease suresi doldugu icin baska bir
// worker'a devredilirdi.
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

// TestRun_StopsClaimingAfterCancel, iptalden sonra yeni is alinmadigini
// dogrular.
//
// Isler ayri tenant'lara dagitilir ki tenant kotasi devreye girip isleri
// hizlica geri birakmasin; olcmek istedigimiz sey yuva siniri ve iptal.
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

	assert.Equal(t, before, store.claimed.Load(), "iptalden sonra is alinmaya devam edildi")
	assert.Less(t, before, int64(100), "yuva siniri is alimini sinirlamadi")
}
