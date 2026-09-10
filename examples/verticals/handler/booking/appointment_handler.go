package booking

import (
	"strconv"

	"github.com/canakyuz/keystone/examples/verticals/domain/booking"
	bookingUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/booking"
	"github.com/gofiber/fiber/v2"
)

type AppointmentHandler struct {
	service *bookingUsecase.AppointmentService
}

func NewAppointmentHandler(service *bookingUsecase.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{service: service}
}

func (h *AppointmentHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req bookingUsecase.CreateAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	appointment, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(appointment)
}

func (h *AppointmentHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	appointment, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == booking.ErrAppointmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Appointment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointment)
}

func (h *AppointmentHandler) List(c *fiber.Ctx) error {
	filters := booking.AppointmentListFilters{Limit: 10, Offset: 0}

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

	if clientEmail := c.Query("client_email"); clientEmail != "" {
		filters.ClientEmail = &clientEmail
	}

	if status := c.Query("status"); status != "" {
		s := booking.AppointmentStatus(status)
		filters.Status = &s
	}

	if appType := c.Query("type"); appType != "" {
		t := booking.AppointmentType(appType)
		filters.Type = &t
	}

	result, err := h.service.List(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *AppointmentHandler) GetByUser(c *fiber.Ctx) error {
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

func (h *AppointmentHandler) GetByClient(c *fiber.Ctx) error {
	clientEmail := c.Query("email")
	if clientEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email parameter is required"})
	}

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

	result, err := h.service.GetByClient(c.Context(), clientEmail, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *AppointmentHandler) GetUpcoming(c *fiber.Ctx) error {
	limit := 10

	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	appointments, err := h.service.GetUpcoming(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointments)
}

func (h *AppointmentHandler) GetByDateRange(c *fiber.Ctx) error {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "start_date and end_date are required"})
	}

	appointments, err := h.service.GetByDateRange(c.Context(), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointments)
}

func (h *AppointmentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req bookingUsecase.UpdateAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	appointment, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		if err == booking.ErrAppointmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Appointment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointment)
}

func (h *AppointmentHandler) Confirm(c *fiber.Ctx) error {
	id := c.Params("id")

	appointment, err := h.service.Confirm(c.Context(), id)
	if err != nil {
		if err == booking.ErrAppointmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Appointment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointment)
}

func (h *AppointmentHandler) Cancel(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	appointment, err := h.service.Cancel(c.Context(), id, req.Reason)
	if err != nil {
		if err == booking.ErrAppointmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Appointment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointment)
}

func (h *AppointmentHandler) Complete(c *fiber.Ctx) error {
	id := c.Params("id")

	appointment, err := h.service.Complete(c.Context(), id)
	if err != nil {
		if err == booking.ErrAppointmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Appointment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointment)
}

func (h *AppointmentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == booking.ErrAppointmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Appointment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AppointmentHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}
