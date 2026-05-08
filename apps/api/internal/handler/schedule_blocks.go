package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ScheduleBlockHandler endpoints para gestionar bloqueos de horario.
type ScheduleBlockHandler struct {
	scheduleRepo domain.ScheduleRepository
}

// NewScheduleBlockHandler crea el handler de bloqueos de horario.
func NewScheduleBlockHandler(scheduleRepo domain.ScheduleRepository) *ScheduleBlockHandler {
	return &ScheduleBlockHandler{scheduleRepo: scheduleRepo}
}

// Create crea un bloqueo de horario (individual o recurrente).
func (h *ScheduleBlockHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var req domain.ScheduleBlock
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}

	if req.IsRecurring {
		if len(req.RecurrenceDays) == 0 {
			return fiber.NewError(http.StatusBadRequest, "recurrence_days es requerido para bloqueos recurrentes")
		}
		for _, d := range req.RecurrenceDays {
			if d < 0 || d > 6 {
				return fiber.NewError(http.StatusBadRequest, "recurrence_days debe contener valores entre 0 (Dom) y 6 (Sáb)")
			}
		}
	} else {
		if req.StartsAt.IsZero() || req.EndsAt.IsZero() {
			return fiber.NewError(http.StatusBadRequest, "starts_at y ends_at son requeridos")
		}
		if !req.EndsAt.After(req.StartsAt) {
			return fiber.NewError(http.StatusBadRequest, "ends_at debe ser posterior a starts_at")
		}
	}

	req.ID = uuid.New()
	req.TenantID = tenantID

	if err := h.scheduleRepo.CreateBlock(c.Context(), &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(req)
}

// List retorna los bloqueos de horario del tenant.
func (h *ScheduleBlockHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var profID *uuid.UUID
	if pidStr := c.Query("professional_id"); pidStr != "" {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			return fiber.NewError(http.StatusBadRequest, "professional_id inválido")
		}
		profID = &pid
	}

	blocks, err := h.scheduleRepo.ListBlocks(c.Context(), tenantID, profID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": blocks})
}

// Delete elimina un bloqueo de horario.
func (h *ScheduleBlockHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID inválido")
	}

	if err := h.scheduleRepo.DeleteBlock(c.Context(), tenantID, id); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
