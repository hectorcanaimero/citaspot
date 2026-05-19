package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type publicSvc struct {
	authRepo       domain.AuthRepository
	profRepo       domain.ProfessionalRepository
	serviceRepo    domain.ServiceRepository
	availSvc       domain.AvailabilityService
	apptSvc        domain.AppointmentSvc
	customerRepo   domain.CustomerRepository
	treatmentRepo  domain.TreatmentRepository
	moduleRepo     domain.TenantModuleRepository
}

// NewPublicSvc crea el servicio de booking público (sin auth).
func NewPublicSvc(
	authRepo domain.AuthRepository,
	profRepo domain.ProfessionalRepository,
	serviceRepo domain.ServiceRepository,
	availSvc domain.AvailabilityService,
	apptSvc domain.AppointmentSvc,
	customerRepo domain.CustomerRepository,
	treatmentRepo domain.TreatmentRepository,
	moduleRepo domain.TenantModuleRepository,
) domain.PublicSvc {
	return &publicSvc{
		authRepo:      authRepo,
		profRepo:      profRepo,
		serviceRepo:   serviceRepo,
		availSvc:      availSvc,
		apptSvc:       apptSvc,
		customerRepo:  customerRepo,
		treatmentRepo: treatmentRepo,
		moduleRepo:    moduleRepo,
	}
}

// GetProfile retorna el perfil público del negocio (sin datos sensibles).
func (s *publicSvc) GetProfile(ctx context.Context, slug string) (*domain.PublicProfile, error) {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	services, err := s.serviceRepo.ListActive(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("publicSvc.GetProfile: services: %w", err)
	}

	professionals, err := s.profRepo.ListByTenantPublic(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("publicSvc.GetProfile: professionals: %w", err)
	}

	links, err := s.profRepo.ListServiceLinks(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("publicSvc.GetProfile: service links: %w", err)
	}

	return &domain.PublicProfile{
		Slug:               tenant.Slug,
		Name:               tenant.Name,
		BusinessType:       tenant.BusinessType,
		City:               tenant.City,
		Country:            tenant.Country,
		Timezone:           tenant.Timezone,
		Services:             services,
		Professionals:        professionals,
		ServiceProfessionals: links,
		BookingIntroText:   tenant.Settings.BookingIntroText,
		BookingSuccessText: tenant.Settings.BookingSuccessText,
		BotName:            tenant.Settings.BotName,
		BotGreeting:        tenant.Settings.BotGreeting,
		LogoURL:            tenant.Settings.LogoURL,
		CoverURL:           tenant.Settings.CoverURL,
		Description:        tenant.Settings.Description,
	}, nil
}

// GetAvailability retorna slots disponibles para booking público.
func (s *publicSvc) GetAvailability(ctx context.Context, slug string, query *domain.AvailabilityQuery) ([]*domain.TimeSlot, error) {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Siempre usar el timezone del tenant para cálculos de disponibilidad.
	// El timezone del cliente puede diferir del negocio y causar errores si el
	// container no tiene tzdata (e.g. America/Sao_Paulo falla en Alpine).
	// Si el tenant no tiene timezone, dejarlo vacío: el engine usa "UTC" como fallback.
	query.Timezone = tenant.Timezone

	return s.availSvc.GetAvailableSlots(ctx, tenant.ID, query)
}

// Book crea una cita desde la página pública de reservas.
func (s *publicSvc) Book(ctx context.Context, slug string, req *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if req.Source == "" {
		req.Source = "web"
	}

	return s.apptSvc.Create(ctx, tenant.ID, req)
}

// ListMyAppointments retorna citas futuras de un cliente identificado por teléfono.
// Si el cliente no existe, retorna lista vacía (no es un error).
func (s *publicSvc) ListMyAppointments(ctx context.Context, slug, phone string) ([]*domain.PublicAppointment, error) {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	phone = domain.NormalizePhone(phone)

	customer, err := s.customerRepo.FindByPhone(ctx, tenant.ID, phone)
	if err != nil {
		return nil, fmt.Errorf("publicSvc.ListMyAppointments: find customer: %w", err)
	}
	if customer == nil {
		return []*domain.PublicAppointment{}, nil
	}

	appts, err := s.apptSvc.ListUpcomingByCustomer(ctx, tenant.ID, customer.ID)
	if err != nil {
		return nil, fmt.Errorf("publicSvc.ListMyAppointments: list: %w", err)
	}

	result := make([]*domain.PublicAppointment, len(appts))
	for i, a := range appts {
		result[i] = &domain.PublicAppointment{
			ID:               a.ID,
			ServiceID:        a.ServiceID,
			ServiceName:      a.ServiceName,
			ProfessionalID:   a.ProfessionalID,
			ProfessionalName: a.ProfessionalName,
			StartsAt:         a.StartsAt,
			EndsAt:           a.EndsAt,
			Status:           a.Status,
		}
	}
	return result, nil
}

// CancelAppointment cancela una cita verificando propiedad por teléfono.
func (s *publicSvc) CancelAppointment(ctx context.Context, slug string, appointmentID uuid.UUID, phone string) error {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return err
	}

	phone = domain.NormalizePhone(phone)

	// Obtener la cita
	appt, err := s.apptSvc.GetByID(ctx, tenant.ID, appointmentID)
	if err != nil {
		return err
	}

	// Verificar propiedad por teléfono
	customer, err := s.customerRepo.FindByPhone(ctx, tenant.ID, phone)
	if err != nil {
		return fmt.Errorf("publicSvc.CancelAppointment: find customer: %w", err)
	}
	if customer == nil || customer.ID != appt.CustomerID {
		return domain.ErrForbidden
	}

	// Verificar estado válido para cancelar
	if appt.Status != "pending" && appt.Status != "confirmed" {
		return fmt.Errorf("solo se pueden cancelar citas pendientes o confirmadas: %w", domain.ErrValidation)
	}

	return s.apptSvc.Cancel(ctx, tenant.ID, appointmentID, "cancelado por el cliente")
}

// RescheduleAppointment reagenda una cita verificando propiedad por teléfono.
func (s *publicSvc) RescheduleAppointment(ctx context.Context, slug string, appointmentID uuid.UUID, phone string, startsAt time.Time) error {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return err
	}

	phone = domain.NormalizePhone(phone)

	// Obtener la cita con detalles (necesitamos el service_duration para calcular ends_at)
	appt, err := s.apptSvc.GetByID(ctx, tenant.ID, appointmentID)
	if err != nil {
		return err
	}

	// Verificar propiedad por teléfono
	customer, err := s.customerRepo.FindByPhone(ctx, tenant.ID, phone)
	if err != nil {
		return fmt.Errorf("publicSvc.RescheduleAppointment: find customer: %w", err)
	}
	if customer == nil || customer.ID != appt.CustomerID {
		return domain.ErrForbidden
	}

	// Verificar estado válido para reagendar
	if appt.Status != "pending" && appt.Status != "confirmed" {
		return fmt.Errorf("solo se pueden reagendar citas pendientes o confirmadas: %w", domain.ErrValidation)
	}

	return s.apptSvc.Reschedule(ctx, tenant.ID, appointmentID, &domain.RescheduleRequest{
		StartsAt: startsAt,
	})
}

// ListMyTreatments retorna treatments activos del cliente por teléfono.
// Bloquea con ErrForbidden si el tenant no tiene módulo dental activo —
// alineado con el gating del Plane #31 para que la IA no exponga info de
// tenants no dentales.
func (s *publicSvc) ListMyTreatments(ctx context.Context, slug, phone string) ([]*domain.CustomerTreatmentSummary, error) {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	active, err := s.moduleRepo.IsActive(ctx, tenant.ID, domain.ModuleDental)
	if err != nil {
		return nil, fmt.Errorf("publicSvc.ListMyTreatments: module check: %w", err)
	}
	if !active {
		return nil, domain.ErrForbidden
	}

	phone = domain.NormalizePhone(phone)
	if phone == "" {
		return []*domain.CustomerTreatmentSummary{}, nil
	}

	return s.treatmentRepo.ListByCustomerPhonePublic(ctx, tenant.ID, phone)
}
