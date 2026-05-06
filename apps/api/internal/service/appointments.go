package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type appointmentSvc struct {
	apptRepo     domain.AppointmentRepository
	serviceRepo  domain.ServiceRepository
	customerRepo domain.CustomerRepository
}

// NewAppointmentSvc crea el servicio de citas.
func NewAppointmentSvc(
	apptRepo domain.AppointmentRepository,
	serviceRepo domain.ServiceRepository,
	customerRepo domain.CustomerRepository,
) domain.AppointmentSvc {
	return &appointmentSvc{
		apptRepo:     apptRepo,
		serviceRepo:  serviceRepo,
		customerRepo: customerRepo,
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
	return appt, nil
}

// Update actualiza el estado y/o notas de una cita.
func (s *appointmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	return s.apptRepo.UpdateStatus(ctx, tenantID, id, req)
}

// Cancel cancela una cita con motivo opcional.
func (s *appointmentSvc) Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	return s.apptRepo.UpdateStatus(ctx, tenantID, id, &domain.UpdateAppointmentRequest{
		Status:             "cancelled",
		CancellationReason: reason,
	})
}
