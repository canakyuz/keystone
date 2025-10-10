package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"nexpaces-api/internal/domain/tenant"
	templateRepo "nexpaces-api/internal/repository/template"
	"nexpaces-api/pkg/logger"
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

	// TODO: Plan -> izolasyon eslestirmesini tamamla ve shared/database senaryolarini burada yonet.
	templateSQL, tplErr := s.templates.GetTemplateByPlan(ctx, string(t.Plan))
	switch {
	case tplErr == nil && strings.TrimSpace(templateSQL) != "":
		if _, err = tx.ExecContext(ctx, templateSQL); err != nil {
			return fmt.Errorf("failed to execute schema template for plan %s: %w", t.Plan, err)
		}
	case errors.Is(tplErr, templateRepo.ErrTemplateNotFound):
		// No template is acceptable; continue with empty schema
	case tplErr != nil:
		return fmt.Errorf("failed to fetch schema template: %w", tplErr)
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
