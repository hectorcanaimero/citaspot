package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type professionalService struct {
	profRepo     domain.ProfessionalRepository
	scheduleRepo domain.ScheduleRepository
}

// NewProfessionalService crea el servicio de profesionales.
func NewProfessionalService(profRepo domain.ProfessionalRepository, scheduleRepo domain.ScheduleRepository) domain.ProfessionalSvc {
	return &professionalService{
		profRepo:     profRepo,
		scheduleRepo: scheduleRepo,
	}
}

// List retorna todos los profesionales del tenant.
func (s *professionalService) List(ctx context.Context, tenantID uuid.UUID, includeArchived bool) ([]*domain.Professional, error) {
	return s.profRepo.List(ctx, tenantID, includeArchived)
}

// Create crea un nuevo profesional.
func (s *professionalService) Create(ctx context.Context, tenantID uuid.UUID, input *domain.ProfessionalInput) (*domain.Professional, error) {
	color := input.Color
	if color == "" {
		color = "#3B82F6" // azul por defecto
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	p := &domain.Professional{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      input.Name,
		Specialty: input.Specialty,
		Bio:       input.Bio,
		AvatarURL: input.AvatarURL,
		Color:     color,
		IsActive:  isActive,
	}

	if err := s.profRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("professionalService.Create: %w", err)
	}
	return p, nil
}

// GetByID retorna un profesional por ID.
func (s *professionalService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Professional, error) {
	return s.profRepo.GetByID(ctx, tenantID, id)
}

// Update actualiza los datos de un profesional.
func (s *professionalService) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.ProfessionalInput) (*domain.Professional, error) {
	p, err := s.profRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		p.Name = input.Name
	}
	if input.Specialty != "" {
		p.Specialty = input.Specialty
	}
	if input.Bio != "" {
		p.Bio = input.Bio
	}
	if input.AvatarURL != "" {
		p.AvatarURL = input.AvatarURL
	}
	if input.Color != "" {
		p.Color = input.Color
	}
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}
	if input.IsArchived != nil {
		p.IsArchived = *input.IsArchived
	}

	if err := s.profRepo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("professionalService.Update: %w", err)
	}
	return p, nil
}

// GetSchedule retorna los horarios semanales de un profesional.
func (s *professionalService) GetSchedule(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Schedule, error) {
	// Validar que el profesional pertenece al tenant
	if _, err := s.profRepo.GetByID(ctx, tenantID, professionalID); err != nil {
		return nil, err
	}
	return s.scheduleRepo.GetSchedules(ctx, tenantID, professionalID)
}

// SetSchedule reemplaza los horarios semanales de un profesional.
func (s *professionalService) SetSchedule(ctx context.Context, tenantID, professionalID uuid.UUID, schedules []*domain.Schedule) ([]*domain.Schedule, error) {
	// Validar que el profesional pertenece al tenant
	if _, err := s.profRepo.GetByID(ctx, tenantID, professionalID); err != nil {
		return nil, err
	}

	// Validar cada día
	for _, sch := range schedules {
		if sch.DayOfWeek < 0 || sch.DayOfWeek > 6 {
			return nil, fmt.Errorf("%w: day_of_week debe estar entre 0 (Dom) y 6 (Sáb)", domain.ErrValidation)
		}
		if sch.StartTime >= sch.EndTime {
			return nil, fmt.Errorf("%w: start_time debe ser anterior a end_time", domain.ErrValidation)
		}
	}

	return s.scheduleRepo.UpsertSchedules(ctx, tenantID, professionalID, schedules)
}
