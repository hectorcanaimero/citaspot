package service

import (
	"context"
	"fmt"

	"github.com/citaspot/api/internal/domain"
)

type publicSvc struct {
	authRepo     domain.AuthRepository
	profRepo     domain.ProfessionalRepository
	serviceRepo  domain.ServiceRepository
	availSvc     domain.AvailabilityService
	apptSvc      domain.AppointmentSvc
	customerRepo domain.CustomerRepository
}

// NewPublicSvc crea el servicio de booking público (sin auth).
func NewPublicSvc(
	authRepo domain.AuthRepository,
	profRepo domain.ProfessionalRepository,
	serviceRepo domain.ServiceRepository,
	availSvc domain.AvailabilityService,
	apptSvc domain.AppointmentSvc,
	customerRepo domain.CustomerRepository,
) domain.PublicSvc {
	return &publicSvc{
		authRepo:     authRepo,
		profRepo:     profRepo,
		serviceRepo:  serviceRepo,
		availSvc:     availSvc,
		apptSvc:      apptSvc,
		customerRepo: customerRepo,
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

	return &domain.PublicProfile{
		Slug:               tenant.Slug,
		Name:               tenant.Name,
		BusinessType:       tenant.BusinessType,
		City:               tenant.City,
		Country:            tenant.Country,
		Timezone:           tenant.Timezone,
		Services:           services,
		Professionals:      professionals,
		BookingIntroText:   tenant.Settings.BookingIntroText,
		BookingSuccessText: tenant.Settings.BookingSuccessText,
		BotName:            tenant.Settings.BotName,
		BotGreeting:        tenant.Settings.BotGreeting,
	}, nil
}

// GetAvailability retorna slots disponibles para booking público.
func (s *publicSvc) GetAvailability(ctx context.Context, slug string, query *domain.AvailabilityQuery) ([]*domain.TimeSlot, error) {
	tenant, err := s.authRepo.FindTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Usar el timezone del tenant si no se especifica
	if query.Timezone == "" {
		query.Timezone = tenant.Timezone
	}

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
