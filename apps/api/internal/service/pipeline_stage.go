package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type pipelineStageSvc struct {
	repo domain.PipelineStageRepository
}

func NewPipelineStageSvc(repo domain.PipelineStageRepository) domain.PipelineStageSvc {
	return &pipelineStageSvc{repo: repo}
}

func (s *pipelineStageSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.PipelineStageInput) (*domain.PipelineStage, error) {
	color := input.Color
	if color == "" {
		color = "#6366f1"
	}
	isDefault := false
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}
	autoRules := true
	if input.AutoRulesEnabled != nil {
		autoRules = *input.AutoRulesEnabled
	}

	stage := &domain.PipelineStage{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             input.Name,
		Position:         input.Position,
		Color:            color,
		IsDefault:        isDefault,
		AutoRulesEnabled: autoRules,
	}

	if err := s.repo.Create(ctx, stage); err != nil {
		return nil, fmt.Errorf("pipelineStageSvc.Create: %w", err)
	}
	return stage, nil
}

func (s *pipelineStageSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.PipelineStage, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *pipelineStageSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PipelineStage, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *pipelineStageSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.PipelineStageInput) (*domain.PipelineStage, error) {
	stage, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		stage.Name = input.Name
	}
	stage.Position = input.Position
	if input.Color != "" {
		stage.Color = input.Color
	}
	if input.IsDefault != nil {
		stage.IsDefault = *input.IsDefault
	}
	if input.AutoRulesEnabled != nil {
		stage.AutoRulesEnabled = *input.AutoRulesEnabled
	}

	if err := s.repo.Update(ctx, stage); err != nil {
		return nil, fmt.Errorf("pipelineStageSvc.Update: %w", err)
	}
	return stage, nil
}

func (s *pipelineStageSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *pipelineStageSvc) Reorder(ctx context.Context, tenantID uuid.UUID, items []domain.ReorderStageInput) error {
	if err := s.repo.Reorder(ctx, tenantID, items); err != nil {
		return fmt.Errorf("pipelineStageSvc.Reorder: %w", err)
	}
	return nil
}
