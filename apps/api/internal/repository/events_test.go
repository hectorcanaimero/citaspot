package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

func TestEventRepository_Insert_NilPool(t *testing.T) {
	// Verifica que NewEventRepository retorna implementación válida
	// y que Insert con pool nil retorna error (no panic).
	repo := NewEventRepository(nil)
	if repo == nil {
		t.Fatal("NewEventRepository returned nil")
	}

	event := domain.Event{
		TenantID:   uuid.New(),
		EventType:  "appointment.created",
		ActorType:  "customer",
		EntityType: "appointment",
		EntityID:   uuid.New(),
		Payload:    map[string]any{"source": "web"},
		OccurredAt: time.Now(),
	}

	err := repo.Insert(context.Background(), event)
	if err == nil {
		t.Fatal("expected error with nil pool, got nil")
	}
}

func TestEventRepository_InsertBatch_EmptySlice(t *testing.T) {
	repo := NewEventRepository(nil)

	// Batch vacío no debería intentar la DB.
	err := repo.InsertBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected nil error for empty batch, got: %v", err)
	}
}
