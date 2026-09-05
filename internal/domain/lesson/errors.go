package lesson

import "errors"

// Student errors
var (
	// Validation errors
	ErrInvalidStudentID  = errors.New("invalid student ID")
	ErrTenantIDRequired  = errors.New("tenant ID is required")
	ErrFirstNameRequired = errors.New("first name is required")
	ErrLastNameRequired  = errors.New("last name is required")
	ErrEmailRequired     = errors.New("email is required")
	ErrInvalidLevel      = errors.New("invalid student level")
	ErrInvalidStatus     = errors.New("invalid status")

	// Business logic errors
	ErrStudentNotFound         = errors.New("student not found")
	ErrStudentAlreadyExists    = errors.New("student already exists")
	ErrStudentAlreadyActive    = errors.New("student is already active")
	ErrStudentAlreadySuspended = errors.New("student is already suspended")
	ErrStudentAlreadyGraduated = errors.New("student is already graduated")
)

// Lesson errors
var (
	// Validation errors
	ErrInvalidLessonID   = errors.New("invalid lesson ID")
	ErrStudentIDRequired = errors.New("student ID is required")
	ErrTitleRequired     = errors.New("title is required")
	ErrSubjectRequired   = errors.New("subject is required")
	ErrInvalidDuration   = errors.New("invalid duration")
	ErrInvalidLessonType = errors.New("invalid lesson type")

	// Business logic errors
	ErrLessonNotFound                  = errors.New("lesson not found")
	ErrLessonNotScheduled              = errors.New("lesson is not scheduled")
	ErrLessonAlreadyCompleted          = errors.New("lesson is already completed")
	ErrLessonAlreadyCancelled          = errors.New("lesson is already cancelled")
	ErrCannotCancelCompletedLesson     = errors.New("cannot cancel completed lesson")
	ErrCannotMarkCompletedLessonNoShow = errors.New("cannot mark completed lesson as no-show")
)

// Assignment errors
var (
	// Validation errors
	ErrInvalidAssignmentID = errors.New("invalid assignment ID")
	ErrDescriptionRequired = errors.New("description is required")
	ErrInvalidDueDate      = errors.New("invalid due date")

	// Business logic errors
	ErrAssignmentNotFound         = errors.New("assignment not found")
	ErrAssignmentAlreadySubmitted = errors.New("assignment is already submitted")
	ErrAssignmentAlreadyGraded    = errors.New("assignment is already graded")
	ErrAssignmentNotSubmitted     = errors.New("assignment not submitted yet")
)

// Multi-tenant errors
var (
	ErrCrossTenantAccess = errors.New("cross-tenant access is not allowed")
	ErrTenantMismatch    = errors.New("tenant ID mismatch")
)

// Permission errors
var (
	ErrUnauthorized            = errors.New("unauthorized access")
	ErrInsufficientPermissions = errors.New("insufficient permissions")
)
