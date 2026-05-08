package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

var validTreatmentTransitions = map[string][]string{
	"proposed":    {"accepted", "abandoned"},
	"accepted":    {"in_progress", "abandoned"},
	"in_progress": {"completed", "abandoned"},
	"completed":   {},
	"abandoned":   {"proposed"},
}

type treatmentSvc struct {
	repo domain.TreatmentRepository
}

func NewTreatmentSvc(repo domain.TreatmentRepository) domain.TreatmentSvc {
	return &treatmentSvc{repo: repo}
}

func (s *treatmentSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.TreatmentInput) (*domain.Treatment, error) {
	currency := input.Currency
	if currency == "" {
		currency = "USD"
	}

	t := &domain.Treatment{
		ID:             uuid.New(),
		TenantID:       tenantID,
		CustomerID:     input.CustomerID,
		ProfessionalID: input.ProfessionalID,
		Name:           input.Name,
		TreatmentType:  input.TreatmentType,
		Status:         "proposed",
		TotalSessions:  input.TotalSessions,
		EstimatedCost:  input.EstimatedCost,
		Currency:       currency,
		ToothNumbers:   input.ToothNumbers,
		Notes:          input.Notes,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("treatmentSvc.Create: %w", err)
	}
	return t, nil
}

func (s *treatmentSvc) List(ctx context.Context, tenantID uuid.UUID, q *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
	return s.repo.List(ctx, tenantID, q)
}

func (s *treatmentSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Treatment, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *treatmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.TreatmentInput) (*domain.Treatment, error) {
	t, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		t.Name = input.Name
	}
	if input.TreatmentType != "" {
		t.TreatmentType = input.TreatmentType
	}
	if input.TotalSessions != nil {
		t.TotalSessions = input.TotalSessions
	}
	if input.EstimatedCost != nil {
		t.EstimatedCost = input.EstimatedCost
	}
	if input.Currency != "" {
		t.Currency = input.Currency
	}
	if input.ToothNumbers != nil {
		t.ToothNumbers = input.ToothNumbers
	}
	if input.Notes != "" {
		t.Notes = input.Notes
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("treatmentSvc.Update: %w", err)
	}
	return t, nil
}

func (s *treatmentSvc) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateTreatmentStatusInput) (*domain.Treatment, error) {
	t, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	allowed, ok := validTreatmentTransitions[t.Status]
	if !ok {
		return nil, fmt.Errorf("treatmentSvc.UpdateStatus: estado actual '%s' desconocido: %w", t.Status, domain.ErrInvalidStatusTransition)
	}
	valid := false
	for _, st := range allowed {
		if st == input.Status {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("treatmentSvc.UpdateStatus: no se puede pasar de '%s' a '%s': %w", t.Status, input.Status, domain.ErrInvalidStatusTransition)
	}

	if err := s.repo.UpdateStatus(ctx, tenantID, id, input.Status); err != nil {
		return nil, fmt.Errorf("treatmentSvc.UpdateStatus: %w", err)
	}

	return s.repo.GetByID(ctx, tenantID, id)
}
