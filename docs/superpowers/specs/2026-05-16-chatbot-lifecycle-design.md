# Chatbot Lifecycle — Consulta, Cancelación, Reagendamiento + P1

**Date:** 2026-05-16
**Status:** Approved
**Scope:** P0 (query, cancel, reschedule) + P1 (smart link, confirmation rule)

---

## Context

The CitaSpot chatbot currently handles new appointment booking and RAG-powered Q&A. It cannot query, cancel, or reschedule existing appointments — the second half of the appointment lifecycle. This spec covers adding those capabilities plus smart booking link sharing and confirmation automation.

---

## 1. Go API — New Public Endpoints

All endpoints under `/api/v1/public/:slug/` — no JWT required, tenant resolved by slug, verified by customer phone.

### 1.1 GET /public/:slug/my-appointments?phone=+58412...

- Resolve tenant by slug
- Find customer by phone (via `customerRepo.FindOrCreateByPhone` — but only find, don't create)
- If no customer found → return empty array
- Query appointments: `status IN ('pending','confirmed')`, `starts_at > NOW()`, ordered by `starts_at ASC`
- Return: `[]{id, service_name, professional_name, starts_at, ends_at, status}`
- Max 10 results

### 1.2 POST /public/:slug/appointments/:id/cancel

Body: `{ "phone": "+58412..." }`

- Resolve tenant by slug
- Get appointment by ID
- Verify appointment.customer.phone == request.phone (ownership check)
- Verify appointment.status IN ('pending', 'confirmed') — can't cancel completed/already cancelled
- Set status = 'cancelled', cancelled_at = now()
- Emit `appointment.cancelled` event (for rules engine)
- Return 204

### 1.3 POST /public/:slug/appointments/:id/reschedule

Body: `{ "phone": "+58412...", "starts_at": "2026-05-23T15:00:00Z", "professional_id": "uuid (optional)" }`

- Resolve tenant by slug
- Get appointment by ID with details (need service.duration_min)
- Verify phone ownership (same as cancel)
- Verify status IN ('pending', 'confirmed')
- Calculate ends_at from starts_at + service.duration_min
- Check no conflict at new time
- Call existing `appointmentRepo.Reschedule()`
- Emit `appointment.rescheduled` event
- Return 204

### Security Model

Phone verification is sufficient because:
- In WhatsApp context, phone comes from Evolution API (verified by WhatsApp itself)
- Public API: phone must match customer record — can't cancel someone else's appointment
- No sensitive data exposed (only future appointment summary)

---

## 2. Python AI — Intent Detection

### New Intents

Add to `Intent` enum:
```python
MY_APPOINTMENTS = "MY_APPOINTMENTS"  # consultar citas
RESCHEDULE = "RESCHEDULE"            # reagendar cita
```

### Updated Classification Prompt

Add to `_SYSTEM_PROMPT`:
```
- MY_APPOINTMENTS: quiere consultar, ver o saber sobre sus citas existentes (¿cuándo es mi cita?, mis citas, qué tengo agendado)
- RESCHEDULE: quiere cambiar fecha/hora de una cita existente (reagendar, mover, cambiar la cita)
```

Keep CANCEL as-is — disambiguation happens in orchestrator by state.

---

## 3. Python AI — New States

Add to `ConvState` enum:
```python
AWAITING_CANCEL_SELECT = "AWAITING_CANCEL_SELECT"
AWAITING_CANCEL_CONFIRM = "AWAITING_CANCEL_CONFIRM"
AWAITING_RESCHEDULE_SELECT = "AWAITING_RESCHEDULE_SELECT"
AWAITING_RESCHEDULE_DATE = "AWAITING_RESCHEDULE_DATE"
AWAITING_RESCHEDULE_SLOT = "AWAITING_RESCHEDULE_SLOT"
AWAITING_RESCHEDULE_CONFIRM = "AWAITING_RESCHEDULE_CONFIRM"
```

Redis state adds:
```json
{
  "pending_appointments": [{"id": "uuid", "service": "Corte", "professional": "Ana", "starts_at": "ISO", "ends_at": "ISO"}],
  "pending_cancel_id": "uuid",
  "pending_reschedule_id": "uuid",
  "pending_reschedule_service": "Corte"
}
```

---

## 4. Orchestrator Flows

### 4.1 MY_APPOINTMENTS (query)

```
IDLE + MY_APPOINTMENTS → fetch appointments by phone
  → 0 results: "No encontré citas agendadas. ¿Te gustaría agendar una?"
  → N results: formatted list with date/time/service/professional
  → Stay in IDLE
```

### 4.2 CANCEL (in IDLE state)

```
IDLE + CANCEL → fetch appointments by phone
  → 0 results: "No encontré citas para cancelar."
  → 1 result: auto-select, ask confirm → AWAITING_CANCEL_CONFIRM
  → N results: show list → AWAITING_CANCEL_SELECT

AWAITING_CANCEL_SELECT + number → select appointment → AWAITING_CANCEL_CONFIRM
AWAITING_CANCEL_CONFIRM + CONFIRM → POST /cancel → reset → IDLE
AWAITING_CANCEL_CONFIRM + CANCEL → abort → IDLE
```

### 4.3 RESCHEDULE

```
IDLE + RESCHEDULE → fetch appointments by phone
  → 0 results: "No encontré citas para reagendar."
  → 1 result: auto-select → AWAITING_RESCHEDULE_DATE
  → N results: show list → AWAITING_RESCHEDULE_SELECT

AWAITING_RESCHEDULE_SELECT + number → select → AWAITING_RESCHEDULE_DATE
AWAITING_RESCHEDULE_DATE + date text → parse date → fetch availability → AWAITING_RESCHEDULE_SLOT
AWAITING_RESCHEDULE_SLOT + number → select slot → AWAITING_RESCHEDULE_CONFIRM
AWAITING_RESCHEDULE_CONFIRM + CONFIRM → POST /reschedule → reset → IDLE
AWAITING_RESCHEDULE_CONFIRM + CANCEL → abort → IDLE
```

### 4.4 Smart Booking Link (P1)

In `_start_booking_flow()`:
- If `len(services) > 4` OR `len(professionals) > 3`: send booking link instead of list
- Message: "Tenemos varias opciones disponibles. Te comparto este link para que elijas: {url}"
- URL: `https://citaspot.com/book/{slug}` (from config or built from slug)
- Reset state to IDLE (no booking flow needed)

---

## 5. Messages (ES/EN/PT)

New keys to add:

```python
# Appointment query
"no_appointments": "No encontré citas agendadas a tu nombre. ¿Te gustaría agendar una? 😊"
"my_appointments_header": "Tus próximas citas:"
"appointment_line": "{idx}. *{service}* — {datetime} con {professional}"

# Cancel flow
"cancel_which": "¿Cuál cita querés cancelar?"
"cancel_confirm": "¿Confirmas que querés cancelar tu cita de *{service}* el {datetime}?"
"cancel_success": "Tu cita ha sido cancelada. ¿Puedo ayudarte con algo más?"
"cancel_failed": "No pude cancelar la cita. Por favor intenta de nuevo o escríbenos."
"no_appointments_cancel": "No encontré citas pendientes para cancelar."

# Reschedule flow
"reschedule_which": "¿Cuál cita querés reagendar?"
"reschedule_date": "¿Para qué fecha querés mover tu *{service}*? Dime día y mes (ej: 23-05)."
"reschedule_confirm": "¿Confirmas mover tu *{service}* al {datetime}?"
"reschedule_success": "¡Listo! Tu cita fue movida al {datetime}. ¡Nos vemos! 🙌"
"reschedule_failed": "No pude reagendar la cita. Por favor intenta de nuevo."
"no_appointments_reschedule": "No encontré citas pendientes para reagendar."

# Smart booking link
"booking_link": "Tenemos varias opciones disponibles. Te comparto este link para que elijas cómodamente:\n{url}"
```

---

## 6. P1: Confirmation Automation Rule

Not a chatbot change — uses existing rules engine.

**Default rule template** (created during onboarding or available as preset):
- Name: "Confirmación de cita 24h antes"
- Trigger: temporal
- Schedule: `{ interval_days: -1, reference_field: "starts_at" }`
- Action: send_whatsapp with template:
  "Hola {{customer_name}}, te recordamos tu cita de {{service_name}} mañana a las {{appointment_time}}. ¿Confirmas asistencia? Responde *Sí* o *No*."
- Cooldown: 24h

**Note:** The temporal rule engine needs to support negative interval_days (X days BEFORE reference). Verify this works in the current implementation.

---

## 7. Files to Modify

### Go API
- `apps/api/internal/handler/public.go` — 3 new handler methods
- `apps/api/internal/service/public.go` (or new file) — business logic for new endpoints
- `apps/api/internal/domain/interfaces.go` — new interface methods
- `apps/api/internal/repository/appointments.go` — new query: ListByCustomerPhone
- `apps/api/internal/repository/customers.go` — FindByPhone (find-only, no create)
- Router file — register new routes

### Python AI
- `apps/ai/app/agent/intent.py` — 2 new intents + updated prompt
- `apps/ai/app/agent/messages.py` — ~15 new message keys × 3 languages
- `apps/ai/app/agent/orchestrator.py` — 6 new states + 3 new flows + smart link logic

### Web (minimal)
- Automation rule template/preset if needed

---

## 8. Verification

1. **Unit tests (Go):** Test new endpoints with valid/invalid phone, missing appointments, wrong ownership
2. **Unit tests (Python):** Test intent detection for new intents, test state transitions
3. **Integration:** Full flow test: query → cancel → reschedule via orchestrator
4. **Smart link:** Verify threshold logic (>4 services triggers link)
5. **Lint:** `make lint` passes
