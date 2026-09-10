package service

import (
	"strconv"

	"github.com/canakyuz/keystone/examples/verticals/domain/service"
	serviceUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/service"
	"github.com/gofiber/fiber/v2"
)

type ServiceHandler struct {
	service *serviceUsecase.ServiceService
}

func NewServiceHandler(service *serviceUsecase.ServiceService) *ServiceHandler {
	return &ServiceHandler{service: service}
}

func (h *ServiceHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req serviceUsecase.CreateServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	svc, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(svc)
}

func (h *ServiceHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	svc, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == service.ErrServiceNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Service not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(svc)
}

func (h *ServiceHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	svc, err := h.service.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == service.ErrServiceNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Service not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(svc)
}

func (h *ServiceHandler) List(c *fiber.Ctx) error {
	filters := service.ServiceListFilters{Limit: 10, Offset: 0}

	if limit := c.Query("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			filters.Limit = val
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil {
			filters.Offset = val
		}
	}

	if category := c.Query("category"); category != "" {
		cat := service.ServiceCategory(category)
		filters.Category = &cat
	}

	if status := c.Query("status"); status != "" {
		st := service.ServiceStatus(status)
		filters.Status = &st
	}

	if featured := c.Query("featured"); featured == "true" {
		f := true
		filters.Featured = &f
	}

	result, err := h.service.List(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *ServiceHandler) GetFeatured(c *fiber.Ctx) error {
	limit := 10

	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	services, err := h.service.GetFeatured(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(services)
}

func (h *ServiceHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req serviceUsecase.UpdateServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	svc, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		if err == service.ErrServiceNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Service not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(svc)
}

func (h *ServiceHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == service.ErrServiceNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Service not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ServiceHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}
