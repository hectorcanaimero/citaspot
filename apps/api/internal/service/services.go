package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type serviceSvc struct {
	repo domain.ServiceRepository
}

// NewServiceSvc crea el servicio de servicios del negocio.
func NewServiceSvc(repo domain.ServiceRepository) domain.ServiceSvc {
	return &serviceSvc{repo: repo}
}

// List retorna todos los servicios del tenant.
func (s *serviceSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Service, error) {
	return s.repo.List(ctx, tenantID)
}

// Create crea un nuevo servicio.
func (s *serviceSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.ServiceInput) (*domain.Service, error) {
	currency := input.Currency
	if currency == "" {
		currency = "USD"
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	svc := &domain.Service{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        input.Name,
		Description: input.Description,
		DurationMin: input.DurationMin,
		Price:       input.Price,
		Currency:    currency,
		BufferMin:   input.BufferMin,
		IsActive:    isActive,
		SortOrder:   input.SortOrder,
	}

	if err := s.repo.Create(ctx, svc); err != nil {
		return nil, fmt.Errorf("serviceSvc.Create: %w", err)
	}
	return svc, nil
}

// GetByID retorna un servicio por ID.
func (s *serviceSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Service, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Update actualiza un servicio existente.
func (s *serviceSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.ServiceInput) (*domain.Service, error) {
	svc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		svc.Name = input.Name
	}
	if input.Description != "" {
		svc.Description = input.Description
	}
	if input.DurationMin > 0 {
		svc.DurationMin = input.DurationMin
	}
	if input.Price != nil {
		svc.Price = input.Price
	}
	if input.Currency != "" {
		svc.Currency = input.Currency
	}
	if input.BufferMin >= 0 {
		svc.BufferMin = input.BufferMin
	}
	if input.IsActive != nil {
		svc.IsActive = *input.IsActive
	}
	svc.SortOrder = input.SortOrder

	if err := s.repo.Update(ctx, svc); err != nil {
		return nil, fmt.Errorf("serviceSvc.Update: %w", err)
	}
	return svc, nil
}
