// Package operation, operasyon ve is kayitlarinin kalici depolanmasini saglar.
//
// Bu paketteki garantilerin cogu uygulama kodunda degil, veritabani
// kisitlarinda ve sorgu bicimlerinde durur. Gerekce: iki surec ayni anda
// calistiginda, uygulama seviyesinde yapilan "once kontrol et sonra yaz"
// dizisi yaris kosulunu engellemez.
package operation

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
)

// pgUniqueViolation, PostgreSQL'in benzersizlik ihlali kodudur.
const pgUniqueViolation = "23505"

// defaultIdempotencyTTL, anahtarin varsayilan gecerlilik suresidir.
//
// Idempotency garantisi suresiz DEGILDIR: bu sure dolduktan sonra ayni
// anahtar yeni bir islem yaratir. Sinir API dokumantasyonunda belirtilmelidir.
const defaultIdempotencyTTL = 24 * time.Hour

// Repository, operasyon ve is kayitlarina erisir.
type Repository struct {
	db *sql.DB
}

// New, repository olusturur.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateRequest, yeni bir operasyon talebini tanimlar.
type CreateRequest struct {
	TenantID  string
	Kind      domain.Kind
	CreatedBy string

	// Scope, idempotency anahtarinin gecerli oldugu kapsamdir.
	// Bir musterinin anahtari digerinin istegini eslestirmemelidir.
	Scope string

	// IdempotencyKey bos olabilir; o durumda tekrar korumasi uygulanmaz.
	IdempotencyKey string

	// RequestBody, parmak izi hesaplanacak normalize edilmis istek govdesidir.
	RequestBody []byte

	IdempotencyTTL time.Duration
	MaxAttempts    int
}

// CreateResult, olusturma sonucudur.
type CreateResult struct {
	Operation *domain.Operation
	JobID     string

	// Replayed, istegin daha once gorulmus bir idempotency anahtariyla
	// geldigini ve mevcut operasyonun dondurulduguni bildirir.
	Replayed bool
}

// Fingerprint, istek govdesinin ozetini uretir.
func Fingerprint(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Create, operasyonu, isi ve idempotency kaydini tek transaction icinde yazar.
//
// NEDEN tek transaction: ucu ayri yazilsaydi, aralarinda surec kapandiginda
// yetim kayit olusurdu. Operasyon var ama isi yoksa, kullaniciya "islemde"
// gorunen ama hicbir worker'in almayacagi bir kayit kalirdi.
//
// Ayni idempotency anahtari tekrar geldiginde:
//   - Istek govdesi ayni ise mevcut operasyon dondurulur (Replayed = true).
//   - Farkli ise ErrIdempotencyConflict doner. Sessizce eski sonucu
//     dondurmek istemciyi yanlis yonlendirirdi.
//
// Yaris kosulu iki asamada cozulur. Once mevcut anahtar okunur; bu, sik
// gorulen tekrar durumunu ucuz yoldan karsilar. Ayni anda gelen iki istek
// bu kontrolu birlikte gecerse, ikincisi INSERT sirasinda benzersizlik
// kisitini ihlal eder ve ayni yola duser. Yalnizca okumaya guvenmek yeterli
// degildir; benzersizligi veritabani zorlar.
func (r *Repository) Create(ctx context.Context, req CreateRequest) (*CreateResult, error) {
	fingerprint := Fingerprint(req.RequestBody)

	if req.IdempotencyKey != "" {
		result, err := r.lookupIdempotent(ctx, req, fingerprint)
		if err != nil || result != nil {
			return result, err
		}
	}

	result, err := r.insertAll(ctx, req, fingerprint)
	if errors.Is(err, errKeyExists) {
		// Yaris: baska bir istek arada anahtari yazdi. Onun operasyonunu
		// donduruyoruz.
		return r.lookupIdempotentStrict(ctx, req, fingerprint)
	}

	return result, err
}

// errKeyExists, benzersizlik ihlalini dahili olarak isaretler.
var errKeyExists = errors.New("idempotency key already exists")

// lookupIdempotent, anahtar daha once gorulduyse sonucu dondurur.
// Gorulmediyse (nil, nil) doner.
func (r *Repository) lookupIdempotent(
	ctx context.Context, req CreateRequest, fingerprint string,
) (*CreateResult, error) {
	var operationID, storedFingerprint string

	err := r.db.QueryRowContext(ctx, `
		SELECT operation_id, request_fingerprint
		FROM idempotency_keys
		WHERE scope = $1 AND idempotency_key = $2 AND expires_at > NOW()`,
		req.Scope, req.IdempotencyKey,
	).Scan(&operationID, &storedFingerprint)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("could not read idempotency record: %w", err)
	case storedFingerprint != fingerprint:
		return nil, domain.ErrIdempotencyConflict
	}

	op, err := r.GetOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}

	return &CreateResult{Operation: op, Replayed: true}, nil
}

// lookupIdempotentStrict, yaris sonrasi mevcut kaydin bulunmasini zorunlu kilar.
func (r *Repository) lookupIdempotentStrict(
	ctx context.Context, req CreateRequest, fingerprint string,
) (*CreateResult, error) {
	result, err := r.lookupIdempotent(ctx, req, fingerprint)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("could not resolve idempotency race: key disappeared")
	}

	return result, nil
}

// insertAll, operasyon, is ve idempotency kaydini tek transaction icinde yazar.
func (r *Repository) insertAll(
	ctx context.Context, req CreateRequest, fingerprint string,
) (*CreateResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	op, jobID, err := insertOperationAndJobTx(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	if req.IdempotencyKey != "" {
		if err := insertIdempotencyKey(ctx, tx, req, fingerprint, op.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	return &CreateResult{Operation: op, JobID: jobID}, nil
}

// insertIdempotencyKey, anahtari yazar. Kisit ihlalinde errKeyExists doner.
func insertIdempotencyKey(
	ctx context.Context, tx *sql.Tx, req CreateRequest, fingerprint, operationID string,
) error {
	ttl := req.IdempotencyTTL
	if ttl <= 0 {
		ttl = defaultIdempotencyTTL
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_keys
			(scope, idempotency_key, request_fingerprint, operation_id, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + make_interval(secs => $5))`,
		req.Scope, req.IdempotencyKey, fingerprint, operationID, int(ttl.Seconds()),
	)
	if err == nil {
		return nil
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) && string(pgErr.Code) == pgUniqueViolation {
		return errKeyExists
	}

	return fmt.Errorf("could not write idempotency record: %w", err)
}

// GetOperation, operasyonu kimligiyle getirir.
func (r *Repository) GetOperation(ctx context.Context, id string) (*domain.Operation, error) {
	op := &domain.Operation{}
	var errCode, errMessage, createdBy sql.NullString
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, kind, status, error_code, error_message,
		       created_by, created_at, updated_at, completed_at
		FROM operations
		WHERE id = $1`, id,
	).Scan(&op.ID, &op.TenantID, &op.Kind, &op.Status, &errCode, &errMessage,
		&createdBy, &op.CreatedAt, &op.UpdatedAt, &completedAt)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("could not read operation: %w", err)
	}

	op.ErrorCode = errCode.String
	op.ErrorMessage = errMessage.String
	op.CreatedBy = createdBy.String
	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	return op, nil
}

// insertOperationAndJobTx, operasyon ve ona bagli isi verilen transaction
// icinde yazar. Tenant kurulumu akisi da ayni yardimciyi kullanir, boylece
// iki yolun kayit bicimi ayrisamaz.
func insertOperationAndJobTx(ctx context.Context, tx *sql.Tx, req CreateRequest) (*domain.Operation, string, error) {
	op := &domain.Operation{TenantID: req.TenantID, Kind: req.Kind, Status: domain.StatusPending}

	err := tx.QueryRowContext(ctx, `
		INSERT INTO operations (tenant_id, kind, status, created_by)
		VALUES ($1, $2, 'pending', NULLIF($3, '')::UUID)
		RETURNING id, created_at, updated_at`,
		req.TenantID, string(req.Kind), req.CreatedBy,
	).Scan(&op.ID, &op.CreatedAt, &op.UpdatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("could not write operation: %w", err)
	}

	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	var jobID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO provisioning_jobs (operation_id, tenant_id, status, max_attempts)
		VALUES ($1, $2, 'pending', $3)
		RETURNING id`,
		op.ID, req.TenantID, maxAttempts,
	).Scan(&jobID)
	if err != nil {
		return nil, "", fmt.Errorf("could not write job: %w", err)
	}

	return op, jobID, nil
}
