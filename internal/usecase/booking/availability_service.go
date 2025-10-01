package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"nexpaces-api/internal/domain/booking"
	"nexpaces-api/pkg/logger"
)

type AvailabilityService struct {
	repo   booking.AvailabilityRepository
	logger logger.Logger
}

func NewAvailabilityService(repo booking.AvailabilityRepository, logger logger.Logger) *AvailabilityService {
	return &AvailabilityService{
		repo:   repo,
		logger: logger,
	}
}

func (s *AvailabilityService) Create(ctx context.Context, tenantID string, req *CreateAvailabilityRequest) (*AvailabilityResponse, error) {
	availability := &booking.Availability{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		UserID:        req.UserID,
		Title:         req.Title,
		Description:   req.Description,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Status:        booking.AvailabilityStatusAvailable,
		Recurrence:    booking.RecurrenceType(req.Recurrence),
		RecurrenceEnd: req.RecurrenceEnd,
		DaysOfWeek:    req.DaysOfWeek,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, availability); err != nil {
		s.logger.WithFields(logger.Fields{
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to create availability")
		return nil, fmt.Errorf("failed to create availability: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id":       tenantID,
		"availability_id": availability.ID,
	}).Info("Availability created successfully")

	return s.toAvailabilityResponse(availability), nil
}

func (s *AvailabilityService) GetByID(ctx context.Context, id string) (*AvailabilityResponse, error) {
	availability, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toAvailabilityResponse(availability), nil
}

func (s *AvailabilityService) List(ctx context.Context, filters booking.AvailabilityListFilters) (*AvailabilityListResponse, error) {
	availabilities, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]AvailabilityResponse, len(availabilities))
	for i, a := range availabilities {
		responses[i] = *s.toAvailabilityResponse(a)
	}

	return &AvailabilityListResponse{
		Availabilities: responses,
		Total:          total,
		Limit:          filters.Limit,
		Offset:         filters.Offset,
	}, nil
}

func (s *AvailabilityService) GetByUser(ctx context.Context, userID string, limit, offset int) (*AvailabilityListResponse, error) {
	availabilities, total, err := s.repo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]AvailabilityResponse, len(availabilities))
	for i, a := range availabilities {
		responses[i] = *s.toAvailabilityResponse(a)
	}

	return &AvailabilityListResponse{
		Availabilities: responses,
		Total:          total,
		Limit:          limit,
		Offset:         offset,
	}, nil
}

func (s *AvailabilityService) GetByDateRange(ctx context.Context, startDate, endDate string) ([]AvailabilityResponse, error) {
	availabilities, err := s.repo.GetByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	responses := make([]AvailabilityResponse, len(availabilities))
	for i, a := range availabilities {
		responses[i] = *s.toAvailabilityResponse(a)
	}

	return responses, nil
}

func (s *AvailabilityService) Update(ctx context.Context, id string, req *UpdateAvailabilityRequest) (*AvailabilityResponse, error) {
	availability, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		availability.Title = req.Title
	}
	if req.Description != "" {
		availability.Description = req.Description
	}
	if req.StartTime != nil {
		availability.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		availability.EndTime = *req.EndTime
	}
	if req.Status != "" {
		availability.Status = booking.AvailabilityStatus(req.Status)
	}

	if err := s.repo.Update(ctx, availability); err != nil {
		s.logger.WithFields(logger.Fields{
			"availability_id": id,
			"error":           err.Error(),
		}).Error("Failed to update availability")
		return nil, fmt.Errorf("failed to update availability: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"availability_id": id,
	}).Info("Availability updated successfully")

	return s.toAvailabilityResponse(availability), nil
}

func (s *AvailabilityService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{
			"availability_id": id,
			"error":           err.Error(),
		}).Error("Failed to delete availability")
		return fmt.Errorf("failed to delete availability: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"availability_id": id,
	}).Info("Availability deleted successfully")

	return nil
}

func (s *AvailabilityService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetStats(ctx)
}

func (s *AvailabilityService) toAvailabilityResponse(a *booking.Availability) *AvailabilityResponse {
	return &AvailabilityResponse{
		ID:            a.ID,
		TenantID:      a.TenantID,
		UserID:        a.UserID,
		Title:         a.Title,
		Description:   a.Description,
		StartTime:     a.StartTime,
		EndTime:       a.EndTime,
		Status:        string(a.Status),
		Recurrence:    string(a.Recurrence),
		RecurrenceEnd: a.RecurrenceEnd,
		DaysOfWeek:    a.DaysOfWeek,
		Metadata:      a.Metadata,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}
