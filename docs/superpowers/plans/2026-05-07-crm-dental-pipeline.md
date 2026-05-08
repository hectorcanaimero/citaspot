# CRM Dental Pipeline — P0 + P1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the 6 feature gaps identified in the dental client analysis to make CitaSpot sellable to medical/dental verticals.

**Architecture:** All 6 features share a common foundation: TenantSettings (JSONB column already exists but isn't mapped to Go). Build settings infrastructure first, then layer features on top. WA notifications are best-effort — never fail the main DB operation.

**Tech Stack:** Go 1.24 (Fiber), Python (FastAPI), Next.js 14, PostgreSQL + JSONB, pgx/v5

---

## File Structure

### New files
- `apps/api/db/migrations/015_tenant_settings_and_blocks.sql` — settings defaults + recurring blocks columns
- `apps/api/db/migrations/016_flexible_reminders.sql` — reminders_sent JSONB column
- `apps/api/internal/handler/settings.go` — GET/PATCH /settings endpoints
- `apps/api/internal/handler/schedule_blocks.go` — CRUD for schedule blocks

### Modified files (Go API)
- `apps/api/internal/domain/types.go` — TenantSettings, RescheduleRequest, PublicProfile extensions, ScheduleBlock extensions
- `apps/api/internal/domain/interfaces.go` — New methods on AuthRepository, AppointmentSvc, AppointmentRepository, ReminderRepository, NotificationRepository + new ScheduleBlockRepository
- `apps/api/internal/repository/auth.go` — GetTenantSettings, UpdateTenantSettings, extend FindTenantByID/BySlug to scan settings
- `apps/api/internal/repository/appointments.go` — Add Reschedule method
- `apps/api/internal/repository/reminders.go` — Replace hardcoded 24h/2h with dynamic FindDueReminders
- `apps/api/internal/repository/notifications.go` — Add MarkReminderSent (generic)
- `apps/api/internal/repository/schedules.go` — Modify GetBlocks to handle recurring blocks, add CreateBlock/DeleteBlock
- `apps/api/internal/service/appointments.go` — Add WAClient/NotifRepo/AuthRepo deps, send WA on status changes, add Reschedule
- `apps/api/internal/service/public.go` — Extend GetProfile to include settings fields
- `apps/api/internal/worker/reminder.go` — Read per-tenant settings, dynamic reminder times
- `apps/api/internal/handler/appointments.go` — Add Reschedule handler
- `apps/api/cmd/server/main.go` — Wire new deps, register new routes

### Modified files (Python AI)
- `apps/ai/app/agent/orchestrator.py` — Use bot_name from profile in system prompt
- `apps/ai/app/agent/messages.py` — Parameterize system_intro with bot_name

### Modified files (Next.js)
- `apps/web/lib/api.ts` — Extend PublicProfile and add settings API methods
- `apps/web/app/book/[slug]/page.tsx` — Show intro/success text from profile
- `apps/web/app/(dashboard)/settings/page.tsx` — Add editable settings fields (bot, booking text, reminders)

---

## Task 1: Database Migrations

**Files:**
- Create: `apps/api/db/migrations/015_tenant_settings_and_blocks.sql`
- Create: `apps/api/db/migrations/016_flexible_reminders.sql`

- [ ] **Step 1: Write migration 015 — tenant settings defaults + recurring blocks**

```sql
-- 015_tenant_settings_and_blocks.sql

-- Populate default settings for all existing tenants
UPDATE tenants SET settings = jsonb_build_object(
    'reminder_minutes', '[1440, 120]'::jsonb,
    'booking_intro_text', '',
    'booking_success_text', '',
    'bot_name', '',
    'bot_greeting', ''
) WHERE settings = '{}' OR settings IS NULL;

-- Set default for new tenants
ALTER TABLE tenants ALTER COLUMN settings SET DEFAULT jsonb_build_object(
    'reminder_minutes', '[1440, 120]'::jsonb,
    'booking_intro_text', '',
    'booking_success_text', '',
    'bot_name', '',
    'bot_greeting', ''
);

-- Recurring schedule blocks support
ALTER TABLE schedule_blocks ADD COLUMN is_recurring BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE schedule_blocks ADD COLUMN recurrence_days SMALLINT[];
-- For recurring blocks: starts_at/ends_at store time-of-day anchored to epoch (2000-01-01)
-- recurrence_days: array of day_of_week values (0=Sun, 6=Sat)

COMMENT ON COLUMN schedule_blocks.is_recurring IS 'TRUE = repeats weekly on recurrence_days';
COMMENT ON COLUMN schedule_blocks.recurrence_days IS 'Days of week (0=Sun..6=Sat) when this block repeats';
```

- [ ] **Step 2: Write migration 016 — flexible reminders tracking**

```sql
-- 016_flexible_reminders.sql

-- Generic reminders tracking: stores which reminder times have been sent
-- Keys are minutes-before-appointment as strings, values are booleans
ALTER TABLE appointments ADD COLUMN reminders_sent JSONB NOT NULL DEFAULT '{}';

-- Backfill from existing boolean columns
UPDATE appointments SET reminders_sent = jsonb_build_object(
    '1440', reminder_24h_sent,
    '120', reminder_2h_sent
);

-- Index for reminder worker queries
CREATE INDEX idx_appointments_reminders_jsonb ON appointments(tenant_id, starts_at, status)
    WHERE reminders_sent != '{"1440":true,"120":true}';
```

- [ ] **Step 3: Run migrations**

```bash
cd apps/api && go run cmd/migrate/main.go
```

- [ ] **Step 4: Commit**

```bash
git add apps/api/db/migrations/015_tenant_settings_and_blocks.sql apps/api/db/migrations/016_flexible_reminders.sql
git commit -m "feat(db): add tenant settings defaults, recurring blocks, flexible reminders"
```

---

## Task 2: Domain Types and Interfaces

**Files:**
- Modify: `apps/api/internal/domain/types.go`
- Modify: `apps/api/internal/domain/interfaces.go`

- [ ] **Step 1: Add TenantSettings struct and extend Tenant**

In `types.go`, after the `Tenant` struct (line 30), add:

```go
// TenantSettings configuración personalizable del tenant (almacenada en JSONB).
type TenantSettings struct {
	ReminderMinutes    []int  `json:"reminder_minutes"`
	BookingIntroText   string `json:"booking_intro_text"`
	BookingSuccessText string `json:"booking_success_text"`
	BotName            string `json:"bot_name"`
	BotGreeting        string `json:"bot_greeting"`
}
```

Add `Settings TenantSettings` field to the `Tenant` struct, `WAStatus string` field, and `LogoURL string`:

```go
type Tenant struct {
    // ... existing fields ...
    WAStatus       string         `json:"wa_status,omitempty"`
    Settings       TenantSettings `json:"settings"`
    // ...
}
```

- [ ] **Step 2: Add RescheduleRequest type**

After `UpdateAppointmentRequest` (line 253):

```go
// RescheduleRequest datos para reagendar una cita.
type RescheduleRequest struct {
	ProfessionalID *uuid.UUID `json:"professional_id"`
	ServiceID      *uuid.UUID `json:"service_id"`
	StartsAt       time.Time  `json:"starts_at" validate:"required"`
}
```

- [ ] **Step 3: Extend PublicProfile with settings fields**

```go
type PublicProfile struct {
	Slug               string          `json:"slug"`
	Name               string          `json:"name"`
	BusinessType       string          `json:"business_type"`
	City               string          `json:"city,omitempty"`
	Country            string          `json:"country,omitempty"`
	Timezone           string          `json:"timezone"`
	Services           []*Service      `json:"services"`
	Professionals      []*Professional `json:"professionals"`
	BookingIntroText   string          `json:"booking_intro_text,omitempty"`
	BookingSuccessText string          `json:"booking_success_text,omitempty"`
	BotName            string          `json:"bot_name,omitempty"`
	BotGreeting        string          `json:"bot_greeting,omitempty"`
}
```

- [ ] **Step 4: Extend ScheduleBlock with recurring fields**

```go
type ScheduleBlock struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ProfessionalID *uuid.UUID `json:"professional_id,omitempty"`
	StartsAt       time.Time  `json:"starts_at"`
	EndsAt         time.Time  `json:"ends_at"`
	Reason         string     `json:"reason,omitempty"`
	IsRecurring    bool       `json:"is_recurring"`
	RecurrenceDays []int      `json:"recurrence_days,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
```

- [ ] **Step 5: Update interfaces in interfaces.go**

Add to `AuthRepository`:
```go
GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (*TenantSettings, error)
UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, s *TenantSettings) error
```

Add to `AppointmentSvc`:
```go
Reschedule(ctx context.Context, tenantID, id uuid.UUID, req *RescheduleRequest) error
```

Add to `AppointmentRepository`:
```go
Reschedule(ctx context.Context, tenantID, id uuid.UUID, professionalID uuid.UUID, startsAt, endsAt time.Time) error
```

Replace `ReminderRepository` methods:
```go
type ReminderRepository interface {
	FindDueReminders(ctx context.Context, minutesBefore int) ([]*ReminderJob, error)
	GetDistinctReminderMinutes(ctx context.Context) ([]int, error)
}
```

Replace `NotificationRepository`:
```go
type NotificationRepository interface {
	LogNotification(ctx context.Context, log *NotificationLog) error
	MarkReminderSent(ctx context.Context, appointmentID uuid.UUID, minutesBefore int) error
}
```

Add new `ScheduleBlockRepository`:
```go
type ScheduleBlockRepository interface {
	Create(ctx context.Context, b *ScheduleBlock) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, professionalID *uuid.UUID) ([]*ScheduleBlock, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
```

- [ ] **Step 6: Verify compilation**

```bash
cd apps/api && go build ./...
```

Expected: Compilation errors in files that implement old interfaces — that's fine, we'll fix them in the next tasks.

- [ ] **Step 7: Commit**

```bash
git add apps/api/internal/domain/
git commit -m "feat(domain): add TenantSettings, RescheduleRequest, recurring blocks, flexible reminders interfaces"
```

---

## Task 3: Auth Repository — Settings CRUD + Extend Tenant Scan

**Files:**
- Modify: `apps/api/internal/repository/auth.go`

- [ ] **Step 1: Add GetTenantSettings method**

```go
func (r *authRepository) GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSettings, error) {
	var raw []byte
	err := r.db.QueryRow(ctx,
		"SELECT COALESCE(settings, '{}')::TEXT FROM tenants WHERE id = $1", tenantID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("authRepository.GetTenantSettings: %w", err)
	}
	var s domain.TenantSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("authRepository.GetTenantSettings: unmarshal: %w", err)
	}
	return &s, nil
}
```

- [ ] **Step 2: Add UpdateTenantSettings method**

```go
func (r *authRepository) UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, s *domain.TenantSettings) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantSettings: marshal: %w", err)
	}
	tag, err := r.db.Exec(ctx,
		"UPDATE tenants SET settings = $2, updated_at = NOW() WHERE id = $1",
		tenantID, raw,
	)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantSettings: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 3: Extend FindTenantByID to scan settings and wa_status**

Update the query to include `settings, wa_status` columns, and update the Scan call to populate `t.Settings` (using an intermediate `[]byte` for JSONB) and `t.WAStatus`.

```go
func (r *authRepository) FindTenantByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, slug, name, business_type, email,
		       city, country, timezone, plan, plan_status, onboarding_done, trial_ends_at,
		       wa_status, COALESCE(settings, '{}')::TEXT,
		       created_at, updated_at
		FROM tenants WHERE id = $1
	`
	t := &domain.Tenant{}
	var settingsRaw []byte
	var waStatus *string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Slug, &t.Name, &t.BusinessType, &t.Email,
		&t.City, &t.Country, &t.Timezone, &t.Plan, &t.PlanStatus, &t.OnboardingDone, &t.TrialEndsAt,
		&waStatus, &settingsRaw,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("authRepository.FindTenantByID: %w", err)
	}
	if waStatus != nil {
		t.WAStatus = *waStatus
	}
	_ = json.Unmarshal(settingsRaw, &t.Settings)
	return t, nil
}
```

Apply the same pattern to `FindTenantBySlug` and `scanUserWithTenant`.

- [ ] **Step 4: Add `encoding/json` import if not already present**

- [ ] **Step 5: Verify compilation**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/repository/auth.go
git commit -m "feat(repo): add tenant settings CRUD and extend tenant queries with settings/wa_status"
```

---

## Task 4: Appointment Service — WA Notifications on Status Change

**Files:**
- Modify: `apps/api/internal/service/appointments.go`

- [ ] **Step 1: Add dependencies to appointmentSvc struct**

```go
type appointmentSvc struct {
	apptRepo     domain.AppointmentRepository
	serviceRepo  domain.ServiceRepository
	customerRepo domain.CustomerRepository
	authRepo     domain.AuthRepository
	waClient     domain.WAClient
	notifRepo    domain.NotificationRepository
}

func NewAppointmentSvc(
	apptRepo domain.AppointmentRepository,
	serviceRepo domain.ServiceRepository,
	customerRepo domain.CustomerRepository,
	authRepo domain.AuthRepository,
	waClient domain.WAClient,
	notifRepo domain.NotificationRepository,
) domain.AppointmentSvc {
	return &appointmentSvc{
		apptRepo:     apptRepo,
		serviceRepo:  serviceRepo,
		customerRepo: customerRepo,
		authRepo:     authRepo,
		waClient:     waClient,
		notifRepo:    notifRepo,
	}
}
```

- [ ] **Step 2: Add notification helper method**

```go
// notifyAppointmentStatus envía una notificación WA al paciente sobre el estado de su cita.
// Best-effort: si falla, loguea el error pero no retorna error.
func (s *appointmentSvc) notifyAppointmentStatus(ctx context.Context, tenantID, apptID uuid.UUID, notifType string) {
	if s.waClient == nil {
		return
	}

	tenant, err := s.authRepo.FindTenantByID(ctx, tenantID)
	if err != nil || tenant.WAStatus != "connected" {
		return
	}

	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil || appt.CustomerPhone == "" {
		return
	}

	loc, _ := time.LoadLocation(tenant.Timezone)
	if loc == nil {
		loc = time.UTC
	}
	fecha := appt.StartsAt.In(loc).Format("02/01/2006")
	hora := appt.StartsAt.In(loc).Format("3:04 PM")

	var text string
	switch notifType {
	case "pending":
		text = fmt.Sprintf(
			"¡Hola %s! 📋 Tu cita para %s con %s el %s a las %s ha sido registrada y está *pendiente de aprobación*. Te avisaremos cuando sea confirmada.",
			appt.CustomerName, appt.ServiceName, appt.ProfessionalName, fecha, hora,
		)
	case "confirmed":
		text = fmt.Sprintf(
			"¡Hola %s! ✅ Tu cita ha sido *confirmada*:\n\n"+
				"📌 *%s*\n"+
				"👩‍⚕️ %s\n"+
				"📅 %s\n"+
				"🕐 %s\n\n"+
				"¡Te esperamos!",
			appt.CustomerName, appt.ServiceName, appt.ProfessionalName, fecha, hora,
		)
	case "cancelled":
		text = fmt.Sprintf(
			"Hola %s, lamentamos informarte que tu cita de %s el %s a las %s ha sido *cancelada*. Puedes reservar nuevamente cuando lo desees.",
			appt.CustomerName, appt.ServiceName, fecha, hora,
		)
	case "rescheduled":
		text = fmt.Sprintf(
			"¡Hola %s! 🔄 Tu cita ha sido *reagendada*:\n\n"+
				"📌 *%s*\n"+
				"👩‍⚕️ %s\n"+
				"📅 %s\n"+
				"🕐 %s\n\n"+
				"¡Te esperamos!",
			appt.CustomerName, appt.ServiceName, appt.ProfessionalName, fecha, hora,
		)
	default:
		return
	}

	waMessageID, sendErr := s.waClient.SendText(ctx, tenant.Slug, appt.CustomerPhone, text)

	// Log notification
	aid := appt.ID
	nl := &domain.NotificationLog{
		ID:            uuid.New(),
		TenantID:      tenantID,
		AppointmentID: &aid,
		Type:          notifType,
		WAMessageID:   waMessageID,
		Content:       text,
		SentAt:        time.Now(),
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
		log.Printf("appointmentSvc.notify: failed to send %s notification: %v", notifType, sendErr)
	} else {
		nl.Status = "sent"
	}
	if err := s.notifRepo.LogNotification(ctx, nl); err != nil {
		log.Printf("appointmentSvc.notify: log error: %v", err)
	}
}
```

- [ ] **Step 3: Add WA notification after Create**

At the end of `Create`, after `s.apptRepo.Create(ctx, appt)` succeeds (line 103-105):

```go
if err := s.apptRepo.Create(ctx, appt); err != nil {
    return nil, fmt.Errorf("appointmentSvc.Create: %w", err)
}

// Best-effort: notificar al paciente que la cita está pendiente
s.notifyAppointmentStatus(ctx, tenantID, appt.ID, "pending")

return appt, nil
```

- [ ] **Step 4: Add WA notification after Update (status change)**

Replace the `Update` method:

```go
func (s *appointmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, req); err != nil {
		return err
	}

	// Best-effort: notificar al paciente del cambio de estado
	if req.Status != "" {
		s.notifyAppointmentStatus(ctx, tenantID, id, req.Status)
	}
	return nil
}
```

- [ ] **Step 5: Add WA notification after Cancel**

```go
func (s *appointmentSvc) Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, &domain.UpdateAppointmentRequest{
		Status:             "cancelled",
		CancellationReason: reason,
	}); err != nil {
		return err
	}
	s.notifyAppointmentStatus(ctx, tenantID, id, "cancelled")
	return nil
}
```

- [ ] **Step 6: Add Reschedule method**

```go
func (s *appointmentSvc) Reschedule(ctx context.Context, tenantID, id uuid.UUID, req *domain.RescheduleRequest) error {
	// 1. Obtener cita actual
	appt, err := s.apptRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: get: %w", err)
	}

	// 2. Determinar professional y service IDs
	profID := appt.ProfessionalID
	if req.ProfessionalID != nil {
		profID = *req.ProfessionalID
	}
	svcID := appt.ServiceID
	if req.ServiceID != nil {
		svcID = *req.ServiceID
	}

	// 3. Obtener servicio para calcular duración
	svc, err := s.serviceRepo.GetByID(ctx, tenantID, svcID)
	if err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: service: %w", err)
	}
	endsAt := req.StartsAt.Add(time.Duration(svc.DurationMin) * time.Minute)

	// 4. Verificar conflictos (excluyendo esta cita)
	excludeID := id
	conflict, err := s.apptRepo.CheckConflict(ctx, tenantID, profID, req.StartsAt, endsAt, &excludeID)
	if err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: conflict: %w", err)
	}
	if conflict {
		return domain.ErrSlotUnavailable
	}

	// 5. Aplicar cambios en DB
	if err := s.apptRepo.Reschedule(ctx, tenantID, id, profID, req.StartsAt, endsAt); err != nil {
		return fmt.Errorf("appointmentSvc.Reschedule: %w", err)
	}

	// 6. Notificar al paciente
	s.notifyAppointmentStatus(ctx, tenantID, id, "rescheduled")
	return nil
}
```

- [ ] **Step 7: Add necessary imports**

Add `log` and `time` to imports if not present.

- [ ] **Step 8: Verify compilation**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add apps/api/internal/service/appointments.go
git commit -m "feat(appointments): add WA notifications on status changes and reschedule support"
```

---

## Task 5: Appointments Repository — Reschedule Method

**Files:**
- Modify: `apps/api/internal/repository/appointments.go`

- [ ] **Step 1: Add Reschedule method**

```go
// Reschedule actualiza el profesional, horario y servicio de una cita existente.
func (r *appointmentRepository) Reschedule(ctx context.Context, tenantID, id, professionalID uuid.UUID, startsAt, endsAt time.Time) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE appointments
			SET professional_id = $3,
			    starts_at       = $4,
			    ends_at         = $5,
			    updated_at      = NOW()
			WHERE tenant_id = $1 AND id = $2
			  AND status NOT IN ('cancelled', 'completed')
		`, tenantID, id, professionalID, startsAt, endsAt)
		if err != nil {
			return fmt.Errorf("appointmentRepository.Reschedule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/repository/appointments.go
git commit -m "feat(repo): add appointment reschedule method"
```

---

## Task 6: Reminders — Flexible Per-Tenant Configuration

**Files:**
- Modify: `apps/api/internal/repository/reminders.go`
- Modify: `apps/api/internal/repository/notifications.go`
- Modify: `apps/api/internal/worker/reminder.go`

- [ ] **Step 1: Replace hardcoded reminder queries**

Rewrite `reminders.go`:

```go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type reminderRepository struct {
	db *pgxpool.Pool
}

func NewReminderRepository(db *pgxpool.Pool) domain.ReminderRepository {
	return &reminderRepository{db: db}
}

const reminderColumns = `
	a.id, a.tenant_id,
	t.slug, t.timezone,
	c.name, c.phone,
	p.name, s.name,
	a.starts_at
`

// FindDueReminders retorna citas que necesitan recordatorio para un tiempo dado (en minutos antes).
// Opera sin RLS — acceso global para el cron.
func (r *reminderRepository) FindDueReminders(ctx context.Context, minutesBefore int) ([]*domain.ReminderJob, error) {
	// Ventana: +/- 15 minutos alrededor del tiempo objetivo
	windowMin := minutesBefore - 15
	windowMax := minutesBefore + 15

	query := `
		SELECT ` + reminderColumns + `
		FROM appointments a
		JOIN tenants       t ON t.id = a.tenant_id
		JOIN customers     c ON c.id = a.customer_id
		JOIN professionals p ON p.id = a.professional_id
		JOIN services      s ON s.id = a.service_id
		WHERE a.status IN ('pending', 'confirmed')
		  AND c.wa_opt_in = TRUE
		  AND c.phone IS NOT NULL
		  AND NOT COALESCE(a.reminders_sent->>$3::TEXT, 'false')::BOOLEAN
		  AND a.starts_at BETWEEN NOW() + ($1 || ' minutes')::INTERVAL
		                       AND NOW() + ($2 || ' minutes')::INTERVAL
		  AND (t.settings->'reminder_minutes') @> to_jsonb($3::INT)
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query, windowMin, windowMax, minutesBefore)
	if err != nil {
		return nil, fmt.Errorf("reminderRepository.FindDueReminders(%d): %w", minutesBefore, err)
	}
	defer rows.Close()

	reminderType := fmt.Sprintf("reminder_%d", minutesBefore)
	return scanReminderJobs(rows, reminderType)
}

// GetDistinctReminderMinutes retorna todos los tiempos de recordatorio únicos configurados por tenants.
func (r *reminderRepository) GetDistinctReminderMinutes(ctx context.Context) ([]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT value::INT
		FROM tenants, jsonb_array_elements(COALESCE(settings->'reminder_minutes', '[1440, 120]'::jsonb)) AS value
		ORDER BY value::INT DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("reminderRepository.GetDistinctReminderMinutes: %w", err)
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var m int
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("reminderRepository.GetDistinctReminderMinutes: scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func scanReminderJobs(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}, reminderType string) ([]*domain.ReminderJob, error) {
	var result []*domain.ReminderJob
	for rows.Next() {
		j := &domain.ReminderJob{Type: reminderType}
		if err := rows.Scan(
			&j.AppointmentID, &j.TenantID,
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

- [ ] **Step 2: Update notifications.go — replace MarkReminder24hSent/MarkReminder2hSent with generic MarkReminderSent**

```go
// MarkReminderSent marca un recordatorio específico como enviado.
func (r *notificationRepository) MarkReminderSent(ctx context.Context, appointmentID uuid.UUID, minutesBefore int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE appointments
		SET reminders_sent = COALESCE(reminders_sent, '{}') || jsonb_build_object($2::TEXT, true),
		    updated_at = NOW()
		WHERE id = $1
	`, appointmentID, fmt.Sprintf("%d", minutesBefore))
	if err != nil {
		return fmt.Errorf("notificationRepository.MarkReminderSent: %w", err)
	}
	return nil
}
```

Remove the old `MarkReminder24hSent` and `MarkReminder2hSent` methods.

- [ ] **Step 3: Rewrite reminder worker for dynamic times**

```go
func (w *ReminderWorker) run(ctx context.Context) {
	// 1. Obtener todos los tiempos de recordatorio configurados
	minutes, err := w.reminderRepo.GetDistinctReminderMinutes(ctx)
	if err != nil {
		slog.Error("ReminderWorker: GetDistinctReminderMinutes error", "error", err)
		return
	}

	totalSent := 0
	for _, m := range minutes {
		jobs, err := w.reminderRepo.FindDueReminders(ctx, m)
		if err != nil {
			slog.Error("ReminderWorker: FindDueReminders error", "minutes", m, "error", err)
			continue
		}
		for _, job := range jobs {
			if err := w.send(ctx, job, m); err != nil {
				slog.Error("ReminderWorker: send error", "appointment", job.AppointmentID, "minutes", m, "error", err)
				continue
			}
			if err := w.notifRepo.MarkReminderSent(ctx, job.AppointmentID, m); err != nil {
				slog.Error("ReminderWorker: mark sent error", "error", err)
			}
			totalSent++
		}
	}
	if totalSent > 0 {
		slog.Info("ReminderWorker: recordatorios procesados", "total", totalSent)
	}
}
```

- [ ] **Step 4: Update send method for dynamic message text**

```go
func (w *ReminderWorker) send(ctx context.Context, job *domain.ReminderJob, minutesBefore int) error {
	connected, err := w.waClient.IsConnected(ctx, job.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("send: instancia %s no conectada", job.TenantSlug)
	}

	time.Sleep(1 * time.Second)

	loc, _ := time.LoadLocation(job.TenantTimezone)
	if loc == nil {
		loc = time.UTC
	}
	horaLocal := job.StartsAt.In(loc).Format("3:04 PM")

	// Generar texto según el tiempo
	var text string
	switch {
	case minutesBefore >= 1440:
		text = fmt.Sprintf(
			"¡Hola %s! 👋 Te recordamos que mañana tienes una cita con %s a las %s para %s. ¿Tienes alguna pregunta? Puedes respondernos aquí.",
			job.CustomerName, job.ProfessionalName, horaLocal, job.ServiceName,
		)
	case minutesBefore >= 60:
		hours := minutesBefore / 60
		text = fmt.Sprintf(
			"¡Hola %s! ⏰ Tu cita con %s es en %d hora(s) (%s). ¡Te esperamos! 😊",
			job.CustomerName, job.ProfessionalName, hours, horaLocal,
		)
	default:
		text = fmt.Sprintf(
			"¡Hola %s! ⏰ Tu cita con %s es en %d minutos (%s). ¡Te esperamos! 😊",
			job.CustomerName, job.ProfessionalName, minutesBefore, horaLocal,
		)
	}

	waMessageID, sendErr := w.waClient.SendText(ctx, job.TenantSlug, job.CustomerPhone, text)

	apptID := job.AppointmentID
	nl := &domain.NotificationLog{
		ID:            uuid.New(),
		TenantID:      job.TenantID,
		AppointmentID: &apptID,
		Type:          fmt.Sprintf("reminder_%d", minutesBefore),
		WAMessageID:   waMessageID,
		Content:       text,
		SentAt:        time.Now(),
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
	} else {
		nl.Status = "sent"
	}

	if err := w.notifRepo.LogNotification(ctx, nl); err != nil {
		slog.Error("ReminderWorker: log notification error", "error", err)
	}

	return sendErr
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/repository/reminders.go apps/api/internal/repository/notifications.go apps/api/internal/worker/reminder.go
git commit -m "feat(reminders): dynamic per-tenant reminder times from settings JSONB"
```

---

## Task 7: Settings Handler + Reschedule Handler + Route Registration

**Files:**
- Create: `apps/api/internal/handler/settings.go`
- Modify: `apps/api/internal/handler/appointments.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Create settings handler**

```go
package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// SettingsHandler endpoints para configuración del tenant.
type SettingsHandler struct {
	authRepo domain.AuthRepository
}

// NewSettingsHandler crea el handler de settings.
func NewSettingsHandler(authRepo domain.AuthRepository) *SettingsHandler {
	return &SettingsHandler{authRepo: authRepo}
}

// Get retorna la configuración actual del tenant.
func (h *SettingsHandler) Get(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	settings, err := h.authRepo.GetTenantSettings(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(settings)
}

// Update actualiza la configuración del tenant.
func (h *SettingsHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var req domain.TenantSettings
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}

	// Validar reminder_minutes
	if len(req.ReminderMinutes) > 5 {
		return fiber.NewError(http.StatusBadRequest, "máximo 5 tiempos de recordatorio")
	}
	for _, m := range req.ReminderMinutes {
		if m < 15 || m > 10080 {
			return fiber.NewError(http.StatusBadRequest, "tiempo de recordatorio debe ser entre 15 minutos y 7 días")
		}
	}

	if err := h.authRepo.UpdateTenantSettings(c.Context(), tenantID, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
```

- [ ] **Step 2: Add Reschedule handler to appointments.go**

```go
// Reschedule reagenda una cita a nuevo horario y/o profesional.
func (h *AppointmentHandler) Reschedule(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID inválido")
	}

	var req domain.RescheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}
	if req.StartsAt.IsZero() {
		return fiber.NewError(http.StatusBadRequest, "starts_at es requerido")
	}

	if err := h.apptSvc.Reschedule(c.Context(), tenantID, id, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
```

- [ ] **Step 3: Update main.go — wire new dependencies and routes**

Update `NewAppointmentSvc` call (line 123):
```go
apptSvc := service.NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, authRepo, waClient, notifRepo)
```

Add settings handler and routes after billing routes (around line 410):
```go
settingsHandler := handler.NewSettingsHandler(authRepo)

// Settings
protected.Get("/settings", settingsHandler.Get)
protected.Patch("/settings", settingsHandler.Update)
```

Add reschedule route to appointments group (after line 387):
```go
appts.Patch("/:id/reschedule", apptHandler.Reschedule)
```

- [ ] **Step 4: Verify compilation**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/handler/settings.go apps/api/internal/handler/appointments.go apps/api/cmd/server/main.go
git commit -m "feat(api): add settings CRUD endpoint and appointment reschedule route"
```

---

## Task 8: Recurring Schedule Blocks

**Files:**
- Modify: `apps/api/internal/repository/schedules.go` — handle recurring blocks in GetBlocks
- Create: `apps/api/internal/handler/schedule_blocks.go` — CRUD endpoints

- [ ] **Step 1: Modify GetBlocks to include recurring blocks**

In `schedules.go`, update the `GetBlocks` query to UNION with recurring blocks:

```go
func (r *scheduleRepository) GetBlocks(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*domain.ScheduleBlock, error) {
	rows, err := r.db.Query(ctx, `
		-- One-time blocks
		SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days, created_at
		FROM schedule_blocks
		WHERE tenant_id = $1
		  AND (professional_id = $2 OR professional_id IS NULL)
		  AND is_recurring = FALSE
		  AND starts_at < $4
		  AND ends_at   > $3
		UNION ALL
		-- Recurring blocks that match any day in the requested range
		SELECT id, tenant_id, professional_id,
		       -- Project recurring time-of-day onto queried date
		       ($3::DATE + starts_at::TIME) AT TIME ZONE 'UTC',
		       ($3::DATE + ends_at::TIME) AT TIME ZONE 'UTC',
		       reason, is_recurring, recurrence_days, created_at
		FROM schedule_blocks
		WHERE tenant_id = $1
		  AND (professional_id = $2 OR professional_id IS NULL)
		  AND is_recurring = TRUE
		  AND EXTRACT(DOW FROM $3 AT TIME ZONE 'UTC')::INT = ANY(recurrence_days)
		ORDER BY starts_at ASC
	`, tenantID, professionalID, from, to)
	if err != nil {
		return nil, fmt.Errorf("scheduleRepository.GetBlocks: %w", err)
	}
	defer rows.Close()

	var blocks []*domain.ScheduleBlock
	for rows.Next() {
		b := &domain.ScheduleBlock{}
		var profID *uuid.UUID
		if err := rows.Scan(
			&b.ID, &b.TenantID, &profID,
			&b.StartsAt, &b.EndsAt, &b.Reason,
			&b.IsRecurring, &b.RecurrenceDays, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scheduleRepository.GetBlocks: scan: %w", err)
		}
		b.ProfessionalID = profID
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}
```

- [ ] **Step 2: Add CreateBlock and DeleteBlock methods**

```go
// CreateBlock crea un bloqueo de horario.
func (r *scheduleRepository) CreateBlock(ctx context.Context, b *domain.ScheduleBlock) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO schedule_blocks (id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, b.ID, b.TenantID, b.ProfessionalID, b.StartsAt, b.EndsAt, b.Reason, b.IsRecurring, b.RecurrenceDays)
	if err != nil {
		return fmt.Errorf("scheduleRepository.CreateBlock: %w", err)
	}
	return nil
}

// ListBlocks retorna todos los bloqueos de un tenant, opcionalmente filtrados por profesional.
func (r *scheduleRepository) ListBlocks(ctx context.Context, tenantID uuid.UUID, professionalID *uuid.UUID) ([]*domain.ScheduleBlock, error) {
	var query string
	var args []any

	if professionalID != nil {
		query = `
			SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days, created_at
			FROM schedule_blocks
			WHERE tenant_id = $1 AND (professional_id = $2 OR professional_id IS NULL)
			ORDER BY is_recurring DESC, starts_at ASC
		`
		args = []any{tenantID, *professionalID}
	} else {
		query = `
			SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days, created_at
			FROM schedule_blocks
			WHERE tenant_id = $1
			ORDER BY is_recurring DESC, starts_at ASC
		`
		args = []any{tenantID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("scheduleRepository.ListBlocks: %w", err)
	}
	defer rows.Close()

	var blocks []*domain.ScheduleBlock
	for rows.Next() {
		b := &domain.ScheduleBlock{}
		var profID *uuid.UUID
		if err := rows.Scan(
			&b.ID, &b.TenantID, &profID, &b.StartsAt, &b.EndsAt, &b.Reason,
			&b.IsRecurring, &b.RecurrenceDays, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scheduleRepository.ListBlocks: scan: %w", err)
		}
		b.ProfessionalID = profID
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// DeleteBlock elimina un bloqueo de horario.
func (r *scheduleRepository) DeleteBlock(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM schedule_blocks WHERE tenant_id = $1 AND id = $2", tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("scheduleRepository.DeleteBlock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 3: Create schedule blocks handler**

```go
package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ScheduleBlockHandler endpoints para gestionar bloqueos de horario.
type ScheduleBlockHandler struct {
	scheduleRepo domain.ScheduleRepository
}

func NewScheduleBlockHandler(scheduleRepo domain.ScheduleRepository) *ScheduleBlockHandler {
	return &ScheduleBlockHandler{scheduleRepo: scheduleRepo}
}

// Create crea un bloqueo de horario (individual o recurrente).
func (h *ScheduleBlockHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var req domain.ScheduleBlock
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}

	// Validaciones
	if req.IsRecurring {
		if len(req.RecurrenceDays) == 0 {
			return fiber.NewError(http.StatusBadRequest, "recurrence_days es requerido para bloqueos recurrentes")
		}
		for _, d := range req.RecurrenceDays {
			if d < 0 || d > 6 {
				return fiber.NewError(http.StatusBadRequest, "recurrence_days debe contener valores entre 0 (Dom) y 6 (Sáb)")
			}
		}
	} else {
		if req.StartsAt.IsZero() || req.EndsAt.IsZero() {
			return fiber.NewError(http.StatusBadRequest, "starts_at y ends_at son requeridos")
		}
		if !req.EndsAt.After(req.StartsAt) {
			return fiber.NewError(http.StatusBadRequest, "ends_at debe ser posterior a starts_at")
		}
	}

	req.ID = uuid.New()
	req.TenantID = tenantID

	if err := h.scheduleRepo.CreateBlock(c.Context(), &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(req)
}

// List retorna los bloqueos de horario del tenant.
func (h *ScheduleBlockHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var profID *uuid.UUID
	if pidStr := c.Query("professional_id"); pidStr != "" {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			return fiber.NewError(http.StatusBadRequest, "professional_id inválido")
		}
		profID = &pid
	}

	blocks, err := h.scheduleRepo.ListBlocks(c.Context(), tenantID, profID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": blocks})
}

// Delete elimina un bloqueo de horario.
func (h *ScheduleBlockHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID inválido")
	}

	if err := h.scheduleRepo.DeleteBlock(c.Context(), tenantID, id); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
```

- [ ] **Step 4: Register routes in main.go**

After instantiating `scheduleRepo` (line 110), add handler and routes:

```go
blockHandler := handler.NewScheduleBlockHandler(scheduleRepo)

// Schedule Blocks
blocks := protected.Group("/schedule-blocks")
blocks.Post("/", blockHandler.Create)
blocks.Get("/", blockHandler.List)
blocks.Delete("/:id", blockHandler.Delete)
```

- [ ] **Step 5: Update ScheduleRepository interface**

Add `CreateBlock`, `ListBlocks`, `DeleteBlock` methods to the `ScheduleRepository` interface in `domain/interfaces.go`.

- [ ] **Step 6: Verify compilation**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add apps/api/internal/repository/schedules.go apps/api/internal/handler/schedule_blocks.go apps/api/cmd/server/main.go apps/api/internal/domain/interfaces.go
git commit -m "feat(schedules): add recurring block support and CRUD endpoints for schedule blocks"
```

---

## Task 9: Public Profile Extensions (Booking Text + Bot Config)

**Files:**
- Modify: `apps/api/internal/service/public.go`
- Modify: `apps/ai/app/agent/orchestrator.py`
- Modify: `apps/ai/app/agent/messages.py`

- [ ] **Step 1: Extend GetProfile to include settings**

In `public.go`, after fetching tenant (line 40-43), read settings and populate profile:

```go
return &domain.PublicProfile{
	Slug:               tenant.Slug,
	Name:               tenant.Name,
	BusinessType:       tenant.BusinessType,
	City:               tenant.City,
	Country:            tenant.Country,
	Timezone:           tenant.Timezone,
	Services:           services,
	Professionals:      professionals,
	BookingIntroText:   tenant.Settings.BookingIntroText,
	BookingSuccessText: tenant.Settings.BookingSuccessText,
	BotName:            tenant.Settings.BotName,
	BotGreeting:        tenant.Settings.BotGreeting,
}, nil
```

This works because `FindTenantBySlug` now scans settings (updated in Task 3).

- [ ] **Step 2: Update AI orchestrator to use bot_name**

In `orchestrator.py`, modify `_build_system_prompt` (line 131):

```python
def _build_system_prompt(profile: dict[str, Any] | None, rag_context: str) -> str:
    """Construye el system prompt del asistente con contexto del negocio."""
    msgs = get_messages(_LANG)
    business_name = profile.get("name", "el negocio") if profile else "el negocio"
    bot_name = profile.get("bot_name", "") if profile else ""

    services_text = ""
    if profile and profile.get("services"):
        lines = []
        for s in profile["services"]:
            price_str = f"${s['price']} USD" if s.get("price") else "consultar en cita"
            lines.append(f"- {s['name']} ({price_str}, {s['duration_min']} min)")
        services_text = f"\n\n{msgs['system_services_header']}\n" + "\n".join(lines)

    rag_section = (
        f"\n\n{msgs['system_rag_header']}\n{rag_context}" if rag_context else ""
    )

    intro = msgs["system_intro"].format(
        business_name=business_name,
        bot_name=bot_name if bot_name else "el asistente virtual",
    )
    warning = msgs["system_warning"]

    return f"{intro}{services_text}{rag_section}\n\n{warning}"
```

Note: also fixed the services_text to handle services without price (`price` can be None/null).

- [ ] **Step 3: Update messages.py system_intro template**

In `messages.py`, update the `system_intro` for all 3 languages:

Spanish (line 44):
```python
"system_intro": (
    "Eres {bot_name} de {business_name}, un negocio de salud y belleza.\n"
    "Tu objetivo es ayudar a los clientes a agendar citas, responder preguntas sobre "
    "servicios y dar información útil.\n\n"
    "Pautas de comunicación:\n"
    "- Tono amable, profesional y conciso\n"
    "- Máximo 3 oraciones por respuesta\n"
    "- Si el cliente quiere agendar: guíalo paso a paso (servicio → profesional → fecha → confirmar)\n"
    "- Si no puedes resolver algo: ofrece escalar con un humano"
),
```

English:
```python
"system_intro": (
    "You are {bot_name} at {business_name}, a health and beauty business.\n"
    ...
),
```

Portuguese:
```python
"system_intro": (
    "Você é {bot_name} de {business_name}, um negócio de saúde e beleza.\n"
    ...
),
```

- [ ] **Step 4: Verify Python syntax**

```bash
cd apps/ai && python -c "from app.agent.messages import get_messages; print('OK')"
```

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/service/public.go apps/ai/app/agent/orchestrator.py apps/ai/app/agent/messages.py
git commit -m "feat: extend public profile with settings and parameterize bot name in AI"
```

---

## Task 10: Frontend — Booking Page + Settings Admin

**Files:**
- Modify: `apps/web/lib/api.ts`
- Modify: `apps/web/app/book/[slug]/page.tsx`
- Modify: `apps/web/app/(dashboard)/settings/page.tsx`

- [ ] **Step 1: Extend TypeScript types and add settings API**

In `api.ts`, extend `PublicProfile` interface:

```typescript
export interface PublicProfile {
  slug: string;
  name: string;
  business_type: string;
  city?: string;
  country?: string;
  timezone?: string;
  services: Service[];
  professionals: Professional[];
  booking_intro_text?: string;
  booking_success_text?: string;
  bot_name?: string;
  bot_greeting?: string;
}

export interface TenantSettings {
  reminder_minutes: number[];
  booking_intro_text: string;
  booking_success_text: string;
  bot_name: string;
  bot_greeting: string;
}
```

Add settings API methods to the authenticated API object:

```typescript
export const settingsApi = {
  async get(): Promise<TenantSettings> {
    return fetchAuth('/api/v1/settings');
  },

  async update(data: Partial<TenantSettings>): Promise<void> {
    await fetchAuth('/api/v1/settings', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  },
};
```

- [ ] **Step 2: Update booking wizard to show custom text**

In `book/[slug]/page.tsx`, update the Header component to show intro text:

```tsx
{/* Header — before service selection */}
{step === 'service' && profile.booking_intro_text && (
  <p className="text-gray-600 mt-2">{profile.booking_intro_text}</p>
)}
```

Update the success screen to show custom success text:

```tsx
{/* Success screen */}
{step === 'success' && (
  <>
    {/* ... existing success content ... */}
    {profile.booking_success_text && (
      <p className="text-gray-600 mt-4">{profile.booking_success_text}</p>
    )}
  </>
)}
```

- [ ] **Step 3: Add settings form to admin settings page**

In `settings/page.tsx`, add a new section within the "Negocio" tab with form fields:

- `bot_name` — input text, label "Nombre del asistente"
- `bot_greeting` — textarea, label "Saludo inicial del bot"
- `booking_intro_text` — textarea, label "Texto antes del formulario de reserva"
- `booking_success_text` — textarea, label "Texto después de confirmar la reserva"
- `reminder_minutes` — multi-select with presets: 1440 (24h), 720 (12h), 120 (2h), 60 (1h), 30 (30min)
- Save button that calls `settingsApi.update()`

Use `useState` for form state, `useEffect` to load initial values from `settingsApi.get()`.

- [ ] **Step 4: Verify frontend compiles**

```bash
cd apps/web && npx next build --no-lint 2>&1 | head -20
```

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/api.ts apps/web/app/book/\\[slug\\]/page.tsx apps/web/app/\\(dashboard\\)/settings/page.tsx
git commit -m "feat(web): add settings form, custom booking text, and reminder config in admin"
```

---

## Verification

### End-to-End Testing

1. **Settings API:**
   ```bash
   # GET settings
   curl -H "Authorization: Bearer $TOKEN" http://localhost:3001/api/v1/settings
   # PATCH settings
   curl -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"bot_name":"Luna","reminder_minutes":[1440,60],"booking_intro_text":"Bienvenido a nuestra clínica"}' \
     http://localhost:3001/api/v1/settings
   ```

2. **Reschedule:**
   ```bash
   curl -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"starts_at":"2026-05-10T14:00:00Z"}' \
     http://localhost:3001/api/v1/appointments/{id}/reschedule
   ```

3. **Schedule Blocks:**
   ```bash
   # Create recurring block (all Saturdays and Sundays)
   curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"professional_id":null,"is_recurring":true,"recurrence_days":[0,6],"starts_at":"2000-01-01T00:00:00Z","ends_at":"2000-01-01T23:59:59Z","reason":"Fin de semana"}' \
     http://localhost:3001/api/v1/schedule-blocks
   ```

4. **WA Notifications:** Create an appointment from admin, verify WA "pending" notification. Confirm it, verify "confirmed" notification. Cancel it, verify "cancelled" notification.

5. **Public Booking Page:** Navigate to `/book/{slug}`, verify custom intro text appears. Complete a booking, verify custom success text appears.

6. **Bot Name:** Send a WA message to the bot, verify system prompt uses custom bot name.

7. **Compilation check:**
   ```bash
   cd apps/api && go build ./...
   cd apps/ai && python -c "from app.agent.orchestrator import process_message; print('OK')"
   cd apps/web && npx next build --no-lint
   ```
