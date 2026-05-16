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

var validTreatmentTransitions = map[string][]string{
	"proposed":    {"accepted", "abandoned"},
	"accepted":    {"in_progress", "abandoned"},
	"in_progress": {"completed", "abandoned"},
	"completed":   {},
	"abandoned":   {"proposed"},
}

type treatmentSvc struct {
	repo      domain.TreatmentRepository
	publisher domain.MessagePublisher
	events    domain.EventRepository
}

func NewTreatmentSvc(repo domain.TreatmentRepository, publisher domain.MessagePublisher, events domain.EventRepository) domain.TreatmentSvc {
	return &treatmentSvc{repo: repo, publisher: publisher, events: events}
}

func (s *treatmentSvc) persistEvent(ctx context.Context, event domain.Event) {
	if s.events == nil {
		return
	}
	if err := s.events.Insert(ctx, event); err != nil {
		slog.Warn("treatmentSvc.persistEvent: failed", "event_type", event.EventType, "entity_id", event.EntityID, "error", err)
	}
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

	s.emitTreatmentEvent(ctx, tenantID, t, "proposed")
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

	s.emitTreatmentEvent(ctx, tenantID, t, input.Status)

	return s.repo.GetByID(ctx, tenantID, id)
}

// publishRuleEvent publica un evento de dominio en la cola rules.events.
func (s *treatmentSvc) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("treatmentSvc.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("treatmentSvc.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}

// emitTreatmentEvent emite un evento de tratamiento para el motor de reglas.
func (s *treatmentSvc) emitTreatmentEvent(ctx context.Context, tenantID uuid.UUID, t *domain.Treatment, newStatus string) {
	eventType := ""
	switch newStatus {
	case "proposed":
		eventType = "treatment.proposed"
	case "accepted":
		eventType = "treatment.accepted"
	case "completed":
		eventType = "treatment.completed"
	case "abandoned":
		eventType = "treatment.abandoned"
	default:
		return
	}

	// Persist analytics event — map "proposed" to "treatment.created" for analytics
	analyticsType := eventType
	if newStatus == "proposed" {
		analyticsType = "treatment.created"
	}
	profID := t.ProfessionalID
	s.persistEvent(ctx, domain.Event{
		TenantID:   tenantID,
		EventType:  analyticsType,
		ActorType:  "professional",
		ActorID:    &profID,
		EntityType: "treatment",
		EntityID:   t.ID,
		Payload: map[string]any{
			"customer_id": t.CustomerID.String(),
			"plan_type":   t.TreatmentType,
		},
		OccurredAt: time.Now(),
	})

	previousStatus := t.Status
	if previousStatus == newStatus {
		// Caso creación: t.Status ya quedó seteado a newStatus en Create antes de emitir.
		previousStatus = ""
	}

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  eventType,
		CustomerID: t.CustomerID,
		EntityID:   t.ID,
		EntityType: "treatment",
		Payload: map[string]any{
			"treatment_id":    t.ID.String(),
			"customer_id":     t.CustomerID.String(),
			"professional_id": t.ProfessionalID.String(),
			"treatment_type":  t.TreatmentType,
			"name":            t.Name,
			"status":          newStatus,
			"previous_status": previousStatus,
		},
		Timestamp: time.Now(),
	})
}
