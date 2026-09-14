// Package webhook stores where a tenant wants to be notified.
//
// The destinations it records are the ones the outbox delivers to, so every write here is
// also a decision about where this platform will later send requests. Validation of the
// address happens before it arrives; the dialler in pkg/outbound refuses it again at
// delivery, because a name that resolved to a public address today can resolve to an
// internal one tomorrow.
package webhook

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

// MaxPerTenant caps how many destinations one tenant keeps.
//
// Every event is written once per active endpoint. Without a cap, a tenant could register
// thousands and turn one provisioning into thousands of outbound requests from this platform.
const MaxPerTenant = 10

var (
	// ErrNotFound is returned for an endpoint that does not exist in the caller's tenant.
	// An endpoint of another tenant is the same answer on purpose.
	ErrNotFound = errors.New("webhook endpoint not found")

	// ErrLimitReached is returned when the tenant already holds MaxPerTenant endpoints.
	ErrLimitReached = errors.New("webhook endpoint limit reached")
)

// Endpoint is a destination as a reader sees it. The secret is not part of it; only its
// last four characters are, so a person can tell two endpoints' secrets apart.
type Endpoint struct {
	ID         string
	URL        string
	Active     bool
	SecretHint string
	CreatedAt  time.Time

	Delivered int
	Pending   int
	Dead      int
}

// Repository reads and writes tenant_webhook_endpoints.
type Repository struct {
	db    *sql.DB
	trail *auditrepo.Repository
}

// New creates the repository. The trail is required: every change here is recorded in the
// transaction that makes it.
func New(db *sql.DB, trail *auditrepo.Repository) *Repository {
	return &Repository{db: db, trail: trail}
}

// Create registers a destination and records it, in one transaction.
//
// The count and the insert are serialised per tenant with a transaction-level advisory
// lock. Without it two concurrent requests could both see nine endpoints and both insert.
func (r *Repository) Create(
	ctx context.Context, tenantID, url, secret string, entry auditrepo.Entry,
) (*Endpoint, error) {
	endpoint := &Endpoint{URL: url, Active: true, SecretHint: hint(secret)}

	err := r.inTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, tenantID); err != nil {
			return fmt.Errorf("could not serialise the endpoint limit: %w", err)
		}

		var count int
		if err := tx.QueryRowContext(ctx,
			`SELECT count(*) FROM tenant_webhook_endpoints WHERE tenant_id = $1`, tenantID,
		).Scan(&count); err != nil {
			return fmt.Errorf("could not count the endpoints: %w", err)
		}
		if count >= MaxPerTenant {
			return ErrLimitReached
		}

		if err := tx.QueryRowContext(ctx, `
			INSERT INTO tenant_webhook_endpoints (tenant_id, url, secret)
			VALUES ($1, $2, $3)
			RETURNING id, created_at`,
			tenantID, url, secret,
		).Scan(&endpoint.ID, &endpoint.CreatedAt); err != nil {
			return fmt.Errorf("could not register the endpoint: %w", err)
		}

		entry.SubjectID = endpoint.ID

		return r.trail.AppendTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, err
	}

	return endpoint, nil
}

// List returns the tenant's destinations with a count of deliveries in each state.
//
// Complexity: one grouped query over the tenant's endpoints and their events; the join
// runs through outbox_events(endpoint_id).
func (r *Repository) List(ctx context.Context, tenantID string) ([]Endpoint, error) {
	var endpoints []Endpoint

	err := r.inTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT e.id, e.url, e.active, right(e.secret, 4), e.created_at,
			       count(o.id) FILTER (WHERE o.status = 'delivered'),
			       count(o.id) FILTER (WHERE o.status IN ('pending', 'delivering')),
			       count(o.id) FILTER (WHERE o.status = 'dead')
			FROM tenant_webhook_endpoints e
			LEFT JOIN outbox_events o ON o.endpoint_id = e.id
			WHERE e.tenant_id = $1
			GROUP BY e.id
			ORDER BY e.created_at DESC`, tenantID)
		if err != nil {
			return fmt.Errorf("could not list the endpoints: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var e Endpoint
			if err := rows.Scan(&e.ID, &e.URL, &e.Active, &e.SecretHint, &e.CreatedAt,
				&e.Delivered, &e.Pending, &e.Dead); err != nil {
				return fmt.Errorf("could not read an endpoint: %w", err)
			}
			endpoints = append(endpoints, e)
		}

		return rows.Err()
	})

	return endpoints, err
}

// SetActive turns delivery to an endpoint on or off, and records it.
//
// There is no delete. Events are tied to their endpoint, and removing it would remove the
// record of what was delivered; turning it off keeps the history and stops the traffic.
func (r *Repository) SetActive(
	ctx context.Context, tenantID, id string, active bool, entry auditrepo.Entry,
) error {
	return r.inTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE tenant_webhook_endpoints
			SET active = $3, updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2`,
			id, tenantID, active,
		)
		if err != nil {
			return fmt.Errorf("could not change the endpoint: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("could not read the change: %w", err)
		}
		if rows == 0 {
			return ErrNotFound
		}

		entry.SubjectID = id

		return r.trail.AppendTx(ctx, tx, entry)
	})
}

// inTenantTx runs one unit of work scoped to the tenant. Both tables carry the fail-closed
// tenant policy; the scope is transaction-local and never reaches a pooled connection.
func (r *Repository) inTenantTx(ctx context.Context, tenantID string, work func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		return fmt.Errorf("could not scope the transaction: %w", err)
	}

	if err := work(tx); err != nil {
		return err
	}

	return tx.Commit()
}

func hint(secret string) string {
	if len(secret) <= 4 {
		return ""
	}

	return secret[len(secret)-4:]
}
