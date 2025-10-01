package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"nexspaces-api/internal/domain/service"
)

type ServicePostgresRepository struct {
	db *sql.DB
}

func NewServicePostgresRepository(db *sql.DB) *ServicePostgresRepository {
	return &ServicePostgresRepository{db: db}
}

func (r *ServicePostgresRepository) Create(ctx context.Context, s *service.Service) error {
	query := `INSERT INTO services (id, tenant_id, name, slug, description, category, status, features, base_price, currency, pricing_model, billing_cycle, duration, max_clients, is_public, featured, image, metadata, created_at, updated_at, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`
	metadataJSON, _ := json.Marshal(s.Metadata)
	_, err := r.db.ExecContext(ctx, query, s.ID, s.TenantID, s.Name, s.Slug, s.Description, s.Category, s.Status, pq.Array(s.Features), s.BasePrice, s.Currency, s.PricingModel, s.BillingCycle, s.Duration, s.MaxClients, s.IsPublic, s.Featured, s.Image, metadataJSON, s.CreatedAt, s.UpdatedAt, s.CreatedBy, s.UpdatedBy)
	return err
}

func (r *ServicePostgresRepository) GetByID(ctx context.Context, id string) (*service.Service, error) {
	query := `SELECT id, tenant_id, name, slug, description, category, status, features, base_price, currency, pricing_model, billing_cycle, duration, max_clients, is_public, featured, image, metadata, created_at, updated_at FROM services WHERE id = $1 AND deleted_at IS NULL`
	s := &service.Service{}
	var metadataJSON []byte
	var features []string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.Description, &s.Category, &s.Status, pq.Array(&features), &s.BasePrice, &s.Currency, &s.PricingModel, &s.BillingCycle, &s.Duration, &s.MaxClients, &s.IsPublic, &s.Featured, &s.Image, &metadataJSON, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrServiceNotFound
	}
	s.Features = features
	json.Unmarshal(metadataJSON, &s.Metadata)
	return s, err
}

func (r *ServicePostgresRepository) GetBySlug(ctx context.Context, slug string) (*service.Service, error) {
	query := `SELECT id, tenant_id, name, slug, description, category, status, base_price, currency, pricing_model, featured FROM services WHERE slug = $1 AND deleted_at IS NULL`
	s := &service.Service{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.Description, &s.Category, &s.Status, &s.BasePrice, &s.Currency, &s.PricingModel, &s.Featured)
	if err == sql.ErrNoRows {
		return nil, service.ErrServiceNotFound
	}
	return s, err
}

func (r *ServicePostgresRepository) List(ctx context.Context, filters service.ServiceListFilters) ([]*service.Service, int64, error) {
	whereClause, args := buildServiceWhereClause(filters)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM services WHERE %s", whereClause)
	var total int64
	r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, name, slug, category, status, base_price, currency, featured FROM services WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, filters.Limit, filters.Offset)

	rows, _ := r.db.QueryContext(ctx, query, args...)
	defer rows.Close()

	services := []*service.Service{}
	for rows.Next() {
		s := &service.Service{}
		rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.Category, &s.Status, &s.BasePrice, &s.Currency, &s.Featured)
		services = append(services, s)
	}
	return services, total, nil
}

func (r *ServicePostgresRepository) GetFeatured(ctx context.Context, limit int) ([]*service.Service, error) {
	query := `SELECT id, tenant_id, name, slug, description, base_price, currency FROM services WHERE featured = TRUE AND status = 'active' AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $1`
	rows, _ := r.db.QueryContext(ctx, query, limit)
	defer rows.Close()
	services := []*service.Service{}
	for rows.Next() {
		s := &service.Service{}
		rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.Description, &s.BasePrice, &s.Currency)
		services = append(services, s)
	}
	return services, nil
}

func (r *ServicePostgresRepository) GetByCategory(ctx context.Context, category service.ServiceCategory, limit, offset int) ([]*service.Service, int64, error) {
	query := `SELECT id, tenant_id, name, slug, base_price, currency FROM services WHERE category = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, _ := r.db.QueryContext(ctx, query, category, limit, offset)
	defer rows.Close()
	services := []*service.Service{}
	for rows.Next() {
		s := &service.Service{}
		rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.BasePrice, &s.Currency)
		services = append(services, s)
	}
	return services, int64(len(services)), nil
}

func (r *ServicePostgresRepository) Update(ctx context.Context, s *service.Service) error {
	query := `UPDATE services SET name=$2, description=$3, status=$4, base_price=$5, featured=$6, updated_at=$7 WHERE id=$1 AND deleted_at IS NULL`
	s.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query, s.ID, s.Name, s.Description, s.Status, s.BasePrice, s.Featured, s.UpdatedAt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrServiceNotFound
	}
	return nil
}

func (r *ServicePostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE services SET deleted_at=$2 WHERE id=$1`, id, time.Now())
	return err
}

func (r *ServicePostgresRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT COUNT(*) as total, COUNT(*) FILTER (WHERE status='active') as active, COUNT(*) FILTER (WHERE featured=TRUE) as featured FROM services WHERE deleted_at IS NULL`
	var total, active, featured int64
	r.db.QueryRowContext(ctx, query).Scan(&total, &active, &featured)
	return map[string]interface{}{"total": total, "active": active, "featured": featured}, nil
}

func buildServiceWhereClause(filters service.ServiceListFilters) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argCount := 1
	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argCount))
		args = append(args, *filters.Category)
		argCount++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}
	if filters.Featured != nil {
		conditions = append(conditions, fmt.Sprintf("featured = $%d", argCount))
		args = append(args, *filters.Featured)
		argCount++
	}
	return strings.Join(conditions, " AND "), args
}
