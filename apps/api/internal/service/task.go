package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type taskSvc struct {
	repo domain.TaskRepository
}

func NewTaskSvc(repo domain.TaskRepository) domain.TaskSvc {
	return &taskSvc{repo: repo}
}

func (s *taskSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.TaskInput) (*domain.Task, error) {
	t := &domain.Task{
		ID:            uuid.New(),
		TenantID:      tenantID,
		AssignedTo:    input.AssignedTo,
		CustomerID:    input.CustomerID,
		AppointmentID: input.AppointmentID,
		TreatmentID:   input.TreatmentID,
		Title:         input.Title,
		Description:   input.Description,
		Status:        "pending",
		DueAt:         input.DueAt,
		Source:        "manual",
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("taskSvc.Create: %w", err)
	}
	return t, nil
}

func (s *taskSvc) List(ctx context.Context, tenantID uuid.UUID, q *domain.TaskListQuery) ([]*domain.Task, error) {
	return s.repo.List(ctx, tenantID, q)
}

func (s *taskSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *taskSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.TaskInput) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	t.Title = input.Title
	if input.Description != "" {
		t.Description = input.Description
	}
	t.AssignedTo = input.AssignedTo
	t.CustomerID = input.CustomerID
	t.AppointmentID = input.AppointmentID
	t.TreatmentID = input.TreatmentID
	t.DueAt = input.DueAt

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("taskSvc.Update: %w", err)
	}
	return t, nil
}

func (s *taskSvc) Complete(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if err := s.repo.Complete(ctx, tenantID, id); err != nil {
		return nil, fmt.Errorf("taskSvc.Complete: %w", err)
	}
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *taskSvc) Dismiss(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if err := s.repo.Dismiss(ctx, tenantID, id); err != nil {
		return nil, fmt.Errorf("taskSvc.Dismiss: %w", err)
	}
	return s.repo.GetByID(ctx, tenantID, id)
}
