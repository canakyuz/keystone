// Package upload records the files the API stores.
//
// The file itself lives on disk. What is kept here is the row that says it exists, whose
// tenant it belongs to and who stored it, written with its audit entry in one transaction.
package upload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
)

// ErrNoTrail reports that a record was asked of a repository built without an audit trail.
var ErrNoTrail = errors.New("upload repository: no audit trail configured")

// Record is one stored file.
type Record struct {
	ID          string
	TenantID    string
	Category    string
	Path        string
	ContentType string
	SizeBytes   int64
	UploadedBy  string
	CreatedAt   time.Time
}

// Repository writes uploads.
type Repository struct {
	db    *sql.DB
	trail *auditrepo.Repository
}

// New creates the repository. The trail is required: a stored file is recorded in the
// transaction that records the file.
func New(db *sql.DB, trail *auditrepo.Repository) *Repository {
	return &Repository{db: db, trail: trail}
}

// Create records a stored file and its audit entry in one transaction, filling in the
// record's id, uploader and time.
//
// The uploader is read from the request context rather than passed in, so a call site
// cannot record the wrong one; see audit.ActorFrom. A file stored with no subject on the
// context is recorded as the system's.
func (r *Repository) Create(ctx context.Context, record *Record) error {
	if r.trail == nil {
		return ErrNoTrail
	}

	actorID, actorType := auditrepo.ActorFrom(ctx)
	if actorType == auditrepo.ActorUser {
		record.UploadedBy = actorID
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// uploads and audit_log both carry the fail-closed tenant policy. The scope is
	// transaction-local and never reaches a pooled connection.
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, record.TenantID); err != nil {
		return fmt.Errorf("could not scope the transaction: %w", err)
	}

	if err := tx.QueryRowContext(ctx, `
		INSERT INTO uploads (tenant_id, category, path, content_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::UUID)
		RETURNING id, created_at`,
		record.TenantID, record.Category, record.Path, record.ContentType, record.SizeBytes, record.UploadedBy,
	).Scan(&record.ID, &record.CreatedAt); err != nil {
		return fmt.Errorf("could not record the upload: %w", err)
	}

	if err := r.trail.AppendTx(ctx, tx, auditrepo.Entry{
		TenantID:    record.TenantID,
		ActorID:     actorID,
		ActorType:   actorType,
		Action:      "upload.created",
		SubjectType: "upload",
		SubjectID:   record.ID,
		Metadata: map[string]any{
			"category":     record.Category,
			"path":         record.Path,
			"content_type": record.ContentType,
			"size_bytes":   record.SizeBytes,
		},
	}); err != nil {
		return err
	}

	return tx.Commit()
}

// Paths returns the served paths recorded for one tenant's files, as a set.
//
// It reads under the tenant's own scope, so a row in another tenant can never account for a
// file in this one's directory.
func (r *Repository) Paths(ctx context.Context, tenantID string) (map[string]struct{}, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("could not begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		return nil, fmt.Errorf("could not scope the transaction: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `SELECT path FROM uploads WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("could not read the upload records: %w", err)
	}
	defer rows.Close()

	paths := make(map[string]struct{})
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("could not read an upload record: %w", err)
		}
		paths[path] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not read the upload records: %w", err)
	}

	return paths, tx.Commit()
}
