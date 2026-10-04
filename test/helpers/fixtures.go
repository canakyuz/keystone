package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	pq "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestTenant represents a test tenant fixture
type TestTenant struct {
	ID         string
	Name       string
	Slug       string
	Email      string
	Status     string
	Plan       string
	SchemaName string
}

// TestUser represents a test user fixture
type TestUser struct {
	ID        string
	TenantID  string
	Email     string
	Password  string
	FirstName string
	LastName  string
	Role      string
	Status    string
}

// TestWebsite represents a test website fixture
type TestWebsite struct {
	ID         string
	TenantID   string
	Name       string
	Slug       string
	TemplateID string
	Status     string
}

// CreateTestTenant creates a test tenant
func CreateTestTenant(t *testing.T, db *sql.DB, slug string) *TestTenant {
	t.Helper()

	tenant := &TestTenant{
		ID:     uuid.New().String(),
		Name:   "Test Tenant " + slug,
		Slug:   slug,
		Email:  slug + "@example.com",
		Status: "active",
		Plan:   "free",
	}

	var schemaName string
	err := db.QueryRowContext(context.Background(), "SELECT generate_schema_name($1)", tenant.Slug).Scan(&schemaName)
	require.NoError(t, err)

	tenant.SchemaName = schemaName

	createSchema := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pq.QuoteIdentifier(schemaName))
	_, err = db.ExecContext(context.Background(), createSchema)
	require.NoError(t, err)

	query := `
		INSERT INTO tenants (id, name, slug, email, schema_name, status, plan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`

	_, err = db.ExecContext(
		context.Background(),
		query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Email,
		tenant.SchemaName,
		tenant.Status,
		tenant.Plan,
	)
	require.NoError(t, err)

	return tenant
}

// CreateTestUser creates a test user
func CreateTestUser(t *testing.T, db *sql.DB, tenantID, email, role string) *TestUser {
	t.Helper()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &TestUser{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
		Role:      role,
		Status:    "active",
	}

	query := `
		INSERT INTO users (
			id, tenant_id, email, password_hash, first_name, last_name,
			role, status, email_verified, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`

	err = WithTenantContext(context.Background(), db, tenantID, func() error {
		_, err := db.ExecContext(
			context.Background(),
			query,
			user.ID,
			user.TenantID,
			user.Email,
			user.Password,
			user.FirstName,
			user.LastName,
			user.Role,
			user.Status,
			true,
		)
		return err
	})
	require.NoError(t, err)

	return user
}
