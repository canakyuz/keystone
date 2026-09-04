package user

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/domain/user"
	"github.com/canakyuz/keystone/pkg/database"
	"github.com/canakyuz/keystone/test/helpers"
)

// newTestRepository, testler için repository kurar.
// Logger nil geçilir: connectionManager tüm log çağrılarını nil-guard ile sarar,
// böylece testlerde log gürültüsü olmadan gerçek search_path davranışı korunur.
// ctxFor, repository'lerin beklediği tenant schema'sını taşıyan context üretir.
// Production'da bunu TenantContextMiddleware yerleştirir; testte HTTP katmanı yok.
func ctxFor(tn *helpers.TestTenant) context.Context {
	return helpers.WithTenantSchema(context.Background(), tn.SchemaName)
}

func newTestRepository(db *sql.DB) *PostgresRepository {
	return NewPostgresRepository(db, database.NewTenantConnectionManager(db, nil))
}

func TestPostgresRepository_Create(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant
	testTenant := helpers.CreateTestTenant(t, db, "user-create-test")

	tests := []struct {
		name    string
		user    *user.User
		wantErr bool
	}{
		{
			name: "create valid user",
			user: &user.User{
				ID:        "7f9c1e10-0001-4000-8000-000000000001",
				TenantID:  testTenant.ID,
				Email:     "test@example.com",
				Password:  "hashed_password",
				FirstName: "John",
				LastName:  "Doe",
				Role:      user.RoleAdmin,
				Status:    user.UserStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "create user with duplicate email in same tenant should fail",
			user: &user.User{
				ID:        "7f9c1e10-0002-4000-8000-000000000002",
				TenantID:  testTenant.ID,
				Email:     "test@example.com", // Duplicate email
				Password:  "hashed_password",
				FirstName: "Jane",
				LastName:  "Doe",
				Role:      user.RoleViewer,
				Status:    user.UserStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
				return repo.Create(ctxFor(testTenant), tt.user)
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify user was created
				created, err := repo.GetByID(ctxFor(testTenant), testTenant.ID, tt.user.ID)
				require.NoError(t, err)
				assert.Equal(t, tt.user.Email, created.Email)
				assert.Equal(t, tt.user.FirstName, created.FirstName)
				assert.Equal(t, tt.user.TenantID, created.TenantID)
			}
		})
	}
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant and user
	testTenant := helpers.CreateTestTenant(t, db, "user-get-test")
	testUser := helpers.CreateTestUser(t, db, testTenant.ID, "test@example.com", "owner")

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "get existing user",
			userID:  testUser.ID,
			wantErr: false,
		},
		{
			name:    "get non-existent user",
			userID:  "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var usr *user.User
			var err error

			err = helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
				usr, err = repo.GetByID(ctxFor(testTenant), testTenant.ID, tt.userID)
				return err
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, usr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, usr)
				assert.Equal(t, tt.userID, usr.ID)
			}
		})
	}
}

func TestPostgresRepository_GetByEmail(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant and user
	testTenant := helpers.CreateTestTenant(t, db, "email-test")
	testUser := helpers.CreateTestUser(t, db, testTenant.ID, "test@example.com", "admin")

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "get user by existing email",
			email:   testUser.Email,
			wantErr: false,
		},
		{
			name:    "get user by non-existent email",
			email:   "nonexistent@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var usr *user.User
			var err error

			err = helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
				usr, err = repo.GetByEmail(ctxFor(testTenant), testTenant.ID, tt.email)
				return err
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, usr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, usr)
				assert.Equal(t, tt.email, usr.Email)
			}
		})
	}
}

func TestPostgresRepository_GetByEmailGlobal(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant and user
	testTenant := helpers.CreateTestTenant(t, db, "global-email-test")
	testUser := helpers.CreateTestUser(t, db, testTenant.ID, "global@example.com", "owner")

	t.Run("get user by email globally (no tenant context)", func(t *testing.T) {
		usr, err := repo.GetByEmailGlobal(ctxFor(testTenant), testUser.Email)
		require.NoError(t, err)
		assert.NotNil(t, usr)
		assert.Equal(t, testUser.Email, usr.Email)
		assert.Equal(t, testTenant.ID, usr.TenantID)
	})

	t.Run("get user by non-existent email globally", func(t *testing.T) {
		usr, err := repo.GetByEmailGlobal(ctxFor(testTenant), "nonexistent@example.com")
		assert.Error(t, err)
		assert.Nil(t, usr)
	})
}

func TestPostgresRepository_Update(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant and user
	testTenant := helpers.CreateTestTenant(t, db, "update-test")
	testUser := helpers.CreateTestUser(t, db, testTenant.ID, "update@example.com", "editor")

	t.Run("update user successfully", func(t *testing.T) {
		err := helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			// Get user
			usr, err := repo.GetByID(ctxFor(testTenant), testTenant.ID, testUser.ID)
			require.NoError(t, err)

			// Update fields
			usr.FirstName = "Updated"
			usr.LastName = "Name"
			usr.Role = user.RoleAdmin
			usr.UpdatedAt = time.Now()

			// Update user
			return repo.Update(ctxFor(testTenant), usr)
		})
		require.NoError(t, err)

		// Verify update
		err = helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			updated, err := repo.GetByID(ctxFor(testTenant), testTenant.ID, testUser.ID)
			require.NoError(t, err)
			assert.Equal(t, "Updated", updated.FirstName)
			assert.Equal(t, "Name", updated.LastName)
			assert.Equal(t, user.RoleAdmin, updated.Role)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestPostgresRepository_UpdateLastLogin(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant and user
	testTenant := helpers.CreateTestTenant(t, db, "login-test")
	testUser := helpers.CreateTestUser(t, db, testTenant.ID, "login@example.com", "viewer")

	t.Run("update last login timestamp", func(t *testing.T) {
		err := helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			return repo.UpdateLastLogin(ctxFor(testTenant), testTenant.ID, testUser.ID)
		})
		require.NoError(t, err)

		// Verify last login was updated
		err = helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			usr, err := repo.GetByID(ctxFor(testTenant), testTenant.ID, testUser.ID)
			require.NoError(t, err)
			assert.NotNil(t, usr.LastLoginAt)
			assert.WithinDuration(t, time.Now(), *usr.LastLoginAt, 5*time.Second)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestPostgresRepository_Delete(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant and user
	testTenant := helpers.CreateTestTenant(t, db, "delete-test")
	testUser := helpers.CreateTestUser(t, db, testTenant.ID, "delete@example.com", "viewer")

	t.Run("soft delete user", func(t *testing.T) {
		err := helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			return repo.Delete(ctxFor(testTenant), testTenant.ID, testUser.ID)
		})
		require.NoError(t, err)

		// Verify user is soft deleted
		err = helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			usr, err := repo.GetByID(ctxFor(testTenant), testTenant.ID, testUser.ID)
			assert.Error(t, err)
			assert.Nil(t, usr)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestPostgresRepository_GetOwner(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create test tenant with owner
	testTenant := helpers.CreateTestTenant(t, db, "owner-test")
	ownerUser := helpers.CreateTestUser(t, db, testTenant.ID, "owner@example.com", "owner")
	helpers.CreateTestUser(t, db, testTenant.ID, "admin@example.com", "admin")

	t.Run("get tenant owner", func(t *testing.T) {
		err := helpers.WithTenantContext(context.Background(), db, testTenant.ID, func() error {
			usr, err := repo.GetOwner(ctxFor(testTenant), testTenant.ID)
			require.NoError(t, err)
			assert.NotNil(t, usr)
			assert.Equal(t, ownerUser.ID, usr.ID)
			assert.Equal(t, user.RoleOwner, usr.Role)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestPostgresRepository_MultiTenantIsolation(t *testing.T) {
	db := helpers.SetupTestDB(t)
	repo := newTestRepository(db)

	// Create two tenants with same email in different tenants
	tenant1 := helpers.CreateTestTenant(t, db, "tenant-1")
	tenant2 := helpers.CreateTestTenant(t, db, "tenant-2")

	user1 := helpers.CreateTestUser(t, db, tenant1.ID, "same@example.com", "owner")
	user2 := helpers.CreateTestUser(t, db, tenant2.ID, "same@example.com", "owner")

	t.Run("same email in different tenants should be isolated", func(t *testing.T) {
		// Access user1 in tenant1 context
		err := helpers.WithTenantContext(context.Background(), db, tenant1.ID, func() error {
			usr, err := repo.GetByEmail(ctxFor(tenant1), tenant1.ID, "same@example.com")
			require.NoError(t, err)
			assert.Equal(t, user1.ID, usr.ID)
			assert.Equal(t, tenant1.ID, usr.TenantID)
			return nil
		})
		require.NoError(t, err)

		// Access user2 in tenant2 context
		err = helpers.WithTenantContext(context.Background(), db, tenant2.ID, func() error {
			usr, err := repo.GetByEmail(ctxFor(tenant2), tenant2.ID, "same@example.com")
			require.NoError(t, err)
			assert.Equal(t, user2.ID, usr.ID)
			assert.Equal(t, tenant2.ID, usr.TenantID)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("cross-tenant access should fail with RLS", func(t *testing.T) {
		// Try to access tenant2's user in tenant1 context
		err := helpers.WithTenantContext(context.Background(), db, tenant1.ID, func() error {
			usr, err := repo.GetByID(ctxFor(tenant1), tenant1.ID, user2.ID)
			assert.Error(t, err) // Should fail due to RLS
			assert.Nil(t, usr)
			return nil
		})
		require.NoError(t, err)
	})
}
