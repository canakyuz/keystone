package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
)

// ErrNoJob, devralinabilir is olmadigini bildirir.
var ErrNoJob = errors.New("devralınabilir iş yok")

// Claim, calistirilabilir bir isi belirli sure icin sahiplenir.
//
// SORGU BICIMI
// FOR UPDATE SKIP LOCKED kullanilir. Iki worker ayni anda calistiginda,
// biri satiri kilitler, digeri o satiri atlayip bir sonrakine gecer. Bunun
// alternatifi uygulama tarafinda kilit yonetmekti; veritabani zaten bu isi
// yapabiliyorken ikinci bir kilit katmani eklemek gereksiz karmasiklik ve
// yeni bir hata kaynagi olurdu.
//
// SKIP LOCKED olmasaydi ikinci worker birincinin islemini bitirmesini
// beklerdi ve kuyruk fiilen tek islemciye duserdi.
//
// LEASE
// Is, kalici bir "isleniyor" bayragiyla degil, sureli bir sahiplenmeyle
// isaretlenir. Worker kurulum ortasinda kapanirsa lease suresi dolar ve is
// yeniden devralinabilir. Kalici bayrak kullanilsaydi is sonsuza kadar o
// bayrakla kalirdi.
//
// FENCE
// Her sahiplenmede fence bir artar. Sonuc bildirimi guncel fence degeriyle
// yapilmak zorundadir; boylece lease'i suresi dolmus eski bir worker geri
// dondugunde bildirimi reddedilir.
//
// Karmasiklik: partial index sayesinde yalnizca calistirilabilir isler
// taranir, tamamlanmis isler plana girmez.
func (r *Repository) Claim(
	ctx context.Context, workerID string, leaseDuration time.Duration,
) (*domain.Job, error) {
	if workerID == "" {
		return nil, errors.New("worker kimliği zorunlu")
	}

	job := &domain.Job{}
	var leaseOwner sql.NullString
	var leaseExpires sql.NullTime
	var lastError sql.NullString

	err := r.db.QueryRowContext(ctx, `
		WITH claimable AS (
			SELECT id
			FROM provisioning_jobs
			WHERE status IN ('pending', 'running')
			  AND next_attempt_at <= NOW()
			  AND (lease_expires_at IS NULL OR lease_expires_at <= NOW())
			ORDER BY next_attempt_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE provisioning_jobs j
		SET status           = 'running',
		    lease_owner      = $1,
		    lease_expires_at = NOW() + make_interval(secs => $2),
		    fence            = j.fence + 1,
		    attempts         = j.attempts + 1,
		    updated_at       = NOW()
		FROM claimable c
		WHERE j.id = c.id
		RETURNING j.id, j.operation_id, j.tenant_id, j.status,
		          j.attempts, j.max_attempts, j.next_attempt_at,
		          j.lease_owner, j.lease_expires_at, j.fence,
		          j.last_error, j.created_at, j.updated_at`,
		workerID, int(leaseDuration.Seconds()),
	).Scan(&job.ID, &job.OperationID, &job.TenantID, &job.Status,
		&job.Attempts, &job.MaxAttempts, &job.NextAttempt,
		&leaseOwner, &leaseExpires, &job.Fence,
		&lastError, &job.CreatedAt, &job.UpdatedAt)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNoJob
	case err != nil:
		return nil, fmt.Errorf("iş sahiplenilemedi: %w", err)
	}

	job.LeaseOwner = leaseOwner.String
	job.LastError = lastError.String
	if leaseExpires.Valid {
		job.LeaseExpires = &leaseExpires.Time
	}

	return job, nil
}

// RenewLease, uzun suren bir isin sahiplenmesini uzatir.
//
// Fence dogrulamasi burada da yapilir: eski bir worker, devredilmis bir isin
// lease'ini uzatarak guncel sahibini kesintiye ugratamamalidir.
//
// Fence ARTMAZ. Yenileme yeni bir sahiplenme degildir; artirmak, worker'in
// elindeki fence degerini gecersiz kilardi.
func (r *Repository) RenewLease(
	ctx context.Context, jobID, workerID string, fence int64, leaseDuration time.Duration,
) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET lease_expires_at = NOW() + make_interval(secs => $1),
		    updated_at       = NOW()
		WHERE id = $2 AND lease_owner = $3 AND fence = $4 AND status = 'running'`,
		int(leaseDuration.Seconds()), jobID, workerID, fence,
	)
	if err != nil {
		return fmt.Errorf("lease yenilenemedi: %w", err)
	}

	return requireOneRow(result, domain.ErrStaleFence)
}

// CompleteSuccess, isi ve operasyonu TEK transaction icinde basarili kapatir.
//
// NEDEN tek transaction: ikisi ayri yazilsaydi, aralarinda surec kapandiginda
// kullaniciya "tamamlandi" gorunen ama tenant'i hala aktiflesmemis bir kayit
// kalabilirdi. Kullaniciya verilen sozun kendisi tutarsiz olurdu.
//
// activateTenant true ise tenant ayni transaction icinde aktife alinir.
func (r *Repository) CompleteSuccess(
	ctx context.Context, jobID, workerID string, fence int64, activateTenant bool,
) error {
	return r.completeInTx(ctx, jobID, workerID, fence, func(ctx context.Context, tx *sql.Tx, tenantID string) error {
		if err := markJobSucceeded(ctx, tx, jobID); err != nil {
			return err
		}
		if err := markOperationSucceeded(ctx, tx, jobID); err != nil {
			return err
		}
		if !activateTenant {
			return nil
		}

		_, err := tx.ExecContext(ctx, `
			UPDATE tenants SET status = 'active', updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL`, tenantID)
		if err != nil {
			return fmt.Errorf("tenant aktifleştirilemedi: %w", err)
		}

		return nil
	})
}

// CompleteFailure, basarisiz denemeyi kaydeder.
//
// Deneme hakki kaldiysa is yeniden denenmek uzere planlanir; kalmadiysa
// 'dead' isaretlenir ve operasyon hatayla kapatilir.
func (r *Repository) CompleteFailure(
	ctx context.Context, jobID, workerID string, fence int64,
	errCode, errMessage string, retryAfter time.Duration,
) error {
	return r.completeInTx(ctx, jobID, workerID, fence, func(ctx context.Context, tx *sql.Tx, _ string) error {
		var attempts, maxAttempts int
		err := tx.QueryRowContext(ctx,
			`SELECT attempts, max_attempts FROM provisioning_jobs WHERE id = $1`, jobID,
		).Scan(&attempts, &maxAttempts)
		if err != nil {
			return fmt.Errorf("deneme sayısı okunamadı: %w", err)
		}

		if attempts >= maxAttempts {
			return exhaustJob(ctx, tx, jobID, errCode, errMessage)
		}

		return rescheduleJob(ctx, tx, jobID, errMessage, retryAfter)
	})
}

// completeInTx, fence dogrulamasini yapip verilen islemi transaction icinde
// calistirir.
//
// Fence kontrolu SELECT ... FOR UPDATE ile yapilir: satir kilitlendigi icin,
// kontrol ile guncelleme arasinda baska bir worker isi devralamaz.
func (r *Repository) completeInTx(
	ctx context.Context, jobID, workerID string, fence int64,
	fn func(context.Context, *sql.Tx, string) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction başlatılamadı: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentFence int64
	var currentOwner sql.NullString
	var tenantID string

	err = tx.QueryRowContext(ctx, `
		SELECT fence, lease_owner, tenant_id
		FROM provisioning_jobs
		WHERE id = $1
		FOR UPDATE`, jobID,
	).Scan(&currentFence, &currentOwner, &tenantID)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.ErrNotFound
	case err != nil:
		return fmt.Errorf("iş okunamadı: %w", err)
	}

	// Hem fence hem sahip dogrulanir. Fence tek basina yeterlidir, ancak
	// sahip kontrolu hatali bir cagriyi daha erken ve daha anlasilir bicimde
	// yakalar.
	if currentFence != fence || currentOwner.String != workerID {
		return fmt.Errorf("%w: iş fence=%d owner=%s, bildirim fence=%d owner=%s",
			domain.ErrStaleFence, currentFence, currentOwner.String, fence, workerID)
	}

	if err := fn(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit başarısız: %w", err)
	}

	return nil
}

func markJobSucceeded(ctx context.Context, tx *sql.Tx, jobID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET status = 'succeeded', lease_owner = NULL, lease_expires_at = NULL,
		    last_error = NULL, updated_at = NOW()
		WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("iş tamamlanamadı: %w", err)
	}

	return nil
}

func markOperationSucceeded(ctx context.Context, tx *sql.Tx, jobID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE operations
		SET status = 'succeeded', completed_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT operation_id FROM provisioning_jobs WHERE id = $1)`, jobID)
	if err != nil {
		return fmt.Errorf("operasyon tamamlanamadı: %w", err)
	}

	return nil
}

// exhaustJob, deneme hakki bitmis isi olu isaretler ve operasyonu kapatir.
func exhaustJob(ctx context.Context, tx *sql.Tx, jobID, errCode, errMessage string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET status = 'dead', lease_owner = NULL, lease_expires_at = NULL,
		    last_error = $2, updated_at = NOW()
		WHERE id = $1`, jobID, errMessage)
	if err != nil {
		return fmt.Errorf("iş ölü işaretlenemedi: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE operations
		SET status = 'failed', error_code = $2, error_message = $3,
		    completed_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT operation_id FROM provisioning_jobs WHERE id = $1)`,
		jobID, errCode, errMessage)
	if err != nil {
		return fmt.Errorf("operasyon başarısız işaretlenemedi: %w", err)
	}

	return nil
}

// rescheduleJob, isi yeniden denenmek uzere planlar.
//
// Lease birakilir: is hemen degil, next_attempt_at geldiginde devralinabilir.
func rescheduleJob(ctx context.Context, tx *sql.Tx, jobID, errMessage string, retryAfter time.Duration) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE provisioning_jobs
		SET status           = 'pending',
		    lease_owner      = NULL,
		    lease_expires_at = NULL,
		    next_attempt_at  = NOW() + make_interval(secs => $2),
		    last_error       = $3,
		    updated_at       = NOW()
		WHERE id = $1`, jobID, int(retryAfter.Seconds()), errMessage)
	if err != nil {
		return fmt.Errorf("iş yeniden planlanamadı: %w", err)
	}

	return nil
}

// requireOneRow, guncellemenin tam olarak bir satiri etkilemesini zorunlu kilar.
func requireOneRow(result sql.Result, onMismatch error) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("etkilenen satır sayısı okunamadı: %w", err)
	}
	if affected != 1 {
		return onMismatch
	}

	return nil
}
