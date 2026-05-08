package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type appointmentSvc struct {
	apptRepo     domain.AppointmentRepository
	serviceRepo  domain.ServiceRepository
	customerRepo domain.CustomerRepository
	authRepo     domain.AuthRepository
	waClient     domain.WAClient
	notifRepo    domain.NotificationRepository
	publisher    domain.MessagePublisher
}

// NewAppointmentSvc crea el servicio de citas.
func NewAppointmentSvc(
	apptRepo domain.AppointmentRepository,
	serviceRepo domain.ServiceRepository,
	customerRepo domain.CustomerRepository,
	authRepo domain.AuthRepository,
	waClient domain.WAClient,
	notifRepo domain.NotificationRepository,
	publisher domain.MessagePublisher,
) domain.AppointmentSvc {
	return &appointmentSvc{
		apptRepo:     apptRepo,
		serviceRepo:  serviceRepo,
		customerRepo: customerRepo,
		authRepo:     authRepo,
		waClient:     waClient,
		notifRepo:    notifRepo,
		publisher:    publisher,
	}
}

// List retorna las citas de un día para el tenant.
func (s *appointmentSvc) List(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*domain.AppointmentWithDetails, error) {
	return s.apptRepo.ListByDate(ctx, tenantID, date, timezone)
}

// ListFiltered retorna citas con filtros dinámicos y paginación.
func (s *appointmentSvc) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	return s.apptRepo.ListFiltered(ctx, tenantID, q)
}

// GetByID retorna una cita por ID con detalles del cliente/profesional/servicio.
func (s *appointmentSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AppointmentWithDetails, error) {
	return s.apptRepo.GetByID(ctx, tenantID, id)
}

// Create crea una nueva cita con validación de conflictos.
func (s *appointmentSvc) Create(ctx context.Context, tenantID uuid.UUID, req *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
	// 1. Obtener servicio para calcular ends_at
	svc, err := s.serviceRepo.GetByID(ctx, tenantID, req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("appointmentSvc.Create: service: %w", err)
	}

	endsAt := req.StartsAt.Add(time.Duration(svc.DurationMin) * time.Minute)

	// 2. Verificar conflicto de horario
	conflict, err := s.apptRepo.CheckConflict(ctx, tenantID, req.ProfessionalID, req.StartsAt, endsAt, nil)
	if err != nil {
		return nil, fmt.Errorf("appointmentSvc.Create: conflict check: %w", err)
	}
	if conflict {
		return nil, domain.ErrSlotUnavailable
	}

	// 3. Resolver o crear cliente
	var customerID uuid.UUID
	if req.CustomerID != nil {
		customerID = *req.CustomerID
	} else {
		// Booking público: buscar o crear por teléfono
		if req.CustomerPhone == "" {
			return nil, fmt.Errorf("%w: customer_id o customer_phone son requeridos", domain.ErrValidation)
		}
		customer, err := s.customerRepo.FindOrCreateByPhone(ctx, tenantID, req.CustomerName, req.CustomerPhone)
		if err != nil {
			return nil, fmt.Errorf("appointmentSvc.Create: customer: %w", err)
		}
		customerID = customer.ID
	}

	// 4. Definir source por defecto
	source := req.Source
	if source == "" {
		source = "dashboard"
	}

	// 5. Crear cita
	appt := &domain.Appointment{
		ID:             uuid.New(),
		TenantID:       tenantID,
		CustomerID:     customerID,
		ProfessionalID: req.ProfessionalID,
		ServiceID:      req.ServiceID,
		StartsAt:       req.StartsAt,
		EndsAt:         endsAt,
		Status:         "pending",
		Source:         source,
		Price:          svc.Price, // snapshot del precio actual
		Notes:          req.Notes,
	}

	if err := s.apptRepo.Create(ctx, appt); err != nil {
		return nil, fmt.Errorf("appointmentSvc.Create: %w", err)
	}

	// Best-effort: notificar al paciente que la cita está pendiente
	s.notifyAppointmentStatus(ctx, tenantID, appt.ID, "pending")
	s.emitAppointmentEvent(ctx, tenantID, appt.ID, "created")

	return appt, nil
}

// Update actualiza el estado y/o notas de una cita.
func (s *appointmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, req); err != nil {
		return err
	}
	if req.Status != "" {
		s.notifyAppointmentStatus(ctx, tenantID, id, req.Status)
		s.emitAppointmentEvent(ctx, tenantID, id, req.Status)
	}
	return nil
}

// Cancel cancela una cita con motivo opcional.
func (s *appointmentSvc) Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, &domain.UpdateAppointmentRequest{
		Status:             "cancelled",
		CancellationReason: reason,
	}); err != nil {
		return err
	}
	s.notifyAppointmentStatus(ctx, tenantID, id, "cancelled")
	s.emitAppointmentEvent(ctx, tenantID, id, "cancelled")
	return nil
}

// Reschedule reagenda una cita a nuevo horario y/o profesional.
func (s *appointmentSvc) Reschedule(ctx context.Context, tenantID, id uuid.UUID, req *domain.RescheduleRequest) error {
	appt, err := s.apptRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: get: %w", err)
	}

	profID := appt.ProfessionalID
	if req.ProfessionalID != nil {
		profID = *req.ProfessionalID
	}
	svcID := appt.ServiceID
	if req.ServiceID != nil {
		svcID = *req.ServiceID
	}

	svc, err := s.serviceRepo.GetByID(ctx, tenantID, svcID)
	if err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: service: %w", err)
	}
	endsAt := req.StartsAt.Add(time.Duration(svc.DurationMin) * time.Minute)

	excludeID := id
	conflict, err := s.apptRepo.CheckConflict(ctx, tenantID, profID, req.StartsAt, endsAt, &excludeID)
	if err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: conflict: %w", err)
	}
	if conflict {
		return domain.ErrSlotUnavailable
	}

	if err := s.apptRepo.Reschedule(ctx, tenantID, id, profID, req.StartsAt, endsAt); err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: %w", err)
	}

	s.notifyAppointmentStatus(ctx, tenantID, id, "rescheduled")
	return nil
}

// notifyAppointmentStatus envía una notificación WA al paciente sobre el estado de su cita.
// Best-effort: si falla, loguea el error pero no retorna error.
func (s *appointmentSvc) notifyAppointmentStatus(ctx context.Context, tenantID, apptID uuid.UUID, notifType string) {
	if s.waClient == nil {
		return
	}

	tenant, err := s.authRepo.FindTenantByID(ctx, tenantID)
	if err != nil || tenant.WAStatus != "connected" {
		return
	}

	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil || appt.CustomerPhone == "" {
		return
	}

	loc, _ := time.LoadLocation(tenant.Timezone)
	if loc == nil {
		loc = time.UTC
	}
	fecha := appt.StartsAt.In(loc).Format("02/01/2006")
	hora := appt.StartsAt.In(loc).Format("3:04 PM")

	var text string
	switch notifType {
	case "pending":
		text = fmt.Sprintf(
			"¡Hola %s! 📋 Tu cita para %s con %s el %s a las %s ha sido registrada y está *pendiente de aprobación*. Te avisaremos cuando sea confirmada.",
			appt.CustomerName, appt.ServiceName, appt.ProfessionalName, fecha, hora,
		)
	case "confirmed":
		text = fmt.Sprintf(
			"¡Hola %s! ✅ Tu cita ha sido *confirmada*:\n\n"+
				"📌 *%s*\n"+
				"👩‍⚕️ %s\n"+
				"📅 %s\n"+
				"🕐 %s\n\n"+
				"¡Te esperamos!",
			appt.CustomerName, appt.ServiceName, appt.ProfessionalName, fecha, hora,
		)
	case "cancelled":
		text = fmt.Sprintf(
			"Hola %s, lamentamos informarte que tu cita de %s el %s a las %s ha sido *cancelada*. Puedes reservar nuevamente cuando lo desees.",
			appt.CustomerName, appt.ServiceName, fecha, hora,
		)
	case "rescheduled":
		text = fmt.Sprintf(
			"¡Hola %s! 🔄 Tu cita ha sido *reagendada*:\n\n"+
				"📌 *%s*\n"+
				"👩‍⚕️ %s\n"+
				"📅 %s\n"+
				"🕐 %s\n\n"+
				"¡Te esperamos!",
			appt.CustomerName, appt.ServiceName, appt.ProfessionalName, fecha, hora,
		)
	default:
		return
	}

	waMessageID, sendErr := s.waClient.SendText(ctx, tenant.Slug, appt.CustomerPhone, text)

	aid := appt.ID
	nl := &domain.NotificationLog{
		ID:            uuid.New(),
		TenantID:      tenantID,
		AppointmentID: &aid,
		Type:          notifType,
		WAMessageID:   waMessageID,
		Content:       text,
		SentAt:        time.Now(),
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
		log.Printf("appointmentSvc.notify: failed to send %s notification: %v", notifType, sendErr)
	} else {
		nl.Status = "sent"
	}
	if err := s.notifRepo.LogNotification(ctx, nl); err != nil {
		log.Printf("appointmentSvc.notify: log error: %v", err)
	}
}

// publishRuleEvent publica un evento de dominio en la cola rules.events.
func (s *appointmentSvc) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("appointmentSvc.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("appointmentSvc.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}

// emitAppointmentEvent emite un evento de cita para el motor de reglas.
// El parámetro `trigger` puede ser un status real ("completed", "cancelled",
// "no_show", "confirmed") o el pseudo-trigger "created" para la creación.
func (s *appointmentSvc) emitAppointmentEvent(ctx context.Context, tenantID, apptID uuid.UUID, trigger string) {
	eventType := ""
	switch trigger {
	case "created":
		eventType = "appointment.created"
	case "confirmed":
		eventType = "appointment.confirmed"
	case "completed":
		eventType = "appointment.completed"
	case "cancelled":
		eventType = "appointment.cancelled"
	case "no_show":
		eventType = "appointment.no_show"
	default:
		return
	}

	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil {
		slog.Warn("appointmentSvc.emitAppointmentEvent: get error", "error", err)
		return
	}

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  eventType,
		CustomerID: appt.CustomerID,
		EntityID:   apptID,
		EntityType: "appointment",
		Payload: map[string]any{
			"appointment_id":  apptID.String(),
			"customer_id":     appt.CustomerID.String(),
			"professional_id": appt.ProfessionalID.String(),
			"service_id":      appt.ServiceID.String(),
			"status":          appt.Status,
			"trigger":         trigger,
			"starts_at":       appt.StartsAt.Format(time.RFC3339),
			"customer_phone":  appt.CustomerPhone,
			"customer_name":   appt.CustomerName,
		},
		Timestamp: time.Now(),
	})
}
