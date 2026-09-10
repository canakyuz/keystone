// Package operation provides durable storage for operation and job records.
//
// Most of the guarantees in this package live in database constraints and query
// shapes rather than in application code. The reason: with two processes running at
// once, a "check then write" sequence performed at the application level does not
// prevent the race.
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

// pgUniqueViolation is PostgreSQL's unique violation code.
const pgUniqueViolation = "23505"

// defaultIdempotencyTTL is the key's default validity window.
//
// The idempotency guarantee is NOT indefinite: once this window expires the same key
// creates a new operation. The limit must be stated in the API documentation.
const defaultIdempotencyTTL = 24 * time.Hour

// Repository accesses operation and job records.
type Repository struct {
	db *sql.DB
}

// New creates the repository.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateRequest describes a request for a new operation.
type CreateRequest struct {
	TenantID  string
	Kind      domain.Kind
	CreatedBy string

	// Scope is where the idempotency key is valid.
	// One customer's key must not match another customer's request.
	Scope string

	// IdempotencyKey may be empty, in which case no replay protection applies.
	IdempotencyKey string

	// RequestID is the correlation id of the HTTP request. It is stored so the worker's
	// log lines can be tied back to the request that asked for the work; see migration
	// 033. It may be empty.
	RequestID string

	// TraceContext is the W3C traceparent of the request; see migration 034. It may be
	// empty.
	TraceContext string

	// RequestBody is the normalised request body the fingerprint is computed from.
	RequestBody []byte

	IdempotencyTTL time.Duration
	MaxAttempts    int
}

// CreateResult is the outcome of creation.
type CreateResult struct {
	Operation *domain.Operation
	JobID     string

	// Replayed reports that the request arrived with an idempotency key already seen,
	// and that the existing operation was returned.
	Replayed bool
}

// Fingerprint produces the digest of a request body.
func Fingerprint(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Create writes the operation, the job and the idempotency record in one transaction.
//
// WHY one transaction: written as three separate steps, a crash in between would
// leave an orphan. With an operation but no job, the user would see a record that
// looks "in progress" but that no worker will ever pick up.
//
// When the same idempotency key arrives again:
//   - If the request body is identical, the existing operation is returned
//     (Replayed = true).
//   - If it differs, ErrIdempotencyConflict is returned. Silently returning the old
//     result would mislead the client.
//
// The race is resolved in two stages. The existing key is read first, which handles
// the common repeat case cheaply. If two simultaneous requests pass that check
// together, the second violates the uniqueness constraint during INSERT and lands on
// the same path. Relying on the read alone is not enough; the database enforces
// uniqueness.
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
		// A race: another request wrote the key in between. We return its operation.
		return r.lookupIdempotentStrict(ctx, req, fingerprint)
	}

	return result, err
}

// errKeyExists flags a uniqueness violation internally.
var errKeyExists = errors.New("idempotency key already exists")

// lookupIdempotent returns the result if the key was seen before.
// It returns (nil, nil) if it was not.
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

// lookupIdempotentStrict insists the existing record be found after a race.
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

// insertAll writes the operation, job and idempotency record in one transaction.
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

// insertIdempotencyKey writes the key. It returns errKeyExists on a constraint
// violation.
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

// GetOperation fetches an operation by its id.
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

// insertOperationAndJobTx writes an operation and its job inside the given
// transaction. The tenant provisioning flow uses the same helper, so the two paths
// cannot drift apart in how they record things.
func insertOperationAndJobTx(ctx context.Context, tx *sql.Tx, req CreateRequest) (*domain.Operation, string, error) {
	op := &domain.Operation{TenantID: req.TenantID, Kind: req.Kind, Status: domain.StatusPending}

	err := tx.QueryRowContext(ctx, `
		INSERT INTO operations (tenant_id, kind, status, created_by, request_id, trace_context)
		VALUES ($1, $2, 'pending', NULLIF($3, '')::UUID, NULLIF($4, ''), NULLIF($5, ''))
		RETURNING id, created_at, updated_at`,
		req.TenantID, string(req.Kind), req.CreatedBy, req.RequestID, req.TraceContext,
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
