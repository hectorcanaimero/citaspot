package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type ruleSvc struct {
	repo domain.RuleRepository
}

func NewRuleSvc(repo domain.RuleRepository) domain.RuleSvc {
	return &ruleSvc{repo: repo}
}

func (s *ruleSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	cooldown := input.CooldownHours
	if cooldown == 0 {
		cooldown = 24
	}

	if input.TriggerType == "event" && input.TriggerEvent == "" {
		return nil, fmt.Errorf("ruleSvc.Create: trigger_event es requerido para reglas tipo 'event': %w", domain.ErrValidation)
	}
	if input.TriggerType == "temporal" && input.TriggerSchedule == nil {
		return nil, fmt.Errorf("ruleSvc.Create: trigger_schedule es requerido para reglas tipo 'temporal': %w", domain.ErrValidation)
	}

	rule := &domain.Rule{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            input.Name,
		Description:     input.Description,
		TriggerType:     input.TriggerType,
		TriggerEvent:    input.TriggerEvent,
		TriggerSchedule: input.TriggerSchedule,
		Conditions:      input.Conditions,
		Actions:         input.Actions,
		IsActive:        isActive,
		IsTemplate:      false,
		CooldownHours:   cooldown,
		Priority:        input.Priority,
	}
	if rule.Conditions == nil {
		rule.Conditions = []domain.RuleCondition{}
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleSvc.Create: %w", err)
	}
	return rule, nil
}

func (s *ruleSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *ruleSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Rule, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *ruleSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
	rule, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		rule.Name = input.Name
	}
	rule.Description = input.Description
	rule.TriggerType = input.TriggerType
	rule.TriggerEvent = input.TriggerEvent
	rule.TriggerSchedule = input.TriggerSchedule
	if input.Conditions != nil {
		rule.Conditions = input.Conditions
	}
	if input.Actions != nil {
		rule.Actions = input.Actions
	}
	if input.IsActive != nil {
		rule.IsActive = *input.IsActive
	}
	if input.CooldownHours > 0 {
		rule.CooldownHours = input.CooldownHours
	}
	rule.Priority = input.Priority

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleSvc.Update: %w", err)
	}
	return rule, nil
}

func (s *ruleSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}
