package lesson

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/lesson"
	"github.com/lib/pq"
)

type LessonPostgresRepository struct {
	db *sql.DB
}

func NewLessonPostgresRepository(db *sql.DB) *LessonPostgresRepository {
	return &LessonPostgresRepository{db: db}
}

func (r *LessonPostgresRepository) Create(ctx context.Context, l *lesson.Lesson) error {
	query := `INSERT INTO lessons (id, tenant_id, student_id, title, description, subject, topic, status, type, scheduled_at, duration, location, meeting_url, materials, homework, notes, metadata, created_at, updated_at, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`
	metadataJSON, _ := json.Marshal(l.Metadata)
	_, err := r.db.ExecContext(ctx, query, l.ID, l.TenantID, l.StudentID, l.Title, l.Description, l.Subject, l.Topic, l.Status, l.Type, l.ScheduledAt, l.Duration, l.Location, l.MeetingURL, pq.Array(l.Materials), l.Homework, l.Notes, metadataJSON, l.CreatedAt, l.UpdatedAt, l.CreatedBy, l.UpdatedBy)
	return err
}

func (r *LessonPostgresRepository) GetByID(ctx context.Context, id string) (*lesson.Lesson, error) {
	query := `SELECT id, tenant_id, student_id, title, description, subject, topic, status, type, scheduled_at, started_at, completed_at, duration, location, meeting_url, materials, homework, notes, student_notes, performance_score, attendance_status, homework_completed, next_lesson_topic, next_lesson_date, metadata, created_at, updated_at FROM lessons WHERE id = $1 AND deleted_at IS NULL`
	l := &lesson.Lesson{}
	var metadataJSON []byte
	var materials []string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&l.ID, &l.TenantID, &l.StudentID, &l.Title, &l.Description, &l.Subject, &l.Topic, &l.Status, &l.Type, &l.ScheduledAt, &l.StartedAt, &l.CompletedAt, &l.Duration, &l.Location, &l.MeetingURL, pq.Array(&materials), &l.Homework, &l.Notes, &l.StudentNotes, &l.PerformanceScore, &l.AttendanceStatus, &l.HomeworkCompleted, &l.NextLessonTopic, &l.NextLessonDate, &metadataJSON, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, lesson.ErrLessonNotFound
	}
	l.Materials = materials
	json.Unmarshal(metadataJSON, &l.Metadata)
	return l, err
}

func (r *LessonPostgresRepository) List(ctx context.Context, filters lesson.LessonListFilters) ([]*lesson.Lesson, int64, error) {
	whereClause, args := buildLessonWhereClause(filters)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM lessons WHERE %s", whereClause)
	var total int64
	r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, student_id, title, subject, status, type, scheduled_at, duration, created_at, updated_at FROM lessons WHERE %s ORDER BY scheduled_at DESC LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, filters.Limit, filters.Offset)

	rows, _ := r.db.QueryContext(ctx, query, args...)
	defer rows.Close()

	lessons := []*lesson.Lesson{}
	for rows.Next() {
		l := &lesson.Lesson{}
		rows.Scan(&l.ID, &l.TenantID, &l.StudentID, &l.Title, &l.Subject, &l.Status, &l.Type, &l.ScheduledAt, &l.Duration, &l.CreatedAt, &l.UpdatedAt)
		lessons = append(lessons, l)
	}
	return lessons, total, nil
}

func (r *LessonPostgresRepository) GetByStudent(ctx context.Context, studentID string, limit, offset int) ([]*lesson.Lesson, int64, error) {
	query := `SELECT id, tenant_id, student_id, title, subject, status, scheduled_at, duration FROM lessons WHERE student_id = $1 AND deleted_at IS NULL ORDER BY scheduled_at DESC LIMIT $2 OFFSET $3`
	rows, _ := r.db.QueryContext(ctx, query, studentID, limit, offset)
	defer rows.Close()
	lessons := []*lesson.Lesson{}
	for rows.Next() {
		l := &lesson.Lesson{}
		rows.Scan(&l.ID, &l.TenantID, &l.StudentID, &l.Title, &l.Subject, &l.Status, &l.ScheduledAt, &l.Duration)
		lessons = append(lessons, l)
	}
	return lessons, int64(len(lessons)), nil
}

func (r *LessonPostgresRepository) Update(ctx context.Context, l *lesson.Lesson) error {
	query := `UPDATE lessons SET title=$2, description=$3, subject=$4, topic=$5, status=$6, started_at=$7, completed_at=$8, performance_score=$9, attendance_status=$10, homework_completed=$11, notes=$12, updated_at=$13 WHERE id=$1 AND deleted_at IS NULL`
	l.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query, l.ID, l.Title, l.Description, l.Subject, l.Topic, l.Status, l.StartedAt, l.CompletedAt, l.PerformanceScore, l.AttendanceStatus, l.HomeworkCompleted, l.Notes, l.UpdatedAt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return lesson.ErrLessonNotFound
	}
	return nil
}

func (r *LessonPostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE lessons SET deleted_at=$2 WHERE id=$1`, id, time.Now())
	return err
}

func (r *LessonPostgresRepository) GetUpcoming(ctx context.Context, limit int) ([]*lesson.Lesson, error) {
	query := `SELECT id, tenant_id, student_id, title, subject, scheduled_at, duration FROM lessons WHERE status='scheduled' AND scheduled_at > NOW() AND deleted_at IS NULL ORDER BY scheduled_at ASC LIMIT $1`
	rows, _ := r.db.QueryContext(ctx, query, limit)
	defer rows.Close()
	lessons := []*lesson.Lesson{}
	for rows.Next() {
		l := &lesson.Lesson{}
		rows.Scan(&l.ID, &l.TenantID, &l.StudentID, &l.Title, &l.Subject, &l.ScheduledAt, &l.Duration)
		lessons = append(lessons, l)
	}
	return lessons, nil
}

func (r *LessonPostgresRepository) GetStats(ctx context.Context) (map[string]any, error) {
	query := `SELECT COUNT(*) as total, COUNT(*) FILTER (WHERE status='completed') as completed, COUNT(*) FILTER (WHERE status='scheduled') as scheduled, COUNT(*) FILTER (WHERE status='cancelled') as cancelled FROM lessons WHERE deleted_at IS NULL`
	var total, completed, scheduled, cancelled int64
	r.db.QueryRowContext(ctx, query).Scan(&total, &completed, &scheduled, &cancelled)
	return map[string]any{"total": total, "completed": completed, "scheduled": scheduled, "cancelled": cancelled}, nil
}

func buildLessonWhereClause(filters lesson.LessonListFilters) (string, []any) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	argCount := 1
	if filters.StudentID != nil {
		conditions = append(conditions, fmt.Sprintf("student_id = $%d", argCount))
		args = append(args, *filters.StudentID)
		argCount++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}
	return strings.Join(conditions, " AND "), args
}
