package lesson

import (
	"time"

	"github.com/google/uuid"
)

// LessonStatus represents lesson status
type LessonStatus string

const (
	LessonStatusScheduled LessonStatus = "scheduled"
	LessonStatusCompleted LessonStatus = "completed"
	LessonStatusCancelled LessonStatus = "cancelled"
	LessonStatusNoShow    LessonStatus = "no-show"
)

// LessonType represents lesson type
type LessonType string

const (
	LessonTypeOneOnOne LessonType = "one-on-one"
	LessonTypeGroup    LessonType = "group"
	LessonTypeOnline   LessonType = "online"
	LessonTypeInPerson LessonType = "in-person"
)

// Lesson represents a tutoring lesson/session
type Lesson struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"` // ⚠️ CRITICAL: Multi-tenant isolation
	StudentID   string       `json:"student_id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Subject     string       `json:"subject"`         // "Math", "Physics", etc.
	Topic       string       `json:"topic,omitempty"` // "Quadratic Equations"
	Status      LessonStatus `json:"status"`
	Type        LessonType   `json:"type"`

	// Schedule
	ScheduledAt time.Time  `json:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int        `json:"duration"` // Duration in minutes
	Location    string     `json:"location,omitempty"`
	MeetingURL  string     `json:"meeting_url,omitempty"` // Zoom/Meet link

	// Content
	Materials    []string `json:"materials,omitempty"` // URLs to materials
	Homework     string   `json:"homework,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	StudentNotes string   `json:"student_notes,omitempty"`

	// Assessment
	PerformanceScore  int    `json:"performance_score,omitempty"` // 1-10
	AttendanceStatus  string `json:"attendance_status,omitempty"` // "present", "absent", "late"
	HomeworkCompleted bool   `json:"homework_completed"`

	// Next lesson planning
	NextLessonTopic string     `json:"next_lesson_topic,omitempty"`
	NextLessonDate  *time.Time `json:"next_lesson_date,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Audit
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy string     `json:"created_by,omitempty"`
	UpdatedBy string     `json:"updated_by,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// NewLesson creates a new lesson
func NewLesson(tenantID, studentID, title, subject string, scheduledAt time.Time, duration int, lessonType LessonType) (*Lesson, error) {
	now := time.Now()
	lesson := &Lesson{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		StudentID:   studentID,
		Title:       title,
		Subject:     subject,
		Status:      LessonStatusScheduled,
		Type:        lessonType,
		ScheduledAt: scheduledAt,
		Duration:    duration,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := lesson.Validate(); err != nil {
		return nil, err
	}

	return lesson, nil
}

// Validate validates lesson data
func (l *Lesson) Validate() error {
	if l.ID == "" {
		return ErrInvalidLessonID
	}
	if l.TenantID == "" {
		return ErrTenantIDRequired
	}
	if l.StudentID == "" {
		return ErrStudentIDRequired
	}
	if l.Title == "" {
		return ErrTitleRequired
	}
	if l.Subject == "" {
		return ErrSubjectRequired
	}
	if l.Duration <= 0 {
		return ErrInvalidDuration
	}
	if !l.Status.IsValid() {
		return ErrInvalidStatus
	}
	if !l.Type.IsValid() {
		return ErrInvalidLessonType
	}
	return nil
}

// Start marks lesson as started
func (l *Lesson) Start() error {
	if l.Status != LessonStatusScheduled {
		return ErrLessonNotScheduled
	}

	now := time.Now()
	l.StartedAt = &now
	l.UpdatedAt = now

	return nil
}

// Complete marks lesson as completed
func (l *Lesson) Complete(performanceScore int, notes string) error {
	if l.Status == LessonStatusCompleted {
		return ErrLessonAlreadyCompleted
	}

	now := time.Now()
	l.Status = LessonStatusCompleted
	l.CompletedAt = &now
	l.PerformanceScore = performanceScore
	l.Notes = notes
	l.AttendanceStatus = "present"
	l.UpdatedAt = now

	return nil
}

// Cancel cancels the lesson
func (l *Lesson) Cancel(reason string) error {
	if l.Status == LessonStatusCompleted {
		return ErrCannotCancelCompletedLesson
	}
	if l.Status == LessonStatusCancelled {
		return ErrLessonAlreadyCancelled
	}

	l.Status = LessonStatusCancelled
	l.UpdatedAt = time.Now()

	if l.Metadata == nil {
		l.Metadata = make(map[string]interface{})
	}
	l.Metadata["cancellation_reason"] = reason
	l.Metadata["cancelled_at"] = time.Now()

	return nil
}

// MarkNoShow marks student as no-show
func (l *Lesson) MarkNoShow() error {
	if l.Status == LessonStatusCompleted {
		return ErrCannotMarkCompletedLessonNoShow
	}

	l.Status = LessonStatusNoShow
	l.AttendanceStatus = "absent"
	l.UpdatedAt = time.Now()

	return nil
}

// MarkHomeworkCompleted marks homework as completed
func (l *Lesson) MarkHomeworkCompleted(completed bool) {
	l.HomeworkCompleted = completed
	l.UpdatedAt = time.Now()
}

// SetNextLesson sets next lesson planning
func (l *Lesson) SetNextLesson(topic string, date time.Time) {
	l.NextLessonTopic = topic
	l.NextLessonDate = &date
	l.UpdatedAt = time.Now()
}

// IsCompleted checks if lesson is completed
func (l *Lesson) IsCompleted() bool {
	return l.Status == LessonStatusCompleted
}

// IsCancelled checks if lesson is cancelled
func (l *Lesson) IsCancelled() bool {
	return l.Status == LessonStatusCancelled
}

// IsScheduled checks if lesson is scheduled
func (l *Lesson) IsScheduled() bool {
	return l.Status == LessonStatusScheduled
}

// IsNoShow checks if lesson is no-show
func (l *Lesson) IsNoShow() bool {
	return l.Status == LessonStatusNoShow
}

// IsValid checks if status is valid
func (s LessonStatus) IsValid() bool {
	switch s {
	case LessonStatusScheduled, LessonStatusCompleted, LessonStatusCancelled, LessonStatusNoShow:
		return true
	default:
		return false
	}
}

// IsValid checks if type is valid
func (t LessonType) IsValid() bool {
	switch t {
	case LessonTypeOneOnOne, LessonTypeGroup, LessonTypeOnline, LessonTypeInPerson:
		return true
	default:
		return false
	}
}

// String returns string representation
func (s LessonStatus) String() string {
	return string(s)
}

// String returns string representation
func (t LessonType) String() string {
	return string(t)
}
