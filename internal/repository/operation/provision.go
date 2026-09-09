package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
)

// ErrSlugTaken, slug'in baska bir tenant tarafindan kullanildigini bildirir.
var ErrSlugTaken = errors.New("slug already taken")

// ProvisionRequest, yeni bir tenant kurulumu talebidir.
type ProvisionRequest struct {
	Name  string
	Slug  string
	Email string
	Plan  string

	CreatedBy string

	Scope          string
	IdempotencyKey string
	RequestBody    []byte
	MaxAttempts    int
}

// ProvisionResult, kurulum talebinin sonucudur.
type ProvisionResult struct {
	Operation *domain.Operation
	TenantID  string
	JobID     string
	Replayed  bool
}

// CreateTenantProvision, tenant kaydini, operasyonu, isi ve idempotency
// kaydini TEK transaction icinde yazar.
//
// NEDEN dordu birlikte: aralarindan herhangi birinde surec kapanirsa yetim
// kayit olusur.
//
//   - Tenant var, operasyon yok: kullanicinin goremeyecegi yarim bir kayit.
//   - Operasyon var, is yok: kullaniciya "islemde" gorunen ama hicbir
//     worker'in almayacagi bir kayit.
//   - Is var, idempotency kaydi yok: istemcinin tekrari ikinci bir kurulum
//     baslatir.
//
// Tenant 'pending' durumunda yazilir. 'active' olmasi worker'in kurulum
// adimlarini tamamlamasina baglidir; boylece semasi hazir olmayan bir
// tenant istek kabul edemez.
func (r *Repository) CreateTenantProvision(
	ctx context.Context, req ProvisionRequest,
) (*ProvisionResult, error) {
	fingerprint := Fingerprint(req.RequestBody)

	if req.IdempotencyKey != "" {
		replayed, err := r.lookupProvision(ctx, req, fingerprint)
		if err != nil || replayed != nil {
			return replayed, err
		}
	}

	result, err := r.insertProvision(ctx, req, fingerprint)
	if errors.Is(err, errKeyExists) {
		return r.lookupProvisionStrict(ctx, req, fingerprint)
	}

	return result, err
}

// insertProvision, dort kaydi tek transaction icinde yazar.
func (r *Repository) insertProvision(
	ctx context.Context, req ProvisionRequest, fingerprint string,
) (*ProvisionResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	tenantID, err := insertPendingTenant(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	createReq := CreateRequest{
		TenantID:       tenantID,
		Kind:           domain.KindTenantProvision,
		CreatedBy:      req.CreatedBy,
		Scope:          req.Scope,
		IdempotencyKey: req.IdempotencyKey,
		MaxAttempts:    req.MaxAttempts,
	}

	op, jobID, err := insertOperationAndJobTx(ctx, tx, createReq)
	if err != nil {
		return nil, err
	}

	if req.IdempotencyKey != "" {
		if err := insertIdempotencyKey(ctx, tx, createReq, fingerprint, op.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	return &ProvisionResult{Operation: op, TenantID: tenantID, JobID: jobID}, nil
}

// insertPendingTenant, tenant kaydini 'pending' durumunda yazar.
//
// Sema adi burada uretilir ve kaydedilir, ancak sema HENUZ olusturulmaz.
// Sema olusturma worker'in isidir; API istegi uzun suren bir DDL islemini
// beklememelidir.
func insertPendingTenant(ctx context.Context, tx *sql.Tx, req ProvisionRequest) (string, error) {
	var schemaName string
	if err := tx.QueryRowContext(ctx, `SELECT generate_schema_name($1)`, req.Slug).Scan(&schemaName); err != nil {
		return "", fmt.Errorf("could not generate schema name: %w", err)
	}

	plan := req.Plan
	if plan == "" {
		plan = "free"
	}

	var tenantID string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO tenants (name, slug, email, schema_name, status, plan)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id`,
		req.Name, req.Slug, req.Email, schemaName, plan,
	).Scan(&tenantID)

	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && string(pgErr.Code) == pgUniqueViolation {
			return "", ErrSlugTaken
		}
		return "", fmt.Errorf("could not write tenant: %w", err)
	}

	return tenantID, nil
}

// lookupProvision, anahtar daha once gorulduyse mevcut sonucu dondurur.
func (r *Repository) lookupProvision(
	ctx context.Context, req ProvisionRequest, fingerprint string,
) (*ProvisionResult, error) {
	createReq := CreateRequest{Scope: req.Scope, IdempotencyKey: req.IdempotencyKey}

	existing, err := r.lookupIdempotent(ctx, createReq, fingerprint)
	if err != nil || existing == nil {
		return nil, err
	}

	return &ProvisionResult{
		Operation: existing.Operation,
		TenantID:  existing.Operation.TenantID,
		Replayed:  true,
	}, nil
}

// lookupProvisionStrict, yaris sonrasi kaydin bulunmasini zorunlu kilar.
func (r *Repository) lookupProvisionStrict(
	ctx context.Context, req ProvisionRequest, fingerprint string,
) (*ProvisionResult, error) {
	result, err := r.lookupProvision(ctx, req, fingerprint)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("could not resolve idempotency race: key disappeared")
	}

	return result, nil
}
