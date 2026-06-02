package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/citaspot/api/internal/domain"
)

type appointmentSvc struct {
	apptRepo     domain.AppointmentRepository
	serviceRepo  domain.ServiceRepository
	customerRepo domain.CustomerRepository
	profRepo     domain.ProfessionalRepository
	authRepo     domain.AuthRepository
	waClient     domain.WAClient
	notifRepo    domain.NotificationRepository
	publisher    domain.MessagePublisher
	events       domain.EventRepository
	// userNotifSvc es opcional — si es nil, no se generan notificaciones in-app.
	// Permite construir el service sin acoplar el feed dashboard en tests.
	userNotifSvc domain.UserNotificationSvc
	// rdb es opcional — si es nil, publishRealtime es no-op. Permite degradar
	// el realtime dashboard sin tirar el servicio cuando Redis no está disponible.
	rdb *redis.Client
}

// NewAppointmentSvc crea el servicio de citas.
// El parametro rdb es opcional (puede ser nil) — habilita el publish de eventos
// a Redis Pub/Sub para el dashboard SSE. Si es nil, publishRealtime es no-op.
//
// Para enganchar el feed in-app del dashboard, llamar a SetUserNotificationSvc
// tras la construcción. Se mantiene fuera del constructor para no acoplar
// llamadores existentes que aún no inyectan el feed.
func NewAppointmentSvc(
	apptRepo domain.AppointmentRepository,
	serviceRepo domain.ServiceRepository,
	customerRepo domain.CustomerRepository,
	profRepo domain.ProfessionalRepository,
	authRepo domain.AuthRepository,
	waClient domain.WAClient,
	notifRepo domain.NotificationRepository,
	publisher domain.MessagePublisher,
	events domain.EventRepository,
	rdb *redis.Client,
) domain.AppointmentSvc {
	return &appointmentSvc{
		apptRepo:     apptRepo,
		serviceRepo:  serviceRepo,
		customerRepo: customerRepo,
		profRepo:     profRepo,
		authRepo:     authRepo,
		waClient:     waClient,
		notifRepo:    notifRepo,
		publisher:    publisher,
		events:       events,
		rdb:          rdb,
	}
}

// SetUserNotificationSvc inyecta el servicio de notificaciones in-app.
// Si no se llama, los métodos de mutación del service no generan
// notificaciones — el resto del flujo sigue funcionando.
func (s *appointmentSvc) SetUserNotificationSvc(svc domain.UserNotificationSvc) {
	s.userNotifSvc = svc
}

// List retorna las citas de un día para el tenant.
func (s *appointmentSvc) List(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*domain.AppointmentWithDetails, error) {
	return s.apptRepo.ListByDate(ctx, tenantID, date, timezone)
}

// ListFiltered retorna citas con filtros dinámicos y paginación.
func (s *appointmentSvc) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	return s.apptRepo.ListFiltered(ctx, tenantID, q)
}

// ListUpcomingByCustomer retorna citas futuras (pending/confirmed) de un cliente.
func (s *appointmentSvc) ListUpcomingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]*domain.AppointmentWithDetails, error) {
	return s.apptRepo.ListUpcomingByCustomer(ctx, tenantID, customerID)
}

// ListUpcoming retorna citas futuras (pending/confirmed) del tenant.
// Clampa limit a [1,50] con default 10 para proteger la DB ante valores hostiles.
func (s *appointmentSvc) ListUpcoming(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.AppointmentWithDetails, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	statuses := []string{"pending", "confirmed"}
	return s.apptRepo.ListUpcoming(ctx, tenantID, limit, statuses)
}

// GetByID retorna una cita por ID con detalles del cliente/profesional/servicio.
func (s *appointmentSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AppointmentWithDetails, error) {
	return s.apptRepo.GetByID(ctx, tenantID, id)
}

// Create crea una nueva cita con validación de conflictos.
//
// Contrato con el frontend: req.StartsAt debe ser RFC3339 con offset explícito
// (ej. "2026-05-19T08:00:00-04:00") o con sufijo Z. Go interpreta ambos como
// instantes absolutos. NUNCA enviar timestamps naive sin zona — el servicio
// los trataría como UTC y desplazaría la cita por el offset del tenant.
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
		// Normalizar telefono para evitar duplicados (ej: "4243148415" vs "+584243148415")
		normalizedPhone := domain.NormalizePhone(req.CustomerPhone)
		customer, err := s.customerRepo.FindOrCreateByPhone(ctx, tenantID, req.CustomerName, normalizedPhone)
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
		TreatmentID:    req.TreatmentID, // materializa treatment_session si != nil
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

	// Emitir evento para el Rules Engine (único canal de notificaciones)
	s.emitAppointmentEvent(ctx, tenantID, appt.ID, "created")

	// Notificación realtime al dashboard vía Redis Pub/Sub (best-effort, post-commit).
	s.publishRealtime(ctx, tenantID, appt.ID, "appointment.created")

	// Feed in-app del dashboard (best-effort, post-commit).
	s.pushUserNotification(ctx, tenantID, appt.ID, domain.UserNotificationAppointmentCreated)

	return appt, nil
}

// Update actualiza el estado y/o notas de una cita.
func (s *appointmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	// Obtener estado anterior para detectar transiciones de "completed"
	var oldStatus string
	var customerID uuid.UUID
	if req.Status != "" {
		appt, err := s.apptRepo.GetByID(ctx, tenantID, id)
		if err != nil {
			return fmt.Errorf("appointmentSvc.Update: get old status: %w", err)
		}
		oldStatus = appt.Status
		customerID = appt.CustomerID
	}

	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, req); err != nil {
		return err
	}

	// Actualizar total_visits del cliente cuando cambia a/desde "completed"
	if req.Status != "" && req.Status != oldStatus {
		s.syncCustomerVisits(ctx, tenantID, customerID, oldStatus, req.Status)
	}

	if req.Status != "" {
		s.emitAppointmentEvent(ctx, tenantID, id, req.Status)
	}

	// Notificación realtime al dashboard vía Redis Pub/Sub (best-effort, post-commit).
	s.publishRealtime(ctx, tenantID, id, "appointment.updated")

	return nil
}

// syncCustomerVisits incrementa o decrementa total_visits al transicionar a/desde "completed".
func (s *appointmentSvc) syncCustomerVisits(ctx context.Context, tenantID, customerID uuid.UUID, oldStatus, newStatus string) {
	switch {
	case newStatus == "completed" && oldStatus != "completed":
		// Transición a completado: incrementar visitas y actualizar fecha
		now := time.Now().UTC()
		if err := s.customerRepo.IncrementVisits(ctx, tenantID, customerID, 1, &now); err != nil {
			slog.Warn("appointmentSvc.syncCustomerVisits: increment failed", "customer_id", customerID, "error", err)
		}
	case oldStatus == "completed" && newStatus != "completed":
		// Revertir completado: decrementar visitas (sin tocar last_visit_at)
		if err := s.customerRepo.IncrementVisits(ctx, tenantID, customerID, -1, nil); err != nil {
			slog.Warn("appointmentSvc.syncCustomerVisits: decrement failed", "customer_id", customerID, "error", err)
		}
	}
}

// Cancel cancela una cita con motivo opcional.
func (s *appointmentSvc) Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, &domain.UpdateAppointmentRequest{
		Status:             "cancelled",
		CancellationReason: reason,
	}); err != nil {
		return err
	}
	s.emitAppointmentEvent(ctx, tenantID, id, "cancelled")

	// Notificación realtime al dashboard vía Redis Pub/Sub (best-effort, post-commit).
	s.publishRealtime(ctx, tenantID, id, "appointment.cancelled")

	// Feed in-app del dashboard (best-effort, post-commit).
	s.pushUserNotification(ctx, tenantID, id, domain.UserNotificationAppointmentCancelled)

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

	// Si el cliente provee ends_at, usarlo como override de la duración del servicio.
	// Caso contrario, calcular con la duración por defecto del servicio.
	var endsAt time.Time
	if req.EndsAt != nil {
		endsAt = *req.EndsAt
	} else {
		endsAt = req.StartsAt.Add(time.Duration(svc.DurationMin) * time.Minute)
	}
	if !endsAt.After(req.StartsAt) {
		return domain.ErrValidation
	}

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

	s.emitAppointmentEvent(ctx, tenantID, id, "rescheduled")

	// Notificación realtime al dashboard vía Redis Pub/Sub (best-effort, post-commit).
	s.publishRealtime(ctx, tenantID, id, "appointment.rescheduled")

	// Feed in-app del dashboard (best-effort, post-commit).
	s.pushUserNotification(ctx, tenantID, id, domain.UserNotificationAppointmentRescheduled)

	return nil
}

// pushUserNotification crea una entrada en el feed in-app del dashboard.
// Best-effort: si falla, loguea y sigue — nunca rompe la operación principal.
// No-op si userNotifSvc no fue inyectado (tests, modo degradado).
func (s *appointmentSvc) pushUserNotification(ctx context.Context, tenantID, apptID uuid.UUID, eventType string) {
	if s.userNotifSvc == nil {
		return
	}
	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil {
		slog.Warn("appointmentSvc.pushUserNotification: load appointment", "appt_id", apptID, "error", err)
		return
	}

	// Formatear starts_at en timezone del tenant para el body
	tenant, _ := s.authRepo.FindTenantByID(ctx, tenantID)
	loc := time.UTC
	if tenant != nil && tenant.Timezone != "" {
		if l, err := time.LoadLocation(tenant.Timezone); err == nil {
			loc = l
		}
	}
	localTime := appt.StartsAt.In(loc).Format("02/01/2006 3:04 PM")

	var title, body string
	switch eventType {
	case domain.UserNotificationAppointmentCreated:
		title = fmt.Sprintf("Nueva cita: %s con %s", appt.CustomerName, appt.ProfessionalName)
		body = fmt.Sprintf("%s — %s", appt.ServiceName, localTime)
	case domain.UserNotificationAppointmentCancelled:
		title = fmt.Sprintf("Cita cancelada: %s", appt.CustomerName)
		body = fmt.Sprintf("%s — %s", appt.ServiceName, localTime)
	case domain.UserNotificationAppointmentRescheduled:
		title = fmt.Sprintf("Cita reagendada: %s", appt.CustomerName)
		body = fmt.Sprintf("Nuevo horario: %s", localTime)
	default:
		return
	}

	metadata := map[string]any{
		"appointment_id":     apptID.String(),
		"customer_id":        appt.CustomerID.String(),
		"customer_name":      appt.CustomerName,
		"professional_id":    appt.ProfessionalID.String(),
		"professional_name":  appt.ProfessionalName,
		"service_id":         appt.ServiceID.String(),
		"service_name":       appt.ServiceName,
		"starts_at":          appt.StartsAt.UTC().Format(time.RFC3339),
	}

	if _, err := s.userNotifSvc.Create(ctx, tenantID, eventType, title, body, metadata); err != nil {
		slog.Warn("appointmentSvc.pushUserNotification: create", "appt_id", apptID, "type", eventType, "error", err)
	}
}

// publishRealtime publica un evento al canal Redis `tenant:{id}:appointments`
// para que los suscriptores SSE del dashboard reciban el cambio en vivo.
//
// Best-effort: nunca retorna error y nunca panic. Se invoca SIEMPRE post-commit
// (después de que el repo confirmó la transacción) — el dashboard puede recibir
// notificación de un cambio que falle? NO: si el commit no pasó, esta función
// no se llama. Si el publish a Redis falla, el cambio en DB ya está persistido
// y el dashboard se reconcilia en la próxima carga.
//
// Si rdb es nil (Redis no disponible al startup), no-op silencioso.
func (s *appointmentSvc) publishRealtime(ctx context.Context, tenantID, apptID uuid.UUID, eventType string) {
	if s.rdb == nil {
		return
	}
	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil {
		slog.Warn("appointmentSvc.publishRealtime: load appointment", "appt_id", apptID, "error", err)
		return
	}
	payload, err := json.Marshal(map[string]any{
		"event": eventType,
		"data":  appt,
		"ts":    time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.Warn("appointmentSvc.publishRealtime: marshal", "error", err)
		return
	}
	channel := fmt.Sprintf("tenant:%s:appointments", tenantID)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		slog.Warn("appointmentSvc.publishRealtime: publish", "channel", channel, "error", err)
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

// persistEvent persiste un evento analítico de forma best-effort (nunca retorna error).
func (s *appointmentSvc) persistEvent(ctx context.Context, event domain.Event) {
	if s.events == nil {
		return
	}
	if err := s.events.Insert(ctx, event); err != nil {
		slog.Warn("appointmentSvc.persistEvent: failed", "event_type", event.EventType, "entity_id", event.EntityID, "error", err)
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
	case "rescheduled":
		eventType = "appointment.rescheduled"
	default:
		return
	}

	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil {
		slog.Warn("appointmentSvc.emitAppointmentEvent: get error", "error", err)
		return
	}

	// Persistir evento analítico (best-effort)
	var actorType string
	var actorID *uuid.UUID
	switch trigger {
	case "completed":
		actorType = "professional"
		actorID = &appt.ProfessionalID
	case "no_show":
		actorType = "system"
		actorID = nil
	default:
		actorType = "customer"
		actorID = &appt.CustomerID
	}
	s.persistEvent(ctx, domain.Event{
		TenantID:   tenantID,
		EventType:  eventType,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "appointment",
		EntityID:   apptID,
		Payload: map[string]any{
			"service_id":      appt.ServiceID.String(),
			"professional_id": appt.ProfessionalID.String(),
			"starts_at":       appt.StartsAt.Format(time.RFC3339),
			"source":          appt.Source,
		},
		OccurredAt: time.Now(),
	})

	// Formatear fecha/hora en timezone del tenant para templates legibles
	tenant, _ := s.authRepo.FindTenantByID(ctx, tenantID)
	loc := time.UTC
	if tenant != nil && tenant.Timezone != "" {
		if l, err := time.LoadLocation(tenant.Timezone); err == nil {
			loc = l
		}
	}
	localTime := appt.StartsAt.In(loc)

	// Resolver telefono del profesional para reglas con recipient=professional
	var professionalPhone string
	if s.profRepo != nil {
		if prof, err := s.profRepo.GetByID(ctx, tenantID, appt.ProfessionalID); err == nil && prof != nil {
			professionalPhone = prof.Phone
		}
	}

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  eventType,
		CustomerID: appt.CustomerID,
		EntityID:   apptID,
		EntityType: "appointment",
		Payload: map[string]any{
			"appointment_id":     apptID.String(),
			"customer_id":        appt.CustomerID.String(),
			"professional_id":    appt.ProfessionalID.String(),
			"service_id":         appt.ServiceID.String(),
			"status":             appt.Status,
			"trigger":            trigger,
			"starts_at":          localTime.Format("02/01/2006 3:04 PM"),
			"appointment_date":   localTime.Format("02/01/2006"),
			"appointment_time":   localTime.Format("3:04 PM"),
			"customer_phone":     appt.CustomerPhone,
			"customer_name":     appt.CustomerName,
			"service_name":      appt.ServiceName,
			"professional_name":  appt.ProfessionalName,
			"professional_phone": professionalPhone,
		},
		Timestamp: time.Now(),
	})
}
