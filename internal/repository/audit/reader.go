package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Record is one entry as a reader sees it.
//
// The actor's address is resolved here rather than in the caller, because the caller would
// have to ask for every actor separately: a page of fifty entries is one join, or fifty
// lookups.
type Record struct {
	ID          string
	Action      string
	ActorID     string
	ActorEmail  string
	ActorType   ActorType
	SubjectType string
	SubjectID   string
	Metadata    map[string]any
	CreatedAt   time.Time
}

// Page is one page of the trail, newest first.
type Page struct {
	Records []Record

	// Before is the cursor for the next page: the timestamp of the oldest record here.
	// Empty when the page is the last one.
	Before *time.Time
}

// Reader reads the trail.
//
// Separate from Repository, which only appends and holds no connection of its own. Reading
// and writing an append-only table are different jobs: one belongs to somebody else's
// transaction, the other is a query with a cursor.
type Reader struct {
	db *sql.DB
}

// NewReader creates the reader.
func NewReader(db *sql.DB) *Reader {
	return &Reader{db: db}
}

// maxPage caps what one request can ask for. A cap the client cannot raise is what keeps a
// single query from reading a year of history into memory.
const maxPage = 200

// List returns the tenant's entries, newest first.
//
// The read runs in a transaction that sets the tenant context, because audit_log carries the
// fail-closed tenant policy: without it the query returns nothing rather than somebody
// else's history. The tenant comes from the caller's verified context, so a page can only
// ever be this tenant's own.
//
// Complexity: O(log n + limit) through the (tenant_id, created_at DESC) index.
func (r *Reader) List(ctx context.Context, tenantID string, limit int, before *time.Time) (*Page, error) {
	if limit <= 0 || limit > maxPage {
		limit = 50
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("could not begin the audit read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`, tenantID,
	); err != nil {
		return nil, fmt.Errorf("could not scope the audit read: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT a.id, a.action, a.actor_id, a.actor_type, u.email,
		       a.subject_type, a.subject_id, a.metadata, a.created_at
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.tenant_id = $1
		  AND ($2::timestamptz IS NULL OR a.created_at < $2)
		ORDER BY a.created_at DESC
		LIMIT $3`,
		tenantID, before, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("could not read the audit trail: %w", err)
	}
	defer rows.Close()

	page := &Page{Records: make([]Record, 0, limit)}

	for rows.Next() {
		var record Record
		var actorID, actorEmail sql.NullString
		var metadata []byte

		if err := rows.Scan(
			&record.ID, &record.Action, &actorID, &record.ActorType, &actorEmail,
			&record.SubjectType, &record.SubjectID, &metadata, &record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not read an audit entry: %w", err)
		}

		record.ActorID = actorID.String
		record.ActorEmail = actorEmail.String

		if err := json.Unmarshal(metadata, &record.Metadata); err != nil {
			return nil, fmt.Errorf("could not decode the audit metadata: %w", err)
		}

		page.Records = append(page.Records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not read the audit trail: %w", err)
	}

	// A full page means there may be more. A short one is the end, and saying so saves the
	// client a request that would come back empty.
	if len(page.Records) == limit {
		oldest := page.Records[len(page.Records)-1].CreatedAt
		page.Before = &oldest
	}

	return page, nil
}
