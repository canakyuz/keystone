package lesson

import (
	"context"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/lesson"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/google/uuid"
)

type AssignmentService struct {
	repo   lesson.AssignmentRepository
	logger logger.Logger
}

func NewAssignmentService(repo lesson.AssignmentRepository, logger logger.Logger) *AssignmentService {
	return &AssignmentService{
		repo:   repo,
		logger: logger,
	}
}

func (s *AssignmentService) Create(ctx context.Context, tenantID string, req *CreateAssignmentRequest) (*AssignmentResponse, error) {
	assignment := &lesson.Assignment{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		StudentID:   req.StudentID,
		LessonID:    req.LessonID,
		Title:       req.Title,
		Description: req.Description,
		Subject:     req.Subject,
		Status:      lesson.AssignmentStatus(req.Status),
		DueDate:     req.DueDate,
		MaxScore:    req.MaxScore,
		Files:       req.Files,
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, assignment); err != nil {
		s.logger.WithFields(logger.Fields{
			"tenant_id":  tenantID,
			"student_id": req.StudentID,
			"error":      err.Error(),
		}).Error("Failed to create assignment")
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id":     tenantID,
		"assignment_id": assignment.ID,
		"student_id":    req.StudentID,
	}).Info("Assignment created successfully")

	return s.toAssignmentResponse(assignment), nil
}

func (s *AssignmentService) GetByID(ctx context.Context, id string) (*AssignmentResponse, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toAssignmentResponse(assignment), nil
}

func (s *AssignmentService) List(ctx context.Context, filters lesson.AssignmentListFilters) (*AssignmentListResponse, error) {
	assignments, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		responses[i] = *s.toAssignmentResponse(a)
	}

	return &AssignmentListResponse{
		Assignments: responses,
		Total:       total,
		Limit:       filters.Limit,
		Offset:      filters.Offset,
	}, nil
}

func (s *AssignmentService) GetByStudent(ctx context.Context, studentID string, limit, offset int) (*AssignmentListResponse, error) {
	assignments, total, err := s.repo.GetByStudent(ctx, studentID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		responses[i] = *s.toAssignmentResponse(a)
	}

	return &AssignmentListResponse{
		Assignments: responses,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
	}, nil
}

func (s *AssignmentService) GetOverdue(ctx context.Context) ([]AssignmentResponse, error) {
	assignments, err := s.repo.GetOverdue(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		responses[i] = *s.toAssignmentResponse(a)
	}

	return responses, nil
}

func (s *AssignmentService) Update(ctx context.Context, id string, req *UpdateAssignmentRequest) (*AssignmentResponse, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Title != "" {
		assignment.Title = req.Title
	}
	if req.Description != "" {
		assignment.Description = req.Description
	}
	if req.Status != "" {
		assignment.Status = lesson.AssignmentStatus(req.Status)
	}
	if req.SubmittedAt != nil {
		assignment.SubmittedAt = req.SubmittedAt
	}
	if req.GradedAt != nil {
		assignment.GradedAt = req.GradedAt
	}
	if req.Score != nil {
		assignment.Score = req.Score
	}
	if req.Feedback != "" {
		assignment.Feedback = req.Feedback
	}

	if err := s.repo.Update(ctx, assignment); err != nil {
		s.logger.WithFields(logger.Fields{
			"assignment_id": id,
			"error":         err.Error(),
		}).Error("Failed to update assignment")
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"assignment_id": id,
	}).Info("Assignment updated successfully")

	return s.toAssignmentResponse(assignment), nil
}

func (s *AssignmentService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{
			"assignment_id": id,
			"error":         err.Error(),
		}).Error("Failed to delete assignment")
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"assignment_id": id,
	}).Info("Assignment deleted successfully")

	return nil
}

func (s *AssignmentService) GetStats(ctx context.Context) (map[string]any, error) {
	return s.repo.GetStats(ctx)
}

func (s *AssignmentService) toAssignmentResponse(a *lesson.Assignment) *AssignmentResponse {
	return &AssignmentResponse{
		ID:          a.ID,
		TenantID:    a.TenantID,
		StudentID:   a.StudentID,
		LessonID:    a.LessonID,
		Title:       a.Title,
		Description: a.Description,
		Subject:     a.Subject,
		Status:      string(a.Status),
		DueDate:     a.DueDate,
		SubmittedAt: a.SubmittedAt,
		GradedAt:    a.GradedAt,
		Score:       a.Score,
		MaxScore:    a.MaxScore,
		Feedback:    a.Feedback,
		Files:       a.Files,
		Metadata:    a.Metadata,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
