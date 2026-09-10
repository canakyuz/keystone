package operation

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	"github.com/canakyuz/keystone/test/helpers"
)

const leaseDuration = 30 * time.Second

// setup prepares a real database and one test tenant.
func setup(t *testing.T) (*Repository, *sql.DB, string) {
	t.Helper()

	db := helpers.SetupTestDB(t)
	if db == nil {
		t.Skip("postgres unreachable")
	}

	tenant := helpers.CreateTestTenant(t, db, "op-test")

	return New(db), db, tenant.ID
}

func createRequest(tenantID, key string, body []byte) CreateRequest {
	return CreateRequest{
		TenantID:       tenantID,
		Kind:           domain.KindTenantProvision,
		Scope:          "account:test",
		IdempotencyKey: key,
		RequestBody:    body,
		MaxAttempts:    3,
	}
}

// TestCreate_WritesOperationAndJobAtomically verifies the operation and the job are
// created together.
//
// Written separately, a crash in between would leave a record that looks "in
// progress" to the user but that no worker will ever pick up.
func TestCreate_WritesOperationAndJobAtomically(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	result, err := repo.Create(ctx, createRequest(tenantID, "anahtar-1", []byte(`{"slug":"acme"}`)))
	require.NoError(t, err)

	assert.False(t, result.Replayed)
	assert.Equal(t, domain.StatusPending, result.Operation.Status)
	assert.NotEmpty(t, result.JobID)

	var jobCount int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM provisioning_jobs WHERE operation_id = $1`,
		result.Operation.ID).Scan(&jobCount))

	assert.Equal(t, 1, jobCount, "operasyon var ama isi yok")
}

// TestCreate_SameKeySameBodyReplays verifies a duplicate request does not create new
// work.
//
// Failure scenario: the transaction committed but the HTTP response never reached the
// client. The client sends the same request again.
func TestCreate_SameKeySameBodyReplays(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()
	body := []byte(`{"slug":"acme"}`)

	first, err := repo.Create(ctx, createRequest(tenantID, "anahtar-2", body))
	require.NoError(t, err)

	second, err := repo.Create(ctx, createRequest(tenantID, "anahtar-2", body))
	require.NoError(t, err)

	assert.True(t, second.Replayed, "a repeated request created new work")
	assert.Equal(t, first.Operation.ID, second.Operation.ID)

	var operationCount int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM operations WHERE tenant_id = $1`, tenantID).Scan(&operationCount))

	assert.Equal(t, 1, operationCount, "a duplicate request created a second operation")
}

// TestCreate_SameKeyDifferentBodyConflicts verifies that reusing a key with a
// different body is rejected.
//
// Silently returning the old result would mislead the client: it would believe the
// request it sent had been processed.
func TestCreate_SameKeyDifferentBodyConflicts(t *testing.T) {
	repo, _, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "anahtar-3", []byte(`{"slug":"acme"}`)))
	require.NoError(t, err)

	_, err = repo.Create(ctx, createRequest(tenantID, "anahtar-3", []byte(`{"slug":"baska"}`)))

	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// TestCreate_ConcurrentSameKeyProducesOneOperation verifies that simultaneous
// requests with the same key produce a single operation.
//
// The "check whether it exists, insert if not" sequence does not achieve this on its
// own: two requests can pass the check together and create two operations. The
// database constraint is what enforces uniqueness.
func TestCreate_ConcurrentSameKeyProducesOneOperation(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()
	body := []byte(`{"slug":"acme"}`)

	const concurrent = 20
	var wg sync.WaitGroup
	ids := make([]string, concurrent)
	var failures atomic.Int64

	wg.Add(concurrent)
	for i := 0; i < concurrent; i++ {
		go func(idx int) {
			defer wg.Done()
			result, err := repo.Create(ctx, createRequest(tenantID, "anahtar-yaris", body))
			if err != nil {
				failures.Add(1)
				return
			}
			ids[idx] = result.Operation.ID
		}(i)
	}
	wg.Wait()

	assert.Zero(t, failures.Load(), "eszamanli yinelenen istek hata verdi")

	unique := make(map[string]bool)
	for _, id := range ids {
		if id != "" {
			unique[id] = true
		}
	}
	assert.Len(t, unique, 1, "eszamanli ayni anahtar birden fazla operasyon uretti")

	var operationCount int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM operations WHERE tenant_id = $1`, tenantID).Scan(&operationCount))

	assert.Equal(t, 1, operationCount)
}

// TestClaim_OnlyOneWorkerGetsTheJob verifies two workers cannot claim the same job.
//
// Without FOR UPDATE SKIP LOCKED the second worker would wait for the first's
// transaction to finish and the queue would effectively collapse to one processor.
func TestClaim_OnlyOneWorkerGetsTheJob(t *testing.T) {
	repo, _, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	const workers = 10
	var claimed atomic.Int64
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(idx int) {
			defer wg.Done()
			job, err := repo.Claim(ctx, workerName(idx), leaseDuration)
			if err == nil && job != nil {
				claimed.Add(1)
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int64(1), claimed.Load(), "ayni is birden fazla worker tarafindan devralindi")
}

// TestClaim_ExpiredLeaseIsReclaimable verifies a crashed worker's job can be taken
// over.
//
// Failure scenario: the worker claimed the job and died before starting. With a
// persistent "processing" flag the job would stay stuck behind it forever.
func TestClaim_ExpiredLeaseIsReclaimable(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	first, err := repo.Claim(ctx, "worker-coken", leaseDuration)
	require.NoError(t, err)

	// A second worker cannot claim while the lease is valid.
	_, err = repo.Claim(ctx, "worker-ikinci", leaseDuration)
	assert.ErrorIs(t, err, ErrNoJob, "the job was claimed despite a valid lease")

	// The worker crashed: push the lease into the past.
	_, err = db.Exec(
		`UPDATE provisioning_jobs SET lease_expires_at = NOW() - interval '1 second' WHERE id = $1`,
		first.ID)
	require.NoError(t, err)

	second, err := repo.Claim(ctx, "worker-ikinci", leaseDuration)
	require.NoError(t, err, "suresi dolmus lease devralinamadi")

	assert.Equal(t, first.ID, second.ID)
	assert.Greater(t, second.Fence, first.Fence, "devralmada fence artmadi")
	assert.Equal(t, "worker-ikinci", second.LeaseOwner)
}

// TestComplete_StaleWorkerIsRejected verifies a late report from a worker is
// rejected.
//
// Failure scenario: the old worker lost its lease, then came back and reported
// "completed". Checking lease_owner alone would not be enough; the old worker knows
// its own name and would pass that check.
func TestComplete_StaleWorkerIsRejected(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	stale, err := repo.Claim(ctx, "worker-old", leaseDuration)
	require.NoError(t, err)

	_, err = db.Exec(
		`UPDATE provisioning_jobs SET lease_expires_at = NOW() - interval '1 second' WHERE id = $1`,
		stale.ID)
	require.NoError(t, err)

	current, err := repo.Claim(ctx, "worker-new", leaseDuration)
	require.NoError(t, err)

	// The old worker tries to report with the fence it holds.
	err = repo.CompleteSuccess(ctx, stale.ID, "worker-old", stale.Fence, true)
	assert.ErrorIs(t, err, domain.ErrStaleFence, "gecikmis worker bildirimi kabul edildi")

	// The current worker can report.
	require.NoError(t, repo.CompleteSuccess(ctx, current.ID, "worker-new", current.Fence, true))
}

// TestCompleteSuccess_ActivatesTenantInSameTransaction verifies that activating the
// tenant and closing the operation happen together.
//
// Written separately, a crash in between could leave a record that looks "completed"
// to the user while its tenant was never activated.
func TestCompleteSuccess_ActivatesTenantInSameTransaction(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := db.Exec(`UPDATE tenants SET status = 'pending' WHERE id = $1`, tenantID)
	require.NoError(t, err)

	created, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	job, err := repo.Claim(ctx, "worker-1", leaseDuration)
	require.NoError(t, err)

	require.NoError(t, repo.CompleteSuccess(ctx, job.ID, "worker-1", job.Fence, true))

	op, err := repo.GetOperation(ctx, created.Operation.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, op.Status)
	assert.NotNil(t, op.CompletedAt, "tamamlanan operasyonun bitis zamani yok")

	var tenantStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM tenants WHERE id = $1`, tenantID).Scan(&tenantStatus))
	assert.Equal(t, "active", tenantStatus, "operasyon tamamlandi ama tenant aktiflesmedi")
}

// TestCompleteFailure_RetriesUntilExhausted verifies the job is retried until its
// attempts run out, then marked dead.
func TestCompleteFailure_RetriesUntilExhausted(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	// MaxAttempts is 3: the first two failures reschedule, the third kills the job.
	for attempt := 1; attempt <= 3; attempt++ {
		job, err := repo.Claim(ctx, "worker-1", leaseDuration)
		require.NoErrorf(t, err, "%d. denemede is devralinamadi", attempt)

		// retryAfter is zero so the next attempt is immediately claimable.
		require.NoError(t, repo.CompleteFailure(
			ctx, job.ID, "worker-1", job.Fence, "schema_error", "could not create schema", 0))
	}

	var jobStatus string
	require.NoError(t, db.QueryRow(
		`SELECT status FROM provisioning_jobs WHERE operation_id = $1`,
		created.Operation.ID).Scan(&jobStatus))
	assert.Equal(t, "dead", jobStatus, "hak tukendigi halde is olu isaretlenmedi")

	op, err := repo.GetOperation(ctx, created.Operation.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusFailed, op.Status)
	assert.Equal(t, "schema_error", op.ErrorCode)

	// A dead job can no longer be claimed.
	_, err = repo.Claim(ctx, "worker-1", leaseDuration)
	assert.ErrorIs(t, err, ErrNoJob, "olu is devralindi")
}

// TestRenewLease_RejectsStaleFence verifies an old worker cannot disrupt the current
// holder by extending the lease.
func TestRenewLease_RejectsStaleFence(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	stale, err := repo.Claim(ctx, "worker-old", leaseDuration)
	require.NoError(t, err)

	_, err = db.Exec(
		`UPDATE provisioning_jobs SET lease_expires_at = NOW() - interval '1 second' WHERE id = $1`,
		stale.ID)
	require.NoError(t, err)

	current, err := repo.Claim(ctx, "worker-new", leaseDuration)
	require.NoError(t, err)

	err = repo.RenewLease(ctx, stale.ID, "worker-old", stale.Fence, leaseDuration)
	assert.ErrorIs(t, err, domain.ErrStaleFence, "the old worker was able to extend the lease")

	require.NoError(t, repo.RenewLease(ctx, current.ID, "worker-new", current.Fence, leaseDuration))
}

// TestGetOperation_NotFound verifies a missing operation returns ErrNotFound.
func TestGetOperation_NotFound(t *testing.T) {
	repo, _, _ := setup(t)

	_, err := repo.GetOperation(context.Background(), "00000000-0000-0000-0000-000000000000")

	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func workerName(i int) string {
	return "worker-" + string(rune('a'+i%26))
}

// TestClaim_CarriesTheRequestID verifies the diagnostic chain reaches the worker.
//
// Provisioning is asynchronous, so the request that asked for a tenant and the worker
// that builds it are in different processes and, for a retried job, possibly hours
// apart. Without the request id on the claimed job the two halves can only be matched by
// timestamp, which stops working the moment there is more than one tenant being
// provisioned at a time.
func TestClaim_CarriesTheRequestID(t *testing.T) {
	repo, _, tenantID := setup(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, CreateRequest{
		TenantID:  tenantID,
		Kind:      domain.KindTenantProvision,
		RequestID: "req-abc-123",
	})
	require.NoError(t, err)
	require.NotNil(t, created.Operation)

	job, err := repo.Claim(ctx, "worker-1", time.Minute)
	require.NoError(t, err)

	assert.Equal(t, "req-abc-123", job.RequestID,
		"the claimed job lost the request id, so the worker's logs cannot be tied to the request")
}

// TestClaim_ToleratesAMissingRequestID verifies an operation without one still claims.
//
// The column is nullable: operations created before migration 033 have no request id,
// and a scan that cannot handle NULL would take the worker down on the oldest rows in
// the table.
func TestClaim_ToleratesAMissingRequestID(t *testing.T) {
	repo, _, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, CreateRequest{
		TenantID: tenantID,
		Kind:     domain.KindTenantProvision,
	})
	require.NoError(t, err)

	job, err := repo.Claim(ctx, "worker-1", time.Minute)
	require.NoError(t, err)

	assert.Empty(t, job.RequestID)
}
