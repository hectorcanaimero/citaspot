package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

type PipelineStageHandler struct {
	svc      domain.PipelineStageSvc
	validate *validator.Validate
}

func NewPipelineStageHandler(svc domain.PipelineStageSvc) *PipelineStageHandler {
	return &PipelineStageHandler{svc: svc, validate: validator.New()}
}

func (h *PipelineStageHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	list, err := h.svc.List(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

func (h *PipelineStageHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.PipelineStageInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	stage, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(stage)
}

func (h *PipelineStageHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	stage, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(stage)
}

func (h *PipelineStageHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.PipelineStageInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	stage, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(stage)
}

func (h *PipelineStageHandler) Delete(c *fiber.Ctx) error {
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

func (h *PipelineStageHandler) Reorder(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var items []domain.ReorderStageInput
	if err := c.BodyParser(&items); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if len(items) == 0 {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Lista de etapas vacía"})
	}

	if err := h.svc.Reorder(c.Context(), tenantID, items); err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
