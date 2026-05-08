# CRM Phase 6A: Infrastructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CRM infrastructure (pipeline stages, treatments, tasks, rules CRUD, customer extensions) to the Go API as foundation for the dental CRM vertical.

**Architecture:** Follows existing layered architecture (domain -> repository -> service -> handler). Five new migrations, new domain types added to types.go/interfaces.go, four new entity CRUDs following existing patterns. Rules are CRUD-only in this phase (engine comes in Phase 6B).

**Tech Stack:** Go 1.24, Fiber v2, pgx v5, PostgreSQL with RLS, validator/v10

---

## Task 1: Database Migrations (017-021)

Create five migration files with RLS policies, indexes, and constraints. Order matters due to FK dependencies: pipeline_stages -> treatments -> rules -> tasks -> customer extensions.

### Step 1.1 — Migration 017: pipeline_stages

- [ ] Create `apps/api/db/migrations/017_pipeline_stages.sql`

```sql
-- 017_pipeline_stages.sql
-- Etapas del pipeline CRM (nuevo lead, en tratamiento, completado, etc.)

CREATE TABLE pipeline_stages (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name                TEXT NOT NULL,
  position            INT NOT NULL,
  color               TEXT DEFAULT '#6366f1',
  is_default          BOOLEAN DEFAULT FALSE,
  auto_rules_enabled  BOOLEAN DEFAULT TRUE,
  created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- Solo una etapa por posicion por tenant
CREATE UNIQUE INDEX idx_pipeline_stages_tenant_position ON pipeline_stages(tenant_id, position);

-- Busqueda rapida por tenant
CREATE INDEX idx_pipeline_stages_tenant ON pipeline_stages(tenant_id);

ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;

CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
```

### Step 1.2 — Migration 018: treatments

- [ ] Create `apps/api/db/migrations/018_treatments.sql`

```sql
-- 018_treatments.sql
-- Tratamientos dentales (o de cualquier vertical) asociados a un cliente

CREATE TABLE treatments (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  customer_id         UUID NOT NULL REFERENCES customers(id),
  professional_id     UUID NOT NULL REFERENCES professionals(id),
  name                TEXT NOT NULL,
  treatment_type      TEXT NOT NULL,
                      -- ortodoncia, endodoncia, implante, protesis, cirugia, periodoncia, estetica, general

  status              TEXT NOT NULL DEFAULT 'proposed',
                      -- proposed -> accepted -> in_progress -> completed -> abandoned

  total_sessions      INT,
  completed_sessions  INT DEFAULT 0,
  estimated_cost      NUMERIC(10,2),
  paid_amount         NUMERIC(10,2) DEFAULT 0,
  currency            TEXT DEFAULT 'USD',
  tooth_numbers       INT[],
  notes               TEXT,

  started_at          TIMESTAMPTZ,
  completed_at        TIMESTAMPTZ,
  next_session_at     TIMESTAMPTZ,

  created_at          TIMESTAMPTZ DEFAULT NOW(),
  updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_treatments_tenant         ON treatments(tenant_id);
CREATE INDEX idx_treatments_customer       ON treatments(tenant_id, customer_id);
CREATE INDEX idx_treatments_professional   ON treatments(tenant_id, professional_id);
CREATE INDEX idx_treatments_status         ON treatments(tenant_id, status);

ALTER TABLE treatments ENABLE ROW LEVEL SECURITY;

CREATE POLICY treatments_tenant_isolation ON treatments
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

-- Vincular citas con tratamientos
ALTER TABLE appointments ADD COLUMN treatment_id UUID REFERENCES treatments(id);
CREATE INDEX idx_appointments_treatment ON appointments(treatment_id);
```

### Step 1.3 — Migration 019: rules and rule_executions

- [ ] Create `apps/api/db/migrations/019_rules.sql`

```sql
-- 019_rules.sql
-- Reglas de automatizacion CRM (CRUD only en Phase 6A, engine en Phase 6B)

CREATE TABLE rules (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name              TEXT NOT NULL,
  description       TEXT,
  trigger_type      TEXT NOT NULL,
                    -- 'event' | 'temporal'
  trigger_event     TEXT,
                    -- appointment.completed, appointment.cancelled, appointment.no_show,
                    -- treatment.status_changed, customer.stage_changed, etc.
  trigger_schedule  JSONB,
                    -- Para temporales: {"interval_days": 90, "reference_field": "last_visit_at"}
  conditions        JSONB DEFAULT '[]',
                    -- [{"field": "customer.total_visits", "op": "gte", "value": 3}]
  actions           JSONB NOT NULL,
                    -- [{"type": "send_whatsapp", "template": "recall_reminder"}, ...]
  is_active         BOOLEAN DEFAULT TRUE,
  is_template       BOOLEAN DEFAULT FALSE,
  template_key      TEXT,
  cooldown_hours    INT DEFAULT 24,
  priority          INT DEFAULT 0,
  created_at        TIMESTAMPTZ DEFAULT NOW(),
  updated_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rules_tenant       ON rules(tenant_id);
CREATE INDEX idx_rules_active       ON rules(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_rules_template     ON rules(is_template) WHERE is_template = TRUE;

ALTER TABLE rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY rules_tenant_isolation ON rules
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

-- Registro de ejecucion de reglas (para auditoría y cooldown)
CREATE TABLE rule_executions (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rule_id             UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
  customer_id         UUID REFERENCES customers(id),
  triggered_at        TIMESTAMPTZ DEFAULT NOW(),
  trigger_event       TEXT,
  conditions_snapshot JSONB,
  actions_result      JSONB,
  status              TEXT NOT NULL,
                      -- success | partial | failed
  error_message       TEXT
);

CREATE INDEX idx_rule_executions_tenant    ON rule_executions(tenant_id);
CREATE INDEX idx_rule_executions_rule      ON rule_executions(rule_id, triggered_at DESC);
CREATE INDEX idx_rule_executions_customer  ON rule_executions(customer_id, triggered_at DESC);

ALTER TABLE rule_executions ENABLE ROW LEVEL SECURITY;

CREATE POLICY rule_executions_tenant_isolation ON rule_executions
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
```

### Step 1.4 — Migration 020: tasks

- [ ] Create `apps/api/db/migrations/020_tasks.sql`

```sql
-- 020_tasks.sql
-- Tareas asignables a profesionales (manuales o generadas por reglas)

CREATE TABLE tasks (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  assigned_to     UUID REFERENCES professionals(id),
  customer_id     UUID REFERENCES customers(id),
  appointment_id  UUID REFERENCES appointments(id),
  treatment_id    UUID REFERENCES treatments(id),
  title           TEXT NOT NULL,
  description     TEXT,
  status          TEXT NOT NULL DEFAULT 'pending',
                  -- pending -> in_progress -> completed -> dismissed
  due_at          TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  source          TEXT NOT NULL DEFAULT 'manual',
                  -- manual | rule | system
  rule_id         UUID REFERENCES rules(id),
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tasks_tenant       ON tasks(tenant_id);
CREATE INDEX idx_tasks_assigned     ON tasks(tenant_id, assigned_to) WHERE status IN ('pending', 'in_progress');
CREATE INDEX idx_tasks_customer     ON tasks(customer_id);
CREATE INDEX idx_tasks_status       ON tasks(tenant_id, status);
CREATE INDEX idx_tasks_due          ON tasks(tenant_id, due_at) WHERE status = 'pending';

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;

CREATE POLICY tasks_tenant_isolation ON tasks
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
```

### Step 1.5 — Migration 021: customer CRM extensions

- [ ] Create `apps/api/db/migrations/021_customer_crm_extensions.sql`

```sql
-- 021_customer_crm_extensions.sql
-- Campos CRM adicionales en la tabla customers

ALTER TABLE customers ADD COLUMN stage_id UUID REFERENCES pipeline_stages(id);
ALTER TABLE customers ADD COLUMN last_visit_at TIMESTAMPTZ;
ALTER TABLE customers ADD COLUMN next_recall_at TIMESTAMPTZ;
ALTER TABLE customers ADD COLUMN lifetime_value NUMERIC(10,2) DEFAULT 0;
ALTER TABLE customers ADD COLUMN acquisition_source TEXT;

CREATE INDEX idx_customers_stage       ON customers(tenant_id, stage_id);
CREATE INDEX idx_customers_recall      ON customers(tenant_id, next_recall_at) WHERE next_recall_at IS NOT NULL;
CREATE INDEX idx_customers_ltv         ON customers(tenant_id, lifetime_value DESC);
```

### Step 1.6 — Commit migrations

- [ ] Commit: `feat(db): add CRM migrations 017-021 (pipeline, treatments, rules, tasks, customer extensions)`

---

## Task 2: Domain Types and Interfaces

Add all new structs to `types.go` and all new interfaces to `interfaces.go`.

### Step 2.1 — Add domain types to types.go

- [ ] Add the following types at the end of `apps/api/internal/domain/types.go`, before the closing of the file:

```go
// ── CRM Pipeline ──────────────────────────────────────────────────────────────

// PipelineStage es una etapa del pipeline CRM del tenant.
type PipelineStage struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	Name             string    `json:"name"`
	Position         int       `json:"position"`
	Color            string    `json:"color"`
	IsDefault        bool      `json:"is_default"`
	AutoRulesEnabled bool      `json:"auto_rules_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

// PipelineStageInput datos para crear/actualizar una etapa.
type PipelineStageInput struct {
	Name             string `json:"name"     validate:"required,min=2,max=100"`
	Position         int    `json:"position" validate:"min=0"`
	Color            string `json:"color"    validate:"omitempty,len=7"`
	IsDefault        *bool  `json:"is_default"`
	AutoRulesEnabled *bool  `json:"auto_rules_enabled"`
}

// ReorderStageInput reordena una etapa a una posicion nueva.
type ReorderStageInput struct {
	StageID  uuid.UUID `json:"stage_id"  validate:"required"`
	Position int       `json:"position"  validate:"min=0"`
}

// ── Treatments ────────────────────────────────────────────────────────────────

// Treatment es un plan de tratamiento asociado a un cliente.
type Treatment struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	CustomerID        uuid.UUID  `json:"customer_id"`
	ProfessionalID    uuid.UUID  `json:"professional_id"`
	Name              string     `json:"name"`
	TreatmentType     string     `json:"treatment_type"`
	Status            string     `json:"status"`
	TotalSessions     *int       `json:"total_sessions,omitempty"`
	CompletedSessions int        `json:"completed_sessions"`
	EstimatedCost     *float64   `json:"estimated_cost,omitempty"`
	PaidAmount        float64    `json:"paid_amount"`
	Currency          string     `json:"currency"`
	ToothNumbers      []int      `json:"tooth_numbers,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	NextSessionAt     *time.Time `json:"next_session_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TreatmentInput datos para crear/actualizar un tratamiento.
type TreatmentInput struct {
	CustomerID     uuid.UUID `json:"customer_id"     validate:"required"`
	ProfessionalID uuid.UUID `json:"professional_id" validate:"required"`
	Name           string    `json:"name"            validate:"required,min=2,max=255"`
	TreatmentType  string    `json:"treatment_type"  validate:"required,oneof=ortodoncia endodoncia implante protesis cirugia periodoncia estetica general"`
	TotalSessions  *int      `json:"total_sessions"  validate:"omitempty,min=1"`
	EstimatedCost  *float64  `json:"estimated_cost"  validate:"omitempty,min=0"`
	Currency       string    `json:"currency"        validate:"omitempty,len=3"`
	ToothNumbers   []int     `json:"tooth_numbers"`
	Notes          string    `json:"notes"`
}

// UpdateTreatmentStatusInput datos para cambiar el estado de un tratamiento.
type UpdateTreatmentStatusInput struct {
	Status string `json:"status" validate:"required,oneof=proposed accepted in_progress completed abandoned"`
}

// TreatmentListQuery filtros para listar tratamientos.
type TreatmentListQuery struct {
	CustomerID     *uuid.UUID `json:"customer_id"`
	ProfessionalID *uuid.UUID `json:"professional_id"`
	Status         string     `json:"status"`
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

// Task es una tarea asignable a un profesional.
type Task struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	AssignedTo    *uuid.UUID `json:"assigned_to,omitempty"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty"`
	AppointmentID *uuid.UUID `json:"appointment_id,omitempty"`
	TreatmentID   *uuid.UUID `json:"treatment_id,omitempty"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	Status        string     `json:"status"`
	DueAt         *time.Time `json:"due_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Source        string     `json:"source"`
	RuleID        *uuid.UUID `json:"rule_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TaskInput datos para crear/actualizar una tarea.
type TaskInput struct {
	AssignedTo    *uuid.UUID `json:"assigned_to"`
	CustomerID    *uuid.UUID `json:"customer_id"`
	AppointmentID *uuid.UUID `json:"appointment_id"`
	TreatmentID   *uuid.UUID `json:"treatment_id"`
	Title         string     `json:"title"       validate:"required,min=2,max=255"`
	Description   string     `json:"description"`
	DueAt         *time.Time `json:"due_at"`
}

// TaskListQuery filtros para listar tareas.
type TaskListQuery struct {
	AssignedTo *uuid.UUID `json:"assigned_to"`
	CustomerID *uuid.UUID `json:"customer_id"`
	Status     string     `json:"status"`
	DueBefore  *time.Time `json:"due_before"`
}

// ── Rules ─────────────────────────────────────────────────────────────────────

// RuleCondition es una condicion que debe cumplirse para que la regla se ejecute.
type RuleCondition struct {
	Field string `json:"field"` // customer.total_visits, treatment.status, etc.
	Op    string `json:"op"`    // eq, neq, gt, gte, lt, lte, in, contains
	Value any    `json:"value"`
}

// RuleAction es una accion a ejecutar cuando la regla se dispara.
type RuleAction struct {
	Type     string         `json:"type"`     // send_whatsapp, create_task, move_stage, send_email
	Template string         `json:"template"` // template key para mensajes
	Params   map[string]any `json:"params"`   // parametros adicionales
}

// RuleTriggerSchedule configuracion para reglas temporales.
type RuleTriggerSchedule struct {
	IntervalDays   int    `json:"interval_days"`
	ReferenceField string `json:"reference_field"` // last_visit_at, next_recall_at, etc.
}

// Rule es una regla de automatizacion CRM.
type Rule struct {
	ID              uuid.UUID            `json:"id"`
	TenantID        uuid.UUID            `json:"tenant_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	TriggerType     string               `json:"trigger_type"`
	TriggerEvent    string               `json:"trigger_event,omitempty"`
	TriggerSchedule *RuleTriggerSchedule `json:"trigger_schedule,omitempty"`
	Conditions      []RuleCondition      `json:"conditions"`
	Actions         []RuleAction         `json:"actions"`
	IsActive        bool                 `json:"is_active"`
	IsTemplate      bool                 `json:"is_template"`
	TemplateKey     string               `json:"template_key,omitempty"`
	CooldownHours   int                  `json:"cooldown_hours"`
	Priority        int                  `json:"priority"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// RuleInput datos para crear/actualizar una regla.
type RuleInput struct {
	Name            string               `json:"name"            validate:"required,min=2,max=255"`
	Description     string               `json:"description"`
	TriggerType     string               `json:"trigger_type"    validate:"required,oneof=event temporal"`
	TriggerEvent    string               `json:"trigger_event"   validate:"omitempty"`
	TriggerSchedule *RuleTriggerSchedule `json:"trigger_schedule"`
	Conditions      []RuleCondition      `json:"conditions"`
	Actions         []RuleAction         `json:"actions"         validate:"required,min=1"`
	IsActive        *bool                `json:"is_active"`
	CooldownHours   int                  `json:"cooldown_hours"  validate:"min=0"`
	Priority        int                  `json:"priority"        validate:"min=0"`
}

// RuleExecution es el registro de una ejecucion de regla.
type RuleExecution struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	RuleID             uuid.UUID       `json:"rule_id"`
	CustomerID         *uuid.UUID      `json:"customer_id,omitempty"`
	TriggeredAt        time.Time       `json:"triggered_at"`
	TriggerEvent       string          `json:"trigger_event,omitempty"`
	ConditionsSnapshot []RuleCondition `json:"conditions_snapshot,omitempty"`
	ActionsResult      []RuleAction    `json:"actions_result,omitempty"`
	Status             string          `json:"status"`
	ErrorMessage       string          `json:"error_message,omitempty"`
}
```

### Step 2.2 — Add new domain error

- [ ] Add to `apps/api/internal/domain/errors.go`:

```go
	// ErrInvalidStatusTransition se retorna cuando un cambio de estado no es valido.
	ErrInvalidStatusTransition = errors.New("transición de estado no válida")
```

### Step 2.3 — Add error mapping in handler/errors.go

- [ ] Add a new case in `handleServiceError` in `apps/api/internal/handler/errors.go`, before the `default:` case:

```go
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		return c.Status(http.StatusBadRequest).JSON(newError("invalid_status_transition", err.Error()))
```

### Step 2.4 — Add interfaces to interfaces.go

- [ ] Add the following interfaces at the end of `apps/api/internal/domain/interfaces.go`:

```go
// ── Pipeline Stages ──────────────────────────────────────────────────────────

// PipelineStageRepository operaciones DB para etapas del pipeline.
type PipelineStageRepository interface {
	Create(ctx context.Context, s *PipelineStage) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*PipelineStage, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*PipelineStage, error)
	Update(ctx context.Context, s *PipelineStage) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Reorder(ctx context.Context, tenantID uuid.UUID, items []ReorderStageInput) error
}

// PipelineStageSvc logica de negocio para etapas del pipeline.
type PipelineStageSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *PipelineStageInput) (*PipelineStage, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*PipelineStage, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*PipelineStage, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *PipelineStageInput) (*PipelineStage, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Reorder(ctx context.Context, tenantID uuid.UUID, items []ReorderStageInput) error
}

// ── Treatments ───────────────────────────────────────────────────────────────

// TreatmentRepository operaciones DB para tratamientos.
type TreatmentRepository interface {
	Create(ctx context.Context, t *Treatment) error
	List(ctx context.Context, tenantID uuid.UUID, q *TreatmentListQuery) ([]*Treatment, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Treatment, error)
	Update(ctx context.Context, t *Treatment) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
}

// TreatmentSvc logica de negocio para tratamientos.
type TreatmentSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *TreatmentInput) (*Treatment, error)
	List(ctx context.Context, tenantID uuid.UUID, q *TreatmentListQuery) ([]*Treatment, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Treatment, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *TreatmentInput) (*Treatment, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, input *UpdateTreatmentStatusInput) (*Treatment, error)
}

// ── Tasks ────────────────────────────────────────────────────────────────────

// TaskRepository operaciones DB para tareas.
type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	List(ctx context.Context, tenantID uuid.UUID, q *TaskListQuery) ([]*Task, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, t *Task) error
	Complete(ctx context.Context, tenantID, id uuid.UUID) error
	Dismiss(ctx context.Context, tenantID, id uuid.UUID) error
}

// TaskSvc logica de negocio para tareas.
type TaskSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *TaskInput) (*Task, error)
	List(ctx context.Context, tenantID uuid.UUID, q *TaskListQuery) ([]*Task, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *TaskInput) (*Task, error)
	Complete(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
	Dismiss(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
}

// ── Rules ────────────────────────────────────────────────────────────────────

// RuleRepository operaciones DB para reglas de automatizacion.
type RuleRepository interface {
	Create(ctx context.Context, r *Rule) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Rule, error)
	Update(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	ListTemplates(ctx context.Context) ([]*Rule, error)
}

// RuleSvc logica de negocio para reglas de automatizacion (CRUD only en Phase 6A).
type RuleSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *RuleInput) (*Rule, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Rule, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *RuleInput) (*Rule, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// RuleExecutionRepository operaciones DB para logs de ejecucion de reglas.
type RuleExecutionRepository interface {
	Create(ctx context.Context, e *RuleExecution) error
	ListByRule(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*RuleExecution, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit int) ([]*RuleExecution, error)
}
```

### Step 2.5 — Commit domain types and interfaces

- [ ] Commit: `feat(domain): add CRM types, interfaces, and errors for pipeline, treatments, tasks, rules`

---

## Task 3: Pipeline Stages CRUD

### Step 3.1 — Repository: pipeline_stages.go

- [ ] Create `apps/api/internal/repository/pipeline_stages.go`

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type pipelineStageRepository struct {
	db *pgxpool.Pool
}

// NewPipelineStageRepository crea el repositorio de etapas del pipeline.
func NewPipelineStageRepository(db *pgxpool.Pool) domain.PipelineStageRepository {
	return &pipelineStageRepository{db: db}
}

func scanPipelineStage(row pgx.Row, s *domain.PipelineStage) error {
	return row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Position, &s.Color,
		&s.IsDefault, &s.AutoRulesEnabled, &s.CreatedAt,
	)
}

const pipelineStageColumns = `id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at`

// Create inserta una nueva etapa del pipeline.
func (r *pipelineStageRepository) Create(ctx context.Context, s *domain.PipelineStage) error {
	return withTenant(ctx, r.db, s.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO pipeline_stages (id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`, s.ID, s.TenantID, s.Name, s.Position, s.Color, s.IsDefault, s.AutoRulesEnabled)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.Create: %w", err)
		}
		return nil
	})
}

// List retorna todas las etapas del pipeline ordenadas por posicion.
func (r *pipelineStageRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.PipelineStage, error) {
	var result []*domain.PipelineStage
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+pipelineStageColumns+`
			FROM pipeline_stages
			WHERE tenant_id = $1
			ORDER BY position ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.PipelineStage{}
			if err := scanPipelineStage(rows, s); err != nil {
				return fmt.Errorf("pipelineStageRepository.List: scan: %w", err)
			}
			result = append(result, s)
		}
		return rows.Err()
	})
	return result, err
}

// GetByID retorna una etapa por ID.
func (r *pipelineStageRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PipelineStage, error) {
	var s *domain.PipelineStage
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		s = &domain.PipelineStage{}
		err := scanPipelineStage(tx.QueryRow(ctx, `
			SELECT `+pipelineStageColumns+`
			FROM pipeline_stages
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), s)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("pipelineStageRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Update actualiza una etapa del pipeline.
func (r *pipelineStageRepository) Update(ctx context.Context, s *domain.PipelineStage) error {
	return withTenant(ctx, r.db, s.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE pipeline_stages
			SET name = $3, position = $4, color = $5, is_default = $6, auto_rules_enabled = $7
			WHERE tenant_id = $1 AND id = $2
		`, s.TenantID, s.ID, s.Name, s.Position, s.Color, s.IsDefault, s.AutoRulesEnabled)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Delete elimina una etapa del pipeline.
func (r *pipelineStageRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM pipeline_stages
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.Delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Reorder actualiza las posiciones de multiples etapas en una transaccion.
func (r *pipelineStageRepository) Reorder(ctx context.Context, tenantID uuid.UUID, items []domain.ReorderStageInput) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		for _, item := range items {
			tag, err := tx.Exec(ctx, `
				UPDATE pipeline_stages
				SET position = $3
				WHERE tenant_id = $1 AND id = $2
			`, tenantID, item.StageID, item.Position)
			if err != nil {
				return fmt.Errorf("pipelineStageRepository.Reorder: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("pipelineStageRepository.Reorder: stage %s: %w", item.StageID, domain.ErrNotFound)
			}
		}
		return nil
	})
}
```

### Step 3.2 — Service: pipeline_stage.go

- [ ] Create `apps/api/internal/service/pipeline_stage.go`

```go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type pipelineStageSvc struct {
	repo domain.PipelineStageRepository
}

// NewPipelineStageSvc crea el servicio de etapas del pipeline.
func NewPipelineStageSvc(repo domain.PipelineStageRepository) domain.PipelineStageSvc {
	return &pipelineStageSvc{repo: repo}
}

// Create crea una nueva etapa del pipeline.
func (s *pipelineStageSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.PipelineStageInput) (*domain.PipelineStage, error) {
	color := input.Color
	if color == "" {
		color = "#6366f1"
	}
	isDefault := false
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}
	autoRules := true
	if input.AutoRulesEnabled != nil {
		autoRules = *input.AutoRulesEnabled
	}

	stage := &domain.PipelineStage{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             input.Name,
		Position:         input.Position,
		Color:            color,
		IsDefault:        isDefault,
		AutoRulesEnabled: autoRules,
	}

	if err := s.repo.Create(ctx, stage); err != nil {
		return nil, fmt.Errorf("pipelineStageSvc.Create: %w", err)
	}
	return stage, nil
}

// List retorna todas las etapas del tenant.
func (s *pipelineStageSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.PipelineStage, error) {
	return s.repo.List(ctx, tenantID)
}

// GetByID retorna una etapa por ID.
func (s *pipelineStageSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PipelineStage, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Update actualiza una etapa existente.
func (s *pipelineStageSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.PipelineStageInput) (*domain.PipelineStage, error) {
	stage, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		stage.Name = input.Name
	}
	stage.Position = input.Position
	if input.Color != "" {
		stage.Color = input.Color
	}
	if input.IsDefault != nil {
		stage.IsDefault = *input.IsDefault
	}
	if input.AutoRulesEnabled != nil {
		stage.AutoRulesEnabled = *input.AutoRulesEnabled
	}

	if err := s.repo.Update(ctx, stage); err != nil {
		return nil, fmt.Errorf("pipelineStageSvc.Update: %w", err)
	}
	return stage, nil
}

// Delete elimina una etapa del pipeline.
func (s *pipelineStageSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// Reorder reordena las etapas del pipeline.
func (s *pipelineStageSvc) Reorder(ctx context.Context, tenantID uuid.UUID, items []domain.ReorderStageInput) error {
	if err := s.repo.Reorder(ctx, tenantID, items); err != nil {
		return fmt.Errorf("pipelineStageSvc.Reorder: %w", err)
	}
	return nil
}
```

### Step 3.3 — Handler: pipeline_stages.go

- [ ] Create `apps/api/internal/handler/pipeline_stages.go`

```go
package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// PipelineStageHandler maneja los endpoints de etapas del pipeline.
type PipelineStageHandler struct {
	svc      domain.PipelineStageSvc
	validate *validator.Validate
}

// NewPipelineStageHandler crea el handler de etapas.
func NewPipelineStageHandler(svc domain.PipelineStageSvc) *PipelineStageHandler {
	return &PipelineStageHandler{svc: svc, validate: validator.New()}
}

// List GET /pipeline-stages
func (h *PipelineStageHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	list, err := h.svc.List(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /pipeline-stages
func (h *PipelineStageHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.PipelineStageInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos invalidos: " + validationMessage(err)})
	}

	stage, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(stage)
}

// GetByID GET /pipeline-stages/:id
func (h *PipelineStageHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	stage, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(stage)
}

// Update PATCH /pipeline-stages/:id
func (h *PipelineStageHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	var input domain.PipelineStageInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}

	stage, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(stage)
}

// Delete DELETE /pipeline-stages/:id
func (h *PipelineStageHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	if err := h.svc.Delete(c.Context(), tenantID, id); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// Reorder PUT /pipeline-stages/reorder
func (h *PipelineStageHandler) Reorder(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var items []domain.ReorderStageInput
	if err := c.BodyParser(&items); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}
	if len(items) == 0 {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Lista de etapas vacia"})
	}

	if err := h.svc.Reorder(c.Context(), tenantID, items); err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
```

### Step 3.4 — Commit pipeline stages

- [ ] Commit: `feat(api): add pipeline stages CRUD (repo, service, handler)`

---

## Task 4: Treatments CRUD

### Step 4.1 — Repository: treatments.go

- [ ] Create `apps/api/internal/repository/treatments.go`

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type treatmentRepository struct {
	db *pgxpool.Pool
}

// NewTreatmentRepository crea el repositorio de tratamientos.
func NewTreatmentRepository(db *pgxpool.Pool) domain.TreatmentRepository {
	return &treatmentRepository{db: db}
}

const treatmentColumns = `
	id, tenant_id, customer_id, professional_id, name, treatment_type,
	status, total_sessions, completed_sessions, estimated_cost, paid_amount,
	currency, tooth_numbers, notes, started_at, completed_at, next_session_at,
	created_at, updated_at
`

func scanTreatment(row pgx.Row, t *domain.Treatment) error {
	var notes *string
	err := row.Scan(
		&t.ID, &t.TenantID, &t.CustomerID, &t.ProfessionalID,
		&t.Name, &t.TreatmentType, &t.Status,
		&t.TotalSessions, &t.CompletedSessions,
		&t.EstimatedCost, &t.PaidAmount, &t.Currency,
		&t.ToothNumbers, &notes,
		&t.StartedAt, &t.CompletedAt, &t.NextSessionAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if notes != nil {
		t.Notes = *notes
	}
	return nil
}

// Create inserta un nuevo tratamiento.
func (r *treatmentRepository) Create(ctx context.Context, t *domain.Treatment) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO treatments
				(id, tenant_id, customer_id, professional_id, name, treatment_type,
				 status, total_sessions, estimated_cost, currency, tooth_numbers, notes,
				 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		`, t.ID, t.TenantID, t.CustomerID, t.ProfessionalID,
			t.Name, t.TreatmentType, t.Status,
			t.TotalSessions, t.EstimatedCost, t.Currency,
			t.ToothNumbers, t.Notes)
		if err != nil {
			return fmt.Errorf("treatmentRepository.Create: %w", err)
		}
		return nil
	})
}

// List retorna tratamientos con filtros opcionales.
func (r *treatmentRepository) List(ctx context.Context, tenantID uuid.UUID, q *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
	var result []*domain.Treatment
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := `SELECT ` + treatmentColumns + ` FROM treatments WHERE tenant_id = $1`
		args := []any{tenantID}
		argN := 2

		if q != nil {
			if q.CustomerID != nil {
				query += fmt.Sprintf(" AND customer_id = $%d", argN)
				args = append(args, *q.CustomerID)
				argN++
			}
			if q.ProfessionalID != nil {
				query += fmt.Sprintf(" AND professional_id = $%d", argN)
				args = append(args, *q.ProfessionalID)
				argN++
			}
			if q.Status != "" {
				query += fmt.Sprintf(" AND status = $%d", argN)
				args = append(args, q.Status)
				argN++
			}
		}
		query += " ORDER BY created_at DESC"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("treatmentRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			t := &domain.Treatment{}
			if err := scanTreatment(rows, t); err != nil {
				return fmt.Errorf("treatmentRepository.List: scan: %w", err)
			}
			result = append(result, t)
		}
		return rows.Err()
	})
	return result, err
}

// GetByID retorna un tratamiento por ID.
func (r *treatmentRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Treatment, error) {
	var t *domain.Treatment
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		t = &domain.Treatment{}
		err := scanTreatment(tx.QueryRow(ctx, `
			SELECT `+treatmentColumns+`
			FROM treatments
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), t)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("treatmentRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Update actualiza un tratamiento.
func (r *treatmentRepository) Update(ctx context.Context, t *domain.Treatment) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE treatments
			SET name = $3, treatment_type = $4, total_sessions = $5,
			    estimated_cost = $6, currency = $7, tooth_numbers = $8,
			    notes = $9, updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, t.TenantID, t.ID, t.Name, t.TreatmentType,
			t.TotalSessions, t.EstimatedCost, t.Currency,
			t.ToothNumbers, t.Notes)
		if err != nil {
			return fmt.Errorf("treatmentRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// UpdateStatus actualiza el estado de un tratamiento y timestamps asociados.
func (r *treatmentRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// Construir SET dinamicamente segun el nuevo status
		setClause := "status = $3, updated_at = NOW()"
		switch status {
		case "in_progress":
			setClause += ", started_at = COALESCE(started_at, NOW())"
		case "completed":
			setClause += ", completed_at = NOW()"
		}

		tag, err := tx.Exec(ctx, `
			UPDATE treatments SET `+setClause+`
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id, status)
		if err != nil {
			return fmt.Errorf("treatmentRepository.UpdateStatus: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
```

### Step 4.2 — Service: treatment.go

- [ ] Create `apps/api/internal/service/treatment.go`

```go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// validTreatmentTransitions define las transiciones de estado validas para tratamientos.
var validTreatmentTransitions = map[string][]string{
	"proposed":    {"accepted", "abandoned"},
	"accepted":    {"in_progress", "abandoned"},
	"in_progress": {"completed", "abandoned"},
	"completed":   {},
	"abandoned":   {"proposed"}, // permitir reabrir
}

type treatmentSvc struct {
	repo domain.TreatmentRepository
}

// NewTreatmentSvc crea el servicio de tratamientos.
func NewTreatmentSvc(repo domain.TreatmentRepository) domain.TreatmentSvc {
	return &treatmentSvc{repo: repo}
}

// Create crea un nuevo tratamiento.
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
	return t, nil
}

// List retorna tratamientos con filtros opcionales.
func (s *treatmentSvc) List(ctx context.Context, tenantID uuid.UUID, q *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
	return s.repo.List(ctx, tenantID, q)
}

// GetByID retorna un tratamiento por ID.
func (s *treatmentSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Treatment, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Update actualiza campos editables de un tratamiento.
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

// UpdateStatus cambia el estado de un tratamiento validando la transicion.
func (s *treatmentSvc) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateTreatmentStatusInput) (*domain.Treatment, error) {
	t, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Validar transicion de estado
	allowed, ok := validTreatmentTransitions[t.Status]
	if !ok {
		return nil, fmt.Errorf("treatmentSvc.UpdateStatus: estado actual '%s' desconocido: %w", t.Status, domain.ErrInvalidStatusTransition)
	}
	valid := false
	for _, s := range allowed {
		if s == input.Status {
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

	// Retornar el tratamiento actualizado
	return s.repo.GetByID(ctx, tenantID, id)
}
```

### Step 4.3 — Handler: treatments.go

- [ ] Create `apps/api/internal/handler/treatments.go`

```go
package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// TreatmentHandler maneja los endpoints de tratamientos.
type TreatmentHandler struct {
	svc      domain.TreatmentSvc
	validate *validator.Validate
}

// NewTreatmentHandler crea el handler de tratamientos.
func NewTreatmentHandler(svc domain.TreatmentSvc) *TreatmentHandler {
	return &TreatmentHandler{svc: svc, validate: validator.New()}
}

// List GET /treatments
func (h *TreatmentHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	q := &domain.TreatmentListQuery{
		Status: c.Query("status"),
	}
	if cid := c.Query("customer_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "customer_id invalido"})
		}
		q.CustomerID = &id
	}
	if pid := c.Query("professional_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id invalido"})
		}
		q.ProfessionalID = &id
	}

	list, err := h.svc.List(c.Context(), tenantID, q)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /treatments
func (h *TreatmentHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.TreatmentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos invalidos: " + validationMessage(err)})
	}

	t, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(t)
}

// GetByID GET /treatments/:id
func (h *TreatmentHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	t, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

// Update PATCH /treatments/:id
func (h *TreatmentHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	var input domain.TreatmentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}

	t, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

// UpdateStatus PATCH /treatments/:id/status
func (h *TreatmentHandler) UpdateStatus(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	var input domain.UpdateTreatmentStatusInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos invalidos: " + validationMessage(err)})
	}

	t, err := h.svc.UpdateStatus(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}
```

### Step 4.4 — Commit treatments

- [ ] Commit: `feat(api): add treatments CRUD (repo, service, handler)`

---

## Task 5: Tasks CRUD

### Step 5.1 — Repository: tasks.go

- [ ] Create `apps/api/internal/repository/tasks.go`

```go
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type taskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository crea el repositorio de tareas.
func NewTaskRepository(db *pgxpool.Pool) domain.TaskRepository {
	return &taskRepository{db: db}
}

const taskColumns = `
	id, tenant_id, assigned_to, customer_id, appointment_id, treatment_id,
	title, description, status, due_at, completed_at, source, rule_id, created_at
`

func scanTask(row pgx.Row, t *domain.Task) error {
	var description *string
	err := row.Scan(
		&t.ID, &t.TenantID, &t.AssignedTo, &t.CustomerID,
		&t.AppointmentID, &t.TreatmentID,
		&t.Title, &description, &t.Status,
		&t.DueAt, &t.CompletedAt, &t.Source, &t.RuleID, &t.CreatedAt,
	)
	if err != nil {
		return err
	}
	if description != nil {
		t.Description = *description
	}
	return nil
}

// Create inserta una nueva tarea.
func (r *taskRepository) Create(ctx context.Context, t *domain.Task) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO tasks
				(id, tenant_id, assigned_to, customer_id, appointment_id, treatment_id,
				 title, description, status, due_at, source, rule_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`, t.ID, t.TenantID, t.AssignedTo, t.CustomerID,
			t.AppointmentID, t.TreatmentID,
			t.Title, t.Description, t.Status,
			t.DueAt, t.Source, t.RuleID)
		if err != nil {
			return fmt.Errorf("taskRepository.Create: %w", err)
		}
		return nil
	})
}

// List retorna tareas con filtros opcionales.
func (r *taskRepository) List(ctx context.Context, tenantID uuid.UUID, q *domain.TaskListQuery) ([]*domain.Task, error) {
	var result []*domain.Task
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := `SELECT ` + taskColumns + ` FROM tasks WHERE tenant_id = $1`
		args := []any{tenantID}
		argN := 2

		if q != nil {
			if q.AssignedTo != nil {
				query += fmt.Sprintf(" AND assigned_to = $%d", argN)
				args = append(args, *q.AssignedTo)
				argN++
			}
			if q.CustomerID != nil {
				query += fmt.Sprintf(" AND customer_id = $%d", argN)
				args = append(args, *q.CustomerID)
				argN++
			}
			if q.Status != "" {
				query += fmt.Sprintf(" AND status = $%d", argN)
				args = append(args, q.Status)
				argN++
			}
			if q.DueBefore != nil {
				query += fmt.Sprintf(" AND due_at <= $%d", argN)
				args = append(args, *q.DueBefore)
				argN++
			}
		}
		query += " ORDER BY COALESCE(due_at, '9999-12-31'::timestamptz) ASC, created_at DESC"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("taskRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			t := &domain.Task{}
			if err := scanTask(rows, t); err != nil {
				return fmt.Errorf("taskRepository.List: scan: %w", err)
			}
			result = append(result, t)
		}
		return rows.Err()
	})
	return result, err
}

// GetByID retorna una tarea por ID.
func (r *taskRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	var t *domain.Task
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		t = &domain.Task{}
		err := scanTask(tx.QueryRow(ctx, `
			SELECT `+taskColumns+`
			FROM tasks
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), t)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("taskRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Update actualiza una tarea.
func (r *taskRepository) Update(ctx context.Context, t *domain.Task) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET assigned_to = $3, customer_id = $4, appointment_id = $5, treatment_id = $6,
			    title = $7, description = $8, due_at = $9
			WHERE tenant_id = $1 AND id = $2
		`, t.TenantID, t.ID, t.AssignedTo, t.CustomerID,
			t.AppointmentID, t.TreatmentID, t.Title, t.Description, t.DueAt)
		if err != nil {
			return fmt.Errorf("taskRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Complete marca una tarea como completada.
func (r *taskRepository) Complete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'completed', completed_at = $3
			WHERE tenant_id = $1 AND id = $2 AND status IN ('pending', 'in_progress')
		`, tenantID, id, time.Now())
		if err != nil {
			return fmt.Errorf("taskRepository.Complete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Dismiss descarta una tarea.
func (r *taskRepository) Dismiss(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'dismissed'
			WHERE tenant_id = $1 AND id = $2 AND status IN ('pending', 'in_progress')
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("taskRepository.Dismiss: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
```

### Step 5.2 — Service: task.go

- [ ] Create `apps/api/internal/service/task.go`

```go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type taskSvc struct {
	repo domain.TaskRepository
}

// NewTaskSvc crea el servicio de tareas.
func NewTaskSvc(repo domain.TaskRepository) domain.TaskSvc {
	return &taskSvc{repo: repo}
}

// Create crea una nueva tarea manual.
func (s *taskSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.TaskInput) (*domain.Task, error) {
	t := &domain.Task{
		ID:            uuid.New(),
		TenantID:      tenantID,
		AssignedTo:    input.AssignedTo,
		CustomerID:    input.CustomerID,
		AppointmentID: input.AppointmentID,
		TreatmentID:   input.TreatmentID,
		Title:         input.Title,
		Description:   input.Description,
		Status:        "pending",
		DueAt:         input.DueAt,
		Source:        "manual",
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("taskSvc.Create: %w", err)
	}
	return t, nil
}

// List retorna tareas con filtros opcionales.
func (s *taskSvc) List(ctx context.Context, tenantID uuid.UUID, q *domain.TaskListQuery) ([]*domain.Task, error) {
	return s.repo.List(ctx, tenantID, q)
}

// GetByID retorna una tarea por ID.
func (s *taskSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Update actualiza una tarea existente.
func (s *taskSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.TaskInput) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	t.Title = input.Title
	if input.Description != "" {
		t.Description = input.Description
	}
	t.AssignedTo = input.AssignedTo
	t.CustomerID = input.CustomerID
	t.AppointmentID = input.AppointmentID
	t.TreatmentID = input.TreatmentID
	t.DueAt = input.DueAt

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("taskSvc.Update: %w", err)
	}
	return t, nil
}

// Complete marca una tarea como completada.
func (s *taskSvc) Complete(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if err := s.repo.Complete(ctx, tenantID, id); err != nil {
		return nil, fmt.Errorf("taskSvc.Complete: %w", err)
	}
	return s.repo.GetByID(ctx, tenantID, id)
}

// Dismiss descarta una tarea.
func (s *taskSvc) Dismiss(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if err := s.repo.Dismiss(ctx, tenantID, id); err != nil {
		return nil, fmt.Errorf("taskSvc.Dismiss: %w", err)
	}
	return s.repo.GetByID(ctx, tenantID, id)
}
```

### Step 5.3 — Handler: tasks.go

- [ ] Create `apps/api/internal/handler/tasks.go`

```go
package handler

import (
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// TaskHandler maneja los endpoints de tareas.
type TaskHandler struct {
	svc      domain.TaskSvc
	validate *validator.Validate
}

// NewTaskHandler crea el handler de tareas.
func NewTaskHandler(svc domain.TaskSvc) *TaskHandler {
	return &TaskHandler{svc: svc, validate: validator.New()}
}

// List GET /tasks
func (h *TaskHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	q := &domain.TaskListQuery{
		Status: c.Query("status"),
	}
	if aid := c.Query("assigned_to"); aid != "" {
		id, err := uuid.Parse(aid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "assigned_to invalido"})
		}
		q.AssignedTo = &id
	}
	if cid := c.Query("customer_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "customer_id invalido"})
		}
		q.CustomerID = &id
	}
	if db := c.Query("due_before"); db != "" {
		t, err := time.Parse(time.RFC3339, db)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "due_before invalido (usar RFC3339)"})
		}
		q.DueBefore = &t
	}

	list, err := h.svc.List(c.Context(), tenantID, q)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /tasks
func (h *TaskHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.TaskInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos invalidos: " + validationMessage(err)})
	}

	t, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(t)
}

// GetByID GET /tasks/:id
func (h *TaskHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	t, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

// Update PATCH /tasks/:id
func (h *TaskHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	var input domain.TaskInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}

	t, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

// Complete POST /tasks/:id/complete
func (h *TaskHandler) Complete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	t, err := h.svc.Complete(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

// Dismiss POST /tasks/:id/dismiss
func (h *TaskHandler) Dismiss(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	t, err := h.svc.Dismiss(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}
```

### Step 5.4 — Commit tasks

- [ ] Commit: `feat(api): add tasks CRUD (repo, service, handler)`

---

## Task 6: Rules CRUD

### Step 6.1 — Repository: rules.go

- [ ] Create `apps/api/internal/repository/rules.go`

```go
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type ruleRepository struct {
	db *pgxpool.Pool
}

// NewRuleRepository crea el repositorio de reglas.
func NewRuleRepository(db *pgxpool.Pool) domain.RuleRepository {
	return &ruleRepository{db: db}
}

const ruleColumns = `
	id, tenant_id, name, description, trigger_type, trigger_event, trigger_schedule,
	conditions, actions, is_active, is_template, template_key,
	cooldown_hours, priority, created_at, updated_at
`

func scanRule(row pgx.Row, r *domain.Rule) error {
	var (
		description *string
		triggerEvent *string
		templateKey  *string
		triggerScheduleJSON []byte
		conditionsJSON      []byte
		actionsJSON         []byte
	)

	err := row.Scan(
		&r.ID, &r.TenantID, &r.Name, &description,
		&r.TriggerType, &triggerEvent, &triggerScheduleJSON,
		&conditionsJSON, &actionsJSON,
		&r.IsActive, &r.IsTemplate, &templateKey,
		&r.CooldownHours, &r.Priority, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if description != nil {
		r.Description = *description
	}
	if triggerEvent != nil {
		r.TriggerEvent = *triggerEvent
	}
	if templateKey != nil {
		r.TemplateKey = *templateKey
	}

	// Deserializar JSONB
	if len(triggerScheduleJSON) > 0 && string(triggerScheduleJSON) != "null" {
		r.TriggerSchedule = &domain.RuleTriggerSchedule{}
		if err := json.Unmarshal(triggerScheduleJSON, r.TriggerSchedule); err != nil {
			return fmt.Errorf("scanRule: unmarshal trigger_schedule: %w", err)
		}
	}
	if len(conditionsJSON) > 0 {
		if err := json.Unmarshal(conditionsJSON, &r.Conditions); err != nil {
			return fmt.Errorf("scanRule: unmarshal conditions: %w", err)
		}
	}
	if len(actionsJSON) > 0 {
		if err := json.Unmarshal(actionsJSON, &r.Actions); err != nil {
			return fmt.Errorf("scanRule: unmarshal actions: %w", err)
		}
	}
	return nil
}

// Create inserta una nueva regla.
func (r *ruleRepository) Create(ctx context.Context, rule *domain.Rule) error {
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Create: marshal conditions: %w", err)
	}
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Create: marshal actions: %w", err)
	}
	var triggerScheduleJSON []byte
	if rule.TriggerSchedule != nil {
		triggerScheduleJSON, err = json.Marshal(rule.TriggerSchedule)
		if err != nil {
			return fmt.Errorf("ruleRepository.Create: marshal trigger_schedule: %w", err)
		}
	}

	return withTenant(ctx, r.db, rule.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO rules
				(id, tenant_id, name, description, trigger_type, trigger_event, trigger_schedule,
				 conditions, actions, is_active, is_template, template_key,
				 cooldown_hours, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		`, rule.ID, rule.TenantID, rule.Name, rule.Description,
			rule.TriggerType, rule.TriggerEvent, triggerScheduleJSON,
			conditionsJSON, actionsJSON,
			rule.IsActive, rule.IsTemplate, rule.TemplateKey,
			rule.CooldownHours, rule.Priority)
		if err != nil {
			return fmt.Errorf("ruleRepository.Create: %w", err)
		}
		return nil
	})
}

// List retorna todas las reglas del tenant.
func (r *ruleRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	var result []*domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1
			ORDER BY priority DESC, name ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("ruleRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.List: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}

// GetByID retorna una regla por ID.
func (r *ruleRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Rule, error) {
	var rule *domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rule = &domain.Rule{}
		err := scanRule(tx.QueryRow(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), rule)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("ruleRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// Update actualiza una regla.
func (r *ruleRepository) Update(ctx context.Context, rule *domain.Rule) error {
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Update: marshal conditions: %w", err)
	}
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Update: marshal actions: %w", err)
	}
	var triggerScheduleJSON []byte
	if rule.TriggerSchedule != nil {
		triggerScheduleJSON, err = json.Marshal(rule.TriggerSchedule)
		if err != nil {
			return fmt.Errorf("ruleRepository.Update: marshal trigger_schedule: %w", err)
		}
	}

	return withTenant(ctx, r.db, rule.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE rules
			SET name = $3, description = $4, trigger_type = $5, trigger_event = $6,
			    trigger_schedule = $7, conditions = $8, actions = $9,
			    is_active = $10, cooldown_hours = $11, priority = $12, updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, rule.TenantID, rule.ID, rule.Name, rule.Description,
			rule.TriggerType, rule.TriggerEvent, triggerScheduleJSON,
			conditionsJSON, actionsJSON,
			rule.IsActive, rule.CooldownHours, rule.Priority)
		if err != nil {
			return fmt.Errorf("ruleRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Delete elimina una regla.
func (r *ruleRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM rules
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("ruleRepository.Delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// ListActive retorna reglas activas del tenant (para el engine en Phase 6B).
func (r *ruleRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	var result []*domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1 AND is_active = TRUE AND is_template = FALSE
			ORDER BY priority DESC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("ruleRepository.ListActive: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.ListActive: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}

// ListTemplates retorna las reglas template globales (sin filtro de tenant).
func (r *ruleRepository) ListTemplates(ctx context.Context) ([]*domain.Rule, error) {
	var result []*domain.Rule
	rows, err := r.db.Query(ctx, `
		SELECT `+ruleColumns+`
		FROM rules
		WHERE is_template = TRUE
		ORDER BY priority DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ruleRepository.ListTemplates: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		rule := &domain.Rule{}
		if err := scanRule(rows, rule); err != nil {
			return nil, fmt.Errorf("ruleRepository.ListTemplates: scan: %w", err)
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}
```

### Step 6.2 — Repository: rule_executions.go

- [ ] Create `apps/api/internal/repository/rule_executions.go`

```go
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type ruleExecutionRepository struct {
	db *pgxpool.Pool
}

// NewRuleExecutionRepository crea el repositorio de ejecuciones de reglas.
func NewRuleExecutionRepository(db *pgxpool.Pool) domain.RuleExecutionRepository {
	return &ruleExecutionRepository{db: db}
}

func scanRuleExecution(row pgx.Row, e *domain.RuleExecution) error {
	var (
		triggerEvent       *string
		errorMessage       *string
		conditionsJSON     []byte
		actionsResultJSON  []byte
	)

	err := row.Scan(
		&e.ID, &e.TenantID, &e.RuleID, &e.CustomerID,
		&e.TriggeredAt, &triggerEvent,
		&conditionsJSON, &actionsResultJSON,
		&e.Status, &errorMessage,
	)
	if err != nil {
		return err
	}
	if triggerEvent != nil {
		e.TriggerEvent = *triggerEvent
	}
	if errorMessage != nil {
		e.ErrorMessage = *errorMessage
	}
	if len(conditionsJSON) > 0 {
		_ = json.Unmarshal(conditionsJSON, &e.ConditionsSnapshot)
	}
	if len(actionsResultJSON) > 0 {
		_ = json.Unmarshal(actionsResultJSON, &e.ActionsResult)
	}
	return nil
}

// Create inserta un registro de ejecucion de regla.
func (r *ruleExecutionRepository) Create(ctx context.Context, e *domain.RuleExecution) error {
	conditionsJSON, _ := json.Marshal(e.ConditionsSnapshot)
	actionsJSON, _ := json.Marshal(e.ActionsResult)

	return withTenant(ctx, r.db, e.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO rule_executions
				(id, tenant_id, rule_id, customer_id, triggered_at, trigger_event,
				 conditions_snapshot, actions_result, status, error_message)
			VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7, $8, $9)
		`, e.ID, e.TenantID, e.RuleID, e.CustomerID,
			e.TriggerEvent, conditionsJSON, actionsJSON,
			e.Status, e.ErrorMessage)
		if err != nil {
			return fmt.Errorf("ruleExecutionRepository.Create: %w", err)
		}
		return nil
	})
}

// ListByRule retorna las ultimas ejecuciones de una regla.
func (r *ruleExecutionRepository) ListByRule(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var result []*domain.RuleExecution
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, rule_id, customer_id, triggered_at, trigger_event,
			       conditions_snapshot, actions_result, status, error_message
			FROM rule_executions
			WHERE tenant_id = $1 AND rule_id = $2
			ORDER BY triggered_at DESC
			LIMIT $3
		`, tenantID, ruleID, limit)
		if err != nil {
			return fmt.Errorf("ruleExecutionRepository.ListByRule: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			e := &domain.RuleExecution{}
			if err := scanRuleExecution(rows, e); err != nil {
				return fmt.Errorf("ruleExecutionRepository.ListByRule: scan: %w", err)
			}
			result = append(result, e)
		}
		return rows.Err()
	})
	return result, err
}

// ListByCustomer retorna las ultimas ejecuciones de reglas para un cliente.
func (r *ruleExecutionRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var result []*domain.RuleExecution
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, rule_id, customer_id, triggered_at, trigger_event,
			       conditions_snapshot, actions_result, status, error_message
			FROM rule_executions
			WHERE tenant_id = $1 AND customer_id = $2
			ORDER BY triggered_at DESC
			LIMIT $3
		`, tenantID, customerID, limit)
		if err != nil {
			return fmt.Errorf("ruleExecutionRepository.ListByCustomer: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			e := &domain.RuleExecution{}
			if err := scanRuleExecution(rows, e); err != nil {
				return fmt.Errorf("ruleExecutionRepository.ListByCustomer: scan: %w", err)
			}
			result = append(result, e)
		}
		return rows.Err()
	})
	return result, err
}
```

### Step 6.3 — Service: rule.go

- [ ] Create `apps/api/internal/service/rule.go`

```go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type ruleSvc struct {
	repo domain.RuleRepository
}

// NewRuleSvc crea el servicio de reglas (CRUD only en Phase 6A).
func NewRuleSvc(repo domain.RuleRepository) domain.RuleSvc {
	return &ruleSvc{repo: repo}
}

// Create crea una nueva regla.
func (s *ruleSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	cooldown := input.CooldownHours
	if cooldown == 0 {
		cooldown = 24
	}

	// Validacion: reglas tipo 'event' requieren trigger_event
	if input.TriggerType == "event" && input.TriggerEvent == "" {
		return nil, fmt.Errorf("ruleSvc.Create: trigger_event es requerido para reglas tipo 'event': %w", domain.ErrValidation)
	}
	// Validacion: reglas tipo 'temporal' requieren trigger_schedule
	if input.TriggerType == "temporal" && input.TriggerSchedule == nil {
		return nil, fmt.Errorf("ruleSvc.Create: trigger_schedule es requerido para reglas tipo 'temporal': %w", domain.ErrValidation)
	}

	rule := &domain.Rule{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            input.Name,
		Description:     input.Description,
		TriggerType:     input.TriggerType,
		TriggerEvent:    input.TriggerEvent,
		TriggerSchedule: input.TriggerSchedule,
		Conditions:      input.Conditions,
		Actions:         input.Actions,
		IsActive:        isActive,
		IsTemplate:      false,
		CooldownHours:   cooldown,
		Priority:        input.Priority,
	}
	if rule.Conditions == nil {
		rule.Conditions = []domain.RuleCondition{}
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleSvc.Create: %w", err)
	}
	return rule, nil
}

// List retorna todas las reglas del tenant.
func (s *ruleSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	return s.repo.List(ctx, tenantID)
}

// GetByID retorna una regla por ID.
func (s *ruleSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Rule, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Update actualiza una regla existente.
func (s *ruleSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
	rule, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		rule.Name = input.Name
	}
	rule.Description = input.Description
	rule.TriggerType = input.TriggerType
	rule.TriggerEvent = input.TriggerEvent
	rule.TriggerSchedule = input.TriggerSchedule
	if input.Conditions != nil {
		rule.Conditions = input.Conditions
	}
	if input.Actions != nil {
		rule.Actions = input.Actions
	}
	if input.IsActive != nil {
		rule.IsActive = *input.IsActive
	}
	if input.CooldownHours > 0 {
		rule.CooldownHours = input.CooldownHours
	}
	rule.Priority = input.Priority

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleSvc.Update: %w", err)
	}
	return rule, nil
}

// Delete elimina una regla.
func (s *ruleSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}
```

### Step 6.4 — Handler: rules.go

- [ ] Create `apps/api/internal/handler/rules.go`

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// RuleHandler maneja los endpoints de reglas de automatizacion.
type RuleHandler struct {
	svc        domain.RuleSvc
	execRepo   domain.RuleExecutionRepository
	validate   *validator.Validate
}

// NewRuleHandler crea el handler de reglas.
func NewRuleHandler(svc domain.RuleSvc, execRepo domain.RuleExecutionRepository) *RuleHandler {
	return &RuleHandler{svc: svc, execRepo: execRepo, validate: validator.New()}
}

// List GET /rules
func (h *RuleHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	list, err := h.svc.List(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

// Create POST /rules
func (h *RuleHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.RuleInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos invalidos: " + validationMessage(err)})
	}

	rule, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(rule)
}

// GetByID GET /rules/:id
func (h *RuleHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	rule, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(rule)
}

// Update PATCH /rules/:id
func (h *RuleHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	var input domain.RuleInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON invalido"})
	}

	rule, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(rule)
}

// Delete DELETE /rules/:id
func (h *RuleHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	if err := h.svc.Delete(c.Context(), tenantID, id); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// ListExecutions GET /rules/:id/executions
func (h *RuleHandler) ListExecutions(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID invalido"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	execs, err := h.execRepo.ListByRule(c.Context(), tenantID, id, limit)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": execs})
}
```

### Step 6.5 — Commit rules

- [ ] Commit: `feat(api): add rules CRUD with executions log (repo, service, handler)`

---

## Task 7: Customer CRM Extensions

Update the existing Customer struct and repository to support the new CRM fields.

### Step 7.1 — Update Customer struct in types.go

- [ ] In `apps/api/internal/domain/types.go`, update the `Customer` struct to add the new CRM fields:

Find the existing Customer struct and replace it with:

```go
// Customer es un cliente del negocio.
type Customer struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	Name              string     `json:"name"`
	Phone             string     `json:"phone,omitempty"`
	Email             string     `json:"email,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	Tags              []string   `json:"tags"`
	WaOptIn           bool       `json:"wa_opt_in"`
	TotalVisits       int        `json:"total_visits"`
	StageID           *uuid.UUID `json:"stage_id,omitempty"`
	LastVisitAt       *time.Time `json:"last_visit_at,omitempty"`
	NextRecallAt      *time.Time `json:"next_recall_at,omitempty"`
	LifetimeValue     float64    `json:"lifetime_value"`
	AcquisitionSource string     `json:"acquisition_source,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
```

### Step 7.2 — Update scanCustomer in customers.go

- [ ] In `apps/api/internal/repository/customers.go`, update the `scanCustomer` function and all queries to include the new columns.

Update `scanCustomer`:

```go
func scanCustomer(row interface{ Scan(dest ...any) error }, c *domain.Customer) error {
	var email, notes, acquisitionSource *string
	if err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Phone, &email,
		&notes, &c.Tags, &c.WaOptIn, &c.TotalVisits,
		&c.StageID, &c.LastVisitAt, &c.NextRecallAt,
		&c.LifetimeValue, &acquisitionSource,
		&c.CreatedAt,
	); err != nil {
		return err
	}
	if email != nil {
		c.Email = *email
	}
	if notes != nil {
		c.Notes = *notes
	}
	if acquisitionSource != nil {
		c.AcquisitionSource = *acquisitionSource
	}
	return nil
}
```

Update the column list in `FindOrCreateByPhone` SELECT:
```sql
SELECT id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
       stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
       created_at
FROM customers
WHERE tenant_id = $1 AND phone = $2
```

Update the INSERT ... RETURNING in `FindOrCreateByPhone`:
```sql
INSERT INTO customers (id, tenant_id, name, phone, wa_opt_in, created_at)
VALUES ($1, $2, $3, $4, TRUE, NOW())
RETURNING id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
          stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
          created_at
```

Update `GetByID` SELECT:
```sql
SELECT id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
       stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
       created_at
FROM customers
WHERE tenant_id = $1 AND id = $2
```

Update `List` SELECT (both branches — with and without search):
```sql
SELECT id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
       stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
       created_at
FROM customers
WHERE tenant_id = $1 ...
```

### Step 7.3 — Commit customer extensions

- [ ] Commit: `feat(api): add CRM fields to Customer (stage, LTV, recall, acquisition source)`

---

## Task 8: Appointment treatment_id Extension

### Step 8.1 — Update Appointment struct in types.go

- [ ] In `apps/api/internal/domain/types.go`, add `TreatmentID` to the `Appointment` struct:

Add after `ServiceID uuid.UUID`:
```go
	TreatmentID        *uuid.UUID `json:"treatment_id,omitempty"`
```

### Step 8.2 — Update appointment repository scan

- [ ] In `apps/api/internal/repository/appointments.go`, update the scan functions and INSERT query to include `treatment_id`.

This requires finding where `Appointment` is scanned and adding `&a.TreatmentID` to the Scan call. Also update the `Create` method to include `treatment_id` in the INSERT.

**Note:** The exact changes depend on the current scan implementation. The agent executing this task should:
1. Read `apps/api/internal/repository/appointments.go`
2. Find all `Scan` calls that produce an `Appointment`
3. Add `treatment_id` to the column list and scan target
4. Add `treatment_id` to the `Create` INSERT

### Step 8.3 — Commit appointment extension

- [ ] Commit: `feat(api): add treatment_id FK to Appointment`

---

## Task 9: Dental Seed Data

### Step 9.1 — Create seed file

- [ ] Create `apps/api/internal/seed/dental.go`

```go
package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultDentalPipelineStages son las etapas por defecto para un pipeline dental.
var DefaultDentalPipelineStages = []struct {
	Name     string
	Position int
	Color    string
}{
	{Name: "Nuevo Paciente", Position: 0, Color: "#6366f1"},
	{Name: "Consulta Inicial", Position: 1, Color: "#8b5cf6"},
	{Name: "Plan de Tratamiento", Position: 2, Color: "#a855f7"},
	{Name: "En Tratamiento", Position: 3, Color: "#f59e0b"},
	{Name: "Seguimiento", Position: 4, Color: "#10b981"},
	{Name: "Completado", Position: 5, Color: "#22c55e"},
	{Name: "Inactivo", Position: 6, Color: "#6b7280"},
}

// SeedDentalPipeline inserta las etapas por defecto del pipeline dental para un tenant.
func SeedDentalPipeline(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed.SeedDentalPipeline: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("seed.SeedDentalPipeline: set_config: %w", err)
	}

	for i, s := range DefaultDentalPipelineStages {
		isDefault := i == 0 // "Nuevo Paciente" es la etapa por defecto
		_, err := tx.Exec(ctx, `
			INSERT INTO pipeline_stages (id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW())
			ON CONFLICT (tenant_id, position) DO NOTHING
		`, uuid.New(), tenantID, s.Name, s.Position, s.Color, isDefault)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalPipeline: stage '%s': %w", s.Name, err)
		}
	}

	return tx.Commit(ctx)
}

// DefaultDentalRuleTemplates son las plantillas de reglas para clinicas dentales.
var DefaultDentalRuleTemplates = []struct {
	Name         string
	Description  string
	TriggerType  string
	TriggerEvent string
	ActionsJSON  string
	TemplateKey  string
}{
	{
		Name:         "Recordatorio de recall (6 meses)",
		Description:  "Enviar WhatsApp cuando el paciente no ha visitado en 6 meses",
		TriggerType:  "temporal",
		TemplateKey:  "dental_recall_6m",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"recall_reminder","params":{"interval_days":180}}]`,
	},
	{
		Name:         "Seguimiento post-tratamiento",
		Description:  "Crear tarea de seguimiento cuando un tratamiento se completa",
		TriggerType:  "event",
		TriggerEvent: "treatment.completed",
		TemplateKey:  "dental_post_treatment",
		ActionsJSON:  `[{"type":"create_task","template":"post_treatment_followup","params":{"due_days":7}}]`,
	},
	{
		Name:         "Paciente no asistio",
		Description:  "Crear tarea de contacto cuando un paciente no asiste a su cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.no_show",
		TemplateKey:  "dental_no_show",
		ActionsJSON:  `[{"type":"create_task","template":"no_show_followup","params":{"due_days":1}}]`,
	},
}

// SeedDentalRuleTemplates inserta las plantillas de reglas globales (is_template = true).
// Se ejecuta una sola vez al inicializar el sistema, no por tenant.
func SeedDentalRuleTemplates(ctx context.Context, pool *pgxpool.Pool) error {
	for _, r := range DefaultDentalRuleTemplates {
		// Verificar si ya existe por template_key
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM rules WHERE is_template = TRUE AND template_key = $1)
		`, r.TemplateKey).Scan(&exists)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalRuleTemplates: check '%s': %w", r.TemplateKey, err)
		}
		if exists {
			continue
		}

		// Templates no necesitan tenant_id valido pero la columna es NOT NULL.
		// Usar un UUID fijo para templates globales.
		templateTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000000")

		_, err = pool.Exec(ctx, `
			INSERT INTO rules
				(id, tenant_id, name, description, trigger_type, trigger_event,
				 conditions, actions, is_active, is_template, template_key,
				 cooldown_hours, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, '[]'::jsonb, $7::jsonb, TRUE, TRUE, $8, 24, 0, NOW(), NOW())
		`, uuid.New(), templateTenantID, r.Name, r.Description,
			r.TriggerType, r.TriggerEvent, r.ActionsJSON, r.TemplateKey)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalRuleTemplates: insert '%s': %w", r.TemplateKey, err)
		}
	}
	return nil
}
```

### Step 9.2 — Commit seed data

- [ ] Commit: `feat(api): add dental seed data for pipeline stages and rule templates`

---

## Task 10: Route Registration and DI Wiring

### Step 10.1 — Wire new repos, services, handlers in main.go

- [ ] In `apps/api/cmd/server/main.go`, add the new repository instantiations after the existing repos (around line 116):

```go
	pipelineRepo    := repository.NewPipelineStageRepository(pool)
	treatmentRepo   := repository.NewTreatmentRepository(pool)
	taskRepo        := repository.NewTaskRepository(pool)
	ruleRepo        := repository.NewRuleRepository(pool)
	ruleExecRepo    := repository.NewRuleExecutionRepository(pool)
```

- [ ] Add new service instantiations after the existing services (around line 132):

```go
	pipelineSvc    := service.NewPipelineStageSvc(pipelineRepo)
	treatmentSvc   := service.NewTreatmentSvc(treatmentRepo)
	taskSvc        := service.NewTaskSvc(taskRepo)
	ruleSvc        := service.NewRuleSvc(ruleRepo)
```

- [ ] Add new handler instantiations after the existing handlers (around line 145):

```go
	pipelineHandler  := handler.NewPipelineStageHandler(pipelineSvc)
	treatmentHandler := handler.NewTreatmentHandler(treatmentSvc)
	taskHandler      := handler.NewTaskHandler(taskSvc)
	ruleHandler      := handler.NewRuleHandler(ruleSvc, ruleExecRepo)
```

### Step 10.2 — Register routes

- [ ] In `apps/api/cmd/server/main.go`, add route groups after the existing `blocks` group (around line 423):

```go
	// Pipeline Stages
	stages := protected.Group("/pipeline-stages")
	stages.Get("/", pipelineHandler.List)
	stages.Post("/", pipelineHandler.Create)
	stages.Put("/reorder", pipelineHandler.Reorder)
	stages.Get("/:id", pipelineHandler.GetByID)
	stages.Patch("/:id", pipelineHandler.Update)
	stages.Delete("/:id", pipelineHandler.Delete)

	// Treatments
	treatments := protected.Group("/treatments")
	treatments.Get("/", treatmentHandler.List)
	treatments.Post("/", treatmentHandler.Create)
	treatments.Get("/:id", treatmentHandler.GetByID)
	treatments.Patch("/:id", treatmentHandler.Update)
	treatments.Patch("/:id/status", treatmentHandler.UpdateStatus)

	// Tasks
	tasks := protected.Group("/tasks")
	tasks.Get("/", taskHandler.List)
	tasks.Post("/", taskHandler.Create)
	tasks.Get("/:id", taskHandler.GetByID)
	tasks.Patch("/:id", taskHandler.Update)
	tasks.Post("/:id/complete", taskHandler.Complete)
	tasks.Post("/:id/dismiss", taskHandler.Dismiss)

	// Rules
	rules := protected.Group("/rules")
	rules.Get("/", ruleHandler.List)
	rules.Post("/", ruleHandler.Create)
	rules.Get("/:id", ruleHandler.GetByID)
	rules.Patch("/:id", ruleHandler.Update)
	rules.Delete("/:id", ruleHandler.Delete)
	rules.Get("/:id/executions", ruleHandler.ListExecutions)
```

### Step 10.3 — Commit route registration

- [ ] Commit: `feat(api): wire CRM repos, services, handlers, and routes in main.go`

---

## Summary of Files Created/Modified

### New files (11)
| File | Purpose |
|------|---------|
| `apps/api/db/migrations/017_pipeline_stages.sql` | Pipeline stages table + RLS |
| `apps/api/db/migrations/018_treatments.sql` | Treatments table + RLS + appointment FK |
| `apps/api/db/migrations/019_rules.sql` | Rules + rule_executions tables + RLS |
| `apps/api/db/migrations/020_tasks.sql` | Tasks table + RLS |
| `apps/api/db/migrations/021_customer_crm_extensions.sql` | ALTER customers with CRM fields |
| `apps/api/internal/repository/pipeline_stages.go` | Pipeline stages repo |
| `apps/api/internal/repository/treatments.go` | Treatments repo |
| `apps/api/internal/repository/tasks.go` | Tasks repo |
| `apps/api/internal/repository/rules.go` | Rules repo |
| `apps/api/internal/repository/rule_executions.go` | Rule executions repo |
| `apps/api/internal/seed/dental.go` | Dental seed data |

### New files (5) — continued
| File | Purpose |
|------|---------|
| `apps/api/internal/service/pipeline_stage.go` | Pipeline stages service |
| `apps/api/internal/service/treatment.go` | Treatments service |
| `apps/api/internal/service/task.go` | Tasks service |
| `apps/api/internal/service/rule.go` | Rules service |
| `apps/api/internal/handler/pipeline_stages.go` | Pipeline stages handler |

### New files (3) — continued
| File | Purpose |
|------|---------|
| `apps/api/internal/handler/treatments.go` | Treatments handler |
| `apps/api/internal/handler/tasks.go` | Tasks handler |
| `apps/api/internal/handler/rules.go` | Rules handler |

### Modified files (5)
| File | Change |
|------|--------|
| `apps/api/internal/domain/types.go` | Add PipelineStage, Treatment, Task, Rule, etc. + extend Customer/Appointment |
| `apps/api/internal/domain/interfaces.go` | Add repo + service interfaces for all new entities |
| `apps/api/internal/domain/errors.go` | Add ErrInvalidStatusTransition |
| `apps/api/internal/handler/errors.go` | Add error mapping for ErrInvalidStatusTransition |
| `apps/api/cmd/server/main.go` | Wire repos, services, handlers, routes |
| `apps/api/internal/repository/customers.go` | Update scanCustomer and queries for CRM fields |
| `apps/api/internal/repository/appointments.go` | Add treatment_id to scan and INSERT |

### API Endpoints Added (21)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/pipeline-stages` | List stages |
| POST | `/api/v1/pipeline-stages` | Create stage |
| PUT | `/api/v1/pipeline-stages/reorder` | Reorder stages |
| GET | `/api/v1/pipeline-stages/:id` | Get stage |
| PATCH | `/api/v1/pipeline-stages/:id` | Update stage |
| DELETE | `/api/v1/pipeline-stages/:id` | Delete stage |
| GET | `/api/v1/treatments` | List treatments (filters: customer_id, professional_id, status) |
| POST | `/api/v1/treatments` | Create treatment |
| GET | `/api/v1/treatments/:id` | Get treatment |
| PATCH | `/api/v1/treatments/:id` | Update treatment |
| PATCH | `/api/v1/treatments/:id/status` | Change treatment status |
| GET | `/api/v1/tasks` | List tasks (filters: assigned_to, customer_id, status, due_before) |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks/:id` | Get task |
| PATCH | `/api/v1/tasks/:id` | Update task |
| POST | `/api/v1/tasks/:id/complete` | Complete task |
| POST | `/api/v1/tasks/:id/dismiss` | Dismiss task |
| GET | `/api/v1/rules` | List rules |
| POST | `/api/v1/rules` | Create rule |
| GET | `/api/v1/rules/:id` | Get rule |
| PATCH | `/api/v1/rules/:id` | Update rule |
| DELETE | `/api/v1/rules/:id` | Delete rule |
| GET | `/api/v1/rules/:id/executions` | List rule executions |
