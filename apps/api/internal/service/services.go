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

type serviceSvc struct {
	repo      domain.ServiceRepository
	publisher domain.MessagePublisher
}

// NewServiceSvc crea el servicio de servicios del negocio.
func NewServiceSvc(repo domain.ServiceRepository, publisher domain.MessagePublisher) domain.ServiceSvc {
	return &serviceSvc{repo: repo, publisher: publisher}
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

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  "service.created",
		CustomerID: uuid.Nil,
		EntityID:   svc.ID,
		EntityType: "service",
		Payload: map[string]any{
			"service_id":       svc.ID.String(),
			"name":             svc.Name,
			"price":            svc.Price,
			"duration_minutes": svc.DurationMin,
		},
		Timestamp: time.Now(),
	})

	return svc, nil
}

// GetByID retorna un servicio por ID.
func (s *serviceSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Service, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Delete elimina permanentemente un servicio.
func (s *serviceSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("serviceSvc.Delete: %w", err)
	}
	return nil
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

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  "service.updated",
		CustomerID: uuid.Nil,
		EntityID:   svc.ID,
		EntityType: "service",
		Payload: map[string]any{
			"service_id":       svc.ID.String(),
			"name":             svc.Name,
			"price":            svc.Price,
			"duration_minutes": svc.DurationMin,
		},
		Timestamp: time.Now(),
	})

	return svc, nil
}

// publishRuleEvent publica eventos de reglas a la cola rules.events.
func (s *serviceSvc) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("serviceSvc.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("serviceSvc.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
