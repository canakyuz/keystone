package lesson

import (
	"time"

	"github.com/google/uuid"
)

// StudentStatus represents student status
type StudentStatus string

const (
	StudentStatusActive    StudentStatus = "active"
	StudentStatusInactive  StudentStatus = "inactive"
	StudentStatusSuspended StudentStatus = "suspended"
	StudentStatusGraduated StudentStatus = "graduated"
)

// StudentLevel represents student level
type StudentLevel string

const (
	LevelBeginner     StudentLevel = "beginner"
	LevelIntermediate StudentLevel = "intermediate"
	LevelAdvanced     StudentLevel = "advanced"
)

// Student represents a student entity
type Student struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"` // ⚠️ CRITICAL: Multi-tenant isolation
	FirstName   string        `json:"first_name"`
	LastName    string        `json:"last_name"`
	Email       string        `json:"email"`
	Phone       string        `json:"phone"`
	DateOfBirth *time.Time    `json:"date_of_birth,omitempty"`
	Status      StudentStatus `json:"status"`
	Level       StudentLevel  `json:"level"`
	Grade       string        `json:"grade,omitempty"` // School grade (e.g., "9th Grade")
	School      string        `json:"school,omitempty"`

	// Parent/Guardian info
	ParentName  string `json:"parent_name,omitempty"`
	ParentEmail string `json:"parent_email,omitempty"`
	ParentPhone string `json:"parent_phone,omitempty"`

	// Academic info
	CurrentGPA float64  `json:"current_gpa,omitempty"`
	TargetGPA  float64  `json:"target_gpa,omitempty"`
	Subjects   []string `json:"subjects,omitempty"` // ["Math", "Physics"]
	Goals      string   `json:"goals,omitempty"`
	Notes      string   `json:"notes,omitempty"`

	// Enrollment
	EnrollmentDate time.Time  `json:"enrollment_date"`
	LastLessonDate *time.Time `json:"last_lesson_date,omitempty"`

	// Statistics
	TotalLessons     int     `json:"total_lessons"`
	CompletedLessons int     `json:"completed_lessons"`
	CancelledLessons int     `json:"cancelled_lessons"`
	AttendanceRate   float64 `json:"attendance_rate"`
	AverageScore     float64 `json:"average_score"`

	// Metadata
	Avatar   string                 `json:"avatar,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Audit
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy string     `json:"created_by,omitempty"`
	UpdatedBy string     `json:"updated_by,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// NewStudent creates a new student
func NewStudent(tenantID, firstName, lastName, email, phone string, level StudentLevel) (*Student, error) {
	now := time.Now()
	student := &Student{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		FirstName:      firstName,
		LastName:       lastName,
		Email:          email,
		Phone:          phone,
		Status:         StudentStatusActive,
		Level:          level,
		EnrollmentDate: now,
		Metadata:       make(map[string]interface{}),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := student.Validate(); err != nil {
		return nil, err
	}

	return student, nil
}

// Validate validates student data
func (s *Student) Validate() error {
	if s.ID == "" {
		return ErrInvalidStudentID
	}
	if s.TenantID == "" {
		return ErrTenantIDRequired
	}
	if s.FirstName == "" {
		return ErrFirstNameRequired
	}
	if s.LastName == "" {
		return ErrLastNameRequired
	}
	if s.Email == "" {
		return ErrEmailRequired
	}
	if !s.Status.IsValid() {
		return ErrInvalidStatus
	}
	if !s.Level.IsValid() {
		return ErrInvalidLevel
	}
	return nil
}

// FullName returns student's full name
func (s *Student) FullName() string {
	return s.FirstName + " " + s.LastName
}

// Activate activates the student
func (s *Student) Activate() error {
	if s.Status == StudentStatusActive {
		return ErrStudentAlreadyActive
	}
	s.Status = StudentStatusActive
	s.UpdatedAt = time.Now()
	return nil
}

// Suspend suspends the student
func (s *Student) Suspend(reason string) error {
	if s.Status == StudentStatusSuspended {
		return ErrStudentAlreadySuspended
	}
	s.Status = StudentStatusSuspended
	s.UpdatedAt = time.Now()

	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata["suspension_reason"] = reason
	s.Metadata["suspended_at"] = time.Now()

	return nil
}

// Graduate marks student as graduated
func (s *Student) Graduate() error {
	if s.Status == StudentStatusGraduated {
		return ErrStudentAlreadyGraduated
	}
	s.Status = StudentStatusGraduated
	s.UpdatedAt = time.Now()

	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata["graduated_at"] = time.Now()

	return nil
}

// UpdateLevel updates student level
func (s *Student) UpdateLevel(newLevel StudentLevel) error {
	if !newLevel.IsValid() {
		return ErrInvalidLevel
	}
	s.Level = newLevel
	s.UpdatedAt = time.Now()
	return nil
}

// RecordLesson records a lesson completion
func (s *Student) RecordLesson(completed bool) {
	s.TotalLessons++
	if completed {
		s.CompletedLessons++
	} else {
		s.CancelledLessons++
	}

	// Calculate attendance rate
	if s.TotalLessons > 0 {
		s.AttendanceRate = float64(s.CompletedLessons) / float64(s.TotalLessons) * 100
	}

	now := time.Now()
	s.LastLessonDate = &now
	s.UpdatedAt = now
}

// UpdateAverageScore updates average score
func (s *Student) UpdateAverageScore(score float64) {
	s.AverageScore = score
	s.UpdatedAt = time.Now()
}

// IsActive checks if student is active
func (s *Student) IsActive() bool {
	return s.Status == StudentStatusActive
}

// IsSuspended checks if student is suspended
func (s *Student) IsSuspended() bool {
	return s.Status == StudentStatusSuspended
}

// IsGraduated checks if student is graduated
func (s *Student) IsGraduated() bool {
	return s.Status == StudentStatusGraduated
}

// IsValid checks if status is valid
func (s StudentStatus) IsValid() bool {
	switch s {
	case StudentStatusActive, StudentStatusInactive, StudentStatusSuspended, StudentStatusGraduated:
		return true
	default:
		return false
	}
}

// IsValid checks if level is valid
func (l StudentLevel) IsValid() bool {
	switch l {
	case LevelBeginner, LevelIntermediate, LevelAdvanced:
		return true
	default:
		return false
	}
}

// String returns string representation
func (s StudentStatus) String() string {
	return string(s)
}

// String returns string representation
func (l StudentLevel) String() string {
	return string(l)
}
