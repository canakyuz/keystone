package booking

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/booking"
	"github.com/lib/pq"
)

type AvailabilityPostgresRepository struct {
	db *sql.DB
}

func NewAvailabilityPostgresRepository(db *sql.DB) *AvailabilityPostgresRepository {
	return &AvailabilityPostgresRepository{db: db}
}

func (r *AvailabilityPostgresRepository) Create(ctx context.Context, a *booking.Availability) error {
	query := `INSERT INTO availabilities (id, tenant_id, user_id, title, description, start_time, end_time, status, recurrence, recurrence_end, days_of_week, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	metadataJSON, _ := json.Marshal(a.Metadata)
	_, err := r.db.ExecContext(ctx, query, a.ID, a.TenantID, a.UserID, a.Title, a.Description, a.StartTime, a.EndTime, a.Status, a.Recurrence, a.RecurrenceEnd, pq.Array(a.DaysOfWeek), metadataJSON, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AvailabilityPostgresRepository) GetByID(ctx context.Context, id string) (*booking.Availability, error) {
	query := `SELECT id, tenant_id, user_id, title, description, start_time, end_time, status, recurrence, recurrence_end, days_of_week, metadata, created_at, updated_at FROM availabilities WHERE id = $1 AND deleted_at IS NULL`
	a := &booking.Availability{}
	var metadataJSON []byte
	var daysOfWeek []int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.TenantID, &a.UserID, &a.Title, &a.Description, &a.StartTime, &a.EndTime, &a.Status, &a.Recurrence, &a.RecurrenceEnd, pq.Array(&daysOfWeek), &metadataJSON, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, booking.ErrAvailabilityNotFound
	}
	a.DaysOfWeek = daysOfWeek
	json.Unmarshal(metadataJSON, &a.Metadata)
	return a, err
}

func (r *AvailabilityPostgresRepository) List(ctx context.Context, filters booking.AvailabilityListFilters) ([]*booking.Availability, int64, error) {
	whereClause, args := buildAvailabilityWhereClause(filters)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM availabilities WHERE %s", whereClause)
	var total int64
	r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, user_id, title, start_time, end_time, status FROM availabilities WHERE %s ORDER BY start_time DESC LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, filters.Limit, filters.Offset)

	rows, _ := r.db.QueryContext(ctx, query, args...)
	defer rows.Close()

	availabilities := []*booking.Availability{}
	for rows.Next() {
		a := &booking.Availability{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.Title, &a.StartTime, &a.EndTime, &a.Status)
		availabilities = append(availabilities, a)
	}
	return availabilities, total, nil
}

func (r *AvailabilityPostgresRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*booking.Availability, int64, error) {
	query := `SELECT id, tenant_id, user_id, title, start_time, end_time, status FROM availabilities WHERE user_id = $1 AND deleted_at IS NULL ORDER BY start_time DESC LIMIT $2 OFFSET $3`
	rows, _ := r.db.QueryContext(ctx, query, userID, limit, offset)
	defer rows.Close()
	availabilities := []*booking.Availability{}
	for rows.Next() {
		a := &booking.Availability{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.Title, &a.StartTime, &a.EndTime, &a.Status)
		availabilities = append(availabilities, a)
	}
	return availabilities, int64(len(availabilities)), nil
}

func (r *AvailabilityPostgresRepository) GetByDateRange(ctx context.Context, startDate, endDate string) ([]*booking.Availability, error) {
	query := `SELECT id, tenant_id, user_id, title, start_time, end_time, status FROM availabilities WHERE start_time >= $1 AND end_time <= $2 AND deleted_at IS NULL ORDER BY start_time ASC`
	rows, _ := r.db.QueryContext(ctx, query, startDate, endDate)
	defer rows.Close()
	availabilities := []*booking.Availability{}
	for rows.Next() {
		a := &booking.Availability{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.Title, &a.StartTime, &a.EndTime, &a.Status)
		availabilities = append(availabilities, a)
	}
	return availabilities, nil
}

func (r *AvailabilityPostgresRepository) Update(ctx context.Context, a *booking.Availability) error {
	query := `UPDATE availabilities SET title=$2, description=$3, start_time=$4, end_time=$5, status=$6, updated_at=$7 WHERE id=$1 AND deleted_at IS NULL`
	a.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query, a.ID, a.Title, a.Description, a.StartTime, a.EndTime, a.Status, a.UpdatedAt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return booking.ErrAvailabilityNotFound
	}
	return nil
}

func (r *AvailabilityPostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE availabilities SET deleted_at=$2 WHERE id=$1`, id, time.Now())
	return err
}

func (r *AvailabilityPostgresRepository) GetStats(ctx context.Context) (map[string]any, error) {
	query := `SELECT COUNT(*) as total, COUNT(*) FILTER (WHERE status='available') as available, COUNT(*) FILTER (WHERE status='booked') as booked, COUNT(*) FILTER (WHERE status='unavailable') as unavailable FROM availabilities WHERE deleted_at IS NULL`
	var total, available, booked, unavailable int64
	r.db.QueryRowContext(ctx, query).Scan(&total, &available, &booked, &unavailable)
	return map[string]any{"total": total, "available": available, "booked": booked, "unavailable": unavailable}, nil
}

func buildAvailabilityWhereClause(filters booking.AvailabilityListFilters) (string, []any) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	argCount := 1
	if filters.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argCount))
		args = append(args, *filters.UserID)
		argCount++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}
	return strings.Join(conditions, " AND "), args
}
