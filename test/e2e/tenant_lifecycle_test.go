package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/middleware"
	auditrepo "github.com/canakyuz/keystone/internal/repository/audit"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/validator"
)

// TestSuspendedTenant_LosesAccessOnTheNextRequest verifies suspension takes effect at once.
//
// The schema cache is what refuses a suspended tenant: it resolves only active and trial
// tenants. It was never told about a suspension, so a tenant that had made one request kept
// resolving from the cache, for up to the in-process TTL here and up to the Redis TTL of ten
// minutes in production.
func TestSuspendedTenant_LosesAccessOnTheNextRequest(t *testing.T) {
	h := newHarness(t)

	tenantID := createTenant(t, h, "e2e-suspend-cache")

	schemaCache := middleware.NewTenantSchemaCache(nil, h.appDB, nil, nil)
	service := newTenantService(h, schemaCache)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/scoped",
		middleware.AuthMiddleware(testJWTSecret),
		middleware.TenantContextMiddleware(schemaCache),
		func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) },
	)

	status := func() int {
		resp, err := app.Test(h.get(t, "/scoped", tenantID), 10_000)
		require.NoError(t, err)
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	require.Equal(t, http.StatusOK, status(), "the active tenant was refused before the test began")

	require.NoError(t, service.Suspend(context.Background(), tenantID, "e2e"))
	assert.NotEqual(t, http.StatusOK, status(), "the suspended tenant was still served from the cache")

	require.NoError(t, service.Activate(context.Background(), tenantID))
	assert.Equal(t, http.StatusOK, status(), "the reactivated tenant was still refused from the negative cache")
}

// TestPlanChange_ReachesTheRateLimitAtOnce verifies the limiter reads the new plan on the
// next request rather than when the cached one expires, five minutes later.
func TestPlanChange_ReachesTheRateLimitAtOnce(t *testing.T) {
	h := newHarness(t)

	tenantID := createTenant(t, h, "e2e-plan-cache")

	planCache := middleware.NewTenantPlanCache(nil, h.appDB)
	service := newTenantService(h, planCache)

	before, err := planCache.GetPlan(context.Background(), tenantID)
	require.NoError(t, err)
	require.NotEqual(t, "enterprise", before)

	_, err = service.UpgradePlan(context.Background(), tenantID, &tenantUsecase.UpgradePlanRequest{Plan: "enterprise"})
	require.NoError(t, err)

	after, err := planCache.GetPlan(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, "enterprise", after, "the limiter kept the cached plan")
}

func newTenantService(h *harness, caches ...tenantUsecase.Forgetter) *tenantUsecase.Service {
	return tenantUsecase.NewService(tenantRepo.NewPostgresRepository(h.appDB).WithAudit(auditrepo.New()),
		validator.New(), logger.Default(), caches...)
}
