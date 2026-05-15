# Implementation Plan: Event Analytics System

> **Spec:** `docs/superpowers/specs/2026-05-14-event-analytics-design.md`
> **Date:** 2026-05-14
> **Next migration:** 032
> **Estimated tasks:** 7

---

## Task 1: Migration 032 — Create events table with partitioning and RLS

### 1.1 Write the migration file

**File:** `apps/api/db/migrations/032_create_events_table.sql`

```sql
-- Tabla de eventos analíticos particionada por mes.
-- Best-effort: si falla la inserción, no rompe la operación de negocio.

CREATE TABLE events (
    id          UUID DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    event_type  TEXT NOT NULL,
    actor_type  TEXT NOT NULL,
    actor_id    UUID,
    entity_type TEXT NOT NULL,
    entity_id   UUID NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);

-- Particiones: mes actual + 2 meses siguientes
CREATE TABLE events_y2026m05 PARTITION OF events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE events_y2026m06 PARTITION OF events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE events_y2026m07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

-- Indices (propagados automáticamente a cada partición)
CREATE INDEX idx_events_tenant_time
    ON events (tenant_id, occurred_at DESC);

CREATE INDEX idx_events_tenant_type_time
    ON events (tenant_id, event_type, occurred_at DESC);

CREATE INDEX idx_events_tenant_entity
    ON events (tenant_id, entity_type, entity_id);

CREATE INDEX idx_events_payload
    ON events USING GIN (payload);

-- RLS
ALTER TABLE events ENABLE ROW LEVEL SECURITY;

CREATE POLICY events_tenant_isolation ON events
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 1.2 Apply migration

```bash
cd /Users/al3jandro/project/agendAI && make migrate
```

**Expected:** Migration 032 applied successfully. Verify with:

```bash
docker compose exec postgres psql -U citaspot -d citaspot -c "\dt events*"
```

**Expected output:** `events`, `events_y2026m05`, `events_y2026m06`, `events_y2026m07` listed.

### 1.3 Commit

```
feat(db): add events table with monthly partitioning and RLS
```

---

## Task 2: Domain layer — Event type and EventRepository interface

### 2.1 Create domain/event.go

**File:** `apps/api/internal/domain/event.go`

```go
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
```

### 2.2 Verify compilation

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./...
```

**Expected:** No errors.

### 2.3 Commit

```
feat(domain): add Event type and EventRepository interface
```

---

## Task 3: Repository layer — events.go with Insert and InsertBatch

### 3.1 Write failing test

**File:** `apps/api/internal/repository/events_test.go`

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/citaspot/api/internal/domain"
)

// mockPool implementa lo mínimo para validar la lógica del repositorio.
// Tests de integración reales corren contra la DB con build tag.

func TestEventRepository_Insert_CallsWithTenant(t *testing.T) {
	// Este test verifica que NewEventRepository retorna una implementación válida
	// y que Insert no paniquea con un pool nil (error esperado, no panic).
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

	// Con pool nil, esperamos un error (no un panic)
	err := repo.Insert(context.Background(), event)
	if err == nil {
		t.Fatal("expected error with nil pool, got nil")
	}
}

func TestEventRepository_InsertBatch_EmptySlice(t *testing.T) {
	repo := NewEventRepository(nil)

	// Batch vacío no debería intentar la DB
	err := repo.InsertBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected nil error for empty batch, got: %v", err)
	}
}
```

### 3.2 Run test — expect failure (Insert test fails because implementation doesn't exist yet)

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./internal/repository/ -run TestEventRepository -v
```

**Expected:** Compilation error — `NewEventRepository` undefined.

### 3.3 Implement repository

**File:** `apps/api/internal/repository/events.go`

```go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

	// Todos los eventos del batch deben pertenecer al mismo tenant
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
```

**IMPORTANT:** The `withTenant` helper is defined in `repository/professionals.go` and is package-level, so it's accessible from `events.go` in the same package. The import for `pgx` comes from `github.com/jackc/pgx/v5` — add it to the imports:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)
```

### 3.4 Run tests — expect pass

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./internal/repository/ -run TestEventRepository -v
```

**Expected:** Both tests pass. The nil pool test returns an error (not a panic). The empty batch test returns nil.

### 3.5 Commit

```
feat(repository): add EventRepository with Insert and InsertBatch
```

---

## Task 4: Integrate event persistence into appointmentSvc

### 4.1 Write failing test

**File:** `apps/api/internal/service/appointments_test.go` (add to existing or create)

Create a mock for EventRepository and verify that `emitAppointmentEvent` calls `events.Insert`:

```go
// mockEventRepository es un mock para verificar que los eventos se persisten.
type mockEventRepository struct {
	insertCalled bool
	lastEvent    domain.Event
	insertErr    error
}

func (m *mockEventRepository) Insert(_ context.Context, event domain.Event) error {
	m.insertCalled = true
	m.lastEvent = event
	return m.insertErr
}

func (m *mockEventRepository) InsertBatch(_ context.Context, _ []domain.Event) error {
	return nil
}
```

Test that Create calls events.Insert with event_type "appointment.created":

```go
func TestAppointmentSvc_Create_PersistsEvent(t *testing.T) {
	// ... setup mocks for all deps ...
	eventRepo := &mockEventRepository{}
	svc := NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, authRepo, nil, notifRepo, nil, eventRepo)

	// ... call Create ...

	if !eventRepo.insertCalled {
		t.Fatal("expected events.Insert to be called")
	}
	if eventRepo.lastEvent.EventType != "appointment.created" {
		t.Fatalf("expected event type appointment.created, got %s", eventRepo.lastEvent.EventType)
	}
}
```

Test that event failure does NOT propagate:

```go
func TestAppointmentSvc_Create_EventFailureDoesNotBreak(t *testing.T) {
	eventRepo := &mockEventRepository{insertErr: fmt.Errorf("db down")}
	svc := NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, authRepo, nil, notifRepo, nil, eventRepo)

	// ... call Create — should succeed despite event failure ...

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
```

### 4.2 Run test — expect failure

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./internal/service/ -run TestAppointmentSvc_Create_PersistsEvent -v
```

**Expected:** Compilation error — `NewAppointmentSvc` doesn't accept `eventRepo` parameter yet.

### 4.3 Modify appointmentSvc

**File:** `apps/api/internal/service/appointments.go`

**Changes:**

1. Add `events domain.EventRepository` field to `appointmentSvc` struct:

```go
type appointmentSvc struct {
	apptRepo     domain.AppointmentRepository
	serviceRepo  domain.ServiceRepository
	customerRepo domain.CustomerRepository
	authRepo     domain.AuthRepository
	waClient     domain.WAClient
	notifRepo    domain.NotificationRepository
	publisher    domain.MessagePublisher
	events       domain.EventRepository
}
```

2. Update constructor to accept `events domain.EventRepository`:

```go
func NewAppointmentSvc(
	apptRepo domain.AppointmentRepository,
	serviceRepo domain.ServiceRepository,
	customerRepo domain.CustomerRepository,
	authRepo domain.AuthRepository,
	waClient domain.WAClient,
	notifRepo domain.NotificationRepository,
	publisher domain.MessagePublisher,
	events domain.EventRepository,
) domain.AppointmentSvc {
	return &appointmentSvc{
		apptRepo:     apptRepo,
		serviceRepo:  serviceRepo,
		customerRepo: customerRepo,
		authRepo:     authRepo,
		waClient:     waClient,
		notifRepo:    notifRepo,
		publisher:    publisher,
		events:       events,
	}
}
```

3. Add a `persistEvent` helper method (best-effort, never returns error):

```go
// persistEvent persiste un evento analítico de forma best-effort.
// Si falla, loguea un warning y continúa — NUNCA rompe la operación de negocio.
func (s *appointmentSvc) persistEvent(ctx context.Context, event domain.Event) {
	if s.events == nil {
		return
	}
	if err := s.events.Insert(ctx, event); err != nil {
		slog.Warn("appointmentSvc.persistEvent: failed", "event_type", event.EventType, "entity_id", event.EntityID, "error", err)
	}
}
```

4. Add event persistence call inside `emitAppointmentEvent`, BEFORE the `publishRuleEvent` call. Insert after the `appt, err := s.apptRepo.GetByID(...)` block and before the tenant/timezone block:

```go
func (s *appointmentSvc) emitAppointmentEvent(ctx context.Context, tenantID, apptID uuid.UUID, trigger string) {
	eventType := ""
	switch trigger {
	case "created":
		eventType = "appointment.created"
	case "confirmed":
		eventType = "appointment.confirmed"
	case "completed":
		eventType = "appointment.completed"
	case "cancelled":
		eventType = "appointment.cancelled"
	case "no_show":
		eventType = "appointment.no_show"
	default:
		return
	}

	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil {
		slog.Warn("appointmentSvc.emitAppointmentEvent: get error", "error", err)
		return
	}

	// Persistir evento analítico (best-effort)
	s.persistEvent(ctx, domain.Event{
		TenantID:   tenantID,
		EventType:  eventType,
		ActorType:  "customer",
		ActorID:    &appt.CustomerID,
		EntityType: "appointment",
		EntityID:   apptID,
		Payload: map[string]any{
			"service_id":       appt.ServiceID.String(),
			"professional_id":  appt.ProfessionalID.String(),
			"starts_at":        appt.StartsAt.Format(time.RFC3339),
			"source":           appt.Source,
		},
		OccurredAt: time.Now(),
	})

	// Formatear fecha/hora en timezone del tenant para templates legibles
	tenant, _ := s.authRepo.FindTenantByID(ctx, tenantID)
	// ... rest of the method stays the same ...
```

**Note on actor_type mapping:** For `appointment.completed`, the actor_type should be `"professional"`. For `appointment.no_show`, it should be `"system"`. Adjust the `persistEvent` call to use a local `actorType` variable:

```go
	// Determinar actor_type según el trigger
	actorType := "customer"
	var actorID *uuid.UUID
	switch trigger {
	case "completed":
		actorType = "professional"
		actorID = &appt.ProfessionalID
	case "no_show":
		actorType = "system"
		actorID = nil
	default:
		actorID = &appt.CustomerID
	}

	// Payload específico por tipo de evento
	eventPayload := map[string]any{
		"service_id":      appt.ServiceID.String(),
		"professional_id": appt.ProfessionalID.String(),
		"starts_at":       appt.StartsAt.Format(time.RFC3339),
		"source":          appt.Source,
	}

	s.persistEvent(ctx, domain.Event{
		TenantID:   tenantID,
		EventType:  eventType,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "appointment",
		EntityID:   apptID,
		Payload:    eventPayload,
		OccurredAt: time.Now(),
	})
```

Also handle the `rescheduled` trigger. The `Reschedule` method currently calls `emitAppointmentEvent` with `"rescheduled"` but the switch doesn't have that case. Add it:

```go
	case "rescheduled":
		eventType = "appointment.rescheduled"
```

And the switch in the original already doesn't have "rescheduled" — this is a bug in existing code since it returns early via `default: return`. The rescheduled event is never published to RabbitMQ either. Fix this by adding the case to the existing switch.

### 4.4 Run tests — expect pass

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./internal/service/ -run TestAppointmentSvc -v
```

**Expected:** All tests pass.

### 4.5 Commit

```
feat(appointments): persist analytics events on appointment lifecycle
```

---

## Task 5: Integrate event persistence into treatmentSvc and whatsAppSvc

### 5.1 Modify treatmentSvc

**File:** `apps/api/internal/service/treatment.go`

**Changes:**

1. Add `events domain.EventRepository` field:

```go
type treatmentSvc struct {
	repo      domain.TreatmentRepository
	publisher domain.MessagePublisher
	events    domain.EventRepository
}
```

2. Update constructor:

```go
func NewTreatmentSvc(repo domain.TreatmentRepository, publisher domain.MessagePublisher, events domain.EventRepository) domain.TreatmentSvc {
	return &treatmentSvc{repo: repo, publisher: publisher, events: events}
}
```

3. Add `persistEvent` helper (same pattern as appointmentSvc):

```go
func (s *treatmentSvc) persistEvent(ctx context.Context, event domain.Event) {
	if s.events == nil {
		return
	}
	if err := s.events.Insert(ctx, event); err != nil {
		slog.Warn("treatmentSvc.persistEvent: failed", "event_type", event.EventType, "entity_id", event.EntityID, "error", err)
	}
}
```

4. Add event persistence in `emitTreatmentEvent`, after the switch block and before `publishRuleEvent`:

```go
func (s *treatmentSvc) emitTreatmentEvent(ctx context.Context, tenantID uuid.UUID, t *domain.Treatment, newStatus string) {
	eventType := ""
	switch newStatus {
	case "proposed":
		eventType = "treatment.proposed"
	case "accepted":
		eventType = "treatment.accepted"
	case "completed":
		eventType = "treatment.completed"
	default:
		return
	}

	// Persistir evento analítico: solo treatment.created (proposed) va al catálogo
	analyticsType := eventType
	if newStatus == "proposed" {
		analyticsType = "treatment.created"
	}

	s.persistEvent(ctx, domain.Event{
		TenantID:   tenantID,
		EventType:  analyticsType,
		ActorType:  "professional",
		ActorID:    &t.ProfessionalID,
		EntityType: "treatment",
		EntityID:   t.ID,
		Payload: map[string]any{
			"customer_id": t.CustomerID.String(),
			"plan_type":   t.TreatmentType,
		},
		OccurredAt: time.Now(),
	})

	// ... rest of publishRuleEvent stays the same ...
```

### 5.2 Modify whatsAppSvc

**File:** `apps/api/internal/service/whatsapp.go`

**Changes:**

1. Add `events domain.EventRepository` field:

```go
type whatsAppSvc struct {
	authRepo     domain.AuthRepository
	convRepo     domain.ConversationRepository
	customerRepo domain.CustomerRepository
	publisher    domain.MessagePublisher
	events       domain.EventRepository
}
```

2. Update constructor:

```go
func NewWhatsAppSvc(
	authRepo domain.AuthRepository,
	convRepo domain.ConversationRepository,
	customerRepo domain.CustomerRepository,
	publisher domain.MessagePublisher,
	events domain.EventRepository,
) domain.WhatsAppSvc {
	return &whatsAppSvc{
		authRepo:     authRepo,
		convRepo:     convRepo,
		customerRepo: customerRepo,
		publisher:    publisher,
		events:       events,
	}
}
```

3. Add `persistEvent` helper:

```go
func (s *whatsAppSvc) persistEvent(ctx context.Context, event domain.Event) {
	if s.events == nil {
		return
	}
	if err := s.events.Insert(ctx, event); err != nil {
		slog.Warn("whatsAppSvc.persistEvent: failed", "event_type", event.EventType, "entity_id", event.EntityID, "error", err)
	}
}
```

4. Add event persistence for `customer.created` (after the existing `publishRuleEvent` call, around line 122-135):

```go
	// Emitir evento customer.created si el cliente acaba de ser creado
	if customer != nil && customer.TotalVisits == 0 && customer.CreatedAt.After(time.Now().Add(-5*time.Second)) {
		// Persistir evento analítico
		s.persistEvent(ctx, domain.Event{
			TenantID:   tenant.ID,
			EventType:  "customer.created",
			ActorType:  "system",
			EntityType: "customer",
			EntityID:   customer.ID,
			Payload: map[string]any{
				"acquisition_source": "whatsapp",
				"phone":             customer.Phone,
			},
			OccurredAt: time.Now(),
		})

		s.publishRuleEvent(ctx, domain.RuleEvent{
			// ... existing code ...
		})
	}
```

5. Add event persistence for `message.inbound` (after saving the message to DB, before publishing to RabbitMQ, around line 147-149):

```go
	if err := s.convRepo.SaveMessage(ctx, msg); err != nil {
		slog.Warn("whatsAppSvc.ProcessInbound: save message warning", "error", err)
	}
	_ = s.convRepo.UpdateConversationTimestamp(ctx, conv.ID)

	// Persistir evento analítico de mensaje entrante
	s.persistEvent(ctx, domain.Event{
		TenantID:   tenant.ID,
		EventType:  "message.inbound",
		ActorType:  "customer",
		ActorID:    func() *uuid.UUID { if customer != nil { id := customer.ID; return &id }; return nil }(),
		EntityType: "conversation",
		EntityID:   conv.ID,
		Payload: map[string]any{
			"channel": "whatsapp",
		},
		OccurredAt: time.Now(),
	})
```

**Note:** The `message.outbound` event should be persisted in the outbound worker (`worker/outbound.go`), NOT in whatsapp.go. That's a separate concern — the outbound worker receives the message from RabbitMQ and sends it via Evolution API. Add it there:

**File:** `apps/api/internal/worker/outbound.go` — this receives the message after the AI service processes it. The outbound worker would need an `events domain.EventRepository` field. However, the outbound worker currently only has `waClient` and `notifRepo`. Adding eventRepo there requires modifying its constructor.

For `message.outbound`, add the event persistence in the outbound worker's message handler, after successfully sending via Evolution API. The payload should include `channel`, `response_time_ms` (if available from the AI service response), and `llm_provider` (if available).

**Simplification for v1:** If the outbound worker payload from RabbitMQ doesn't include `response_time_ms` and `llm_provider`, skip those fields for now. They can be added when the AI service includes them in the outbound message payload.

### 5.3 Run tests

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./internal/service/ -v
```

**Expected:** All tests pass.

### 5.4 Commit

```
feat(analytics): persist events in treatment and whatsapp services
```

---

## Task 6: Wire eventRepo in main.go

### 6.1 Modify main.go

**File:** `apps/api/cmd/server/main.go`

**Changes:**

1. Add eventRepo creation after the existing repository block (around line 138):

```go
	eventRepo := repository.NewEventRepository(pool)
```

2. Update `NewAppointmentSvc` call (line ~145) to pass `eventRepo`:

```go
	apptSvc := service.NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, authRepo, waClient, notifRepo, publisher, eventRepo)
```

3. Update `NewTreatmentSvc` call (line ~157) to pass `eventRepo`:

```go
	treatmentSvc := service.NewTreatmentSvc(treatmentRepo, publisher, eventRepo)
```

4. Update `NewWhatsAppSvc` call (line ~149-151) to pass `eventRepo`:

```go
	var waSvc domain.WhatsAppSvc
	if publisher != nil {
		waSvc = service.NewWhatsAppSvc(authRepo, convRepo, customerRepo, publisher, eventRepo)
	}
```

### 6.2 Verify compilation

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./cmd/server/
```

**Expected:** No errors.

### 6.3 Run full test suite

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./...
```

**Expected:** All tests pass. Existing tests that call `NewAppointmentSvc`, `NewTreatmentSvc`, or `NewWhatsAppSvc` will need updating to pass the new `events` parameter (pass `nil` for existing tests that don't care about events).

### 6.4 Commit

```
feat(wiring): inject EventRepository into appointment, treatment, and whatsapp services
```

---

## Task 7: Events maintenance worker — partition creation and retention cleanup

### 7.1 Create worker

**File:** `apps/api/internal/worker/events_maintenance.go`

```go
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// retentionByPlan define los días de retención por plan de suscripción.
var retentionByPlan = map[string]int{
	"basic":   90,
	"starter": 90,
	"trial":   90,
	"pro":     365,
	// enterprise: sin límite — no aparece en el mapa
}

// EventsMaintenanceWorker ejecuta mantenimiento de la tabla events:
// - Diario: limpieza de eventos expirados según el plan del tenant.
// - Mensual: creación de particiones futuras.
type EventsMaintenanceWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
}

// NewEventsMaintenanceWorker crea el worker de mantenimiento de eventos.
func NewEventsMaintenanceWorker(pool *pgxpool.Pool) *EventsMaintenanceWorker {
	return &EventsMaintenanceWorker{
		pool:     pool,
		interval: 24 * time.Hour,
	}
}

// Start inicia el cron en una goroutine. Bloqueante — llamar con go.
func (w *EventsMaintenanceWorker) Start(ctx context.Context) {
	slog.Info("EventsMaintenanceWorker: iniciado (cada 24h)")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Ejecutar inmediatamente al iniciar: asegurar particiones futuras
	w.ensurePartitions(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("EventsMaintenanceWorker: detenido")
			return
		case <-ticker.C:
			w.cleanupExpired(ctx)
			w.ensurePartitions(ctx)
		}
	}
}

// cleanupExpired elimina eventos que superan el período de retención del plan del tenant.
func (w *EventsMaintenanceWorker) cleanupExpired(ctx context.Context) {
	for plan, days := range retentionByPlan {
		result, err := w.pool.Exec(ctx, `
			DELETE FROM events
			WHERE tenant_id IN (
				SELECT id FROM tenants WHERE plan = $1
			)
			AND occurred_at < NOW() - make_interval(days => $2)
		`, plan, days)
		if err != nil {
			slog.Warn("EventsMaintenanceWorker.cleanupExpired: error", "plan", plan, "error", err)
			continue
		}
		if result.RowsAffected() > 0 {
			slog.Info("EventsMaintenanceWorker.cleanupExpired: deleted", "plan", plan, "rows", result.RowsAffected())
		}
	}
}

// ensurePartitions crea particiones para los próximos 2 meses si no existen.
func (w *EventsMaintenanceWorker) ensurePartitions(ctx context.Context) {
	now := time.Now()

	for i := 0; i <= 2; i++ {
		month := now.AddDate(0, i, 0)
		partName := fmt.Sprintf("events_y%dm%02d", month.Year(), month.Month())
		rangeStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		rangeEnd := rangeStart.AddDate(0, 1, 0)

		// CREATE TABLE IF NOT EXISTS no funciona con particiones — verificar primero.
		var exists bool
		err := w.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_class WHERE relname = $1
			)
		`, partName).Scan(&exists)
		if err != nil {
			slog.Warn("EventsMaintenanceWorker.ensurePartitions: check error", "partition", partName, "error", err)
			continue
		}
		if exists {
			continue
		}

		_, err = w.pool.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE %s PARTITION OF events FOR VALUES FROM ('%s') TO ('%s')`,
			partName,
			rangeStart.Format("2006-01-02"),
			rangeEnd.Format("2006-01-02"),
		))
		if err != nil {
			slog.Warn("EventsMaintenanceWorker.ensurePartitions: create error", "partition", partName, "error", err)
		} else {
			slog.Info("EventsMaintenanceWorker.ensurePartitions: created", "partition", partName)
		}
	}
}
```

**IMPORTANT:** The cleanup query runs WITHOUT `withTenant` because it operates across ALL tenants. The events table has RLS, but this worker connects with the pool's default role which should have superuser/owner privileges (same as migration runner). If RLS blocks this, the worker needs to `SET app.tenant_id` per tenant or use `SET ROLE` to bypass RLS. **Verify in testing.**

If RLS blocks the cleanup, add `FORCE ROW LEVEL SECURITY` bypass or run cleanup per-tenant:

```go
// Alternativa segura: iterar por tenant
rows, _ := w.pool.Query(ctx, `SELECT id FROM tenants WHERE plan = $1`, plan)
for rows.Next() {
    var tenantID uuid.UUID
    rows.Scan(&tenantID)
    // DELETE con withTenant por cada tenant
}
```

### 7.2 Wire worker in main.go

**File:** `apps/api/cmd/server/main.go`

Add after the existing worker starts (around line 229-231):

```go
	// Worker de mantenimiento de eventos (particiones + limpieza)
	eventsWorker := worker.NewEventsMaintenanceWorker(pool)
	go eventsWorker.Start(ctx)
```

### 7.3 Write test for ensurePartitions logic

**File:** `apps/api/internal/worker/events_maintenance_test.go`

```go
package worker

import (
	"testing"
	"time"
	"fmt"
)

func TestPartitionNameFormat(t *testing.T) {
	now := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	for i := 0; i <= 2; i++ {
		month := now.AddDate(0, i, 0)
		name := fmt.Sprintf("events_y%dm%02d", month.Year(), month.Month())
		expected := []string{"events_y2026m05", "events_y2026m06", "events_y2026m07"}
		if name != expected[i] {
			t.Fatalf("expected %s, got %s", expected[i], name)
		}
	}
}

func TestRetentionByPlan(t *testing.T) {
	cases := []struct {
		plan string
		days int
		ok   bool
	}{
		{"basic", 90, true},
		{"starter", 90, true},
		{"pro", 365, true},
		{"enterprise", 0, false},
	}
	for _, tc := range cases {
		days, ok := retentionByPlan[tc.plan]
		if ok != tc.ok {
			t.Fatalf("plan %s: expected ok=%v, got %v", tc.plan, tc.ok, ok)
		}
		if ok && days != tc.days {
			t.Fatalf("plan %s: expected %d days, got %d", tc.plan, tc.days, days)
		}
	}
}
```

### 7.4 Run tests

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go test ./internal/worker/ -run TestPartition -v && go test ./internal/worker/ -run TestRetention -v
```

**Expected:** Both pass.

### 7.5 Commit

```
feat(worker): add events maintenance worker for partition creation and retention cleanup
```

---

## Verification Checklist

After all 7 tasks are complete, run these commands:

```bash
# 1. Full test suite
cd /Users/al3jandro/project/agendAI/apps/api && go test ./...

# 2. Build check
cd /Users/al3jandro/project/agendAI/apps/api && go build ./cmd/server/

# 3. Lint (if available)
cd /Users/al3jandro/project/agendAI && make lint

# 4. Verify migration applied
docker compose exec postgres psql -U citaspot -d citaspot -c "SELECT relname FROM pg_class WHERE relname LIKE 'events%' ORDER BY relname;"
```

**Expected output for check 4:**
```
       relname
---------------------
 events
 events_y2026m05
 events_y2026m06
 events_y2026m07
```

---

## Files Created (4)

| File | Purpose |
|---|---|
| `apps/api/db/migrations/032_create_events_table.sql` | Partitioned events table + indexes + RLS |
| `apps/api/internal/domain/event.go` | Event struct + EventRepository interface |
| `apps/api/internal/repository/events.go` | PostgreSQL implementation with withTenant |
| `apps/api/internal/worker/events_maintenance.go` | Daily cleanup + monthly partition creation |

## Files Modified (4)

| File | Change |
|---|---|
| `apps/api/internal/service/appointments.go` | Add `events` field, `persistEvent` helper, call in `emitAppointmentEvent` |
| `apps/api/internal/service/treatment.go` | Add `events` field, `persistEvent` helper, call in `emitTreatmentEvent` |
| `apps/api/internal/service/whatsapp.go` | Add `events` field, `persistEvent` helper, calls for customer.created + message.inbound |
| `apps/api/cmd/server/main.go` | Create eventRepo, pass to 3 service constructors, start maintenance worker |

## Test Files Created (3)

| File | Purpose |
|---|---|
| `apps/api/internal/repository/events_test.go` | Repository unit tests (nil pool, empty batch) |
| `apps/api/internal/service/appointments_test.go` | Event persistence called + best-effort pattern |
| `apps/api/internal/worker/events_maintenance_test.go` | Partition naming + retention config |

## Deferred to v2

- `message.outbound` event (requires modifying outbound worker + AI service payload to include `response_time_ms` and `llm_provider`)
- `notification.sent` and `notification.failed` events (requires modifying reminder worker)
- Query endpoints (GET /api/v1/events with filters) — analytics dashboard
- Event replay / backfill for historical data
