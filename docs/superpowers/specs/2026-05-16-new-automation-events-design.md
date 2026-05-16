# New Automation Events — Design Spec

> **Date:** 2026-05-16
> **Status:** Draft
> **Scope:** 6 new events for the rules/automation engine (`rules.events` RabbitMQ queue)

---

## Context

CitaSpot's automation system publishes events to the `rules.events` RabbitMQ queue. The `RulesEventWorker` consumes them and executes matching rules (conditions → actions like send_whatsapp, create_task, move_stage, update_field).

After fixing 3 gaps (appointment.rescheduled, customer.stage_changed from manual UI, treatment.abandoned), we have **12 events** covering appointments, customers, and treatments. However, several developed features have no automation events: inbound WhatsApp messages, notification failures, professional management, and service catalog changes.

This spec adds **6 high-value events** that cover real automation scenarios for health/beauty businesses.

---

## New Events

All events are published to `rules.events` only (no analytics/PostgreSQL dual emission).

### 1. `message.inbound`

| Field | Value |
|-------|-------|
| **Source** | `WhatsAppSvc.ProcessInbound()` — `apps/api/internal/service/whatsapp.go` |
| **Publisher status** | Already injected |
| **Actor** | customer |
| **Entity type** | conversation |
| **Entity ID** | conversation ID |
| **CustomerID** | resolved customer UUID |

**Payload:**
```json
{
  "channel": "whatsapp",
  "customer_id": "uuid",
  "customer_name": "string",
  "message_preview": "string (first 50 chars)"
}
```

**Emission point:** After inbound message is saved to conversation and customer is resolved (after line ~181 in ProcessInbound, near existing `message.inbound` analytics event).

**Use cases:**
- "When a message arrives outside business hours → create follow-up task"
- "When a message arrives → move customer to stage 'Contacted'"

---

### 2. `notification.failed`

| Field | Value |
|-------|-------|
| **Source** | `ReminderWorker.send()` — `apps/api/internal/worker/reminder.go` |
| **Publisher status** | Needs injection |
| **Actor** | system |
| **Entity type** | notification |
| **Entity ID** | appointment ID |
| **CustomerID** | customer ID from the appointment |

**Payload:**
```json
{
  "type": "reminder_24h | reminder_2h | review",
  "channel": "whatsapp",
  "error": "string",
  "appointment_id": "uuid",
  "customer_id": "uuid"
}
```

**Emission point:** Inside `send()` method when `sendErr != nil`, after logging the notification as failed.

**Safety:** Only emitted from `ReminderWorker`, NOT from `OutboundWorker`. This prevents automation loops where a `notification.failed` event triggers a `send_whatsapp` action that fails and emits another `notification.failed`.

**Use cases:**
- "When a reminder fails → create urgent task to call the patient"

---

### 3. `professional.created`

| Field | Value |
|-------|-------|
| **Source** | `ProfessionalService.Create()` — `apps/api/internal/service/professionals.go` |
| **Publisher status** | Needs injection |
| **Actor** | admin |
| **Entity type** | professional |
| **Entity ID** | professional UUID |
| **CustomerID** | `uuid.Nil` (not applicable) |

**Payload:**
```json
{
  "professional_id": "uuid",
  "name": "string",
  "specialty": "string | null"
}
```

**Use cases:**
- "When a new professional joins → send WhatsApp welcome to team"
- "When a new professional joins → create onboarding tasks"

---

### 4. `professional.archived`

| Field | Value |
|-------|-------|
| **Source** | `ProfessionalService.Update()` — `apps/api/internal/service/professionals.go` |
| **Publisher status** | Same as professional.created (shared service) |
| **Actor** | admin |
| **Entity type** | professional |
| **Entity ID** | professional UUID |
| **CustomerID** | `uuid.Nil` |

**Payload:**
```json
{
  "professional_id": "uuid",
  "name": "string"
}
```

**Emission condition:** Only when `is_archived` changes from `false` to `true`. Not on every update.

**Use cases:**
- "When a professional is archived → notify admin"

---

### 5. `service.created`

| Field | Value |
|-------|-------|
| **Source** | `ServiceSvc.Create()` — `apps/api/internal/service/services.go` |
| **Publisher status** | Needs injection |
| **Actor** | admin |
| **Entity type** | service |
| **Entity ID** | service UUID |
| **CustomerID** | `uuid.Nil` |

**Payload:**
```json
{
  "service_id": "uuid",
  "name": "string",
  "price": "number | null",
  "duration_minutes": "number"
}
```

**Use cases:**
- "When a service is created → create task to configure availability"

---

### 6. `service.updated`

| Field | Value |
|-------|-------|
| **Source** | `ServiceSvc.Update()` — `apps/api/internal/service/services.go` |
| **Publisher status** | Same as service.created (shared service) |
| **Actor** | admin |
| **Entity type** | service |
| **Entity ID** | service UUID |
| **CustomerID** | `uuid.Nil` |

**Payload:**
```json
{
  "service_id": "uuid",
  "name": "string",
  "price": "number | null",
  "duration_minutes": "number"
}
```

**Use cases:**
- "When a price changes → notify internally"

---

## Infrastructure Changes

### Services/workers that need `MessagePublisher` injected

| Component | Constructor | File | main.go line |
|-----------|-------------|------|-------------|
| `ProfessionalService` | `NewProfessionalService(profRepo, scheduleRepo)` → add `publisher` | `service/professionals.go` | ~144 |
| `ServiceSvc` | `NewServiceSvc(serviceRepo)` → add `publisher` | `service/services.go` | ~145 |
| `ReminderWorker` | `NewReminderWorker(reminderRepo, notifRepo, waClient)` → add `publisher` | `worker/reminder.go` | ~231 |

Each component gets the standard `publishRuleEvent()` helper method (best-effort, nil-safe, log-and-continue on error).

### Components that already have publisher (no changes needed)
- `WhatsAppSvc` — just needs to emit the new `message.inbound` event

### Frontend changes

Add all 6 events to `TRIGGER_EVENTS` array in:
- `apps/web/app/dashboard/automations/new/page.tsx`
- `apps/web/app/dashboard/automations/[id]/page.tsx`

Add translations in:
- `apps/web/lib/i18n/locales/es.ts`
- `apps/web/lib/i18n/locales/en.ts`
- `apps/web/lib/i18n/locales/pt.ts`

**Translations:**

| Event | ES | EN | PT |
|-------|----|----|----| 
| `message.inbound` | Mensaje recibido | Message received | Mensagem recebida |
| `notification.failed` | Notificacion fallida | Notification failed | Notificacao falhou |
| `professional.created` | Profesional creado | Professional created | Profissional criado |
| `professional.archived` | Profesional archivado | Professional archived | Profissional arquivado |
| `service.created` | Servicio creado | Service created | Servico criado |
| `service.updated` | Servicio actualizado | Service updated | Servico atualizado |

---

## CustomerID for non-customer events

Events like `professional.created` and `service.created` don't have an associated customer. The `RuleEvent.CustomerID` field will be `uuid.Nil`. The `RulesEventWorker` already handles this — it matches rules by `TenantID + EventType`, not by `CustomerID`. Actions that require a customer (like `send_whatsapp`) won't be applicable for these events unless the rule explicitly targets a customer from the condition/payload.

---

## Verification

1. Create a rule with trigger `message.inbound` → send a WhatsApp message to the connected number → verify the rule fires
2. Simulate a reminder failure (disconnect WhatsApp, wait for reminder cron) → verify `notification.failed` event arrives and rule executes
3. Create a professional from the dashboard → verify `professional.created` fires matching rules
4. Archive a professional → verify `professional.archived` fires
5. Create and update a service → verify both events fire
6. All 6 events visible in the automation rule creator dropdown with correct labels in es/en/pt
7. `go build ./...` compiles
8. `go test ./internal/handler/... ./internal/service/... ./internal/worker/...` passes
9. `npx tsc --noEmit` passes (excluding pre-existing errors)
