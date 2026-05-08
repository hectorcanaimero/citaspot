package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

type TreatmentHandler struct {
	svc      domain.TreatmentSvc
	validate *validator.Validate
}

func NewTreatmentHandler(svc domain.TreatmentSvc) *TreatmentHandler {
	return &TreatmentHandler{svc: svc, validate: validator.New()}
}

func (h *TreatmentHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	q := &domain.TreatmentListQuery{
		Status: c.Query("status"),
	}
	if cid := c.Query("customer_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "customer_id inválido"})
		}
		q.CustomerID = &id
	}
	if pid := c.Query("professional_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id inválido"})
		}
		q.ProfessionalID = &id
	}

	list, err := h.svc.List(c.Context(), tenantID, q)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

func (h *TreatmentHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.TreatmentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	t, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(t)
}

func (h *TreatmentHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	t, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

func (h *TreatmentHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.TreatmentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	t, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

func (h *TreatmentHandler) UpdateStatus(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.UpdateTreatmentStatusInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	t, err := h.svc.UpdateStatus(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}
