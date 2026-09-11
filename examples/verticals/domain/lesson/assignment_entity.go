package lesson

import (
	"time"

	"github.com/google/uuid"
)

// AssignmentStatus represents assignment status
type AssignmentStatus string

const (
	AssignmentStatusPending   AssignmentStatus = "pending"
	AssignmentStatusSubmitted AssignmentStatus = "submitted"
	AssignmentStatusGraded    AssignmentStatus = "graded"
	AssignmentStatusOverdue   AssignmentStatus = "overdue"
)

// Assignment represents a homework/assignment
type Assignment struct {
	ID          string           `json:"id"`
	TenantID    string           `json:"tenant_id"` // CRITICAL: Multi-tenant isolation
	StudentID   string           `json:"student_id"`
	LessonID    string           `json:"lesson_id,omitempty"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Subject     string           `json:"subject"`
	Status      AssignmentStatus `json:"status"`
	DueDate     time.Time        `json:"due_date"`
	SubmittedAt *time.Time       `json:"submitted_at,omitempty"`
	GradedAt    *time.Time       `json:"graded_at,omitempty"`
	Score       *int             `json:"score,omitempty"` // 0-100
	MaxScore    int              `json:"max_score"`
	Feedback    string           `json:"feedback,omitempty"`
	Files       []string         `json:"files,omitempty"` // URLs to uploaded files
	Metadata    map[string]any   `json:"metadata,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	CreatedBy   string           `json:"created_by,omitempty"`
	UpdatedBy   string           `json:"updated_by,omitempty"`
	DeletedAt   *time.Time       `json:"deleted_at,omitempty"`
}

// NewAssignment creates a new assignment
func NewAssignment(tenantID, studentID, title, description, subject string, dueDate time.Time, maxScore int) (*Assignment, error) {
	now := time.Now()
	assignment := &Assignment{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		StudentID:   studentID,
		Title:       title,
		Description: description,
		Subject:     subject,
		Status:      AssignmentStatusPending,
		DueDate:     dueDate,
		MaxScore:    maxScore,
		Metadata:    make(map[string]any),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := assignment.Validate(); err != nil {
		return nil, err
	}

	return assignment, nil
}

// Validate validates assignment data
func (a *Assignment) Validate() error {
	if a.ID == "" {
		return ErrInvalidAssignmentID
	}
	if a.TenantID == "" {
		return ErrTenantIDRequired
	}
	if a.StudentID == "" {
		return ErrStudentIDRequired
	}
	if a.Title == "" {
		return ErrTitleRequired
	}
	if a.Description == "" {
		return ErrDescriptionRequired
	}
	if a.DueDate.IsZero() {
		return ErrInvalidDueDate
	}
	if !a.Status.IsValid() {
		return ErrInvalidStatus
	}
	return nil
}

// Submit marks assignment as submitted
func (a *Assignment) Submit() error {
	if a.Status == AssignmentStatusSubmitted {
		return ErrAssignmentAlreadySubmitted
	}

	now := time.Now()
	a.Status = AssignmentStatusSubmitted
	a.SubmittedAt = &now
	a.UpdatedAt = now

	return nil
}

// Grade grades the assignment
func (a *Assignment) Grade(score int, feedback string) error {
	if a.Status != AssignmentStatusSubmitted {
		return ErrAssignmentNotSubmitted
	}
	if a.Status == AssignmentStatusGraded {
		return ErrAssignmentAlreadyGraded
	}

	now := time.Now()
	a.Status = AssignmentStatusGraded
	a.Score = &score
	a.Feedback = feedback
	a.GradedAt = &now
	a.UpdatedAt = now

	return nil
}

// MarkOverdue marks assignment as overdue
func (a *Assignment) MarkOverdue() {
	if a.Status == AssignmentStatusPending && time.Now().After(a.DueDate) {
		a.Status = AssignmentStatusOverdue
		a.UpdatedAt = time.Now()
	}
}

// IsOverdue checks if assignment is overdue
func (a *Assignment) IsOverdue() bool {
	return time.Now().After(a.DueDate) && a.Status == AssignmentStatusPending
}

// IsPending checks if assignment is pending
func (a *Assignment) IsPending() bool {
	return a.Status == AssignmentStatusPending
}

// IsSubmitted checks if assignment is submitted
func (a *Assignment) IsSubmitted() bool {
	return a.Status == AssignmentStatusSubmitted
}

// IsGraded checks if assignment is graded
func (a *Assignment) IsGraded() bool {
	return a.Status == AssignmentStatusGraded
}

// GetScorePercentage returns score as percentage
func (a *Assignment) GetScorePercentage() float64 {
	if a.Score == nil || a.MaxScore == 0 {
		return 0
	}
	return float64(*a.Score) / float64(a.MaxScore) * 100
}

// IsValid checks if status is valid
func (s AssignmentStatus) IsValid() bool {
	switch s {
	case AssignmentStatusPending, AssignmentStatusSubmitted, AssignmentStatusGraded, AssignmentStatusOverdue:
		return true
	default:
		return false
	}
}

// String returns string representation
func (s AssignmentStatus) String() string {
	return string(s)
}
