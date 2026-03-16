package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// AppointmentHandler maneja los endpoints de citas y disponibilidad.
type AppointmentHandler struct {
	apptSvc  domain.AppointmentSvc
	availSvc domain.AvailabilityService
	validate *validator.Validate
}

// NewAppointmentHandler crea el handler de citas.
func NewAppointmentHandler(apptSvc domain.AppointmentSvc, availSvc domain.AvailabilityService) *AppointmentHandler {
	return &AppointmentHandler{
		apptSvc:  apptSvc,
		availSvc: availSvc,
		validate: validator.New(),
	}
}

// List GET /appointments?date=YYYY-MM-DD&timezone=America/Santo_Domingo
func (h *AppointmentHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	tenant := middleware.TenantFromContext(c)

	date := c.Query("date")
	if date == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El parámetro 'date' es requerido (YYYY-MM-DD)"})
	}
	timezone := c.Query("timezone")
	if timezone == "" && tenant != nil {
		timezone = tenant.Timezone
	}

	list, err := h.apptSvc.List(c.Context(), tenantID, date, timezone)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /appointments
func (h *AppointmentHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var req domain.CreateAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}
	if req.Source == "" {
		req.Source = "dashboard"
	}

	appt, err := h.apptSvc.Create(c.Context(), tenantID, &req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(appt)
}

// GetByID GET /appointments/:id
func (h *AppointmentHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	appt, err := h.apptSvc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(appt)
}

// Update PATCH /appointments/:id
func (h *AppointmentHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var req domain.UpdateAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	if err := h.apptSvc.Update(c.Context(), tenantID, id, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// Cancel DELETE /appointments/:id/cancel
func (h *AppointmentHandler) Cancel(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)

	if err := h.apptSvc.Cancel(c.Context(), tenantID, id, body.Reason); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// Availability GET /appointments/availability?professional_id=&service_id=&date=&timezone=
func (h *AppointmentHandler) Availability(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	tenant := middleware.TenantFromContext(c)

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
	timezone := c.Query("timezone")
	if timezone == "" && tenant != nil {
		timezone = tenant.Timezone
	}

	slots, err := h.availSvc.GetAvailableSlots(c.Context(), tenantID, &domain.AvailabilityQuery{
		ProfessionalID: profID,
		ServiceID:      svcID,
		Date:           date,
		Timezone:       timezone,
	})
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": slots})
}
