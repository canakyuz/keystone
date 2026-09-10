package tenant

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canakyuz/keystone/internal/domain/tenant"
	templateRepo "github.com/canakyuz/keystone/internal/repository/template"
	"github.com/canakyuz/keystone/pkg/logger"
)

//
// Integration tests verify:
// 1. Multiple components working together
// 2. Database interactions
// 3. Transaction rollback scenarios
// 4. Edge cases and error handling
//
// Mock Strategy:
// - Database: sqlmock (simulates PostgreSQL)
// - Template repo: mockTemplateRepository (custom mock)

// mockTemplateRepository simulates template repository for testing
type mockTemplateRepository struct {
	templates map[string]string
	err       error
}

func (m *mockTemplateRepository) GetTemplateByPlan(ctx context.Context, plan string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if template, ok := m.templates[plan]; ok {
		return template, nil
	}
	// Must return the same error that real repository returns
	return "", templateRepo.ErrTemplateNotFound
}

// TestGenerateSchemaName tests schema name generation with database function
func TestGenerateSchemaName(t *testing.T) {
	// Multiple scenarios in one test
	tests := []struct {
		name           string
		base           string
		mockReturn     string
		mockError      error
		expectedSchema string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "valid schema generation",
			base:           "acme",
			mockReturn:     "tenant_acme_abc123",
			mockError:      nil,
			expectedSchema: "tenant_acme_abc123",
			expectError:    false,
		},
		{
			name:          "database error",
			base:          "test",
			mockReturn:    "",
			mockError:     sql.ErrConnDone,
			expectError:   true,
			errorContains: "failed to generate schema name",
		},
		{
			name:          "invalid schema name returned",
			base:          "test",
			mockReturn:    "INVALID_SCHEMA", // Uppercase not allowed
			mockError:     nil,
			expectError:   true,
			errorContains: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db, mock, err := sqlmock.New()
			require.NoError(t, err, "Failed to create sqlmock")
			defer db.Close()

			// Setup expectations
			if tt.mockError != nil {
				mock.ExpectQuery("SELECT generate_schema_name").
					WithArgs(tt.base).
					WillReturnError(tt.mockError)
			} else {
				mock.ExpectQuery("SELECT generate_schema_name").
					WithArgs(tt.base).
					WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow(tt.mockReturn))
			}

			service := NewProvisioningService(db, nil, nil)

			// Execute function
			schema, err := service.GenerateSchemaName(context.Background(), tt.base)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSchema, schema)
			}

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestProvisionTenantSchema_Success tests successful tenant provisioning
func TestProvisionTenantSchema_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockTemplates := &mockTemplateRepository{
		templates: map[string]string{
			"pro": "CREATE TABLE users (id UUID PRIMARY KEY);",
		},
	}

	service := NewProvisioningService(db, mockTemplates, nil)

	// Test tenant
	testTenant := &tenant.Tenant{
		ID:         "test-tenant-id",
		Name:       "Test Tenant",
		Slug:       "test-tenant",
		Email:      "test@example.com",
		SchemaName: "tenant_test_abc123",
		Plan:       tenant.PlanPro,
		Status:     tenant.TenantStatusActive,
	}

	// Transaction expectations
	mock.ExpectBegin()

	// CREATE SCHEMA
	mock.ExpectExec(`CREATE SCHEMA IF NOT EXISTS "tenant_test_abc123"`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// SET search_path
	mock.ExpectExec(`SET LOCAL search_path TO "tenant_test_abc123", public`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Execute template SQL
	mock.ExpectExec("CREATE TABLE users").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// UPDATE tenants SET schema_name
	mock.ExpectExec("UPDATE tenants SET schema_name").
		WithArgs("tenant_test_abc123", "test-tenant-id").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	err = service.ProvisionTenantSchema(context.Background(), testTenant)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProvisionTenantSchema_Rollback tests transaction rollback on error
func TestProvisionTenantSchema_Rollback(t *testing.T) {
	// ROLLBACK SCENARIO: Schema creation fails, transaction rolls back

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewProvisioningService(db, &mockTemplateRepository{}, nil)

	testTenant := &tenant.Tenant{
		ID:         "test-tenant-id",
		SchemaName: "tenant_test_abc123",
		Plan:       tenant.PlanFree,
	}

	// Transaction starts, then fails
	mock.ExpectBegin()

	// CREATE SCHEMA fails (e.g., schema already exists with different owner)
	mock.ExpectExec(`CREATE SCHEMA IF NOT EXISTS`).
		WillReturnError(sql.ErrConnDone)

	// Expect rollback (automatic in defer)
	mock.ExpectRollback()

	err = service.ProvisionTenantSchema(context.Background(), testTenant)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create tenant schema")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProvisionTenantSchema_NoTemplate tests provisioning without template
func TestProvisionTenantSchema_NoTemplate(t *testing.T) {
	// EDGE CASE: Plan has no template, should succeed with empty schema

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// No templates available
	mockTemplates := &mockTemplateRepository{
		templates: map[string]string{},
	}

	service := NewProvisioningService(db, mockTemplates, nil)

	testTenant := &tenant.Tenant{
		ID:         "test-tenant-id",
		SchemaName: "tenant_test_xyz789",
		Plan:       tenant.PlanFree, // Free plan might not have template
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE SCHEMA IF NOT EXISTS "tenant_test_xyz789"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET LOCAL search_path TO "tenant_test_xyz789", public`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// No template execution expected

	mock.ExpectExec("UPDATE tenants SET schema_name").
		WithArgs("tenant_test_xyz789", "test-tenant-id").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = service.ProvisionTenantSchema(context.Background(), testTenant)

	assert.NoError(t, err, "Should succeed even without template")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProvisionTenantSchema_InvalidSchemaName tests validation
func TestProvisionTenantSchema_InvalidSchemaName(t *testing.T) {
	// VALIDATION TEST: Invalid schema name should be rejected early

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewProvisioningService(db, nil, nil)

	testCases := []struct {
		name       string
		schemaName string
		errorMsg   string
	}{
		{
			name:       "uppercase letters",
			schemaName: "TENANT_INVALID",
			errorMsg:   "invalid",
		},
		{
			name:       "empty schema name",
			schemaName: "",
			errorMsg:   "required",
		},
		{
			name:       "special characters",
			schemaName: "tenant-with-dash",
			errorMsg:   "invalid",
		},
		{
			name:       "too long",
			schemaName: "tenant_" + string(make([]byte, 100)),
			errorMsg:   "invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testTenant := &tenant.Tenant{
				ID:         "test-id",
				SchemaName: tc.schemaName,
				Plan:       tenant.PlanFree,
			}

			// No database calls expected (validation fails first)
			err := service.ProvisionTenantSchema(context.Background(), testTenant)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetIsolationStrategy tests plan-to-isolation mapping
func TestGetIsolationStrategy(t *testing.T) {
	// STRATEGY MAPPING TEST

	service := &ProvisioningService{
		logger: logger.New(logger.Config{Level: "debug"}),
	}

	tests := []struct {
		plan             tenant.SubscriptionPlan
		expectedStrategy string
	}{
		{tenant.PlanFree, "schema-per-tenant"},
		{tenant.PlanStarter, "schema-per-tenant"},
		{tenant.PlanPro, "schema-per-tenant"},
		{tenant.PlanEnterprise, "schema-per-tenant"},
		{"unknown-plan", "schema-per-tenant"}, // Fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			strategy := service.getIsolationStrategy(tt.plan)
			assert.Equal(t, tt.expectedStrategy, strategy)
		})
	}
}

// TestProvisionTenantSchema_NilTenant tests nil tenant handling
func TestProvisionTenantSchema_NilTenant(t *testing.T) {
	// NIL CHECK: Defensive programming

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewProvisioningService(db, nil, nil)

	// No database calls expected
	err = service.ProvisionTenantSchema(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant entity is required")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// BenchmarkProvisionTenantSchema benchmarks provisioning performance
func BenchmarkProvisionTenantSchema(b *testing.B) {
	// PERFORMANCE BENCHMARK

	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	service := NewProvisioningService(db, &mockTemplateRepository{}, nil)

	testTenant := &tenant.Tenant{
		ID:         "bench-tenant",
		SchemaName: "tenant_bench",
		Plan:       tenant.PlanFree,
	}

	// Setup expectations for N iterations
	for i := 0; i < b.N; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`CREATE SCHEMA`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`SET LOCAL search_path`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("UPDATE tenants").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = service.ProvisionTenantSchema(context.Background(), testTenant)
	}
}

// Integration test notes
//
// 1. Isolation: every test must run independently.
// 2. **Cleanup:** defer ile resource cleanup
// 3. Expectations: verify every DB call through sqlmock.
// 4. **Edge Cases:** Nil, empty, invalid inputs test et
// 5. Rollback: transaction rollback scenarios matter.
// 6. **Performance:** Benchmark ile performance regression catch et
