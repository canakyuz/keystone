package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
)

// ErrSlugTaken reports that the slug is already used by another tenant.
var ErrSlugTaken = errors.New("slug already taken")

// ProvisionRequest is a request to provision a new tenant.
type ProvisionRequest struct {
	Name  string
	Slug  string
	Email string
	Plan  string

	CreatedBy string

	// RequestID is the correlation id of the HTTP request; see migration 033.
	RequestID string

	// TraceContext is the W3C traceparent of the request; see migration 034.
	TraceContext string

	Scope          string
	IdempotencyKey string
	RequestBody    []byte
	MaxAttempts    int
}

// ProvisionResult is the outcome of a provisioning request.
type ProvisionResult struct {
	Operation *domain.Operation
	TenantID  string
	JobID     string
	Replayed  bool
}

// CreateTenantProvision writes the tenant record, the operation, the job and the
// idempotency record in ONE transaction.
//
// WHY all four together: a crash between any two of them leaves an orphan.
//
//   - Tenant but no operation: a half-record the user cannot see.
//   - Operation but no job: a record that looks "in progress" to the user but that no
//     worker will ever pick up.
//   - Job but no idempotency record: a client retry starts a second provisioning.
//
// The tenant is written in the 'pending' state. Becoming 'active' depends on the
// worker completing the provisioning steps, so a tenant whose schema is not ready
// cannot accept requests.
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

// insertProvision writes the four records in one transaction.
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
		RequestID:      req.RequestID,
		TraceContext:   req.TraceContext,
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

// insertPendingTenant writes the tenant record in the 'pending' state.
//
// The schema name is generated and recorded here, but the schema itself is NOT yet
// created. Creating it is the worker's job; an API request should not wait on a
// long-running DDL operation.
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

// lookupProvision returns the existing result if the key was seen before.
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

// lookupProvisionStrict insists the record be found after a race.
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
