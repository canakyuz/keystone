package lesson

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"nexpaces-api/internal/domain/lesson"
	lessonUsecase "nexpaces-api/internal/usecase/lesson"
)

type AssignmentHandler struct {
	service *lessonUsecase.AssignmentService
}

func NewAssignmentHandler(service *lessonUsecase.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{service: service}
}

// Create godoc
// @Summary Create assignment
// @Tags Assignments
// @Accept json
// @Produce json
// @Param request body lessonUsecase.CreateAssignmentRequest true "Create Assignment Request"
// @Success 201 {object} lessonUsecase.AssignmentResponse
// @Router /assignments [post]
func (h *AssignmentHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req lessonUsecase.CreateAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	assignment, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(assignment)
}

// GetByID godoc
// @Summary Get assignment by ID
// @Tags Assignments
// @Produce json
// @Param id path string true "Assignment ID"
// @Success 200 {object} lessonUsecase.AssignmentResponse
// @Router /assignments/{id} [get]
func (h *AssignmentHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	assignment, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == lesson.ErrAssignmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Assignment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(assignment)
}

// List godoc
// @Summary List assignments
// @Tags Assignments
// @Produce json
// @Param student_id query string false "Filter by student"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} lessonUsecase.AssignmentListResponse
// @Router /assignments [get]
func (h *AssignmentHandler) List(c *fiber.Ctx) error {
	filters := lesson.AssignmentListFilters{
		Limit:  10,
		Offset: 0,
	}

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

	if studentID := c.Query("student_id"); studentID != "" {
		filters.StudentID = &studentID
	}

	if status := c.Query("status"); status != "" {
		s := lesson.AssignmentStatus(status)
		filters.Status = &s
	}

	result, err := h.service.List(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// GetByStudent godoc
// @Summary Get assignments by student
// @Tags Assignments
// @Produce json
// @Param student_id path string true "Student ID"
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} lessonUsecase.AssignmentListResponse
// @Router /students/{student_id}/assignments [get]
func (h *AssignmentHandler) GetByStudent(c *fiber.Ctx) error {
	studentID := c.Params("student_id")
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

	result, err := h.service.GetByStudent(c.Context(), studentID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// GetOverdue godoc
// @Summary Get overdue assignments
// @Tags Assignments
// @Produce json
// @Success 200 {array} lessonUsecase.AssignmentResponse
// @Router /assignments/overdue [get]
func (h *AssignmentHandler) GetOverdue(c *fiber.Ctx) error {
	assignments, err := h.service.GetOverdue(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(assignments)
}

// Update godoc
// @Summary Update assignment
// @Tags Assignments
// @Accept json
// @Produce json
// @Param id path string true "Assignment ID"
// @Param request body lessonUsecase.UpdateAssignmentRequest true "Update Assignment Request"
// @Success 200 {object} lessonUsecase.AssignmentResponse
// @Router /assignments/{id} [put]
func (h *AssignmentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req lessonUsecase.UpdateAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	assignment, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		if err == lesson.ErrAssignmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Assignment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(assignment)
}

// Delete godoc
// @Summary Delete assignment
// @Tags Assignments
// @Param id path string true "Assignment ID"
// @Success 204
// @Router /assignments/{id} [delete]
func (h *AssignmentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == lesson.ErrAssignmentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Assignment not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetStats godoc
// @Summary Get assignment statistics
// @Tags Assignments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /assignments/stats [get]
func (h *AssignmentHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}
