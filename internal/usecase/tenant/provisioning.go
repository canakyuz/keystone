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

// getIsolationStrategy, subscription plan'e göre isolation stratejisini döndürür.
//
// 🎓 BACKEND KONSEPT: Plan-Based Multi-Tenancy Strategy
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
// 🎓 GO KONSEPT: Constant Returns (No Complex Logic)
// Bu fonksiyon şu anda basit bir mapping yapar.
// İleride database'den config okuyabilir (dynamic strategy)
func (s *ProvisioningService) getIsolationStrategy(plan tenant.SubscriptionPlan) string {
	switch plan {
	case tenant.PlanFree, tenant.PlanStarter, tenant.PlanPro:
		return "schema-per-tenant" // Default: Strong isolation, medium cost

	case tenant.PlanEnterprise:
		// 🎓 FUTURE: Dedicated database for enterprise
		// Şu an schema-per-tenant kullan, sonra migrate edilecek
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

	// 🎓 PLAN-TO-ISOLATION MAPPING
	//
	// Multi-Tenant Isolation Strategies:
	// 1. Schema-per-tenant (Current): Her tenant ayrı PostgreSQL schema
	//    - Free, Starter, Pro plans için varsayılan
	//    - Strong isolation, medium cost
	//
	// 2. Database-per-tenant: Her tenant ayrı database
	//    - Enterprise plan için (future)
	//    - Strongest isolation, high cost
	//
	// 3. Shared schema + RLS: Tüm tenant'lar aynı tablolarda
	//    - Development/Demo için (future)
	//    - Weak isolation, lowest cost
	//
	// Current Implementation: Schema-per-tenant for all plans
	isolationStrategy := s.getIsolationStrategy(t.Plan)

	switch isolationStrategy {
	case "schema-per-tenant":
		// Schema zaten oluşturuldu (line 67-70)
		// Şimdi plan-specific template uygula
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
			// Template yok, boş schema ile devam (acceptable)
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
		// 🎓 FUTURE: Enterprise plan için dedicated database
		// Bu durumda yeni bir database oluştur ve connection string döndür
		return fmt.Errorf("database-per-tenant isolation not yet implemented (enterprise plan)")

	case "shared-rls":
		// 🎓 FUTURE: Demo/Development için shared schema + Row Level Security
		// Bu durumda schema oluşturma, sadece RLS policy ekle
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
