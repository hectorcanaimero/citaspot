package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type eventRepository struct {
	db *pgxpool.Pool
}

// NewEventRepository crea el repositorio de eventos analíticos.
func NewEventRepository(db *pgxpool.Pool) domain.EventRepository {
	return &eventRepository{db: db}
}

// Insert persiste un evento individual con RLS via withTenant.
func (r *eventRepository) Insert(ctx context.Context, event domain.Event) error {
	if r.db == nil {
		return fmt.Errorf("eventRepository.Insert: pool is nil")
	}

	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("eventRepository.Insert: marshal payload: %w", err)
	}

	return withTenant(ctx, r.db, event.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO events (tenant_id, event_type, actor_type, actor_id, entity_type, entity_id, payload, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, event.TenantID, event.EventType, event.ActorType, event.ActorID,
			event.EntityType, event.EntityID, payload, event.OccurredAt)
		if err != nil {
			return fmt.Errorf("eventRepository.Insert: exec: %w", err)
		}
		return nil
	})
}

// InsertBatch persiste múltiples eventos en una sola transacción.
func (r *eventRepository) InsertBatch(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	if r.db == nil {
		return fmt.Errorf("eventRepository.InsertBatch: pool is nil")
	}

	// Todos los eventos del batch deben pertenecer al mismo tenant.
	tenantID := events[0].TenantID

	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		for i, event := range events {
			if event.OccurredAt.IsZero() {
				event.OccurredAt = time.Now()
			}
			payload, err := json.Marshal(event.Payload)
			if err != nil {
				return fmt.Errorf("eventRepository.InsertBatch[%d]: marshal: %w", i, err)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO events (tenant_id, event_type, actor_type, actor_id, entity_type, entity_id, payload, occurred_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, event.TenantID, event.EventType, event.ActorType, event.ActorID,
				event.EntityType, event.EntityID, payload, event.OccurredAt)
			if err != nil {
				return fmt.Errorf("eventRepository.InsertBatch[%d]: exec: %w", i, err)
			}
		}
		return nil
	})
}
