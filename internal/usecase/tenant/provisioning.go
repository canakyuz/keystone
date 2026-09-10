package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/canakyuz/keystone/internal/domain/tenant"
	templateRepo "github.com/canakyuz/keystone/internal/repository/template"
	"github.com/canakyuz/keystone/pkg/logger"
)

// ProvisioningService is responsible for provisioning schema-per-tenant resources.
type ProvisioningService struct {
	db        *sql.DB
	templates templateRepo.Repository
	logger    *logger.Logger
}

// NewProvisioningService creates a new provisioning service instance.
func NewProvisioningService(db *sql.DB, templates templateRepo.Repository, log *logger.Logger) *ProvisioningService {
	return &ProvisioningService{
		db:        db,
		templates: templates,
		logger:    log,
	}
}

// getIsolationStrategy returns the isolation strategy for a subscription plan.
//
// # Plan-Based Multi-Tenancy Strategy
//
// Isolation Strategies Comparison:
// ┌─────────────────────┬──────────────┬──────────┬────────────┐
// │ Strategy            │ Isolation    │ Cost     │ Use Case   │
// ├─────────────────────┼──────────────┼──────────┼────────────┤
// │ Schema-per-tenant   │ Strong       │ Medium   │ Most plans │
// │ Database-per-tenant │ Strongest    │ High     │ Enterprise │
// │ Shared + RLS        │ Weak         │ Low      │ Demo/Dev   │
// └─────────────────────┴──────────────┴──────────┴────────────┘
//
// Constant Returns (No Complex Logic)
// It is a plain mapping today. It could read configuration from the database later.
func (s *ProvisioningService) getIsolationStrategy(plan tenant.SubscriptionPlan) string {
	switch plan {
	case tenant.PlanFree, tenant.PlanStarter, tenant.PlanPro:
		return "schema-per-tenant" // Default: Strong isolation, medium cost

	case tenant.PlanEnterprise:
		// Dedicated database for enterprise
		// Uses schema-per-tenant for now; to be migrated later.
		return "schema-per-tenant"

	default:
		// Unknown plan, fallback to schema-per-tenant
		if s.logger != nil {
			s.logger.WithFields(logger.Fields{
				"plan": plan,
			}).Warn("Unknown plan, falling back to schema-per-tenant")
		}
		return "schema-per-tenant"
	}
}

// GenerateSchemaName delegates to the database helper function that guarantees uniqueness.
func (s *ProvisioningService) GenerateSchemaName(ctx context.Context, base string) (string, error) {
	var schemaName string
	if err := s.db.QueryRowContext(ctx, "SELECT generate_schema_name($1)", base).Scan(&schemaName); err != nil {
		return "", fmt.Errorf("failed to generate schema name: %w", err)
	}

	if err := tenant.ValidateSchemaName(schemaName); err != nil {
		return "", err
	}

	return schemaName, nil
}

// ProvisionTenantSchema creates the tenant schema and executes plan-specific SQL template.
func (s *ProvisioningService) ProvisionTenantSchema(ctx context.Context, t *tenant.Tenant) error {
	if t == nil {
		return errors.New("tenant entity is required for provisioning")
	}
	if err := tenant.ValidateSchemaName(t.SchemaName); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tenant provisioning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	createStmt := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pq.QuoteIdentifier(t.SchemaName))
	if _, err = tx.ExecContext(ctx, createStmt); err != nil {
		return fmt.Errorf("failed to create tenant schema %s: %w", t.SchemaName, err)
	}

	setPathStmt := fmt.Sprintf("SET LOCAL search_path TO %s, public", pq.QuoteIdentifier(t.SchemaName))
	if _, err = tx.ExecContext(ctx, setPathStmt); err != nil {
		return fmt.Errorf("failed to set tenant search_path: %w", err)
	}

	//
	// Multi-Tenant Isolation Strategies:
	// 1. Schema-per-tenant (current): a separate PostgreSQL schema per tenant.
	//    The default for the free, starter and pro plans.
	//    - Strong isolation, medium cost
	//
	// 2. Database-per-tenant: a separate database per tenant.
	//    Intended for the enterprise plan; not implemented.
	//    - Strongest isolation, high cost
	//
	// 3. Shared schema plus RLS: every tenant in the same tables.
	//    Intended for development and demo; not implemented.
	//    - Weak isolation, lowest cost
	//
	// Current Implementation: Schema-per-tenant for all plans
	isolationStrategy := s.getIsolationStrategy(t.Plan)

	switch isolationStrategy {
	case "schema-per-tenant":
		// The schema already exists at this point; apply the plan-specific template.
		templateSQL, tplErr := s.templates.GetTemplateByPlan(ctx, string(t.Plan))
		switch {
		case tplErr == nil && strings.TrimSpace(templateSQL) != "":
			// Template bulundu, execute et
			if _, err = tx.ExecContext(ctx, templateSQL); err != nil {
				return fmt.Errorf("failed to execute schema template for plan %s: %w", t.Plan, err)
			}

			if s.logger != nil {
				s.logger.WithFields(logger.Fields{
					"tenant_id": t.ID,
					"plan":      t.Plan,
					"template":  "executed",
				}).Debug("Schema template applied")
			}

		case errors.Is(tplErr, templateRepo.ErrTemplateNotFound):
			// No template: continuing with an empty schema is acceptable.
			if s.logger != nil {
				s.logger.WithFields(logger.Fields{
					"tenant_id": t.ID,
					"plan":      t.Plan,
				}).Debug("No template found for plan, continuing with empty schema")
			}

		case tplErr != nil:
			// Template fetch error (unexpected)
			return fmt.Errorf("failed to fetch schema template: %w", tplErr)
		}

	case "database-per-tenant":
		// Not implemented: a dedicated database for the enterprise plan would be created
		// here, returning its connection string.
		return fmt.Errorf("database-per-tenant isolation not yet implemented (enterprise plan)")

	case "shared-rls":
		// Not implemented: for a shared schema with Row Level Security no schema would be
		// created here, only the RLS policies added.
		return fmt.Errorf("shared-rls isolation not yet implemented (demo/dev plan)")

	default:
		return fmt.Errorf("unknown isolation strategy: %s", isolationStrategy)
	}

	if _, err = tx.ExecContext(ctx, "UPDATE tenants SET schema_name = $1 WHERE id = $2", t.SchemaName, t.ID); err != nil {
		return fmt.Errorf("failed to persist tenant schema name: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tenant provisioning: %w", err)
	}

	if s.logger != nil {
		s.logger.WithFields(logger.Fields{
			"tenant_id":   t.ID,
			"schema_name": t.SchemaName,
			"plan":        t.Plan,
		}).Info("Tenant schema provisioned")
	}

	return nil
}
