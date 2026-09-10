package lesson

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/canakyuz/keystone/examples/verticals/domain/lesson"
)

// StudentPostgresRepository implements StudentRepository using PostgreSQL
type StudentPostgresRepository struct {
	db *sql.DB
}

// NewStudentPostgresRepository creates a new student repository
func NewStudentPostgresRepository(db *sql.DB) *StudentPostgresRepository {
	return &StudentPostgresRepository{db: db}
}

// Create creates a new student
func (r *StudentPostgresRepository) Create(ctx context.Context, s *lesson.Student) error {
	query := `
		INSERT INTO students (
			id, tenant_id, first_name, last_name, email, phone, date_of_birth,
			status, level, grade, school, parent_name, parent_email, parent_phone,
			current_gpa, target_gpa, subjects, goals, notes, enrollment_date,
			avatar, metadata, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)
	`

	metadataJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		s.ID, s.TenantID, s.FirstName, s.LastName, s.Email, s.Phone, s.DateOfBirth,
		s.Status, s.Level, s.Grade, s.School, s.ParentName, s.ParentEmail, s.ParentPhone,
		s.CurrentGPA, s.TargetGPA, pq.Array(s.Subjects), s.Goals, s.Notes, s.EnrollmentDate,
		s.Avatar, metadataJSON, s.CreatedAt, s.UpdatedAt, s.CreatedBy, s.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create student: %w", err)
	}

	return nil
}

// GetByID retrieves a student by ID
func (r *StudentPostgresRepository) GetByID(ctx context.Context, id string) (*lesson.Student, error) {
	query := `
		SELECT id, tenant_id, first_name, last_name, email, phone, date_of_birth,
			   status, level, grade, school, parent_name, parent_email, parent_phone,
			   current_gpa, target_gpa, subjects, goals, notes, enrollment_date,
			   last_lesson_date, total_lessons, completed_lessons, cancelled_lessons,
			   attendance_rate, average_score, avatar, metadata,
			   created_at, updated_at, created_by, updated_by
		FROM students
		WHERE id = $1 AND deleted_at IS NULL
	`

	s := &lesson.Student{}
	var metadataJSON []byte
	var subjects []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.FirstName, &s.LastName, &s.Email, &s.Phone, &s.DateOfBirth,
		&s.Status, &s.Level, &s.Grade, &s.School, &s.ParentName, &s.ParentEmail, &s.ParentPhone,
		&s.CurrentGPA, &s.TargetGPA, pq.Array(&subjects), &s.Goals, &s.Notes, &s.EnrollmentDate,
		&s.LastLessonDate, &s.TotalLessons, &s.CompletedLessons, &s.CancelledLessons,
		&s.AttendanceRate, &s.AverageScore, &s.Avatar, &metadataJSON,
		&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, lesson.ErrStudentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	s.Subjects = subjects
	if err := json.Unmarshal(metadataJSON, &s.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return s, nil
}

// GetByEmail retrieves a student by email
func (r *StudentPostgresRepository) GetByEmail(ctx context.Context, email string) (*lesson.Student, error) {
	query := `
		SELECT id, tenant_id, first_name, last_name, email, phone, date_of_birth,
			   status, level, grade, school, parent_name, parent_email, parent_phone,
			   current_gpa, target_gpa, subjects, goals, notes, enrollment_date,
			   last_lesson_date, total_lessons, completed_lessons, cancelled_lessons,
			   attendance_rate, average_score, avatar, metadata,
			   created_at, updated_at, created_by, updated_by
		FROM students
		WHERE email = $1 AND deleted_at IS NULL
	`

	s := &lesson.Student{}
	var metadataJSON []byte
	var subjects []string

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&s.ID, &s.TenantID, &s.FirstName, &s.LastName, &s.Email, &s.Phone, &s.DateOfBirth,
		&s.Status, &s.Level, &s.Grade, &s.School, &s.ParentName, &s.ParentEmail, &s.ParentPhone,
		&s.CurrentGPA, &s.TargetGPA, pq.Array(&subjects), &s.Goals, &s.Notes, &s.EnrollmentDate,
		&s.LastLessonDate, &s.TotalLessons, &s.CompletedLessons, &s.CancelledLessons,
		&s.AttendanceRate, &s.AverageScore, &s.Avatar, &metadataJSON,
		&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, lesson.ErrStudentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get student by email: %w", err)
	}

	s.Subjects = subjects
	if err := json.Unmarshal(metadataJSON, &s.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return s, nil
}

// List retrieves students with filters
func (r *StudentPostgresRepository) List(ctx context.Context, filters lesson.StudentListFilters) ([]*lesson.Student, int64, error) {
	whereClause, args := buildStudentWhereClause(filters)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM students WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count students: %w", err)
	}

	// List query
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, first_name, last_name, email, phone, date_of_birth,
			   status, level, grade, school, parent_name, parent_email, parent_phone,
			   current_gpa, target_gpa, subjects, goals, notes, enrollment_date,
			   last_lesson_date, total_lessons, completed_lessons, cancelled_lessons,
			   attendance_rate, average_score, avatar, metadata,
			   created_at, updated_at, created_by, updated_by
		FROM students
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, len(args)+1, len(args)+2)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list students: %w", err)
	}
	defer rows.Close()

	students := make([]*lesson.Student, 0)
	for rows.Next() {
		s := &lesson.Student{}
		var metadataJSON []byte
		var subjects []string

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.FirstName, &s.LastName, &s.Email, &s.Phone, &s.DateOfBirth,
			&s.Status, &s.Level, &s.Grade, &s.School, &s.ParentName, &s.ParentEmail, &s.ParentPhone,
			&s.CurrentGPA, &s.TargetGPA, pq.Array(&subjects), &s.Goals, &s.Notes, &s.EnrollmentDate,
			&s.LastLessonDate, &s.TotalLessons, &s.CompletedLessons, &s.CancelledLessons,
			&s.AttendanceRate, &s.AverageScore, &s.Avatar, &metadataJSON,
			&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan student: %w", err)
		}

		s.Subjects = subjects
		if err := json.Unmarshal(metadataJSON, &s.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		students = append(students, s)
	}

	return students, total, nil
}

// Update updates a student
func (r *StudentPostgresRepository) Update(ctx context.Context, s *lesson.Student) error {
	query := `
		UPDATE students SET
			first_name = $2, last_name = $3, email = $4, phone = $5, date_of_birth = $6,
			status = $7, level = $8, grade = $9, school = $10,
			parent_name = $11, parent_email = $12, parent_phone = $13,
			current_gpa = $14, target_gpa = $15, subjects = $16, goals = $17, notes = $18,
			last_lesson_date = $19, total_lessons = $20, completed_lessons = $21,
			cancelled_lessons = $22, attendance_rate = $23, average_score = $24,
			avatar = $25, metadata = $26, updated_at = $27, updated_by = $28
		WHERE id = $1 AND deleted_at IS NULL
	`

	metadataJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	s.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		s.ID, s.FirstName, s.LastName, s.Email, s.Phone, s.DateOfBirth,
		s.Status, s.Level, s.Grade, s.School,
		s.ParentName, s.ParentEmail, s.ParentPhone,
		s.CurrentGPA, s.TargetGPA, pq.Array(s.Subjects), s.Goals, s.Notes,
		s.LastLessonDate, s.TotalLessons, s.CompletedLessons,
		s.CancelledLessons, s.AttendanceRate, s.AverageScore,
		s.Avatar, metadataJSON, s.UpdatedAt, s.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update student: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return lesson.ErrStudentNotFound
	}

	return nil
}

// Delete soft deletes a student
func (r *StudentPostgresRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE students SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return lesson.ErrStudentNotFound
	}

	return nil
}

// GetStats returns student statistics
func (r *StudentPostgresRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'inactive') as inactive,
			COUNT(*) FILTER (WHERE status = 'suspended') as suspended,
			COUNT(*) FILTER (WHERE status = 'graduated') as graduated,
			COUNT(*) FILTER (WHERE level = 'beginner') as beginner,
			COUNT(*) FILTER (WHERE level = 'intermediate') as intermediate,
			COUNT(*) FILTER (WHERE level = 'advanced') as advanced,
			AVG(attendance_rate) as avg_attendance,
			AVG(average_score) as avg_score
		FROM students
		WHERE deleted_at IS NULL
	`

	var total, active, inactive, suspended, graduated int64
	var beginner, intermediate, advanced int64
	var avgAttendance, avgScore float64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&total, &active, &inactive, &suspended, &graduated,
		&beginner, &intermediate, &advanced, &avgAttendance, &avgScore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := map[string]interface{}{
		"total":          total,
		"active":         active,
		"inactive":       inactive,
		"suspended":      suspended,
		"graduated":      graduated,
		"beginner":       beginner,
		"intermediate":   intermediate,
		"advanced":       advanced,
		"avg_attendance": avgAttendance,
		"avg_score":      avgScore,
	}

	return stats, nil
}

// buildStudentWhereClause builds WHERE clause for student queries
func buildStudentWhereClause(filters lesson.StudentListFilters) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argCount := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filters.Status)
		argCount++
	}

	if filters.Level != nil {
		conditions = append(conditions, fmt.Sprintf("level = $%d", argCount))
		args = append(args, *filters.Level)
		argCount++
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)",
			argCount, argCount, argCount,
		))
		args = append(args, searchPattern)
		argCount++
	}

	return strings.Join(conditions, " AND "), args
}
