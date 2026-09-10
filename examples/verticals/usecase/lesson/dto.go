package lesson

import "time"

// Student DTOs
type CreateStudentRequest struct {
	FirstName   string     `json:"first_name" validate:"required"`
	LastName    string     `json:"last_name" validate:"required"`
	Email       string     `json:"email" validate:"required,email"`
	Phone       string     `json:"phone,omitempty"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Status      string     `json:"status" validate:"required,oneof=active inactive suspended graduated"`
	Level       string     `json:"level" validate:"required,oneof=beginner intermediate advanced"`
	Grade       string     `json:"grade,omitempty"`
	School      string     `json:"school,omitempty"`
	ParentName  string     `json:"parent_name,omitempty"`
	ParentEmail string     `json:"parent_email,omitempty"`
	ParentPhone string     `json:"parent_phone,omitempty"`
	CurrentGPA  float64    `json:"current_gpa,omitempty"`
	TargetGPA   float64    `json:"target_gpa,omitempty"`
	Subjects    []string   `json:"subjects,omitempty"`
	Goals       string     `json:"goals,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	Avatar      string     `json:"avatar,omitempty"`
}

type UpdateStudentRequest struct {
	FirstName   string     `json:"first_name,omitempty"`
	LastName    string     `json:"last_name,omitempty"`
	Email       string     `json:"email,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Status      string     `json:"status,omitempty"`
	Level       string     `json:"level,omitempty"`
	Grade       string     `json:"grade,omitempty"`
	School      string     `json:"school,omitempty"`
	ParentName  string     `json:"parent_name,omitempty"`
	ParentEmail string     `json:"parent_email,omitempty"`
	ParentPhone string     `json:"parent_phone,omitempty"`
	CurrentGPA  float64    `json:"current_gpa,omitempty"`
	TargetGPA   float64    `json:"target_gpa,omitempty"`
	Subjects    []string   `json:"subjects,omitempty"`
	Goals       string     `json:"goals,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	Avatar      string     `json:"avatar,omitempty"`
}

type StudentResponse struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	FirstName        string                 `json:"first_name"`
	LastName         string                 `json:"last_name"`
	Email            string                 `json:"email"`
	Phone            string                 `json:"phone,omitempty"`
	DateOfBirth      *time.Time             `json:"date_of_birth,omitempty"`
	Status           string                 `json:"status"`
	Level            string                 `json:"level"`
	Grade            string                 `json:"grade,omitempty"`
	School           string                 `json:"school,omitempty"`
	ParentName       string                 `json:"parent_name,omitempty"`
	ParentEmail      string                 `json:"parent_email,omitempty"`
	ParentPhone      string                 `json:"parent_phone,omitempty"`
	CurrentGPA       float64                `json:"current_gpa,omitempty"`
	TargetGPA        float64                `json:"target_gpa,omitempty"`
	Subjects         []string               `json:"subjects,omitempty"`
	Goals            string                 `json:"goals,omitempty"`
	Notes            string                 `json:"notes,omitempty"`
	EnrollmentDate   time.Time              `json:"enrollment_date"`
	LastLessonDate   *time.Time             `json:"last_lesson_date,omitempty"`
	TotalLessons     int                    `json:"total_lessons"`
	CompletedLessons int                    `json:"completed_lessons"`
	CancelledLessons int                    `json:"cancelled_lessons"`
	AttendanceRate   float64                `json:"attendance_rate"`
	AverageScore     float64                `json:"average_score"`
	Avatar           string                 `json:"avatar,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type StudentListResponse struct {
	Students []StudentResponse `json:"students"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// Lesson DTOs
type CreateLessonRequest struct {
	StudentID   string    `json:"student_id" validate:"required"`
	Title       string    `json:"title" validate:"required"`
	Description string    `json:"description,omitempty"`
	Subject     string    `json:"subject" validate:"required"`
	Topic       string    `json:"topic,omitempty"`
	Status      string    `json:"status" validate:"required,oneof=scheduled in-progress completed cancelled no-show"`
	Type        string    `json:"type" validate:"required,oneof=one-on-one group online in-person"`
	ScheduledAt time.Time `json:"scheduled_at" validate:"required"`
	Duration    int       `json:"duration" validate:"required"`
	Location    string    `json:"location,omitempty"`
	MeetingURL  string    `json:"meeting_url,omitempty"`
	Materials   []string  `json:"materials,omitempty"`
	Homework    string    `json:"homework,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

type UpdateLessonRequest struct {
	Title             string     `json:"title,omitempty"`
	Description       string     `json:"description,omitempty"`
	Subject           string     `json:"subject,omitempty"`
	Topic             string     `json:"topic,omitempty"`
	Status            string     `json:"status,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	PerformanceScore  int        `json:"performance_score,omitempty"`
	AttendanceStatus  string     `json:"attendance_status,omitempty"`
	HomeworkCompleted bool       `json:"homework_completed,omitempty"`
	Notes             string     `json:"notes,omitempty"`
}

type LessonResponse struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	StudentID         string                 `json:"student_id"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description,omitempty"`
	Subject           string                 `json:"subject"`
	Topic             string                 `json:"topic,omitempty"`
	Status            string                 `json:"status"`
	Type              string                 `json:"type"`
	ScheduledAt       time.Time              `json:"scheduled_at"`
	StartedAt         *time.Time             `json:"started_at,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	Duration          int                    `json:"duration"`
	Location          string                 `json:"location,omitempty"`
	MeetingURL        string                 `json:"meeting_url,omitempty"`
	Materials         []string               `json:"materials,omitempty"`
	Homework          string                 `json:"homework,omitempty"`
	Notes             string                 `json:"notes,omitempty"`
	StudentNotes      string                 `json:"student_notes,omitempty"`
	PerformanceScore  int                    `json:"performance_score,omitempty"`
	AttendanceStatus  string                 `json:"attendance_status,omitempty"`
	HomeworkCompleted bool                   `json:"homework_completed"`
	NextLessonTopic   string                 `json:"next_lesson_topic,omitempty"`
	NextLessonDate    *time.Time             `json:"next_lesson_date,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type LessonListResponse struct {
	Lessons []LessonResponse `json:"lessons"`
	Total   int64            `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// Assignment DTOs
type CreateAssignmentRequest struct {
	StudentID   string    `json:"student_id" validate:"required"`
	LessonID    string    `json:"lesson_id,omitempty"`
	Title       string    `json:"title" validate:"required"`
	Description string    `json:"description,omitempty"`
	Subject     string    `json:"subject" validate:"required"`
	Status      string    `json:"status" validate:"required,oneof=pending submitted graded overdue"`
	DueDate     time.Time `json:"due_date" validate:"required"`
	MaxScore    int       `json:"max_score" validate:"required"`
	Files       []string  `json:"files,omitempty"`
}

type UpdateAssignmentRequest struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status,omitempty"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
	Score       *int       `json:"score,omitempty"`
	Feedback    string     `json:"feedback,omitempty"`
}

type AssignmentResponse struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	StudentID   string                 `json:"student_id"`
	LessonID    string                 `json:"lesson_id,omitempty"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Subject     string                 `json:"subject"`
	Status      string                 `json:"status"`
	DueDate     time.Time              `json:"due_date"`
	SubmittedAt *time.Time             `json:"submitted_at,omitempty"`
	GradedAt    *time.Time             `json:"graded_at,omitempty"`
	Score       *int                   `json:"score,omitempty"`
	MaxScore    int                    `json:"max_score"`
	Feedback    string                 `json:"feedback,omitempty"`
	Files       []string               `json:"files,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type AssignmentListResponse struct {
	Assignments []AssignmentResponse `json:"assignments"`
	Total       int64                `json:"total"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}
