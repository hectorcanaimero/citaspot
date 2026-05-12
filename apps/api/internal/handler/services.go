package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ServiceHandler maneja los endpoints de servicios del negocio.
type ServiceHandler struct {
	svc      domain.ServiceSvc
	validate *validator.Validate
}

// NewServiceHandler crea el handler de servicios.
func NewServiceHandler(svc domain.ServiceSvc) *ServiceHandler {
	return &ServiceHandler{svc: svc, validate: validator.New()}
}

// List GET /services
func (h *ServiceHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	list, err := h.svc.List(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /services
func (h *ServiceHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.ServiceInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	svc, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(svc)
}

// GetByID GET /services/:id
func (h *ServiceHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	svc, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(svc)
}

// Delete DELETE /services/:id
func (h *ServiceHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	if err := h.svc.Delete(c.Context(), tenantID, id); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// Update PATCH /services/:id
func (h *ServiceHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.ServiceInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	svc, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(svc)
}
