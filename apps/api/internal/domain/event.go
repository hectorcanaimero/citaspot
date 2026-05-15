package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event representa un evento operacional persistido para analytics.
// La inserción es best-effort: si falla, se loguea y la operación de negocio continúa.
type Event struct {
	TenantID   uuid.UUID      `json:"tenant_id"`
	EventType  string         `json:"event_type"`
	ActorType  string         `json:"actor_type"`
	ActorID    *uuid.UUID     `json:"actor_id,omitempty"`
	EntityType string         `json:"entity_type"`
	EntityID   uuid.UUID      `json:"entity_id"`
	Payload    map[string]any `json:"payload,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// EventRepository persiste eventos analíticos en la tabla events.
type EventRepository interface {
	// Insert persiste un evento individual. Best-effort — el caller loguea y continúa si falla.
	Insert(ctx context.Context, event Event) error
	// InsertBatch persiste múltiples eventos en una sola operación.
	InsertBatch(ctx context.Context, events []Event) error
}
