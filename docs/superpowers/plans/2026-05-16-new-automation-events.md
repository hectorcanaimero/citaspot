# New Automation Events Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 6 new automation events (`message.inbound`, `notification.failed`, `professional.created`, `professional.archived`, `service.created`, `service.updated`) to the rules engine.

**Architecture:** Each event is published to the `rules.events` RabbitMQ queue using the existing `publishRuleEvent()` best-effort pattern. Three services need `MessagePublisher` injected. The `ReminderJob` struct needs a `CustomerID` field added. Frontend needs the 6 events added to `TRIGGER_EVENTS` arrays and i18n translations.

**Tech Stack:** Go (Fiber, amqp091-go), Next.js (TypeScript), RabbitMQ

**Spec:** `docs/superpowers/specs/2026-05-16-new-automation-events-design.md`

---

### Task 1: Add `message.inbound` rule event in WhatsAppSvc

**Files:**
- Modify: `apps/api/internal/service/whatsapp.go:178-193`

This service already has `publisher` and `publishRuleEvent()`. We just need to add the rule event emission next to the existing analytics event.

- [ ] **Step 1: Add rule event after analytics event**

In `apps/api/internal/service/whatsapp.go`, find the block starting at line 178 (`// Persist message.inbound analytics event`). After the `persistEvent` call (after line 192's closing `}`), add:

```go
	// Publish message.inbound to rules engine
	if customer != nil {
		preview := content
		if len(preview) > 50 {
			preview = preview[:50]
		}
		s.publishRuleEvent(ctx, domain.RuleEvent{
			TenantID:   tenant.ID,
			EventType:  "message.inbound",
			CustomerID: customer.ID,
			EntityID:   conv.ID,
			EntityType: "conversation",
			Payload: map[string]any{
				"channel":         "whatsapp",
				"customer_id":     customer.ID.String(),
				"customer_name":   customer.Name,
				"message_preview": preview,
			},
			Timestamp: time.Now(),
		})
	}
```

- [ ] **Step 2: Verify build**

Run: `cd apps/api && go build ./...`
Expected: clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/service/whatsapp.go
git commit -m "feat(automations): emit message.inbound rule event from WhatsAppSvc"
```

---

### Task 2: Add `CustomerID` to `ReminderJob` and query

**Files:**
- Modify: `apps/api/internal/domain/types.go:543-554`
- Modify: `apps/api/internal/repository/reminders.go:21-26,88-107`

The `ReminderJob` struct doesn't have `CustomerID`, but we need it for the `RuleEvent`. Add the field and update the SQL query + scanner.

- [ ] **Step 1: Add CustomerID field to ReminderJob**

In `apps/api/internal/domain/types.go`, add `CustomerID` to the `ReminderJob` struct after `TenantID`:

```go
type ReminderJob struct {
	AppointmentID    uuid.UUID `json:"appointment_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	CustomerID       uuid.UUID `json:"customer_id"`
	TenantSlug       string    `json:"tenant_slug"`
	TenantTimezone   string    `json:"tenant_timezone"`
	CustomerName     string    `json:"customer_name"`
	CustomerPhone    string    `json:"customer_phone"`
	ProfessionalName string    `json:"professional_name"`
	ServiceName      string    `json:"service_name"`
	StartsAt         time.Time `json:"starts_at"`
	Type             string    `json:"type"`
}
```

- [ ] **Step 2: Update reminderColumns SQL**

In `apps/api/internal/repository/reminders.go`, update `reminderColumns` to include `c.id`:

```go
const reminderColumns = `
	a.id, a.tenant_id, c.id,
	t.slug, t.timezone,
	c.name, c.phone,
	p.name, s.name,
	a.starts_at
`
```

- [ ] **Step 3: Update scanReminderJobs to scan CustomerID**

In `apps/api/internal/repository/reminders.go`, update the `scanReminderJobs` function:

```go
func scanReminderJobs(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}, reminderType string) ([]*domain.ReminderJob, error) {
	var result []*domain.ReminderJob
	for rows.Next() {
		j := &domain.ReminderJob{Type: reminderType}
		if err := rows.Scan(
			&j.AppointmentID, &j.TenantID, &j.CustomerID,
			&j.TenantSlug, &j.TenantTimezone,
			&j.CustomerName, &j.CustomerPhone,
			&j.ProfessionalName, &j.ServiceName,
			&j.StartsAt,
		); err != nil {
			return nil, fmt.Errorf("scanReminderJobs: %w", err)
		}
		result = append(result, j)
	}
	return result, rows.Err()
}
```

- [ ] **Step 4: Verify build**

Run: `cd apps/api && go build ./...`
Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/domain/types.go apps/api/internal/repository/reminders.go
git commit -m "feat(domain): add CustomerID to ReminderJob for rule event emission"
```

---

### Task 3: Add `notification.failed` rule event in ReminderWorker

**Files:**
- Modify: `apps/api/internal/worker/reminder.go:16-35,134-145`
- Modify: `apps/api/cmd/server/main.go:231`

Inject `MessagePublisher` into `ReminderWorker` and emit `notification.failed` when `sendErr != nil`.

- [ ] **Step 1: Add publisher field and update constructor**

In `apps/api/internal/worker/reminder.go`, update the struct and constructor:

```go
type ReminderWorker struct {
	reminderRepo  domain.ReminderRepository
	notifRepo     domain.NotificationRepository
	waClient      domain.WAClient
	publisher     domain.MessagePublisher
	interval      time.Duration
}

func NewReminderWorker(
	reminderRepo domain.ReminderRepository,
	notifRepo domain.NotificationRepository,
	waClient domain.WAClient,
	publisher domain.MessagePublisher,
) *ReminderWorker {
	return &ReminderWorker{
		reminderRepo: reminderRepo,
		notifRepo:    notifRepo,
		waClient:     waClient,
		publisher:    publisher,
		interval:     5 * time.Minute,
	}
}
```

- [ ] **Step 2: Add publishRuleEvent helper**

Add at the end of `apps/api/internal/worker/reminder.go`:

```go
func (w *ReminderWorker) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if w.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("ReminderWorker.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := w.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("ReminderWorker.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
```

Add `"encoding/json"` to the import block.

- [ ] **Step 3: Emit notification.failed on send error**

In the `send()` method, after the existing `if sendErr != nil` block (around line 134), add the rule event emission. Replace the entire block:

```go
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()

		w.publishRuleEvent(ctx, domain.RuleEvent{
			TenantID:   job.TenantID,
			EventType:  "notification.failed",
			CustomerID: job.CustomerID,
			EntityID:   job.AppointmentID,
			EntityType: "notification",
			Payload: map[string]any{
				"type":           fmt.Sprintf("reminder_%d", minutesBefore),
				"channel":        "whatsapp",
				"error":          sendErr.Error(),
				"appointment_id": job.AppointmentID.String(),
				"customer_id":    job.CustomerID.String(),
			},
			Timestamp: time.Now(),
		})
	} else {
		nl.Status = "sent"
	}
```

- [ ] **Step 4: Update main.go wiring**

In `apps/api/cmd/server/main.go`, update line 231:

```go
	reminderWorker := worker.NewReminderWorker(reminderRepo, notifRepo, waClient, publisher)
```

- [ ] **Step 5: Verify build**

Run: `cd apps/api && go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/worker/reminder.go apps/api/cmd/server/main.go
git commit -m "feat(automations): emit notification.failed from ReminderWorker"
```

---

### Task 4: Add `professional.created` and `professional.archived` in ProfessionalService

**Files:**
- Modify: `apps/api/internal/service/professionals.go:12-23,31-56,64-96`
- Modify: `apps/api/cmd/server/main.go:144`

- [ ] **Step 1: Add publisher field, update constructor, add helper**

In `apps/api/internal/service/professionals.go`, replace the struct and constructor, and add the helper:

```go
type professionalService struct {
	profRepo     domain.ProfessionalRepository
	scheduleRepo domain.ScheduleRepository
	publisher    domain.MessagePublisher
}

func NewProfessionalService(profRepo domain.ProfessionalRepository, scheduleRepo domain.ScheduleRepository, publisher domain.MessagePublisher) domain.ProfessionalSvc {
	return &professionalService{
		profRepo:     profRepo,
		scheduleRepo: scheduleRepo,
		publisher:    publisher,
	}
}
```

Add the helper at the end of the file:

```go
func (s *professionalService) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("professionalService.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("professionalService.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
```

Add `"encoding/json"`, `"log/slog"`, and `"time"` to the import block.

- [ ] **Step 2: Emit professional.created in Create()**

In the `Create()` method, after the `s.profRepo.Create(ctx, p)` call succeeds (after line 53), add:

```go
	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  "professional.created",
		CustomerID: uuid.Nil,
		EntityID:   p.ID,
		EntityType: "professional",
		Payload: map[string]any{
			"professional_id": p.ID.String(),
			"name":            p.Name,
			"specialty":       p.Specialty,
		},
		Timestamp: time.Now(),
	})
```

- [ ] **Step 3: Emit professional.archived in Update()**

In the `Update()` method, capture the old `IsArchived` value before applying changes. Right after `p, err := s.profRepo.GetByID(...)` (line 65), add:

```go
	wasArchived := p.IsArchived
```

Then after `s.profRepo.Update(ctx, p)` succeeds (after line 93), add:

```go
	if !wasArchived && p.IsArchived {
		s.publishRuleEvent(ctx, domain.RuleEvent{
			TenantID:   tenantID,
			EventType:  "professional.archived",
			CustomerID: uuid.Nil,
			EntityID:   p.ID,
			EntityType: "professional",
			Payload: map[string]any{
				"professional_id": p.ID.String(),
				"name":            p.Name,
			},
			Timestamp: time.Now(),
		})
	}
```

- [ ] **Step 4: Update main.go wiring**

In `apps/api/cmd/server/main.go`, update line 144:

```go
	profSvc    := service.NewProfessionalService(profRepo, scheduleRepo, publisher)
```

- [ ] **Step 5: Verify build**

Run: `cd apps/api && go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/service/professionals.go apps/api/cmd/server/main.go
git commit -m "feat(automations): emit professional.created and professional.archived events"
```

---

### Task 5: Add `service.created` and `service.updated` in ServiceSvc

**Files:**
- Modify: `apps/api/internal/service/services.go:12-19,27-54,70-103`
- Modify: `apps/api/cmd/server/main.go:145`

- [ ] **Step 1: Add publisher field, update constructor, add helper**

In `apps/api/internal/service/services.go`, replace the struct and constructor, and add the helper:

```go
type serviceSvc struct {
	repo      domain.ServiceRepository
	publisher domain.MessagePublisher
}

func NewServiceSvc(repo domain.ServiceRepository, publisher domain.MessagePublisher) domain.ServiceSvc {
	return &serviceSvc{repo: repo, publisher: publisher}
}
```

Add the helper at the end of the file:

```go
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
```

Add `"encoding/json"`, `"log/slog"`, and `"time"` to the import block.

- [ ] **Step 2: Emit service.created in Create()**

After `s.repo.Create(ctx, svc)` succeeds (after line 51), add:

```go
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
```

- [ ] **Step 3: Emit service.updated in Update()**

After `s.repo.Update(ctx, svc)` succeeds (after line 100), add:

```go
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
```

- [ ] **Step 4: Update main.go wiring**

In `apps/api/cmd/server/main.go`, update line 145:

```go
	serviceSvc := service.NewServiceSvc(serviceRepo, publisher)
```

- [ ] **Step 5: Verify build**

Run: `cd apps/api && go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/service/services.go apps/api/cmd/server/main.go
git commit -m "feat(automations): emit service.created and service.updated events"
```

---

### Task 6: Add 6 events to frontend TRIGGER_EVENTS and i18n

**Files:**
- Modify: `apps/web/app/dashboard/automations/new/page.tsx:25-38`
- Modify: `apps/web/app/dashboard/automations/[id]/page.tsx:25-38`
- Modify: `apps/web/lib/i18n/locales/es.ts:~1084-1096`
- Modify: `apps/web/lib/i18n/locales/en.ts:~1089-1102`
- Modify: `apps/web/lib/i18n/locales/pt.ts:~1089-1102`

- [ ] **Step 1: Update TRIGGER_EVENTS in new/page.tsx**

In `apps/web/app/dashboard/automations/new/page.tsx`, replace the `TRIGGER_EVENTS` array:

```typescript
const TRIGGER_EVENTS = [
  'appointment.created',
  'appointment.confirmed',
  'appointment.completed',
  'appointment.cancelled',
  'appointment.rescheduled',
  'appointment.no_show',
  'customer.created',
  'customer.stage_changed',
  'treatment.proposed',
  'treatment.accepted',
  'treatment.completed',
  'treatment.abandoned',
  'message.inbound',
  'notification.failed',
  'professional.created',
  'professional.archived',
  'service.created',
  'service.updated',
] as const;
```

- [ ] **Step 2: Update TRIGGER_EVENTS in [id]/page.tsx**

Apply the exact same array replacement in `apps/web/app/dashboard/automations/[id]/page.tsx`.

- [ ] **Step 3: Add translations in es.ts**

In `apps/web/lib/i18n/locales/es.ts`, in the `triggerEvents` object, add after `'treatment.abandoned'`:

```typescript
        'message.inbound': 'Mensaje recibido',
        'notification.failed': 'Notificacion fallida',
        'professional.created': 'Profesional creado',
        'professional.archived': 'Profesional archivado',
        'service.created': 'Servicio creado',
        'service.updated': 'Servicio actualizado',
```

- [ ] **Step 4: Add translations in en.ts**

In `apps/web/lib/i18n/locales/en.ts`, in the `triggerEvents` object, add after `'treatment.abandoned'`:

```typescript
        'message.inbound': 'Message received',
        'notification.failed': 'Notification failed',
        'professional.created': 'Professional created',
        'professional.archived': 'Professional archived',
        'service.created': 'Service created',
        'service.updated': 'Service updated',
```

- [ ] **Step 5: Add translations in pt.ts**

In `apps/web/lib/i18n/locales/pt.ts`, in the `triggerEvents` object, add after `'treatment.abandoned'`:

```typescript
        'message.inbound': 'Mensagem recebida',
        'notification.failed': 'Notificacao falhou',
        'professional.created': 'Profissional criado',
        'professional.archived': 'Profissional arquivado',
        'service.created': 'Servico criado',
        'service.updated': 'Servico atualizado',
```

- [ ] **Step 6: Verify TypeScript compiles**

Run: `cd apps/web && npx tsc --noEmit 2>&1 | grep -E "automations|es\.ts|en\.ts|pt\.ts" || echo "No errors in modified files"`
Expected: "No errors in modified files"

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/dashboard/automations/new/page.tsx apps/web/app/dashboard/automations/\\[id\\]/page.tsx apps/web/lib/i18n/locales/es.ts apps/web/lib/i18n/locales/en.ts apps/web/lib/i18n/locales/pt.ts
git commit -m "feat(automations): add 6 new trigger events to frontend selector and i18n"
```

---

### Task 7: Final verification

**Files:** None (read-only verification)

- [ ] **Step 1: Full Go build**

Run: `cd apps/api && go build ./...`
Expected: clean build.

- [ ] **Step 2: Go tests**

Run: `cd apps/api && go test ./internal/handler/... ./internal/service/... ./internal/worker/... -v 2>&1 | tail -20`
Expected: all tests PASS.

- [ ] **Step 3: TypeScript check**

Run: `cd apps/web && npx tsc --noEmit`
Expected: only pre-existing error in `__tests__/components/knowledge.test.tsx` (unrelated).

- [ ] **Step 4: Verify event count**

Quick grep to confirm all 18 events (12 existing + 6 new) are in the TRIGGER_EVENTS array:

Run: `grep -c "'" apps/web/app/dashboard/automations/new/page.tsx | head -1`
Expected: the TRIGGER_EVENTS array should have 18 entries.
