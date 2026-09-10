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

type AssignmentPostgresRepository struct {
	db *sql.DB
}

func NewAssignmentPostgresRepository(db *sql.DB) *AssignmentPostgresRepository {
	return &AssignmentPostgresRepository{db: db}
}

func (r *AssignmentPostgresRepository) Create(ctx context.Context, a *lesson.Assignment) error {
	query := `INSERT INTO assignments (id, tenant_id, student_id, lesson_id, title, description, subject, status, due_date, max_score, files, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	metadataJSON, _ := json.Marshal(a.Metadata)
	_, err := r.db.ExecContext(ctx, query, a.ID, a.TenantID, a.StudentID, a.LessonID, a.Title, a.Description, a.Subject, a.Status, a.DueDate, a.MaxScore, pq.Array(a.Files), metadataJSON, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AssignmentPostgresRepository) GetByID(ctx context.Context, id string) (*lesson.Assignment, error) {
	query := `SELECT id, tenant_id, student_id, lesson_id, title, description, subject, status, due_date, submitted_at, graded_at, score, max_score, feedback, files, metadata, created_at, updated_at FROM assignments WHERE id = $1 AND deleted_at IS NULL`
	a := &lesson.Assignment{}
	var metadataJSON []byte
	var files []string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.TenantID, &a.StudentID, &a.LessonID, &a.Title, &a.Description, &a.Subject, &a.Status, &a.DueDate, &a.SubmittedAt, &a.GradedAt, &a.Score, &a.MaxScore, &a.Feedback, pq.Array(&files), &metadataJSON, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, lesson.ErrAssignmentNotFound
	}
	a.Files = files
	json.Unmarshal(metadataJSON, &a.Metadata)
	return a, err
}

func (r *AssignmentPostgresRepository) List(ctx context.Context, filters lesson.AssignmentListFilters) ([]*lesson.Assignment, int64, error) {
	whereClause, args := buildAssignmentWhereClause(filters)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM assignments WHERE %s", whereClause)
	var total int64
	r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, student_id, title, subject, status, due_date FROM assignments WHERE %s ORDER BY due_date DESC LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, filters.Limit, filters.Offset)

	rows, _ := r.db.QueryContext(ctx, query, args...)
	defer rows.Close()

	assignments := []*lesson.Assignment{}
	for rows.Next() {
		a := &lesson.Assignment{}
		rows.Scan(&a.ID, &a.TenantID, &a.StudentID, &a.Title, &a.Subject, &a.Status, &a.DueDate)
		assignments = append(assignments, a)
	}
	return assignments, total, nil
}

func (r *AssignmentPostgresRepository) GetByStudent(ctx context.Context, studentID string, limit, offset int) ([]*lesson.Assignment, int64, error) {
	query := `SELECT id, tenant_id, student_id, title, subject, status, due_date FROM assignments WHERE student_id = $1 AND deleted_at IS NULL ORDER BY due_date DESC LIMIT $2 OFFSET $3`
	rows, _ := r.db.QueryContext(ctx, query, studentID, limit, offset)
	defer rows.Close()
	assignments := []*lesson.Assignment{}
	for rows.Next() {
		a := &lesson.Assignment{}
		rows.Scan(&a.ID, &a.TenantID, &a.StudentID, &a.Title, &a.Subject, &a.Status, &a.DueDate)
		assignments = append(assignments, a)
	}
	return assignments, int64(len(assignments)), nil
}

func (r *AssignmentPostgresRepository) GetOverdue(ctx context.Context) ([]*lesson.Assignment, error) {
	query := `SELECT id, tenant_id, student_id, title, due_date FROM assignments WHERE status='pending' AND due_date < NOW() AND deleted_at IS NULL`
	rows, _ := r.db.QueryContext(ctx, query)
	defer rows.Close()
	assignments := []*lesson.Assignment{}
	for rows.Next() {
		a := &lesson.Assignment{}
		rows.Scan(&a.ID, &a.TenantID, &a.StudentID, &a.Title, &a.DueDate)
		assignments = append(assignments, a)
	}
	return assignments, nil
}

func (r *AssignmentPostgresRepository) Update(ctx context.Context, a *lesson.Assignment) error {
	query := `UPDATE assignments SET title=$2, description=$3, status=$4, submitted_at=$5, graded_at=$6, score=$7, feedback=$8, updated_at=$9 WHERE id=$1 AND deleted_at IS NULL`
	a.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query, a.ID, a.Title, a.Description, a.Status, a.SubmittedAt, a.GradedAt, a.Score, a.Feedback, a.UpdatedAt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return lesson.ErrAssignmentNotFound
	}
	return nil
}

func (r *AssignmentPostgresRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE assignments SET deleted_at=$2 WHERE id=$1`, id, time.Now())
	return err
}

func (r *AssignmentPostgresRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT COUNT(*) as total, COUNT(*) FILTER (WHERE status='pending') as pending, COUNT(*) FILTER (WHERE status='submitted') as submitted, COUNT(*) FILTER (WHERE status='graded') as graded FROM assignments WHERE deleted_at IS NULL`
	var total, pending, submitted, graded int64
	r.db.QueryRowContext(ctx, query).Scan(&total, &pending, &submitted, &graded)
	return map[string]interface{}{"total": total, "pending": pending, "submitted": submitted, "graded": graded}, nil
}

func buildAssignmentWhereClause(filters lesson.AssignmentListFilters) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
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
