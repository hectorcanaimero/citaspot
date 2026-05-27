package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type professionalService struct {
	profRepo     domain.ProfessionalRepository
	scheduleRepo domain.ScheduleRepository
	serviceRepo  domain.ServiceRepository
	apptRepo     domain.AppointmentRepository
	publisher    domain.MessagePublisher
}

// NewProfessionalService crea el servicio de profesionales.
func NewProfessionalService(
	profRepo domain.ProfessionalRepository,
	scheduleRepo domain.ScheduleRepository,
	serviceRepo domain.ServiceRepository,
	apptRepo domain.AppointmentRepository,
	publisher domain.MessagePublisher,
) domain.ProfessionalSvc {
	return &professionalService{
		profRepo:     profRepo,
		scheduleRepo: scheduleRepo,
		serviceRepo:  serviceRepo,
		apptRepo:     apptRepo,
		publisher:    publisher,
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
		Phone:     input.Phone,
		Email:     input.Email,
		Color:     color,
		IsActive:  isActive,
	}

	if err := s.profRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("professionalService.Create: %w", err)
	}

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  "professional.created",
		CustomerID: uuid.Nil,
		EntityID:   p.ID,
		EntityType: "professional",
		Payload: map[string]any{
			"professional_id": p.ID.String(),
			"name":            p.Name,
			"specialty":       p.Specialty,
		},
		Timestamp: time.Now(),
	})

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

	wasArchived := p.IsArchived

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
	if input.Phone != "" {
		p.Phone = input.Phone
	}
	if input.Email != "" {
		p.Email = input.Email
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

	if !wasArchived && p.IsArchived {
		s.publishRuleEvent(ctx, domain.RuleEvent{
			TenantID:   tenantID,
			EventType:  "professional.archived",
			CustomerID: uuid.Nil,
			EntityID:   p.ID,
			EntityType: "professional",
			Payload: map[string]any{
				"professional_id": p.ID.String(),
				"name":            p.Name,
			},
			Timestamp: time.Now(),
		})
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

// ListServices retorna los servicios asignados a un profesional.
func (s *professionalService) ListServices(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Service, error) {
	if _, err := s.profRepo.GetByID(ctx, tenantID, professionalID); err != nil {
		return nil, err
	}
	return s.profRepo.ListServices(ctx, tenantID, professionalID)
}

// AssignService asigna un servicio a un profesional.
func (s *professionalService) AssignService(ctx context.Context, tenantID, professionalID, serviceID uuid.UUID) error {
	if err := s.profRepo.AssignService(ctx, tenantID, professionalID, serviceID); err != nil {
		return fmt.Errorf("professionalService.AssignService: %w", err)
	}
	return nil
}

// RemoveService elimina la asignación de un servicio a un profesional.
func (s *professionalService) RemoveService(ctx context.Context, tenantID, professionalID, serviceID uuid.UUID) error {
	if err := s.profRepo.RemoveService(ctx, tenantID, professionalID, serviceID); err != nil {
		return fmt.Errorf("professionalService.RemoveService: %w", err)
	}
	return nil
}

// ListCustomers retorna los clientes únicos atendidos por el profesional
// (al menos una appointment con status='completed'). Valida primero que el
// profesional pertenezca al tenant.
func (s *professionalService) ListCustomers(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Customer, error) {
	if _, err := s.profRepo.GetByID(ctx, tenantID, professionalID); err != nil {
		return nil, err
	}
	customers, err := s.apptRepo.ListDistinctCustomersByProfessional(ctx, tenantID, professionalID)
	if err != nil {
		return nil, fmt.Errorf("professionalService.ListCustomers: %w", err)
	}
	return customers, nil
}

// AutoAssignAllServicesToFirstProfessional asigna todos los servicios activos al primer
// profesional (orden alfabético via profRepo.List). Sin error si no hay profesionales o servicios.
func (s *professionalService) AutoAssignAllServicesToFirstProfessional(
	ctx context.Context, tenantID uuid.UUID,
) (int, error) {
	profs, err := s.profRepo.List(ctx, tenantID, false)
	if err != nil {
		return 0, fmt.Errorf("professionalService.AutoAssignAllServicesToFirstProfessional: list profs: %w", err)
	}
	if len(profs) == 0 {
		return 0, nil
	}
	services, err := s.serviceRepo.ListActive(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("professionalService.AutoAssignAllServicesToFirstProfessional: list services: %w", err)
	}
	if len(services) == 0 {
		return 0, nil
	}
	ids := make([]uuid.UUID, len(services))
	for i, svc := range services {
		ids[i] = svc.ID
	}
	return s.profRepo.BulkAssignServicesToProfessional(ctx, tenantID, profs[0].ID, ids)
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

// publishRuleEvent publica un evento de regla a RabbitMQ.
func (s *professionalService) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("professionalService.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("professionalService.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
