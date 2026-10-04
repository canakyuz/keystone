package lesson

import "github.com/canakyuz/keystone/pkg/clienterr"

// Student errors
var (
	// Validation errors
	ErrInvalidStudentID  = clienterr.New("invalid student ID")
	ErrTenantIDRequired  = clienterr.New("tenant ID is required")
	ErrFirstNameRequired = clienterr.New("first name is required")
	ErrLastNameRequired  = clienterr.New("last name is required")
	ErrEmailRequired     = clienterr.New("email is required")
	ErrInvalidLevel      = clienterr.New("invalid student level")
	ErrInvalidStatus     = clienterr.New("invalid status")

	// Business logic errors
	ErrStudentNotFound         = clienterr.New("student not found")
	ErrStudentAlreadyExists    = clienterr.New("student already exists")
	ErrStudentAlreadyActive    = clienterr.New("student is already active")
	ErrStudentAlreadySuspended = clienterr.New("student is already suspended")
	ErrStudentAlreadyGraduated = clienterr.New("student is already graduated")
)

// Lesson errors
var (
	// Validation errors
	ErrInvalidLessonID   = clienterr.New("invalid lesson ID")
	ErrStudentIDRequired = clienterr.New("student ID is required")
	ErrTitleRequired     = clienterr.New("title is required")
	ErrSubjectRequired   = clienterr.New("subject is required")
	ErrInvalidDuration   = clienterr.New("invalid duration")
	ErrInvalidLessonType = clienterr.New("invalid lesson type")

	// Business logic errors
	ErrLessonNotFound                  = clienterr.New("lesson not found")
	ErrLessonNotScheduled              = clienterr.New("lesson is not scheduled")
	ErrLessonAlreadyCompleted          = clienterr.New("lesson is already completed")
	ErrLessonAlreadyCancelled          = clienterr.New("lesson is already cancelled")
	ErrCannotCancelCompletedLesson     = clienterr.New("cannot cancel completed lesson")
	ErrCannotMarkCompletedLessonNoShow = clienterr.New("cannot mark completed lesson as no-show")
)

// Assignment errors
var (
	// Validation errors
	ErrInvalidAssignmentID = clienterr.New("invalid assignment ID")
	ErrDescriptionRequired = clienterr.New("description is required")
	ErrInvalidDueDate      = clienterr.New("invalid due date")

	// Business logic errors
	ErrAssignmentNotFound         = clienterr.New("assignment not found")
	ErrAssignmentAlreadySubmitted = clienterr.New("assignment is already submitted")
	ErrAssignmentAlreadyGraded    = clienterr.New("assignment is already graded")
	ErrAssignmentNotSubmitted     = clienterr.New("assignment not submitted yet")
)

// Multi-tenant errors
var (
	ErrCrossTenantAccess = clienterr.New("cross-tenant access is not allowed")
	ErrTenantMismatch    = clienterr.New("tenant ID mismatch")
)

// Permission errors
var (
	ErrUnauthorized            = clienterr.New("unauthorized access")
	ErrInsufficientPermissions = clienterr.New("insufficient permissions")
)
