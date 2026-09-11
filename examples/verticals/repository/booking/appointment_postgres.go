package booking

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/booking"
)

type AppointmentPostgresRepository struct {
	db *sql.DB
}

func NewAppointmentPostgresRepository(db *sql.DB) *AppointmentPostgresRepository {
	return &AppointmentPostgresRepository{db: db}
}

func (r *AppointmentPostgresRepository) Create(ctx context.Context, a *booking.Appointment) error {
	query := `INSERT INTO appointments (id, tenant_id, user_id, client_name, client_email, client_phone, title, description, type, status, start_time, end_time, duration, location, meeting_url, notes, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`
	metadataJSON, _ := json.Marshal(a.Metadata)
	_, err := r.db.ExecContext(ctx, query, a.ID, a.TenantID, a.UserID, a.ClientName, a.ClientEmail, a.ClientPhone, a.Title, a.Description, a.Type, a.Status, a.StartTime, a.EndTime, a.Duration, a.Location, a.MeetingURL, a.Notes, metadataJSON, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AppointmentPostgresRepository) GetByID(ctx context.Context, id string) (*booking.Appointment, error) {
	query := `SELECT id, tenant_id, user_id, client_name, client_email, client_phone, title, description, type, status, start_time, end_time, duration, location, meeting_url, notes, cancellation_reason, cancelled_at, confirmed_at, completed_at, reminder_sent, reminder_sent_at, metadata, created_at, updated_at FROM appointments WHERE id = $1 AND deleted_at IS NULL`
	a := &booking.Appointment{}
	var metadataJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.TenantID, &a.UserID, &a.ClientName, &a.ClientEmail, &a.ClientPhone, &a.Title, &a.Description, &a.Type, &a.Status, &a.StartTime, &a.EndTime, &a.Duration, &a.Location, &a.MeetingURL, &a.Notes, &a.CancellationReason, &a.CancelledAt, &a.ConfirmedAt, &a.CompletedAt, &a.ReminderSent, &a.ReminderSentAt, &metadataJSON, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, booking.ErrAppointmentNotFound
	}
	json.Unmarshal(metadataJSON, &a.Metadata)
	return a, err
}

func (r *AppointmentPostgresRepository) List(ctx context.Context, filters booking.AppointmentListFilters) ([]*booking.Appointment, int64, error) {
	whereClause, args := buildAppointmentWhereClause(filters)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM appointments WHERE %s", whereClause)
	var total int64
	r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, user_id, client_name, client_email, title, type, status, start_time, end_time FROM appointments WHERE %s ORDER BY start_time DESC LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, filters.Limit, filters.Offset)

	rows, _ := r.db.QueryContext(ctx, query, args...)
	defer rows.Close()

	appointments := []*booking.Appointment{}
	for rows.Next() {
		a := &booking.Appointment{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.ClientName, &a.ClientEmail, &a.Title, &a.Type, &a.Status, &a.StartTime, &a.EndTime)
		appointments = append(appointments, a)
	}
	return appointments, total, nil
}

func (r *AppointmentPostgresRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*booking.Appointment, int64, error) {
	query := `SELECT id, tenant_id, user_id, client_name, client_email, title, type, status, start_time, end_time FROM appointments WHERE user_id = $1 AND deleted_at IS NULL ORDER BY start_time DESC LIMIT $2 OFFSET $3`
	rows, _ := r.db.QueryContext(ctx, query, userID, limit, offset)
	defer rows.Close()
	appointments := []*booking.Appointment{}
	for rows.Next() {
		a := &booking.Appointment{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.ClientName, &a.ClientEmail, &a.Title, &a.Type, &a.Status, &a.StartTime, &a.EndTime)
		appointments = append(appointments, a)
	}
	return appointments, int64(len(appointments)), nil
}

func (r *AppointmentPostgresRepository) GetByClient(ctx context.Context, clientEmail string, limit, offset int) ([]*booking.Appointment, int64, error) {
	query := `SELECT id, tenant_id, client_name, client_email, title, type, status, start_time, end_time FROM appointments WHERE client_email = $1 AND deleted_at IS NULL ORDER BY start_time DESC LIMIT $2 OFFSET $3`
	rows, _ := r.db.QueryContext(ctx, query, clientEmail, limit, offset)
	defer rows.Close()
	appointments := []*booking.Appointment{}
	for rows.Next() {
		a := &booking.Appointment{}
		rows.Scan(&a.ID, &a.TenantID, &a.ClientName, &a.ClientEmail, &a.Title, &a.Type, &a.Status, &a.StartTime, &a.EndTime)
		appointments = append(appointments, a)
	}
	return appointments, int64(len(appointments)), nil
}

func (r *AppointmentPostgresRepository) GetUpcoming(ctx context.Context, limit int) ([]*booking.Appointment, error) {
	query := `SELECT id, tenant_id, user_id, client_name, title, start_time, end_time FROM appointments WHERE status IN ('pending', 'confirmed') AND start_time > NOW() AND deleted_at IS NULL ORDER BY start_time ASC LIMIT $1`
	rows, _ := r.db.QueryContext(ctx, query, limit)
	defer rows.Close()
	appointments := []*booking.Appointment{}
	for rows.Next() {
		a := &booking.Appointment{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.ClientName, &a.Title, &a.StartTime, &a.EndTime)
		appointments = append(appointments, a)
	}
	return appointments, nil
}

func (r *AppointmentPostgresRepository) GetByDateRange(ctx context.Context, startDate, endDate string) ([]*booking.Appointment, error) {
	query := `SELECT id, tenant_id, user_id, client_name, title, start_time, end_time, status FROM appointments WHERE start_time >= $1 AND end_time <= $2 AND deleted_at IS NULL ORDER BY start_time ASC`
	rows, _ := r.db.QueryContext(ctx, query, startDate, endDate)
	defer rows.Close()
	appointments := []*booking.Appointment{}
	for rows.Next() {
		a := &booking.Appointment{}
		rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.ClientName, &a.Title, &a.StartTime, &a.EndTime, &a.Status)
		appointments = append(appointments, a)
	}
	return appointments, nil
}

func (r *AppointmentPostgresRepository) Update(ctx context.Context, a *booking.Appointment) error {
	query := `UPDATE appointments SET title=$2, description=$3, status=$4, confirmed_at=$5, cancelled_at=$6, completed_at=$7, cancellation_reason=$8, notes=$9, updated_at=$10 WHERE id=$1 AND deleted_at IS NULL`
	a.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query, a.ID, a.Title, a.Description, a.Status, a.ConfirmedAt, a.CancelledAt, a.CompletedAt, a.CancellationReason, a.Notes, a.UpdatedAt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return booking.ErrAppointmentNotFound
	}
	return nil
}

func (r *AppointmentPostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE appointments SET deleted_at=$2 WHERE id=$1`, id, time.Now())
	return err
}

func (r *AppointmentPostgresRepository) GetStats(ctx context.Context) (map[string]any, error) {
	query := `SELECT COUNT(*) as total, COUNT(*) FILTER (WHERE status='pending') as pending, COUNT(*) FILTER (WHERE status='confirmed') as confirmed, COUNT(*) FILTER (WHERE status='completed') as completed, COUNT(*) FILTER (WHERE status='cancelled') as cancelled FROM appointments WHERE deleted_at IS NULL`
	var total, pending, confirmed, completed, cancelled int64
	r.db.QueryRowContext(ctx, query).Scan(&total, &pending, &confirmed, &completed, &cancelled)
	return map[string]any{"total": total, "pending": pending, "confirmed": confirmed, "completed": completed, "cancelled": cancelled}, nil
}

func buildAppointmentWhereClause(filters booking.AppointmentListFilters) (string, []any) {
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
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argCount))
		args = append(args, *filters.Type)
		argCount++
	}
	return strings.Join(conditions, " AND "), args
}
