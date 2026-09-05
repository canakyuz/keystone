package operation

import (
	"fmt"
	"math"
	"time"
)

// JobStatus, is biriminin durumudur.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"

	// JobDead, deneme hakki tukenmis istir. Otomatik olarak tekrar alinmaz;
	// mudahale gerektirir.
	JobDead JobStatus = "dead"
)

// Job, worker'in devraldigi is birimidir.
type Job struct {
	ID          string
	OperationID string
	TenantID    string
	Status      JobStatus

	Attempts    int
	MaxAttempts int
	NextAttempt time.Time

	// LeaseOwner, isi su an sahiplenen worker'in kimligidir.
	LeaseOwner string

	// LeaseExpires, sahiplenmenin bitis anidir. Bu andan sonra is baska bir
	// worker tarafindan devralinabilir.
	LeaseExpires *time.Time

	// Fence, her sahiplenmede artan sayactir.
	//
	// Sonuc bildirimi guncel fence degeriyle yapilmak zorundadir. Lease
	// suresi dolup is devredildikten sonra geri donen eski worker, elindeki
	// eski fence ile bildirim yapamaz. Yalnizca LeaseOwner'a bakmak yetmez:
	// eski worker kendi adini bilir ve o kontrolu gecerdi.
	Fence int64

	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LeaseValidAt, verilen anda sahiplenmenin hala gecerli olup olmadigini soyler.
func (j *Job) LeaseValidAt(now time.Time) bool {
	return j.LeaseExpires != nil && now.Before(*j.LeaseExpires)
}

// Claimable, isin verilen anda devralinabilir olup olmadigini soyler.
//
// Devralinabilir olmasi icin ya hic sahiplenilmemis olmali, ya da onceki
// sahiplenmenin suresi dolmus olmali. Ayrica yeniden deneme zamani gelmis
// olmalidir.
func (j *Job) Claimable(now time.Time) bool {
	if j.Status != JobPending && j.Status != JobRunning {
		return false
	}
	if now.Before(j.NextAttempt) {
		return false
	}

	return !j.LeaseValidAt(now)
}

// VerifyFence, bildirimin guncel sahiplenmeden geldigini dogrular.
func (j *Job) VerifyFence(fence int64) error {
	if fence != j.Fence {
		return fmt.Errorf("%w: beklenen %d, gelen %d", ErrStaleFence, j.Fence, fence)
	}
	return nil
}

// ExhaustedAfterThisAttempt, bu denemeden sonra hak kalmayacagini soyler.
func (j *Job) ExhaustedAfterThisAttempt() bool {
	return j.Attempts >= j.MaxAttempts
}

// BackoffConfig, yeniden deneme aralarini belirler.
type BackoffConfig struct {
	// Base, ilk bekleme suresidir.
	Base time.Duration

	// Max, tek bir beklemenin ust sinirdir.
	Max time.Duration

	// Jitter, [0,1] araliginda rastgelelik oranidir.
	//
	// Neden gerekli: ayni anda basarisiz olan N is, jitter olmadan ayni anda
	// yeniden dener. Bu, zaten sorunlu olan bagimliliga senkronize bir dalga
	// gonderir (thundering herd).
	Jitter float64
}

// DefaultBackoff, makul varsayilanlari dondurur.
func DefaultBackoff() BackoffConfig {
	return BackoffConfig{Base: time.Second, Max: 5 * time.Minute, Jitter: 0.3}
}

// BackoffFor, verilen deneme sayisi icin bekleme suresini hesaplar.
//
// Us alma tabani ikidir: 1s, 2s, 4s, 8s... Max ile sinirlanir.
// randFraction [0,1) araliginda bir deger almalidir; disaridan verilmesi
// testin deterministik olmasini saglar.
//
// Karmasiklik: O(1).
func BackoffFor(cfg BackoffConfig, attempt int, randFraction float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	// math.Pow yerine ust sinirli kaydirma: cok buyuk attempt degerlerinde
	// tasma olmasin.
	shift := math.Min(float64(attempt-1), 30)
	wait := time.Duration(float64(cfg.Base) * math.Pow(2, shift))

	if wait > cfg.Max || wait <= 0 {
		wait = cfg.Max
	}

	if cfg.Jitter > 0 {
		spread := float64(wait) * math.Min(cfg.Jitter, 1)
		wait += time.Duration(randFraction * spread)
	}

	return wait
}
