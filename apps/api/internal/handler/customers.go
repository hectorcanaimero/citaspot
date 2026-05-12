package handler

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// CustomerHandler maneja los endpoints de clientes.
type CustomerHandler struct {
	repo domain.CustomerRepository
}

// NewCustomerHandler crea el handler de clientes.
func NewCustomerHandler(repo domain.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{repo: repo}
}

// List GET /api/v1/customers
// Parámetros opcionales: search, limit (default 50), offset (default 0).
func (h *CustomerHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	search := c.Query("search", "")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	// Límites razonables
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	customers, err := h.repo.List(c.Context(), tenantID, search, limit, offset)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": customers})
}

// GetByID GET /api/v1/customers/:id
func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	customer, err := h.repo.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(customer)
}

// UpdateStage PATCH /api/v1/customers/:id/stage
func (h *CustomerHandler) UpdateStage(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var body struct {
		StageID *string `json:"stage_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Formato inválido"})
	}

	var stageID *uuid.UUID
	if body.StageID != nil && *body.StageID != "" {
		parsed, err := uuid.Parse(*body.StageID)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "stage_id inválido"})
		}
		stageID = &parsed
	}

	if err := h.repo.UpdateStage(c.Context(), tenantID, id, stageID); err != nil {
		return handleServiceError(c, err)
	}

	customer, err := h.repo.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(customer)
}
