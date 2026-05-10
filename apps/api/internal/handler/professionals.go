package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ProfessionalHandler maneja los endpoints de profesionales y sus horarios.
type ProfessionalHandler struct {
	svc      domain.ProfessionalSvc
	validate *validator.Validate
}

// NewProfessionalHandler crea el handler de profesionales.
func NewProfessionalHandler(svc domain.ProfessionalSvc) *ProfessionalHandler {
	return &ProfessionalHandler{svc: svc, validate: validator.New()}
}

// List GET /professionals
func (h *ProfessionalHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	includeArchived := c.Query("include_archived") == "true"
	list, err := h.svc.List(c.Context(), tenantID, includeArchived)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /professionals
func (h *ProfessionalHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.ProfessionalInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	p, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(p)
}

// GetByID GET /professionals/:id
func (h *ProfessionalHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	p, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(p)
}

// Update PATCH /professionals/:id
func (h *ProfessionalHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.ProfessionalInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	p, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(p)
}

// GetSchedule GET /professionals/:id/schedule
func (h *ProfessionalHandler) GetSchedule(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	schedules, err := h.svc.GetSchedule(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": schedules})
}

// SetSchedule PUT /professionals/:id/schedule
func (h *ProfessionalHandler) SetSchedule(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var schedules []*domain.Schedule
	if err := c.BodyParser(&schedules); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	result, err := h.svc.SetSchedule(c.Context(), tenantID, id, schedules)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": result})
}
