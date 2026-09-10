package lesson

import (
	"strconv"

	"github.com/canakyuz/keystone/examples/verticals/domain/lesson"
	lessonUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/lesson"
	"github.com/gofiber/fiber/v2"
)

type LessonHandler struct {
	service *lessonUsecase.LessonService
}

func NewLessonHandler(service *lessonUsecase.LessonService) *LessonHandler {
	return &LessonHandler{service: service}
}

// Create godoc
// @Summary Create lesson
// @Tags Lessons
// @Accept json
// @Produce json
// @Param request body lessonUsecase.CreateLessonRequest true "Create Lesson Request"
// @Success 201 {object} lessonUsecase.LessonResponse
// @Router /lessons [post]
func (h *LessonHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req lessonUsecase.CreateLessonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	lessonRes, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(lessonRes)
}

// GetByID godoc
// @Summary Get lesson by ID
// @Tags Lessons
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} lessonUsecase.LessonResponse
// @Router /lessons/{id} [get]
func (h *LessonHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	lessonRes, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == lesson.ErrLessonNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Lesson not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(lessonRes)
}

// List godoc
// @Summary List lessons
// @Tags Lessons
// @Produce json
// @Param student_id query string false "Filter by student"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} lessonUsecase.LessonListResponse
// @Router /lessons [get]
func (h *LessonHandler) List(c *fiber.Ctx) error {
	filters := lesson.LessonListFilters{
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
		s := lesson.LessonStatus(status)
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
// @Summary Get lessons by student
// @Tags Lessons
// @Produce json
// @Param student_id path string true "Student ID"
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} lessonUsecase.LessonListResponse
// @Router /students/{student_id}/lessons [get]
func (h *LessonHandler) GetByStudent(c *fiber.Ctx) error {
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

// GetUpcoming godoc
// @Summary Get upcoming lessons
// @Tags Lessons
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Success 200 {array} lessonUsecase.LessonResponse
// @Router /lessons/upcoming [get]
func (h *LessonHandler) GetUpcoming(c *fiber.Ctx) error {
	limit := 10

	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	lessons, err := h.service.GetUpcoming(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(lessons)
}

// Update godoc
// @Summary Update lesson
// @Tags Lessons
// @Accept json
// @Produce json
// @Param id path string true "Lesson ID"
// @Param request body lessonUsecase.UpdateLessonRequest true "Update Lesson Request"
// @Success 200 {object} lessonUsecase.LessonResponse
// @Router /lessons/{id} [put]
func (h *LessonHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req lessonUsecase.UpdateLessonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	lessonRes, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		if err == lesson.ErrLessonNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Lesson not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(lessonRes)
}

// Delete godoc
// @Summary Delete lesson
// @Tags Lessons
// @Param id path string true "Lesson ID"
// @Success 204
// @Router /lessons/{id} [delete]
func (h *LessonHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == lesson.ErrLessonNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Lesson not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetStats godoc
// @Summary Get lesson statistics
// @Tags Lessons
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /lessons/stats [get]
func (h *LessonHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}
