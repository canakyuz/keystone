package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"nexspaces-api/internal/domain/booking"
	"nexspaces-api/pkg/logger"
)

type AppointmentService struct {
	repo   booking.AppointmentRepository
	logger logger.Logger
}

func NewAppointmentService(repo booking.AppointmentRepository, logger logger.Logger) *AppointmentService {
	return &AppointmentService{
		repo:   repo,
		logger: logger,
	}
}

func (s *AppointmentService) Create(ctx context.Context, tenantID string, req *CreateAppointmentRequest) (*AppointmentResponse, error) {
	appointment := &booking.Appointment{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		UserID:      req.UserID,
		ClientName:  req.ClientName,
		ClientEmail: req.ClientEmail,
		ClientPhone: req.ClientPhone,
		Title:       req.Title,
		Description: req.Description,
		Type:        booking.AppointmentType(req.Type),
		Status:      booking.AppointmentStatusPending,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Duration:    req.Duration,
		Location:    req.Location,
		MeetingURL:  req.MeetingURL,
		Notes:       req.Notes,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, appointment); err != nil {
		s.logger.WithFields(logger.Fields{
			"tenant_id":    tenantID,
			"client_email": req.ClientEmail,
			"error":        err.Error(),
		}).Error("Failed to create appointment")
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"tenant_id":      tenantID,
		"appointment_id": appointment.ID,
		"client_email":   req.ClientEmail,
	}).Info("Appointment created successfully")

	return s.toAppointmentResponse(appointment), nil
}

func (s *AppointmentService) GetByID(ctx context.Context, id string) (*AppointmentResponse, error) {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toAppointmentResponse(appointment), nil
}

func (s *AppointmentService) List(ctx context.Context, filters booking.AppointmentListFilters) (*AppointmentListResponse, error) {
	appointments, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]AppointmentResponse, len(appointments))
	for i, a := range appointments {
		responses[i] = *s.toAppointmentResponse(a)
	}

	return &AppointmentListResponse{
		Appointments: responses,
		Total:        total,
		Limit:        filters.Limit,
		Offset:       filters.Offset,
	}, nil
}

func (s *AppointmentService) GetByUser(ctx context.Context, userID string, limit, offset int) (*AppointmentListResponse, error) {
	appointments, total, err := s.repo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]AppointmentResponse, len(appointments))
	for i, a := range appointments {
		responses[i] = *s.toAppointmentResponse(a)
	}

	return &AppointmentListResponse{
		Appointments: responses,
		Total:        total,
		Limit:        limit,
		Offset:       offset,
	}, nil
}

func (s *AppointmentService) GetByClient(ctx context.Context, clientEmail string, limit, offset int) (*AppointmentListResponse, error) {
	appointments, total, err := s.repo.GetByClient(ctx, clientEmail, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]AppointmentResponse, len(appointments))
	for i, a := range appointments {
		responses[i] = *s.toAppointmentResponse(a)
	}

	return &AppointmentListResponse{
		Appointments: responses,
		Total:        total,
		Limit:        limit,
		Offset:       offset,
	}, nil
}

func (s *AppointmentService) GetUpcoming(ctx context.Context, limit int) ([]AppointmentResponse, error) {
	appointments, err := s.repo.GetUpcoming(ctx, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]AppointmentResponse, len(appointments))
	for i, a := range appointments {
		responses[i] = *s.toAppointmentResponse(a)
	}

	return responses, nil
}

func (s *AppointmentService) GetByDateRange(ctx context.Context, startDate, endDate string) ([]AppointmentResponse, error) {
	appointments, err := s.repo.GetByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	responses := make([]AppointmentResponse, len(appointments))
	for i, a := range appointments {
		responses[i] = *s.toAppointmentResponse(a)
	}

	return responses, nil
}

func (s *AppointmentService) Update(ctx context.Context, id string, req *UpdateAppointmentRequest) (*AppointmentResponse, error) {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		appointment.Title = req.Title
	}
	if req.Description != "" {
		appointment.Description = req.Description
	}
	if req.Status != "" {
		appointment.Status = booking.AppointmentStatus(req.Status)
	}
	if req.Notes != "" {
		appointment.Notes = req.Notes
	}
	if req.CancellationReason != "" {
		appointment.CancellationReason = req.CancellationReason
	}

	if err := s.repo.Update(ctx, appointment); err != nil {
		s.logger.WithFields(logger.Fields{
			"appointment_id": id,
			"error":          err.Error(),
		}).Error("Failed to update appointment")
		return nil, fmt.Errorf("failed to update appointment: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"appointment_id": id,
	}).Info("Appointment updated successfully")

	return s.toAppointmentResponse(appointment), nil
}

func (s *AppointmentService) Confirm(ctx context.Context, id string) (*AppointmentResponse, error) {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := appointment.Confirm(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, appointment); err != nil {
		return nil, err
	}

	s.logger.WithFields(logger.Fields{
		"appointment_id": id,
	}).Info("Appointment confirmed")

	return s.toAppointmentResponse(appointment), nil
}

func (s *AppointmentService) Cancel(ctx context.Context, id string, reason string) (*AppointmentResponse, error) {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := appointment.Cancel(reason); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, appointment); err != nil {
		return nil, err
	}

	s.logger.WithFields(logger.Fields{
		"appointment_id": id,
		"reason":         reason,
	}).Info("Appointment cancelled")

	return s.toAppointmentResponse(appointment), nil
}

func (s *AppointmentService) Complete(ctx context.Context, id string) (*AppointmentResponse, error) {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := appointment.Complete(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, appointment); err != nil {
		return nil, err
	}

	s.logger.WithFields(logger.Fields{
		"appointment_id": id,
	}).Info("Appointment completed")

	return s.toAppointmentResponse(appointment), nil
}

func (s *AppointmentService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{
			"appointment_id": id,
			"error":          err.Error(),
		}).Error("Failed to delete appointment")
		return fmt.Errorf("failed to delete appointment: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"appointment_id": id,
	}).Info("Appointment deleted successfully")

	return nil
}

func (s *AppointmentService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetStats(ctx)
}

func (s *AppointmentService) toAppointmentResponse(a *booking.Appointment) *AppointmentResponse {
	return &AppointmentResponse{
		ID:                 a.ID,
		TenantID:           a.TenantID,
		UserID:             a.UserID,
		ClientName:         a.ClientName,
		ClientEmail:        a.ClientEmail,
		ClientPhone:        a.ClientPhone,
		Title:              a.Title,
		Description:        a.Description,
		Type:               string(a.Type),
		Status:             string(a.Status),
		StartTime:          a.StartTime,
		EndTime:            a.EndTime,
		Duration:           a.Duration,
		Location:           a.Location,
		MeetingURL:         a.MeetingURL,
		Notes:              a.Notes,
		CancellationReason: a.CancellationReason,
		CancelledAt:        a.CancelledAt,
		ConfirmedAt:        a.ConfirmedAt,
		CompletedAt:        a.CompletedAt,
		ReminderSent:       a.ReminderSent,
		ReminderSentAt:     a.ReminderSentAt,
		Metadata:           a.Metadata,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
	}
}
