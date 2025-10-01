package lesson

import "context"

// StudentRepository defines student data access interface
type StudentRepository interface {
	Create(ctx context.Context, student *Student) error
	GetByID(ctx context.Context, id string) (*Student, error)
	GetByEmail(ctx context.Context, email string) (*Student, error)
	List(ctx context.Context, filters StudentListFilters) ([]*Student, int64, error)
	Update(ctx context.Context, student *Student) error
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// LessonRepository defines lesson data access interface
type LessonRepository interface {
	Create(ctx context.Context, lesson *Lesson) error
	GetByID(ctx context.Context, id string) (*Lesson, error)
	List(ctx context.Context, filters LessonListFilters) ([]*Lesson, int64, error)
	GetByStudent(ctx context.Context, studentID string, limit, offset int) ([]*Lesson, int64, error)
	Update(ctx context.Context, lesson *Lesson) error
	Delete(ctx context.Context, id string) error
	GetUpcoming(ctx context.Context, limit int) ([]*Lesson, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// AssignmentRepository defines assignment data access interface
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *Assignment) error
	GetByID(ctx context.Context, id string) (*Assignment, error)
	List(ctx context.Context, filters AssignmentListFilters) ([]*Assignment, int64, error)
	GetByStudent(ctx context.Context, studentID string, limit, offset int) ([]*Assignment, int64, error)
	GetOverdue(ctx context.Context) ([]*Assignment, error)
	Update(ctx context.Context, assignment *Assignment) error
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// StudentListFilters defines filters for listing students
type StudentListFilters struct {
	Status    *StudentStatus
	Level     *StudentLevel
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

// LessonListFilters defines filters for listing lessons
type LessonListFilters struct {
	StudentID *string
	Status    *LessonStatus
	Type      *LessonType
	Subject   *string
	FromDate  *string
	ToDate    *string
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

// AssignmentListFilters defines filters for listing assignments
type AssignmentListFilters struct {
	StudentID *string
	Status    *AssignmentStatus
	Subject   *string
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}
