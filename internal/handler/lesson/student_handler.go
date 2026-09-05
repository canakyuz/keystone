package lesson

import (
	"strconv"

	"github.com/canakyuz/keystone/internal/domain/lesson"
	lessonUsecase "github.com/canakyuz/keystone/internal/usecase/lesson"
	"github.com/gofiber/fiber/v2"
)

type StudentHandler struct {
	service *lessonUsecase.StudentService
}

func NewStudentHandler(service *lessonUsecase.StudentService) *StudentHandler {
	return &StudentHandler{service: service}
}

// Create godoc
// @Summary Create student
// @Tags Students
// @Accept json
// @Produce json
// @Param request body lessonUsecase.CreateStudentRequest true "Create Student Request"
// @Success 201 {object} lessonUsecase.StudentResponse
// @Router /students [post]
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var req lessonUsecase.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	student, err := h.service.Create(c.Context(), tenantID, &req)
	if err != nil {
		if err == lesson.ErrStudentAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Student with this email already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(student)
}

// GetByID godoc
// @Summary Get student by ID
// @Tags Students
// @Produce json
// @Param id path string true "Student ID"
// @Success 200 {object} lessonUsecase.StudentResponse
// @Router /students/{id} [get]
func (h *StudentHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	student, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == lesson.ErrStudentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Student not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(student)
}

// GetByEmail godoc
// @Summary Get student by email
// @Tags Students
// @Produce json
// @Param email query string true "Student Email"
// @Success 200 {object} lessonUsecase.StudentResponse
// @Router /students/email [get]
func (h *StudentHandler) GetByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email parameter is required",
		})
	}

	student, err := h.service.GetByEmail(c.Context(), email)
	if err != nil {
		if err == lesson.ErrStudentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Student not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(student)
}

// List godoc
// @Summary List students
// @Tags Students
// @Produce json
// @Param status query string false "Filter by status"
// @Param level query string false "Filter by level"
// @Param search query string false "Search by name or email"
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} lessonUsecase.StudentListResponse
// @Router /students [get]
func (h *StudentHandler) List(c *fiber.Ctx) error {
	filters := lesson.StudentListFilters{
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

	if status := c.Query("status"); status != "" {
		s := lesson.StudentStatus(status)
		filters.Status = &s
	}

	if level := c.Query("level"); level != "" {
		l := lesson.StudentLevel(level)
		filters.Level = &l
	}

	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	result, err := h.service.List(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// Update godoc
// @Summary Update student
// @Tags Students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Param request body lessonUsecase.UpdateStudentRequest true "Update Student Request"
// @Success 200 {object} lessonUsecase.StudentResponse
// @Router /students/{id} [put]
func (h *StudentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req lessonUsecase.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	student, err := h.service.Update(c.Context(), id, &req)
	if err != nil {
		if err == lesson.ErrStudentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Student not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(student)
}

// Delete godoc
// @Summary Delete student
// @Tags Students
// @Param id path string true "Student ID"
// @Success 204
// @Router /students/{id} [delete]
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == lesson.ErrStudentNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Student not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetStats godoc
// @Summary Get student statistics
// @Tags Students
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /students/stats [get]
func (h *StudentHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}
