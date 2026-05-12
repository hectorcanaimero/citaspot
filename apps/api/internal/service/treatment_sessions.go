package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type treatmentSessionSvc struct {
	repo          domain.TreatmentSessionRepository
	treatmentRepo domain.TreatmentRepository
}

// NewTreatmentSessionSvc crea el servicio de sesiones de tratamiento.
func NewTreatmentSessionSvc(repo domain.TreatmentSessionRepository, treatmentRepo domain.TreatmentRepository) domain.TreatmentSessionSvc {
	return &treatmentSessionSvc{repo: repo, treatmentRepo: treatmentRepo}
}

func (s *treatmentSessionSvc) Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *domain.TreatmentSessionInput) (*domain.TreatmentSession, error) {
	// Validar que el treatment pertenece al tenant
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return nil, fmt.Errorf("treatmentSessionSvc.Create: %w", err)
	}
	return s.repo.Create(ctx, tenantID, treatmentID, input)
}

func (s *treatmentSessionSvc) List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*domain.TreatmentSession, error) {
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return nil, fmt.Errorf("treatmentSessionSvc.List: %w", err)
	}
	return s.repo.List(ctx, tenantID, treatmentID)
}

func (s *treatmentSessionSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TreatmentSession, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *treatmentSessionSvc) Update(ctx context.Context, tenantID, treatmentID, id uuid.UUID, input *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error) {
	// Validar que el treatment pertenece al tenant
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return nil, fmt.Errorf("treatmentSessionSvc.Update: %w", err)
	}
	return s.repo.Update(ctx, tenantID, id, input)
}

func (s *treatmentSessionSvc) Delete(ctx context.Context, tenantID, treatmentID, id uuid.UUID) error {
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return fmt.Errorf("treatmentSessionSvc.Delete: %w", err)
	}
	return s.repo.Delete(ctx, tenantID, id)
}
