package outbox

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/canakyuz/keystone/internal/domain/outbox"
	"github.com/canakyuz/keystone/test/helpers"
)

// setup prepares a database, a tenant and one active endpoint.
func setup(t *testing.T) (*Repository, *sql.DB, string, string) {
	t.Helper()

	db := helpers.SetupTestDB(t)
	tenant := helpers.CreateTestTenant(t, db, "outbox-test")

	var endpointID string
	require.NoError(t, db.QueryRow(`
		INSERT INTO tenant_webhook_endpoints (tenant_id, url, secret)
		VALUES ($1, 'https://receiver.test/hook', 'shhh')
		RETURNING id`, tenant.ID).Scan(&endpointID))

	return New(db), db, tenant.ID, endpointID
}

// appendOne writes a single event through a transaction, as production does.
func appendOne(t *testing.T, repo *Repository, db *sql.DB, tenantID string) int {
	t.Helper()

	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	count, err := repo.AppendTx(ctx, tx, AppendRequest{
		TenantID:      tenantID,
		AggregateType: "tenant",
		AggregateID:   tenantID,
		EventType:     "tenant.provisioned",
		Payload:       map[string]string{"tenant_id": tenantID},
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	return count
}

// TestAppendTx_IsRolledBackWithItsTransaction is the claim the whole pattern rests on.
//
// If the event could survive a rolled-back transaction, a notification would go out for
// work that never happened. If it could be lost while the work committed, a tenant would
// go live with nobody ever told. The event and the business data have to share a fate,
// and sharing a transaction is what makes that true rather than likely.
func TestAppendTx_IsRolledBackWithItsTransaction(t *testing.T) {
	repo, db, tenantID, _ := setup(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	count, err := repo.AppendTx(ctx, tx, AppendRequest{
		TenantID:      tenantID,
		AggregateType: "tenant",
		AggregateID:   tenantID,
		EventType:     "tenant.provisioned",
		Payload:       map[string]string{"tenant_id": tenantID},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, tx.Rollback())

	var remaining int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM outbox_events WHERE tenant_id = $1`, tenantID).Scan(&remaining))

	assert.Zero(t, remaining, "the event outlived the transaction that produced it")
}

// TestAppendTx_WritesOnePerActiveEndpoint verifies fan-out and the active filter.
//
// One row per endpoint, so an endpoint that is down cannot hold up delivery to one that
// is up, and a single retry counter never has to describe two receivers failing at
// different rates.
func TestAppendTx_WritesOnePerActiveEndpoint(t *testing.T) {
	repo, db, tenantID, _ := setup(t)

	_, err := db.Exec(`
		INSERT INTO tenant_webhook_endpoints (tenant_id, url, secret, active)
		VALUES ($1, 'https://second.test/hook', 's2', TRUE),
		       ($1, 'https://disabled.test/hook', 's3', FALSE)`, tenantID)
	require.NoError(t, err)

	assert.Equal(t, 2, appendOne(t, repo, db, tenantID),
		"a disabled endpoint received an event, or an active one did not")
}

// TestAppendTx_WithoutEndpointsWritesNothing verifies silence is not an error.
//
// A tenant that asked for no notifications is owed none. Recording an undeliverable event
// would fill the table with rows whose only possible outcome is dead.
func TestAppendTx_WithoutEndpointsWritesNothing(t *testing.T) {
	repo, db, _, endpointID := setup(t)

	_, err := db.Exec(`DELETE FROM tenant_webhook_endpoints WHERE id = $1`, endpointID)
	require.NoError(t, err)

	var tenantID string
	require.NoError(t, db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tenantID))

	assert.Zero(t, appendOne(t, repo, db, tenantID))
}

// TestClaim_OnlyOneWorkerGetsTheEvent verifies an event is delivered by one worker.
//
// Without FOR UPDATE SKIP LOCKED two workers would both send it, which turns
// at-least-once into at-least-twice for every event and every receiver.
func TestClaim_OnlyOneWorkerGetsTheEvent(t *testing.T) {
	repo, db, tenantID, _ := setup(t)
	ctx := context.Background()

	require.Equal(t, 1, appendOne(t, repo, db, tenantID))

	first, err := repo.Claim(ctx, "worker-1", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, first)

	_, err = repo.Claim(ctx, "worker-2", time.Minute)

	assert.ErrorIs(t, err, domain.ErrNoEvent, "a second worker claimed the same event")
}

// TestClaim_ConcurrentWorkersNeverShareAnEvent is the same claim under real concurrency.
func TestClaim_ConcurrentWorkersNeverShareAnEvent(t *testing.T) {
	repo, db, tenantID, _ := setup(t)
	ctx := context.Background()

	const events = 20
	for i := 0; i < events; i++ {
		require.Equal(t, 1, appendOne(t, repo, db, tenantID))
	}

	var claimed atomic.Int64
	seen := sync.Map{}
	var duplicates atomic.Int64

	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			for {
				event, err := repo.Claim(ctx, "worker-"+string(rune('a'+worker)), time.Minute)
				if err != nil {
					return
				}

				if _, loaded := seen.LoadOrStore(event.ID, true); loaded {
					duplicates.Add(1)
				}
				claimed.Add(1)
			}
		}(w)
	}
	wg.Wait()

	assert.Equal(t, int64(events), claimed.Load(), "not every event was claimed exactly once")
	assert.Zero(t, duplicates.Load(), "an event was claimed by two workers")
}

// TestMarkDelivered_StaleWorkerIsRejected verifies fencing holds here too.
//
// A worker whose lease expired may still be waiting on a slow receiver. When it finally
// returns, the event has moved on, and letting it write "delivered" would mark an event
// complete that the current worker is still trying to send.
func TestMarkDelivered_StaleWorkerIsRejected(t *testing.T) {
	repo, db, tenantID, _ := setup(t)
	ctx := context.Background()

	require.Equal(t, 1, appendOne(t, repo, db, tenantID))

	stale, err := repo.Claim(ctx, "worker-old", time.Minute)
	require.NoError(t, err)

	// The old worker's lease lapses and another worker takes over.
	_, err = db.Exec(`UPDATE outbox_events SET lease_expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, stale.ID)
	require.NoError(t, err)

	current, err := repo.Claim(ctx, "worker-new", time.Minute)
	require.NoError(t, err)
	require.Equal(t, stale.ID, current.ID)

	assert.ErrorIs(t,
		repo.MarkDelivered(ctx, stale.ID, "worker-old", stale.Fence),
		domain.ErrStaleFence)

	assert.NoError(t, repo.MarkDelivered(ctx, current.ID, "worker-new", current.Fence))
}

// TestMarkFailed_RetriesUntilExhaustedThenDies verifies the dead-letter state.
//
// An endpoint that has refused every attempt with increasing backoff will refuse the
// next one too, and a queue that retries it forever spends capacity that working
// endpoints need. Dead is a state somebody has to look at, which is the intent.
func TestMarkFailed_RetriesUntilExhaustedThenDies(t *testing.T) {
	repo, db, tenantID, _ := setup(t)
	ctx := context.Background()

	require.Equal(t, 1, appendOne(t, repo, db, tenantID))

	_, err := db.Exec(`UPDATE outbox_events SET max_attempts = 3 WHERE tenant_id = $1`, tenantID)
	require.NoError(t, err)

	for attempt := 1; attempt <= 3; attempt++ {
		event, claimErr := repo.Claim(ctx, "worker-1", time.Minute)
		require.NoErrorf(t, claimErr, "attempt %d could not claim", attempt)

		// Zero backoff so the next attempt is immediately claimable.
		require.NoError(t, repo.MarkFailed(ctx, event.ID, "worker-1", event.Fence, "receiver answered 500", 0))
	}

	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM outbox_events WHERE tenant_id = $1`, tenantID).Scan(&status))
	assert.Equal(t, string(domain.StatusDead), status)

	_, err = repo.Claim(ctx, "worker-1", time.Minute)
	assert.ErrorIs(t, err, domain.ErrNoEvent, "a dead event was claimed again")
}

// TestClaim_RespectsTheRetrySchedule verifies backoff is honoured.
//
// Without it a failing endpoint is retried in a tight loop, which is a denial of service
// aimed at a receiver that is already having a bad day.
func TestClaim_RespectsTheRetrySchedule(t *testing.T) {
	repo, db, tenantID, _ := setup(t)
	ctx := context.Background()

	require.Equal(t, 1, appendOne(t, repo, db, tenantID))

	event, err := repo.Claim(ctx, "worker-1", time.Minute)
	require.NoError(t, err)

	require.NoError(t, repo.MarkFailed(ctx, event.ID, "worker-1", event.Fence, "boom", time.Hour))

	_, err = repo.Claim(ctx, "worker-1", time.Minute)
	assert.ErrorIs(t, err, domain.ErrNoEvent, "an event was claimed before its retry was due")
}

// TestClaim_SkipsDeactivatedEndpoints verifies an operator can stop delivery.
//
// Deactivating an endpoint is what somebody does while a receiver is being repaired. If
// events kept going out anyway the switch would be decorative.
func TestClaim_SkipsDeactivatedEndpoints(t *testing.T) {
	repo, db, tenantID, endpointID := setup(t)
	ctx := context.Background()

	require.Equal(t, 1, appendOne(t, repo, db, tenantID))

	_, err := db.Exec(`UPDATE tenant_webhook_endpoints SET active = FALSE WHERE id = $1`, endpointID)
	require.NoError(t, err)

	_, err = repo.Claim(ctx, "worker-1", time.Minute)
	assert.ErrorIs(t, err, domain.ErrNoEvent)
}

// TestDepth_ReportsZeroesRatherThanOmitting verifies the gauge is always publishable.
//
// A series that stops being published is indistinguishable from a scrape failure, and an
// alert on a missing series is far noisier than one on a zero.
func TestDepth_ReportsZeroesRatherThanOmitting(t *testing.T) {
	repo, _, _, _ := setup(t)

	depth, err := repo.Depth(context.Background())
	require.NoError(t, err)

	for _, status := range []string{"pending", "delivering", "dead"} {
		_, ok := depth[status]
		assert.Truef(t, ok, "status %q is missing from the depth report", status)
	}
}
