package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// PublicHandler maneja los endpoints de booking público (sin auth).
type PublicHandler struct {
	svc      domain.PublicSvc
	validate *validator.Validate
}

// NewPublicHandler crea el handler público.
func NewPublicHandler(svc domain.PublicSvc) *PublicHandler {
	return &PublicHandler{svc: svc, validate: validator.New()}
}

// GetProfile GET /public/:slug
// Retorna el perfil del negocio con sus servicios y profesionales.
func (h *PublicHandler) GetProfile(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Slug requerido"})
	}

	profile, err := h.svc.GetProfile(c.Context(), slug)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(profile)
}

// GetAvailability GET /public/:slug/availability?professional_id=&service_id=&date=&timezone=
func (h *PublicHandler) GetAvailability(c *fiber.Ctx) error {
	slug := c.Params("slug")

	profID, err := uuid.Parse(c.Query("professional_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id inválido"})
	}
	svcID, err := uuid.Parse(c.Query("service_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "service_id inválido"})
	}
	date := c.Query("date")
	if date == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El parámetro 'date' es requerido"})
	}

	slots, err := h.svc.GetAvailability(c.Context(), slug, &domain.AvailabilityQuery{
		ProfessionalID: profID,
		ServiceID:      svcID,
		Date:           date,
		Timezone:       c.Query("timezone"),
	})
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": slots})
}

// Book POST /public/:slug/book
// Crea una reserva sin autenticación.
func (h *PublicHandler) Book(c *fiber.Ctx) error {
	slug := c.Params("slug")

	var req domain.CreateAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	// Validar campos mínimos para booking público
	if req.ProfessionalID == uuid.Nil || req.ServiceID == uuid.Nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id y service_id son requeridos"})
	}
	if req.StartsAt.IsZero() {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "starts_at es requerido"})
	}
	if req.CustomerPhone == "" && req.CustomerName == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "customer_name y customer_phone son requeridos"})
	}
	// Preservar el source enviado por el cliente (ej: "whatsapp")
	if req.Source == "" {
		req.Source = "web"
	}

	appt, err := h.svc.Book(c.Context(), slug, &req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(appt)
}

// ListMyAppointments GET /public/:slug/my-appointments?phone=+58412...
// Retorna citas futuras de un cliente identificado por teléfono.
func (h *PublicHandler) ListMyAppointments(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Slug requerido"})
	}

	phone := c.Query("phone")
	if phone == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El parámetro 'phone' es requerido"})
	}

	appts, err := h.svc.ListMyAppointments(c.Context(), slug, phone)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": appts})
}

// ListMyTreatments GET /public/:slug/my-treatments?phone=+58412...
// Retorna tratamientos activos del cliente. 403 si el tenant no tiene
// módulo dental activo. Usado por el AI service tool list_my_treatments.
func (h *PublicHandler) ListMyTreatments(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Slug requerido"})
	}

	phone := c.Query("phone")
	if phone == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El parámetro 'phone' es requerido"})
	}

	treatments, err := h.svc.ListMyTreatments(c.Context(), slug, phone)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": treatments})
}

// CancelAppointment POST /public/:slug/appointments/:id/cancel
// Cancela una cita verificando propiedad por teléfono.
func (h *PublicHandler) CancelAppointment(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Slug requerido"})
	}

	apptID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de cita inválido"})
	}

	var req domain.PublicCancelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if req.Phone == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El campo 'phone' es requerido"})
	}

	if err := h.svc.CancelAppointment(c.Context(), slug, apptID, req.Phone); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// RescheduleAppointment POST /public/:slug/appointments/:id/reschedule
// Reagenda una cita verificando propiedad por teléfono.
func (h *PublicHandler) RescheduleAppointment(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Slug requerido"})
	}

	apptID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de cita inválido"})
	}

	var req domain.PublicRescheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if req.Phone == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El campo 'phone' es requerido"})
	}
	if req.StartsAt.IsZero() {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El campo 'starts_at' es requerido"})
	}

	if err := h.svc.RescheduleAppointment(c.Context(), slug, apptID, req.Phone, req.StartsAt); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
