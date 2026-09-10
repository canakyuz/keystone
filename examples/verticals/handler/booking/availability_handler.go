package booking

import (
	"strconv"

	"github.com/canakyuz/keystone/examples/verticals/domain/booking"
	bookingUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/booking"
	"github.com/gofiber/fiber/v2"
)

type AvailabilityHandler struct {
	service *bookingUsecase.AvailabilityService
}

func NewAvailabilityHandler(service *bookingUsecase.AvailabilityService) *AvailabilityHandler {
	return &AvailabilityHandler{service: service}
}

func (h *AvailabilityHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req bookingUsecase.CreateAvailabilityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	availability, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(availability)
}

func (h *AvailabilityHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	availability, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == booking.ErrAvailabilityNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Availability not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(availability)
}

func (h *AvailabilityHandler) List(c *fiber.Ctx) error {
	filters := booking.AvailabilityListFilters{Limit: 10, Offset: 0}

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

	if userID := c.Query("user_id"); userID != "" {
		filters.UserID = &userID
	}

	if status := c.Query("status"); status != "" {
		s := booking.AvailabilityStatus(status)
		filters.Status = &s
	}

	result, err := h.service.List(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *AvailabilityHandler) GetByUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	limit := 10
	offset := 0

	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	result, err := h.service.GetByUser(c.Context(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *AvailabilityHandler) GetByDateRange(c *fiber.Ctx) error {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "start_date and end_date are required"})
	}

	availabilities, err := h.service.GetByDateRange(c.Context(), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(availabilities)
}

func (h *AvailabilityHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req bookingUsecase.UpdateAvailabilityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	availability, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		if err == booking.ErrAvailabilityNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Availability not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(availability)
}

func (h *AvailabilityHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == booking.ErrAvailabilityNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Availability not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AvailabilityHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}
