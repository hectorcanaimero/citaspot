package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// CRMHandler maneja endpoints de metricas CRM.
type CRMHandler struct {
	repo domain.CRMMetricsRepository
}

// NewCRMHandler crea un handler de metricas CRM.
// Usa el repository directamente — no hay logica de negocio extra para metricas de lectura.
func NewCRMHandler(repo domain.CRMMetricsRepository) *CRMHandler {
	return &CRMHandler{repo: repo}
}

// Metrics retorna metricas agregadas del CRM para el tenant actual.
func (h *CRMHandler) Metrics(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(403, "tenant no identificado")
	}

	metrics, err := h.repo.GetMetrics(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(metrics)
}
