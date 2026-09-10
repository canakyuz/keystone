package worker

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	"github.com/canakyuz/keystone/internal/domain/tenant"
	"github.com/canakyuz/keystone/pkg/logger"
)

// TenantLoader fetches the tenant a job belongs to.
// The interface is declared on the consumer side: the worker knows only the single
// behaviour it needs, not the whole tenant repository.
type TenantLoader interface {
	GetByID(ctx context.Context, id string) (*tenant.Tenant, error)
}

// SchemaProvisioner prepares the tenant's workspace.
type SchemaProvisioner interface {
	ProvisionTenantSchema(ctx context.Context, t *tenant.Tenant) error
}

// StatusSetter updates the tenant lifecycle state.
//
// The methods are deliberately narrow: not "move to any state" but "make this
// transition". The reasoning relates to the limit written down in ADR-0003. Fencing
// only protects the metadata update in the job table; the tenant write here falls
// outside that protection. A worker that has lost its lease can still make this
// call.
//
// The transitions are therefore conditional on the source state. Pulling an active
// tenant back to 'provisioning' is impossible; a late call from an old worker
// quietly has no effect.
type StatusSetter interface {
	// MarkProvisioning moves the tenant to 'provisioning'.
	// It does nothing if the tenant is already 'active'.
	MarkProvisioning(ctx context.Context, tenantID string) error

	// MarkFailed moves the tenant to 'failed'.
	// It does nothing if the tenant is already 'active'.
	MarkFailed(ctx context.Context, tenantID string) error
}

// ProvisionHandler runs the provisioning job.
//
// REPEATABILITY
// This handler can and must be able to run more than once. When a lease expires, the
// job moves to another worker and the steps run again from the start. In the failure
// matrix this row reads: "schema created, result not recorded -> a new attempt
// proceeds by verifying the existing schema."
//
// Every step is therefore written to be either idempotent, or to verify the current
// state and move past it.
type ProvisionHandler struct {
	tenants     TenantLoader
	status      StatusSetter
	provisioner SchemaProvisioner
	log         *logger.Logger
}

// NewProvisionHandler creates the handler.
func NewProvisionHandler(
	tenants TenantLoader, status StatusSetter, provisioner SchemaProvisioner, log *logger.Logger,
) *ProvisionHandler {
	return &ProvisionHandler{tenants: tenants, status: status, provisioner: provisioner, log: log}
}

// Handle runs a single provisioning job.
//
// Steps:
//  1. Load the tenant record.
//  2. If it is already active, do nothing (a previous attempt's result went unrecorded).
//  3. Move the status to 'provisioning'.
//  4. Create the schema and apply the migrations.
//
// Making the tenant 'active' is NOT this handler's job. Activation happens in the
// same transaction that marks the job done (see Repository.CompleteSuccess). Done
// here, a crash between activation and closing the operation would leave a tenant
// that looks "in progress" to the user while actually being usable.
func (h *ProvisionHandler) Handle(ctx context.Context, job *domain.Job) error {
	t, err := h.tenants.GetByID(ctx, job.TenantID)
	if err != nil {
		return fmt.Errorf("could not load tenant: %w", err)
	}
	if t == nil {
		return errors.New("tenant not found")
	}

	// A previous attempt may have finished the work but failed to record the result.
	// In that case there is no need to run the steps again.
	if t.Status == tenant.TenantStatusActive {
		h.logStep(job, "tenant already active, provisioning skipped")
		return nil
	}

	if err := h.status.MarkProvisioning(ctx, t.ID); err != nil {
		return fmt.Errorf("could not move status to 'provisioning': %w", err)
	}

	h.logStep(job, "preparing schema")

	// ProvisionTenantSchema uses CREATE SCHEMA IF NOT EXISTS and applies the migrations
	// in its own transaction. Interrupted halfway, that transaction rolls back and a new
	// attempt starts from a clean point.
	if err := h.provisioner.ProvisionTenantSchema(ctx, t); err != nil {
		return fmt.Errorf("could not prepare schema: %w", err)
	}

	h.logStep(job, "schema ready")

	return nil
}

// MarkFailed marks the tenant 'failed' once the attempts are exhausted.
//
// Why it is a separate method: not every failed attempt should make the tenant
// 'failed'. While a job will be retried, the tenant stays 'provisioning'. Only when
// the attempts run out does it reach a terminal state.
func (h *ProvisionHandler) MarkFailed(ctx context.Context, tenantID string) error {
	return h.status.MarkFailed(ctx, tenantID)
}

// logStep carries the diagnostic chain.
//
// The chain: request_id -> operation_id -> job_id -> attempt -> worker -> migration
// step. operation_id, job_id and attempt are written here; request_id is not yet
// carried over from the HTTP layer.
func (h *ProvisionHandler) logStep(job *domain.Job, step string) {
	if h.log == nil {
		return
	}

	h.log.WithFields(logger.Fields{
		"operation_id": job.OperationID,
		"job_id":       job.ID,
		"tenant_id":    job.TenantID,
		"attempt":      job.Attempts,
		"lease_owner":  job.LeaseOwner,
		"fence":        job.Fence,
	}).Info(step)
}
