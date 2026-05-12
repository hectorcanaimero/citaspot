package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// TreatmentSessionHandler maneja los endpoints REST de sesiones de tratamiento.
type TreatmentSessionHandler struct {
	svc      domain.TreatmentSessionSvc
	validate *validator.Validate
}

// NewTreatmentSessionHandler crea el handler de sesiones.
func NewTreatmentSessionHandler(svc domain.TreatmentSessionSvc) *TreatmentSessionHandler {
	return &TreatmentSessionHandler{svc: svc, validate: validator.New()}
}

// List GET /api/v1/treatments/:id/sessions
func (h *TreatmentSessionHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	sessions, err := h.svc.List(c.UserContext(), tenantID, treatmentID)
	if err != nil {
		return handleServiceError(c, err)
	}

	if sessions == nil {
		sessions = []*domain.TreatmentSession{}
	}
	return c.JSON(fiber.Map{"data": sessions})
}

// Create POST /api/v1/treatments/:id/sessions
func (h *TreatmentSessionHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	var input domain.TreatmentSessionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	session, err := h.svc.Create(c.UserContext(), tenantID, treatmentID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(session)
}

// GetByID GET /api/v1/treatments/:id/sessions/:sid
func (h *TreatmentSessionHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	sessionID, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de sesión inválido"})
	}

	session, err := h.svc.GetByID(c.UserContext(), tenantID, sessionID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(session)
}

// Update PATCH /api/v1/treatments/:id/sessions/:sid
func (h *TreatmentSessionHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	sessionID, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de sesión inválido"})
	}

	var input domain.UpdateTreatmentSessionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	session, err := h.svc.Update(c.UserContext(), tenantID, treatmentID, sessionID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(session)
}

// Delete DELETE /api/v1/treatments/:id/sessions/:sid
func (h *TreatmentSessionHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	sessionID, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de sesión inválido"})
	}

	if err := h.svc.Delete(c.UserContext(), tenantID, treatmentID, sessionID); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
