// Package audit records what changed, beside the change itself.
//
// The trail is written inside the caller's transaction and never on its own. An audit
// record that can disagree with the data is worse than no record, because somebody will
// eventually trust it: written after the change, a crash in between leaves a state nobody
// can account for; written before, a rollback leaves a record of something that never
// happened.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// ActorType distinguishes who made a change.
type ActorType string

const (
	// ActorUser is a person acting through the API.
	ActorUser ActorType = "user"

	// ActorSystem is the platform acting on its own, a worker completing a job for
	// example. Kept distinct from an unknown actor: "nobody asked for this, it was the
	// scheduler" and "we do not know who did this" are different answers.
	ActorSystem ActorType = "system"

	// ActorAPI is a machine credential.
	ActorAPI ActorType = "api"
)

// Entry is one line of the trail.
type Entry struct {
	TenantID string

	ActorID   string
	ActorType ActorType

	Action      string
	SubjectType string
	SubjectID   string

	// Metadata carries enough context to read the entry without joining to rows that may
	// since have changed. A record of the past that needs the present to be legible is
	// not a record of the past.
	Metadata map[string]any

	RequestID string
}

// Repository appends to the trail.
type Repository struct{}

// New creates the repository.
//
// It holds no connection: every write belongs to somebody else's transaction, so there is
// no path here that could open one of its own.
func New() *Repository {
	return &Repository{}
}

// AppendTx writes one entry inside the caller's transaction.
//
// The transaction is a parameter rather than an implementation detail, and that is the
// whole guarantee. If this opened its own, the entry and the change it describes could
// commit independently, which is the failure this package exists to prevent.
func (r *Repository) AppendTx(ctx context.Context, tx *sql.Tx, entry Entry) error {
	metadata, err := json.Marshal(orEmpty(entry.Metadata))
	if err != nil {
		return fmt.Errorf("could not encode the audit metadata: %w", err)
	}

	actorType := entry.ActorType
	if actorType == "" {
		actorType = ActorSystem
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log (
			tenant_id, actor_id, actor_type, action, subject_type, subject_id, metadata, request_id
		)
		VALUES ($1, NULLIF($2, '')::UUID, $3, $4, $5, $6::UUID, $7::jsonb, NULLIF($8, ''))`,
		entry.TenantID, entry.ActorID, string(actorType),
		entry.Action, entry.SubjectType, entry.SubjectID, string(metadata), entry.RequestID,
	)
	if err != nil {
		return fmt.Errorf("could not append the audit entry: %w", err)
	}

	return nil
}

// orEmpty keeps a nil map out of the encoder.
//
// json.Marshal turns a nil map into "null", and the column is NOT NULL with a '{}'
// default that an explicit null would override. A trail entry whose metadata is the
// literal null is harder to query than one whose metadata is empty.
func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}

	return m
}
