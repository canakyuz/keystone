package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditRepo "github.com/canakyuz/keystone/internal/repository/audit"
	operationRepo "github.com/canakyuz/keystone/internal/repository/operation"
	outboxRepo "github.com/canakyuz/keystone/internal/repository/outbox"
	templateRepo "github.com/canakyuz/keystone/internal/repository/template"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	"github.com/canakyuz/keystone/internal/worker"
	"github.com/canakyuz/keystone/test/helpers"
)

const workerRoleID = "worker-role-test"

// TestWorkerRole_RunsTheWholeJob drives one provisioning job from claim to delivery over
// the worker role and nothing else.
//
// Before migration 039 the worker had no role of its own. Over the application's role its
// claim matched no row while a job was waiting, because the queue tables carry the tenant
// policy and the worker serves every tenant. It ran only as a superuser.
func TestWorkerRole_RunsTheWholeJob(t *testing.T) {
	admin := helpers.SetupTestDB(t)
	appDB := helpers.SetupAppRoleDB(t, admin)
	workerDB := helpers.SetupWorkerRoleDB(t, admin)
	ctx := context.Background()

	const subject = "00000000-0000-4000-8000-0000000000c1"

	// The API accepts the work over its own role.
	accepted, err := operationRepo.New(appDB).CreateTenantProvision(ctx, operationRepo.ProvisionRequest{
		Name:        "Worker Role",
		Slug:        "worker-role",
		Email:       "owner@worker-role.test",
		CreatedBy:   subject,
		Scope:       "subject:" + subject,
		RequestBody: []byte(`{"slug":"worker-role"}`),
	})
	require.NoError(t, err)

	_, err = admin.Exec(`
		INSERT INTO tenant_webhook_endpoints (tenant_id, url, secret)
		VALUES ($1, 'https://hooks.example.com/keystone', 'secret')`, accepted.TenantID)
	require.NoError(t, err)

	// The worker does the rest over its role, built the way cmd/worker builds it.
	outbox := outboxRepo.New(workerDB)
	operations := operationRepo.New(workerDB).WithOutbox(outbox).WithAudit(auditRepo.New())
	tenants := tenantRepo.NewPostgresRepository(workerDB)
	provisioner := tenantUsecase.NewProvisioningService(
		workerDB, templateRepo.NewFileSystemRepository("../../templates/tenants"), nil)
	handler := worker.NewProvisionHandler(tenants, tenants, provisioner, nil)

	job, err := operations.Claim(ctx, workerRoleID, time.Minute)
	require.NoError(t, err, "the worker role could not claim a waiting job")

	require.NoError(t, operations.RenewLease(ctx, job.ID, workerRoleID, job.Fence, time.Minute))
	require.NoError(t, handler.Handle(ctx, job), "the worker role could not provision the tenant")
	require.NoError(t, operations.CompleteSuccess(ctx, job.ID, workerRoleID, job.Fence, true))

	_, err = operations.QueueDepth(ctx)
	require.NoError(t, err, "the worker role could not read the queue depth")

	var status, schema string
	require.NoError(t, admin.QueryRow(
		`SELECT status, schema_name FROM tenants WHERE id = $1`, accepted.TenantID,
	).Scan(&status, &schema))
	assert.Equal(t, "active", status)

	var schemas, trail int
	require.NoError(t, admin.QueryRow(
		`SELECT count(*) FROM information_schema.schemata WHERE schema_name = $1`, schema,
	).Scan(&schemas))
	assert.Equal(t, 1, schemas, "the tenant schema was not created")

	require.NoError(t, admin.QueryRow(
		`SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'tenant.activated'`, accepted.TenantID,
	).Scan(&trail))
	assert.Equal(t, 1, trail, "the activation left no trail")

	event, err := outbox.Claim(ctx, workerRoleID, time.Minute)
	require.NoError(t, err, "the worker role could not claim the notification")
	require.NoError(t, outbox.MarkDelivered(ctx, event.ID, workerRoleID, event.Fence))

	_, err = outbox.Depth(ctx)
	require.NoError(t, err, "the worker role could not read the outbox depth")
}

// TestWorkerRole_HoldsNothingElse: the role can do its job and no more. The worker never
// runs any of these statements, and the database, not just the code, has to refuse each
// one.
func TestWorkerRole_HoldsNothingElse(t *testing.T) {
	admin := helpers.SetupTestDB(t)
	workerDB := helpers.SetupWorkerRoleDB(t, admin)

	for _, stmt := range []string{
		`SELECT count(*) FROM users`,
		`SELECT count(*) FROM payments`,
		`SELECT count(*) FROM idempotency_keys`,
		`SELECT count(*) FROM audit_log`,
		`UPDATE audit_log SET action = 'rewritten'`,
		`DELETE FROM audit_log`,
		`DELETE FROM tenants`,
		`DELETE FROM provisioning_jobs`,
		`INSERT INTO operations (tenant_id, kind, status) VALUES (gen_random_uuid(), 'tenant.provision', 'pending')`,
		`UPDATE tenant_webhook_endpoints SET url = 'https://attacker.example.com/'`,
		`DROP TABLE tenants`,
	} {
		_, err := workerDB.Exec(stmt)
		if assert.Errorf(t, err, "the worker role was allowed to run: %s", stmt) {
			assert.Regexp(t, `permission denied|must be owner`, err.Error(), stmt)
		}
	}
}
