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

// ListFiltered GET /appointments/search?date_from=&date_to=&...
func (h *AppointmentHandler) ListFiltered(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	tenant := middleware.TenantFromContext(c)

	// Parámetros requeridos
	dateFrom := c.Query("date_from")
	if dateFrom == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El parámetro 'date_from' es requerido (YYYY-MM-DD)"})
	}
	dateTo := c.Query("date_to")
	if dateTo == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "El parámetro 'date_to' es requerido (YYYY-MM-DD)"})
	}

	// Timezone: query param o fallback al timezone del tenant
	timezone := c.Query("timezone")
	if timezone == "" && tenant != nil {
		timezone = tenant.Timezone
	}

	// UUIDs opcionales
	var profID *uuid.UUID
	if raw := c.Query("professional_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id inválido"})
		}
		profID = &id
	}

	var svcID *uuid.UUID
	if raw := c.Query("service_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "service_id inválido"})
		}
		svcID = &id
	}

	// Enteros opcionales con fallback a cero (el repositorio aplica los defaults)
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))

	q := &domain.AppointmentListQuery{
		DateFrom:       dateFrom,
		DateTo:         dateTo,
		Timezone:       timezone,
		ProfessionalID: profID,
		ServiceID:      svcID,
		Status:         c.Query("status"),
		Search:         c.Query("search"),
		SortBy:         c.Query("sort_by"),
		SortDir:        c.Query("sort_dir"),
		Page:           page,
		PerPage:        perPage,
	}

	result, err := h.apptSvc.ListFiltered(c.Context(), tenantID, q)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(result)
}

// Upcoming GET /appointments/upcoming?limit=10
// Retorna citas futuras (pending/confirmed) del tenant, ordenadas por starts_at ASC.
// Default limit = 10, max = 50 (clamping en el service).
func (h *AppointmentHandler) Upcoming(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusForbidden).JSON(errorResponse{Error: "tenant no identificado"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "limit inválido"})
	}

	items, err := h.apptSvc.ListUpcoming(c.Context(), tenantID, limit)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": items})
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

// Reschedule reagenda una cita a nuevo horario y/o profesional.
func (h *AppointmentHandler) Reschedule(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID inválido")
	}

	var req domain.RescheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}
	if req.StartsAt.IsZero() {
		return fiber.NewError(http.StatusBadRequest, "starts_at es requerido")
	}

	if err := h.apptSvc.Reschedule(c.Context(), tenantID, id, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
