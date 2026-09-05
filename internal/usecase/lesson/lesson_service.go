package lesson

import (
	"context"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/internal/domain/lesson"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/google/uuid"
)

type LessonService struct {
	repo   lesson.LessonRepository
	logger logger.Logger
}

func NewLessonService(repo lesson.LessonRepository, logger logger.Logger) *LessonService {
	return &LessonService{
		repo:   repo,
		logger: logger,
	}
}

func (s *LessonService) Create(ctx context.Context, tenantID string, req *CreateLessonRequest) (*LessonResponse, error) {
	lessonEntity := &lesson.Lesson{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		StudentID:   req.StudentID,
		Title:       req.Title,
		Description: req.Description,
		Subject:     req.Subject,
		Topic:       req.Topic,
		Status:      lesson.LessonStatus(req.Status),
		Type:        lesson.LessonType(req.Type),
		ScheduledAt: req.ScheduledAt,
		Duration:    req.Duration,
		Location:    req.Location,
		MeetingURL:  req.MeetingURL,
		Materials:   req.Materials,
		Homework:    req.Homework,
		Notes:       req.Notes,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   tenantID,
		UpdatedBy:   tenantID,
	}

	if err := s.repo.Create(ctx, lessonEntity); err != nil {
		s.logger.WithFields(logger.Fields{
			"tenant_id":  tenantID,
			"student_id": req.StudentID,
			"error":      err.Error(),
		}).Error("Failed to create lesson")
		return nil, fmt.Errorf("failed to create lesson: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id":  tenantID,
		"lesson_id":  lessonEntity.ID,
		"student_id": req.StudentID,
	}).Info("Lesson created successfully")

	return s.toLessonResponse(lessonEntity), nil
}

func (s *LessonService) GetByID(ctx context.Context, id string) (*LessonResponse, error) {
	lessonEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toLessonResponse(lessonEntity), nil
}

func (s *LessonService) List(ctx context.Context, filters lesson.LessonListFilters) (*LessonListResponse, error) {
	lessons, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]LessonResponse, len(lessons))
	for i, l := range lessons {
		responses[i] = *s.toLessonResponse(l)
	}

	return &LessonListResponse{
		Lessons: responses,
		Total:   total,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
	}, nil
}

func (s *LessonService) GetByStudent(ctx context.Context, studentID string, limit, offset int) (*LessonListResponse, error) {
	lessons, total, err := s.repo.GetByStudent(ctx, studentID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]LessonResponse, len(lessons))
	for i, l := range lessons {
		responses[i] = *s.toLessonResponse(l)
	}

	return &LessonListResponse{
		Lessons: responses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *LessonService) GetUpcoming(ctx context.Context, limit int) ([]LessonResponse, error) {
	lessons, err := s.repo.GetUpcoming(ctx, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]LessonResponse, len(lessons))
	for i, l := range lessons {
		responses[i] = *s.toLessonResponse(l)
	}

	return responses, nil
}

func (s *LessonService) Update(ctx context.Context, id string, req *UpdateLessonRequest) (*LessonResponse, error) {
	lessonEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Title != "" {
		lessonEntity.Title = req.Title
	}
	if req.Description != "" {
		lessonEntity.Description = req.Description
	}
	if req.Subject != "" {
		lessonEntity.Subject = req.Subject
	}
	if req.Topic != "" {
		lessonEntity.Topic = req.Topic
	}
	if req.Status != "" {
		lessonEntity.Status = lesson.LessonStatus(req.Status)
	}
	if req.StartedAt != nil {
		lessonEntity.StartedAt = req.StartedAt
	}
	if req.CompletedAt != nil {
		lessonEntity.CompletedAt = req.CompletedAt
	}
	if req.PerformanceScore > 0 {
		lessonEntity.PerformanceScore = req.PerformanceScore
	}
	if req.AttendanceStatus != "" {
		lessonEntity.AttendanceStatus = req.AttendanceStatus
	}
	lessonEntity.HomeworkCompleted = req.HomeworkCompleted
	if req.Notes != "" {
		lessonEntity.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, lessonEntity); err != nil {
		s.logger.WithFields(logger.Fields{
			"lesson_id": id,
			"error":     err.Error(),
		}).Error("Failed to update lesson")
		return nil, fmt.Errorf("failed to update lesson: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"lesson_id": id,
	}).Info("Lesson updated successfully")

	return s.toLessonResponse(lessonEntity), nil
}

func (s *LessonService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{
			"lesson_id": id,
			"error":     err.Error(),
		}).Error("Failed to delete lesson")
		return fmt.Errorf("failed to delete lesson: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"lesson_id": id,
	}).Info("Lesson deleted successfully")

	return nil
}

func (s *LessonService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetStats(ctx)
}

func (s *LessonService) toLessonResponse(l *lesson.Lesson) *LessonResponse {
	return &LessonResponse{
		ID:                l.ID,
		TenantID:          l.TenantID,
		StudentID:         l.StudentID,
		Title:             l.Title,
		Description:       l.Description,
		Subject:           l.Subject,
		Topic:             l.Topic,
		Status:            string(l.Status),
		Type:              string(l.Type),
		ScheduledAt:       l.ScheduledAt,
		StartedAt:         l.StartedAt,
		CompletedAt:       l.CompletedAt,
		Duration:          l.Duration,
		Location:          l.Location,
		MeetingURL:        l.MeetingURL,
		Materials:         l.Materials,
		Homework:          l.Homework,
		Notes:             l.Notes,
		StudentNotes:      l.StudentNotes,
		PerformanceScore:  l.PerformanceScore,
		AttendanceStatus:  l.AttendanceStatus,
		HomeworkCompleted: l.HomeworkCompleted,
		NextLessonTopic:   l.NextLessonTopic,
		NextLessonDate:    l.NextLessonDate,
		Metadata:          l.Metadata,
		CreatedAt:         l.CreatedAt,
		UpdatedAt:         l.UpdatedAt,
	}
}
