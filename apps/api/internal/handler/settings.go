package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// SettingsHandler endpoints para configuración del tenant.
type SettingsHandler struct {
	authRepo domain.AuthRepository
}

// NewSettingsHandler crea el handler de settings.
func NewSettingsHandler(authRepo domain.AuthRepository) *SettingsHandler {
	return &SettingsHandler{authRepo: authRepo}
}

// Get retorna la configuración actual del tenant.
func (h *SettingsHandler) Get(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	settings, err := h.authRepo.GetTenantSettings(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(settings)
}

// Update actualiza la configuración del tenant.
func (h *SettingsHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var req domain.TenantSettings
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}

	// Validar reminder_minutes
	if len(req.ReminderMinutes) > 5 {
		return fiber.NewError(http.StatusBadRequest, "máximo 5 tiempos de recordatorio")
	}
	for _, m := range req.ReminderMinutes {
		if m < 15 || m > 10080 {
			return fiber.NewError(http.StatusBadRequest, "tiempo de recordatorio debe ser entre 15 minutos y 7 días")
		}
	}

	if err := h.authRepo.UpdateTenantSettings(c.Context(), tenantID, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
