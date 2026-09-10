package tenant

import (
	"context"
	"fmt"
)

// MarkProvisioning moves the tenant to 'provisioning'.
//
// The transition is only made from 'pending', 'failed', or 'provisioning' itself. An
// active tenant cannot be pulled back by this call.
//
// WHY the source-state condition: this write falls outside fencing protection. A
// worker that has lost its lease can make this call after the current worker has
// activated the tenant. Without the condition it would knock a working tenant back
// into a provisioning state.
//
// Affecting no rows is not an error: it means the transition did not apply, which is
// an expected outcome.
//
// Complexity: O(1), a single update by primary key.
func (r *PostgresRepository) MarkProvisioning(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenants
		SET status = 'provisioning', updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status IN ('pending', 'provisioning', 'failed')`, tenantID)
	if err != nil {
		return fmt.Errorf("could not move tenant to 'provisioning': %w", err)
	}

	return nil
}

// MarkFailed moves the tenant to 'failed'.
//
// An active tenant is never marked failed: its provisioning already completed, and a
// late report must not undo that.
func (r *PostgresRepository) MarkFailed(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenants
		SET status = 'failed', updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status IN ('pending', 'provisioning')`, tenantID)
	if err != nil {
		return fmt.Errorf("could not move tenant to 'failed': %w", err)
	}

	return nil
}
