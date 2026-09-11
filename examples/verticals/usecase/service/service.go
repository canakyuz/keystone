package service

import (
	"context"
	"fmt"
	"time"

	"github.com/canakyuz/keystone/examples/verticals/domain/service"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/google/uuid"
)

type ServiceService struct {
	repo   service.ServiceRepository
	logger logger.Logger
}

func NewServiceService(repo service.ServiceRepository, logger logger.Logger) *ServiceService {
	return &ServiceService{repo: repo, logger: logger}
}

func (s *ServiceService) Create(ctx context.Context, tenantID string, req *CreateServiceRequest) (*ServiceResponse, error) {
	svc := &service.Service{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		Category:     service.ServiceCategory(req.Category),
		Status:       service.ServiceStatusActive,
		Features:     req.Features,
		BasePrice:    req.BasePrice,
		Currency:     req.Currency,
		PricingModel: service.PricingModel(req.PricingModel),
		BillingCycle: req.BillingCycle,
		Duration:     req.Duration,
		MaxClients:   req.MaxClients,
		IsPublic:     req.IsPublic,
		Featured:     false,
		Image:        req.Image,
		Metadata:     make(map[string]any),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    tenantID,
		UpdatedBy:    tenantID,
	}

	if err := s.repo.Create(ctx, svc); err != nil {
		s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "error": err.Error()}).Error("Failed to create service")
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	s.logger.WithFields(logger.Fields{"tenant_id": tenantID, "service_id": svc.ID}).Info("Service created successfully")
	return s.toServiceResponse(svc), nil
}

func (s *ServiceService) GetByID(ctx context.Context, id string) (*ServiceResponse, error) {
	svc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toServiceResponse(svc), nil
}

func (s *ServiceService) GetBySlug(ctx context.Context, slug string) (*ServiceResponse, error) {
	svc, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.toServiceResponse(svc), nil
}

func (s *ServiceService) List(ctx context.Context, filters service.ServiceListFilters) (*ServiceListResponse, error) {
	services, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	responses := make([]ServiceResponse, len(services))
	for i, svc := range services {
		responses[i] = *s.toServiceResponse(svc)
	}

	return &ServiceListResponse{Services: responses, Total: total, Limit: filters.Limit, Offset: filters.Offset}, nil
}

func (s *ServiceService) GetFeatured(ctx context.Context, limit int) ([]ServiceResponse, error) {
	services, err := s.repo.GetFeatured(ctx, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]ServiceResponse, len(services))
	for i, svc := range services {
		responses[i] = *s.toServiceResponse(svc)
	}
	return responses, nil
}

func (s *ServiceService) Update(ctx context.Context, id string, req *UpdateServiceRequest) (*ServiceResponse, error) {
	svc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		svc.Name = req.Name
	}
	if req.Description != "" {
		svc.Description = req.Description
	}
	if len(req.Features) > 0 {
		svc.Features = req.Features
	}
	if req.BasePrice > 0 {
		svc.BasePrice = req.BasePrice
	}
	if req.Status != "" {
		svc.Status = service.ServiceStatus(req.Status)
	}
	svc.Featured = req.Featured
	if req.Image != "" {
		svc.Image = req.Image
	}

	if err := s.repo.Update(ctx, svc); err != nil {
		s.logger.WithFields(logger.Fields{"service_id": id, "error": err.Error()}).Error("Failed to update service")
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	s.logger.WithFields(logger.Fields{"service_id": id}).Info("Service updated successfully")
	return s.toServiceResponse(svc), nil
}

func (s *ServiceService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithFields(logger.Fields{"service_id": id, "error": err.Error()}).Error("Failed to delete service")
		return fmt.Errorf("failed to delete service: %w", err)
	}

	s.logger.WithFields(logger.Fields{"service_id": id}).Info("Service deleted successfully")
	return nil
}

func (s *ServiceService) GetStats(ctx context.Context) (map[string]any, error) {
	return s.repo.GetStats(ctx)
}

func (s *ServiceService) toServiceResponse(svc *service.Service) *ServiceResponse {
	return &ServiceResponse{
		ID:           svc.ID,
		TenantID:     svc.TenantID,
		Name:         svc.Name,
		Slug:         svc.Slug,
		Description:  svc.Description,
		Category:     string(svc.Category),
		Status:       string(svc.Status),
		Features:     svc.Features,
		BasePrice:    svc.BasePrice,
		Currency:     svc.Currency,
		PricingModel: string(svc.PricingModel),
		BillingCycle: svc.BillingCycle,
		Duration:     svc.Duration,
		MaxClients:   svc.MaxClients,
		IsPublic:     svc.IsPublic,
		Featured:     svc.Featured,
		Image:        svc.Image,
		Metadata:     svc.Metadata,
		CreatedAt:    svc.CreatedAt,
		UpdatedAt:    svc.UpdatedAt,
	}
}
