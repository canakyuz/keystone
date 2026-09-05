package lesson

import (
	"context"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/internal/domain/lesson"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/google/uuid"
)

type StudentService struct {
	repo   lesson.StudentRepository
	logger logger.Logger
}

func NewStudentService(repo lesson.StudentRepository, logger logger.Logger) *StudentService {
	return &StudentService{
		repo:   repo,
		logger: logger,
	}
}

func (s *StudentService) Create(ctx context.Context, tenantID string, req *CreateStudentRequest) (*StudentResponse, error) {
	// Check if email already exists
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, lesson.ErrStudentAlreadyExists
	}

	student := &lesson.Student{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		Phone:          req.Phone,
		DateOfBirth:    req.DateOfBirth,
		Status:         lesson.StudentStatus(req.Status),
		Level:          lesson.StudentLevel(req.Level),
		Grade:          req.Grade,
		School:         req.School,
		ParentName:     req.ParentName,
		ParentEmail:    req.ParentEmail,
		ParentPhone:    req.ParentPhone,
		CurrentGPA:     req.CurrentGPA,
		TargetGPA:      req.TargetGPA,
		Subjects:       req.Subjects,
		Goals:          req.Goals,
		Notes:          req.Notes,
		EnrollmentDate: time.Now(),
		Avatar:         req.Avatar,
		Metadata:       make(map[string]interface{}),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      tenantID,
		UpdatedBy:      tenantID,
	}

	if err := s.repo.Create(ctx, student); err != nil {
		s.logger.WithFields(logger.Fields{
			"tenant_id": tenantID,
			"email":     req.Email,
			"error":     err.Error(),
		}).Error("Failed to create student")
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id":  tenantID,
		"student_id": student.ID,
		"email":      student.Email,
	}).Info("Student created successfully")

	return s.toStudentResponse(student), nil
}

func (s *StudentService) GetByID(ctx context.Context, id string) (*StudentResponse, error) {
	student, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toStudentResponse(student), nil
}

func (s *StudentService) GetByEmail(ctx context.Context, email string) (*StudentResponse, error) {
	student, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return s.toStudentResponse(student), nil
}

func (s *StudentService) List(ctx context.Context, filters lesson.StudentListFilters) (*StudentListResponse, error) {
	students, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]StudentResponse, len(students))
	for i, student := range students {
		responses[i] = *s.toStudentResponse(student)
	}

	return &StudentListResponse{
		Students: responses,
		Total:    total,
		Limit:    filters.Limit,
		Offset:   filters.Offset,
	}, nil
}

func (s *StudentService) Update(ctx context.Context, id string, req *UpdateStudentRequest) (*StudentResponse, error) {
	student, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.FirstName != "" {
		student.FirstName = req.FirstName
	}
	if req.LastName != "" {
		student.LastName = req.LastName
	}
	if req.Email != "" {
		student.Email = req.Email
	}
	if req.Phone != "" {
		student.Phone = req.Phone
	}
	if req.DateOfBirth != nil {
		student.DateOfBirth = req.DateOfBirth
	}
	if req.Status != "" {
		student.Status = lesson.StudentStatus(req.Status)
	}
	if req.Level != "" {
		student.Level = lesson.StudentLevel(req.Level)
	}
	if req.Grade != "" {
		student.Grade = req.Grade
	}
	if req.School != "" {
		student.School = req.School
	}
	if req.ParentName != "" {
		student.ParentName = req.ParentName
	}
	if req.ParentEmail != "" {
		student.ParentEmail = req.ParentEmail
	}
	if req.ParentPhone != "" {
		student.ParentPhone = req.ParentPhone
	}
	if req.CurrentGPA > 0 {
		student.CurrentGPA = req.CurrentGPA
	}
	if req.TargetGPA > 0 {
		student.TargetGPA = req.TargetGPA
	}
	if len(req.Subjects) > 0 {
		student.Subjects = req.Subjects
	}
	if req.Goals != "" {
		student.Goals = req.Goals
	}
	if req.Notes != "" {
		student.Notes = req.Notes
	}
	if req.Avatar != "" {
		student.Avatar = req.Avatar
	}

	if err := s.repo.Update(ctx, student); err != nil {
		s.logger.WithFields(logger.Fields{
			"student_id": id,
			"error":      err.Error(),
		}).Error("Failed to update student")
		return nil, fmt.Errorf("failed to update student: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"student_id": id,
	}).Info("Student updated successfully")

	return s.toStudentResponse(student), nil
}

func (s *StudentService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{
			"student_id": id,
			"error":      err.Error(),
		}).Error("Failed to delete student")
		return fmt.Errorf("failed to delete student: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"student_id": id,
	}).Info("Student deleted successfully")

	return nil
}

func (s *StudentService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetStats(ctx)
}

func (s *StudentService) toStudentResponse(student *lesson.Student) *StudentResponse {
	return &StudentResponse{
		ID:               student.ID,
		TenantID:         student.TenantID,
		FirstName:        student.FirstName,
		LastName:         student.LastName,
		Email:            student.Email,
		Phone:            student.Phone,
		DateOfBirth:      student.DateOfBirth,
		Status:           string(student.Status),
		Level:            string(student.Level),
		Grade:            student.Grade,
		School:           student.School,
		ParentName:       student.ParentName,
		ParentEmail:      student.ParentEmail,
		ParentPhone:      student.ParentPhone,
		CurrentGPA:       student.CurrentGPA,
		TargetGPA:        student.TargetGPA,
		Subjects:         student.Subjects,
		Goals:            student.Goals,
		Notes:            student.Notes,
		EnrollmentDate:   student.EnrollmentDate,
		LastLessonDate:   student.LastLessonDate,
		TotalLessons:     student.TotalLessons,
		CompletedLessons: student.CompletedLessons,
		CancelledLessons: student.CancelledLessons,
		AttendanceRate:   student.AttendanceRate,
		AverageScore:     student.AverageScore,
		Avatar:           student.Avatar,
		Metadata:         student.Metadata,
		CreatedAt:        student.CreatedAt,
		UpdatedAt:        student.UpdatedAt,
	}
}
