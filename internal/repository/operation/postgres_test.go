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

// setup, gercek veritabani ve bir test tenant'i hazirlar.
func setup(t *testing.T) (*Repository, *sql.DB, string) {
	t.Helper()

	db := helpers.SetupTestDB(t)
	if db == nil {
		t.Skip("postgres erişilemiyor")
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

// TestCreate_WritesOperationAndJobAtomically, operasyon ve isin birlikte
// olustugunu dogrular.
//
// Ikisi ayri yazilsaydi, aralarinda surec kapandiginda kullaniciya "islemde"
// gorunen ama hicbir worker'in almayacagi bir kayit kalirdi.
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

// TestCreate_SameKeySameBodyReplays, yinelenen istegin yeni bir islem
// yaratmadigini dogrular.
//
// Arıza senaryosu: transaction tamamlandi ama HTTP cevabi istemciye
// ulasmadi. Istemci ayni istegi tekrar gonderir.
func TestCreate_SameKeySameBodyReplays(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()
	body := []byte(`{"slug":"acme"}`)

	first, err := repo.Create(ctx, createRequest(tenantID, "anahtar-2", body))
	require.NoError(t, err)

	second, err := repo.Create(ctx, createRequest(tenantID, "anahtar-2", body))
	require.NoError(t, err)

	assert.True(t, second.Replayed, "tekrar eden istek yeni islem yaratti")
	assert.Equal(t, first.Operation.ID, second.Operation.ID)

	var operationCount int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM operations WHERE tenant_id = $1`, tenantID).Scan(&operationCount))

	assert.Equal(t, 1, operationCount, "yinelenen istek ikinci operasyon yaratti")
}

// TestCreate_SameKeyDifferentBodyConflicts, ayni anahtarin farkli govdeyle
// kullanilmasinin reddedildigini dogrular.
//
// Sessizce eski sonucu dondurmek istemciyi yanlis yonlendirirdi: gonderdigi
// istegin islendigini sanirdi.
func TestCreate_SameKeyDifferentBodyConflicts(t *testing.T) {
	repo, _, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "anahtar-3", []byte(`{"slug":"acme"}`)))
	require.NoError(t, err)

	_, err = repo.Create(ctx, createRequest(tenantID, "anahtar-3", []byte(`{"slug":"baska"}`)))

	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// TestCreate_ConcurrentSameKeyProducesOneOperation, ayni anahtarla ayni anda
// gelen isteklerin tek operasyon urettigini dogrular.
//
// "Once var mi diye bak, yoksa ekle" dizisi tek basina bunu saglamaz: iki
// istek kontrolu birlikte gecip iki ayri operasyon yaratabilir. Benzersizligi
// veritabani kisiti zorlar.
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

// TestClaim_OnlyOneWorkerGetsTheJob, iki worker'in ayni isi devralamadigini
// dogrular.
//
// FOR UPDATE SKIP LOCKED olmasaydi ikinci worker birincinin islemini
// bitirmesini bekler ve kuyruk fiilen tek islemciye duserdi.
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

// TestClaim_ExpiredLeaseIsReclaimable, cokmus worker'in isinin devralinabildigini
// dogrular.
//
// Arıza senaryosu: worker isi aldi ve baslamadan kapandi. Kalici bir
// "isleniyor" bayragi kullanilsaydi is sonsuza kadar o bayrakla kalirdi.
func TestClaim_ExpiredLeaseIsReclaimable(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	first, err := repo.Claim(ctx, "worker-coken", leaseDuration)
	require.NoError(t, err)

	// Ikinci worker, lease gecerliyken devralamaz.
	_, err = repo.Claim(ctx, "worker-ikinci", leaseDuration)
	assert.ErrorIs(t, err, ErrNoJob, "gecerli lease'e ragmen is devralindi")

	// Worker coktu: lease'i gecmise al.
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

// TestComplete_StaleWorkerIsRejected, gecikmis worker bildiriminin
// reddedildigini dogrular.
//
// Arıza senaryosu: eski worker lease'ini kaybettikten sonra geri dondu ve
// "tamamlandi" bildirdi. Yalnizca lease_owner'a bakmak yetmezdi; eski worker
// kendi adini bilir ve o kontrolu gecerdi.
func TestComplete_StaleWorkerIsRejected(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	stale, err := repo.Claim(ctx, "worker-eski", leaseDuration)
	require.NoError(t, err)

	_, err = db.Exec(
		`UPDATE provisioning_jobs SET lease_expires_at = NOW() - interval '1 second' WHERE id = $1`,
		stale.ID)
	require.NoError(t, err)

	current, err := repo.Claim(ctx, "worker-yeni", leaseDuration)
	require.NoError(t, err)

	// Eski worker elindeki fence ile bildirmeye calisir.
	err = repo.CompleteSuccess(ctx, stale.ID, "worker-eski", stale.Fence, true)
	assert.ErrorIs(t, err, domain.ErrStaleFence, "gecikmis worker bildirimi kabul edildi")

	// Guncel worker bildirebilir.
	require.NoError(t, repo.CompleteSuccess(ctx, current.ID, "worker-yeni", current.Fence, true))
}

// TestCompleteSuccess_ActivatesTenantInSameTransaction, tenant aktiflestirme
// ile operasyon kapatmanin birlikte gerceklestigini dogrular.
//
// Ikisi ayri yazilsaydi, aralarinda surec kapandiginda kullaniciya
// "tamamlandi" gorunen ama tenant'i aktiflesmemis bir kayit kalabilirdi.
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

// TestCompleteFailure_RetriesUntilExhausted, deneme hakki bitene kadar
// yeniden denendigini, sonra olu isaretlendigini dogrular.
func TestCompleteFailure_RetriesUntilExhausted(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	// MaxAttempts 3: ilk iki basarisizlik yeniden planlanir, ucuncusu olduruur.
	for attempt := 1; attempt <= 3; attempt++ {
		job, err := repo.Claim(ctx, "worker-1", leaseDuration)
		require.NoErrorf(t, err, "%d. denemede is devralinamadi", attempt)

		// retryAfter sifir: sonraki deneme hemen alinabilsin.
		require.NoError(t, repo.CompleteFailure(
			ctx, job.ID, "worker-1", job.Fence, "schema_error", "şema oluşturulamadı", 0))
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

	// Olu is artik devralinamaz.
	_, err = repo.Claim(ctx, "worker-1", leaseDuration)
	assert.ErrorIs(t, err, ErrNoJob, "olu is devralindi")
}

// TestRenewLease_RejectsStaleFence, eski worker'in lease uzatarak guncel
// sahibi kesintiye ugratamadigini dogrular.
func TestRenewLease_RejectsStaleFence(t *testing.T) {
	repo, db, tenantID := setup(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, createRequest(tenantID, "", []byte(`{}`)))
	require.NoError(t, err)

	stale, err := repo.Claim(ctx, "worker-eski", leaseDuration)
	require.NoError(t, err)

	_, err = db.Exec(
		`UPDATE provisioning_jobs SET lease_expires_at = NOW() - interval '1 second' WHERE id = $1`,
		stale.ID)
	require.NoError(t, err)

	current, err := repo.Claim(ctx, "worker-yeni", leaseDuration)
	require.NoError(t, err)

	err = repo.RenewLease(ctx, stale.ID, "worker-eski", stale.Fence, leaseDuration)
	assert.ErrorIs(t, err, domain.ErrStaleFence, "eski worker lease uzatabildi")

	require.NoError(t, repo.RenewLease(ctx, current.ID, "worker-yeni", current.Fence, leaseDuration))
}

// TestGetOperation_NotFound, olmayan operasyonun ErrNotFound dondurdugunu
// dogrular.
func TestGetOperation_NotFound(t *testing.T) {
	repo, _, _ := setup(t)

	_, err := repo.GetOperation(context.Background(), "00000000-0000-0000-0000-000000000000")

	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func workerName(i int) string {
	return "worker-" + string(rune('a'+i%26))
}
