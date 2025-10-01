package tenant

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nexpaces-api/internal/domain/tenant"
	"nexpaces-api/test/helpers"
)

func TestPostgresRepository_Create(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	tests := []struct {
		name    string
		tenant  *tenant.Tenant
		wantErr bool
	}{
		{
			name: "create valid tenant",
			tenant: &tenant.Tenant{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				Name:      "Test Company",
				Slug:      "test-company",
				Email:     "test@example.com",
				Status:    tenant.TenantStatusActive,
				Plan:      tenant.PlanFree,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "create tenant with duplicate slug should fail",
			tenant: &tenant.Tenant{
				ID:        "550e8400-e29b-41d4-a716-446655440001",
				Name:      "Another Company",
				Slug:      "test-company", // Duplicate slug
				Email:     "another@example.com",
				Status:    tenant.TenantStatusActive,
				Plan:      tenant.PlanFree,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(context.Background(), tt.tenant)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify tenant was created
				created, err := repo.GetByID(context.Background(), tt.tenant.ID)
				require.NoError(t, err)
				assert.Equal(t, tt.tenant.Name, created.Name)
				assert.Equal(t, tt.tenant.Slug, created.Slug)
				assert.Equal(t, tt.tenant.Email, created.Email)
			}
		})
	}
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create test tenant
	testTenant := helpers.CreateTestTenant(t, db, "test-tenant")

	tests := []struct {
		name     string
		tenantID string
		wantErr  bool
	}{
		{
			name:     "get existing tenant",
			tenantID: testTenant.ID,
			wantErr:  false,
		},
		{
			name:     "get non-existent tenant",
			tenantID: "non-existent-id",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn, err := repo.GetByID(context.Background(), tt.tenantID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tn)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tn)
				assert.Equal(t, tt.tenantID, tn.ID)
			}
		})
	}
}

func TestPostgresRepository_GetBySlug(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create test tenant
	testTenant := helpers.CreateTestTenant(t, db, "test-slug")

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{
			name:    "get tenant by existing slug",
			slug:    testTenant.Slug,
			wantErr: false,
		},
		{
			name:    "get tenant by non-existent slug",
			slug:    "non-existent-slug",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn, err := repo.GetBySlug(context.Background(), tt.slug)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tn)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tn)
				assert.Equal(t, tt.slug, tn.Slug)
			}
		})
	}
}

func TestPostgresRepository_Update(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create test tenant
	testTenant := helpers.CreateTestTenant(t, db, "update-test")

	t.Run("update tenant successfully", func(t *testing.T) {
		// Get tenant
		tn, err := repo.GetByID(context.Background(), testTenant.ID)
		require.NoError(t, err)

		// Update fields
		tn.Name = "Updated Name"
		tn.Plan = tenant.PlanPro
		tn.Status = tenant.TenantStatusSuspended
		tn.UpdatedAt = time.Now()

		// Update tenant
		err = repo.Update(context.Background(), tn)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(context.Background(), testTenant.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, tenant.PlanPro, updated.Plan)
		assert.Equal(t, tenant.TenantStatusSuspended, updated.Status)
	})
}

func TestPostgresRepository_Delete(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	t.Run("soft delete tenant", func(t *testing.T) {
		// Create test tenant
		testTenant := helpers.CreateTestTenant(t, db, "delete-test")

		// Delete tenant
		err := repo.Delete(context.Background(), testTenant.ID)
		require.NoError(t, err)

		// Verify tenant is soft deleted
		tn, err := repo.GetByID(context.Background(), testTenant.ID)
		assert.Error(t, err)
		assert.Nil(t, tn)

		// Verify tenant exists in database with deleted_at set
		var deletedAt sql.NullTime
		err = db.QueryRowContext(
			context.Background(),
			"SELECT deleted_at FROM tenants WHERE id = $1",
			testTenant.ID,
		).Scan(&deletedAt)
		require.NoError(t, err)
		assert.True(t, deletedAt.Valid)
	})
}

func TestPostgresRepository_SetCustomDomain(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create test tenant
	testTenant := helpers.CreateTestTenant(t, db, "domain-test")

	t.Run("set custom domain", func(t *testing.T) {
		domain := "custom.example.com"
		err := repo.SetCustomDomain(context.Background(), testTenant.ID, domain)
		require.NoError(t, err)

		// Verify custom domain was set
		tn, err := repo.GetByID(context.Background(), testTenant.ID)
		require.NoError(t, err)
		assert.Equal(t, domain, tn.CustomDomain)
		assert.False(t, tn.CustomDomainVerified)
	})
}

func TestPostgresRepository_VerifyCustomDomain(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create test tenant with custom domain
	testTenant := helpers.CreateTestTenant(t, db, "verify-test")
	err := repo.SetCustomDomain(context.Background(), testTenant.ID, "verify.example.com")
	require.NoError(t, err)

	t.Run("verify custom domain", func(t *testing.T) {
		err := repo.VerifyCustomDomain(context.Background(), testTenant.ID)
		require.NoError(t, err)

		// Verify domain is verified
		tn, err := repo.GetByID(context.Background(), testTenant.ID)
		require.NoError(t, err)
		assert.True(t, tn.CustomDomainVerified)
		assert.NotNil(t, tn.CustomDomainVerifiedAt)
	})
}

func TestPostgresRepository_GetStats(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create test tenants with different statuses
	helpers.CreateTestTenant(t, db, "active-1")
	helpers.CreateTestTenant(t, db, "active-2")

	t.Run("get tenant statistics", func(t *testing.T) {
		stats, err := repo.GetStats(context.Background())
		require.NoError(t, err)

		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats["total"].(int64), int64(2))
		assert.GreaterOrEqual(t, stats["active"].(int64), int64(2))
	})
}

func TestPostgresRepository_TenantIsolation(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := NewPostgresRepository(db)

	// Create two test tenants
	tenant1 := helpers.CreateTestTenant(t, db, "tenant-1")
	tenant2 := helpers.CreateTestTenant(t, db, "tenant-2")

	t.Run("RLS policies prevent cross-tenant access", func(t *testing.T) {
		// Set tenant context for tenant1
		err := helpers.SetTenantContext(context.Background(), db, tenant1.ID)
		require.NoError(t, err)

		// Try to access tenant2's data
		tn, err := repo.GetByID(context.Background(), tenant2.ID)
		assert.Error(t, err) // Should fail due to RLS
		assert.Nil(t, tn)

		// Clear tenant context
		err = helpers.ClearTenantContext(context.Background(), db)
		require.NoError(t, err)

		// Without RLS context, should be able to access
		tn, err = repo.GetByID(context.Background(), tenant2.ID)
		require.NoError(t, err)
		assert.Equal(t, tenant2.ID, tn.ID)
	})
}
