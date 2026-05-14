package handler

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ClinicalNoteHandler maneja endpoints de notas clínicas SOAP.
type ClinicalNoteHandler struct {
	svc      domain.ClinicalNoteSvc
	validate *validator.Validate
}

func NewClinicalNoteHandler(svc domain.ClinicalNoteSvc) *ClinicalNoteHandler {
	return &ClinicalNoteHandler{svc: svc, validate: validator.New()}
}

// Create POST /api/v1/appointments/:id/clinical-note
func (h *ClinicalNoteHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	appointmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de cita inválido"})
	}

	var req domain.CreateClinicalNoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{Error: err.Error()})
	}

	note, err := h.svc.Create(c.Context(), tenantID, appointmentID, &req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(note)
}

// GetByAppointment GET /api/v1/appointments/:id/clinical-note
func (h *ClinicalNoteHandler) GetByAppointment(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	appointmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de cita inválido"})
	}

	note, err := h.svc.GetByAppointmentID(c.Context(), tenantID, appointmentID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(note)
}

// ListByCustomer GET /api/v1/customers/:id/clinical-notes
func (h *ClinicalNoteHandler) ListByCustomer(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	customerID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de cliente inválido"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	notes, total, err := h.svc.ListByCustomer(c.Context(), tenantID, customerID, limit, offset)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": notes, "total": total})
}

// GetByID GET /api/v1/customers/:id/clinical-notes/:noteId
func (h *ClinicalNoteHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	noteID, err := uuid.Parse(c.Params("noteId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de nota inválido"})
	}

	note, err := h.svc.GetByID(c.Context(), tenantID, noteID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(note)
}

// Update PATCH /api/v1/clinical-notes/:noteId
func (h *ClinicalNoteHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	noteID, err := uuid.Parse(c.Params("noteId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de nota inválido"})
	}

	var req domain.UpdateClinicalNoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	if err := h.svc.Update(c.Context(), tenantID, noteID, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// Delete DELETE /api/v1/clinical-notes/:noteId
func (h *ClinicalNoteHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	noteID, err := uuid.Parse(c.Params("noteId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de nota inválido"})
	}

	if err := h.svc.Delete(c.Context(), tenantID, noteID); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
