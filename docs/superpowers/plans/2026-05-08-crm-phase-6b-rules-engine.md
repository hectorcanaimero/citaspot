# CRM Phase 6B: Rules Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the rules engine with event-driven and temporal execution paths, condition evaluator, action executors, and event publishing from existing services.

**Architecture:** Hybrid engine (events + cron). RulesEventWorker consumes from `rules.events` RabbitMQ queue. TemporalRulesWorker runs every 15 min. Both use shared engine core (evaluator + executor + actions). Events published from appointment/treatment/customer services.

**Tech Stack:** Go 1.24, RabbitMQ (amqp091-go), pgx v5, Fiber v2

---

## Task 1: Domain Event Types + Queue Declaration

**Files to modify:**
- `apps/api/internal/domain/types.go`
- `apps/api/internal/client/rabbitmq/publisher.go`

### 1A. Add `RuleEvent` struct to `domain/types.go`

Append after the `RuleExecution` struct (after line 650):

```go
// ── Rule Events ──────────────────────────────────────────────────────────────

// RuleEvent es un evento de dominio publicado en la cola rules.events.
// Los servicios publican estos eventos cuando ocurren acciones relevantes
// para el motor de reglas (cita completada, cliente creado, etc.).
type RuleEvent struct {
	TenantID   uuid.UUID      `json:"tenant_id"`
	EventType  string         `json:"event_type"`  // "appointment.completed", "customer.created", etc.
	CustomerID uuid.UUID      `json:"customer_id"`
	EntityID   uuid.UUID      `json:"entity_id"`   // ID de la entidad que generó el evento
	EntityType string         `json:"entity_type"` // "appointment", "customer", "treatment"
	Payload    map[string]any `json:"payload"`      // datos adicionales del evento
	Timestamp  time.Time      `json:"timestamp"`
}
```

### 1B. Add `rules.events` queue to publisher.go

In `apps/api/internal/client/rabbitmq/publisher.go`, add `"rules.events"` to the queues slice in the `connect()` method:

```go
// Declarar todas las colas del sistema (idempotente)
queues := []string{
	"wa.messages.inbound",
	"wa.messages.outbound",
	"notifications.reminders",
	"knowledge.vectorize",
	"notifications.review",
	"rules.events",
}
```

### 1C. Add new repository interfaces to `domain/interfaces.go`

Append to the `CustomerRepository` interface:

```go
// CustomerRepository operaciones DB para clientes.
type CustomerRepository interface {
	FindOrCreateByPhone(ctx context.Context, tenantID uuid.UUID, name, phone string) (*Customer, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)
	List(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*Customer, error)
	// UpdateStage actualiza la etapa del pipeline de un cliente.
	UpdateStage(ctx context.Context, tenantID, customerID, stageID uuid.UUID) error
	// UpdateField actualiza un campo específico del cliente (next_recall_at, etc.).
	UpdateField(ctx context.Context, tenantID, customerID uuid.UUID, field string, value any) error
}
```

Add a `HasRecentExecution` method to `RuleExecutionRepository`:

```go
// RuleExecutionRepository operaciones DB para logs de ejecucion de reglas.
type RuleExecutionRepository interface {
	Create(ctx context.Context, e *RuleExecution) error
	ListByRule(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*RuleExecution, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit int) ([]*RuleExecution, error)
	// HasRecentExecution verifica si una regla se ejecutó para un cliente dentro del cooldown.
	HasRecentExecution(ctx context.Context, tenantID, ruleID, customerID uuid.UUID, cooldownHours int) (bool, error)
}
```

Add a `ListActiveByTriggerEvent` method to `RuleRepository`:

```go
// RuleRepository operaciones DB para reglas de automatizacion.
type RuleRepository interface {
	Create(ctx context.Context, r *Rule) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Rule, error)
	Update(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	ListTemplates(ctx context.Context) ([]*Rule, error)
	// ListActiveByTriggerEvent retorna reglas activas de un tenant que coinciden con el evento.
	ListActiveByTriggerEvent(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*Rule, error)
	// ListActiveTemporal retorna todas las reglas temporales activas de todos los tenants.
	// Nota: opera sin RLS — necesita acceso global como ReminderRepository.
	ListActiveTemporal(ctx context.Context) ([]*Rule, error)
}
```

### 1D. Add new repository implementations

**File: `apps/api/internal/repository/customers.go`** -- add these two methods:

```go
// UpdateStage actualiza la etapa del pipeline de un cliente.
func (r *customerRepository) UpdateStage(ctx context.Context, tenantID, customerID, stageID uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE customers
			SET stage_id = $3
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, customerID, stageID)
		if err != nil {
			return fmt.Errorf("customerRepository.UpdateStage: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// UpdateField actualiza un campo específico del cliente.
// Campos permitidos: next_recall_at, last_visit_at, total_visits, lifetime_value.
func (r *customerRepository) UpdateField(ctx context.Context, tenantID, customerID uuid.UUID, field string, value any) error {
	// Whitelist de campos permitidos para evitar inyección SQL
	allowed := map[string]bool{
		"next_recall_at": true,
		"last_visit_at":  true,
		"total_visits":   true,
		"lifetime_value": true,
	}
	if !allowed[field] {
		return fmt.Errorf("customerRepository.UpdateField: campo '%s' no permitido: %w", field, domain.ErrValidation)
	}

	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := fmt.Sprintf(`UPDATE customers SET %s = $3 WHERE tenant_id = $1 AND id = $2`, field)
		tag, err := tx.Exec(ctx, query, tenantID, customerID, value)
		if err != nil {
			return fmt.Errorf("customerRepository.UpdateField: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
```

**File: `apps/api/internal/repository/rule_executions.go`** -- add this method:

```go
// HasRecentExecution verifica si una regla se ejecutó para un cliente dentro del cooldown.
func (r *ruleExecutionRepository) HasRecentExecution(ctx context.Context, tenantID, ruleID, customerID uuid.UUID, cooldownHours int) (bool, error) {
	var exists bool
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM rule_executions
				WHERE tenant_id = $1
				  AND rule_id = $2
				  AND customer_id = $3
				  AND status = 'success'
				  AND triggered_at > NOW() - INTERVAL '1 hour' * $4
			)
		`, tenantID, ruleID, customerID, cooldownHours).Scan(&exists)
	})
	if err != nil {
		return false, fmt.Errorf("ruleExecutionRepository.HasRecentExecution: %w", err)
	}
	return exists, nil
}
```

**File: `apps/api/internal/repository/rules.go`** -- add these two methods:

```go
// ListActiveByTriggerEvent retorna reglas activas de un tenant que coinciden con el evento.
func (r *ruleRepository) ListActiveByTriggerEvent(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*domain.Rule, error) {
	var result []*domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1
			  AND is_active = TRUE
			  AND is_template = FALSE
			  AND trigger_type = 'event'
			  AND trigger_event = $2
			ORDER BY priority DESC
		`, tenantID, triggerEvent)
		if err != nil {
			return fmt.Errorf("ruleRepository.ListActiveByTriggerEvent: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.ListActiveByTriggerEvent: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}

// ListActiveTemporal retorna todas las reglas temporales activas de todos los tenants.
// Opera SIN RLS para acceso global — igual que ReminderRepository.
func (r *ruleRepository) ListActiveTemporal(ctx context.Context) ([]*domain.Rule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+ruleColumns+`
		FROM rules
		WHERE is_active = TRUE
		  AND is_template = FALSE
		  AND trigger_type = 'temporal'
		ORDER BY tenant_id, priority DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("ruleRepository.ListActiveTemporal: query: %w", err)
	}
	defer rows.Close()

	var result []*domain.Rule
	for rows.Next() {
		rule := &domain.Rule{}
		if err := scanRule(rows, rule); err != nil {
			return nil, fmt.Errorf("ruleRepository.ListActiveTemporal: scan: %w", err)
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}
```

### Verification

- `go build ./...` compiles without errors
- Queue `rules.events` is declared on publisher startup
- New methods on existing repositories compile and follow `withTenant` pattern
- `ListActiveTemporal` operates without RLS (global query, same as `ReminderRepository.FindDueReminders`)

---

## Task 2: Engine - Condition Evaluator

**File to create:** `apps/api/internal/engine/evaluator.go`

This is a pure function module with zero side effects -- it evaluates a list of `RuleCondition` against an event context map.

```go
// Package engine contiene el motor de evaluación y ejecución de reglas CRM.
package engine

import (
	"fmt"
	"strings"

	"github.com/citaspot/api/internal/domain"
)

// EvaluateConditions evalúa todas las condiciones contra un contexto.
// Retorna true si TODAS las condiciones se cumplen (AND lógico).
// Si no hay condiciones, retorna true (regla sin filtros = siempre ejecuta).
func EvaluateConditions(conditions []domain.RuleCondition, ctx map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	for _, cond := range conditions {
		if !evaluateCondition(cond, ctx) {
			return false
		}
	}
	return true
}

// evaluateCondition evalúa una condición individual.
func evaluateCondition(cond domain.RuleCondition, ctx map[string]any) bool {
	actual, ok := resolveField(cond.Field, ctx)
	if !ok {
		// Campo no existe en el contexto — la condición no se cumple
		return false
	}

	switch cond.Op {
	case "eq":
		return compareEq(actual, cond.Value)
	case "neq":
		return !compareEq(actual, cond.Value)
	case "gt":
		return compareNumeric(actual, cond.Value) > 0
	case "lt":
		return compareNumeric(actual, cond.Value) < 0
	case "gte":
		return compareNumeric(actual, cond.Value) >= 0
	case "lte":
		return compareNumeric(actual, cond.Value) <= 0
	case "contains":
		return compareContains(actual, cond.Value)
	case "in":
		return compareIn(actual, cond.Value)
	default:
		return false
	}
}

// resolveField resuelve un campo con notación de punto: "customer.total_visits"
// En un mapa plano o anidado.
func resolveField(field string, ctx map[string]any) (any, bool) {
	parts := strings.Split(field, ".")
	var current any = ctx

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// compareEq compara igualdad con conversión de tipos.
func compareEq(actual, expected any) bool {
	// Convertir ambos a la misma representación
	a := normalizeValue(actual)
	e := normalizeValue(expected)
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", e)
}

// compareNumeric compara dos valores numéricos.
// Retorna -1 si a < b, 0 si a == b, 1 si a > b.
// Si alguno no es numérico, retorna -2 (condición falsa para gt/lt/etc.).
func compareNumeric(actual, expected any) int {
	a := toFloat64(actual)
	b := toFloat64(expected)
	if a == nil || b == nil {
		return -2 // no comparable
	}
	switch {
	case *a < *b:
		return -1
	case *a > *b:
		return 1
	default:
		return 0
	}
}

// compareContains verifica si el string actual contiene el substring esperado.
func compareContains(actual, expected any) bool {
	aStr := fmt.Sprintf("%v", actual)
	eStr := fmt.Sprintf("%v", expected)
	return strings.Contains(strings.ToLower(aStr), strings.ToLower(eStr))
}

// compareIn verifica si el valor actual está en la lista esperada.
func compareIn(actual, expected any) bool {
	list, ok := expected.([]any)
	if !ok {
		// Intentar como slice de strings
		strList, ok := expected.([]string)
		if !ok {
			return false
		}
		for _, item := range strList {
			if compareEq(actual, item) {
				return true
			}
		}
		return false
	}
	for _, item := range list {
		if compareEq(actual, item) {
			return true
		}
	}
	return false
}

// normalizeValue normaliza un valor para comparación uniforme.
func normalizeValue(v any) any {
	switch val := v.(type) {
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case bool:
		return val
	default:
		return v
	}
}

// toFloat64 convierte un valor a float64 si es numérico.
func toFloat64(v any) *float64 {
	var f float64
	switch val := v.(type) {
	case float64:
		f = val
	case float32:
		f = float64(val)
	case int:
		f = float64(val)
	case int32:
		f = float64(val)
	case int64:
		f = float64(val)
	case string:
		// Intentar parsear string numérico
		n, err := fmt.Sscanf(val, "%f", &f)
		if err != nil || n != 1 {
			return nil
		}
	default:
		return nil
	}
	return &f
}
```

### Verification

- `go build ./internal/engine/...` compiles
- `go test ./internal/engine/...` passes (unit tests for all operators)
- Pure functions, no dependencies on DB or external services

---

## Task 3: Engine - Action Executors

**Files to create:**
- `apps/api/internal/engine/action.go` (interface + registry)
- `apps/api/internal/engine/actions/whatsapp.go`
- `apps/api/internal/engine/actions/task.go`
- `apps/api/internal/engine/actions/stage.go`
- `apps/api/internal/engine/actions/field.go`

### 3A. Action interface and params: `engine/action.go`

```go
package engine

import (
	"context"

	"github.com/google/uuid"
)

// ActionParams contiene los datos necesarios para ejecutar una acción.
type ActionParams struct {
	TenantID   uuid.UUID
	TenantSlug string // slug del tenant para identificar instancia WA
	CustomerID uuid.UUID
	EntityID   uuid.UUID      // ID de la entidad que disparó el evento
	EntityType string         // "appointment", "treatment", "customer"
	Template   string         // template del mensaje (para send_whatsapp)
	Params     map[string]any // parámetros adicionales de la acción
	Context    map[string]any // contexto completo del evento (para template rendering)
}

// ActionExecutor ejecuta una acción específica del motor de reglas.
type ActionExecutor interface {
	// Type retorna el identificador del tipo de acción: "send_whatsapp", "create_task", etc.
	Type() string
	// Execute ejecuta la acción con los parámetros dados.
	Execute(ctx context.Context, params ActionParams) error
}

// ActionRegistry contiene los ejecutores registrados por tipo.
type ActionRegistry struct {
	executors map[string]ActionExecutor
}

// NewActionRegistry crea un registro vacío.
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{executors: make(map[string]ActionExecutor)}
}

// Register registra un ejecutor para un tipo de acción.
func (r *ActionRegistry) Register(executor ActionExecutor) {
	r.executors[executor.Type()] = executor
}

// Get retorna el ejecutor para un tipo de acción, o nil si no existe.
func (r *ActionRegistry) Get(actionType string) ActionExecutor {
	return r.executors[actionType]
}
```

### 3B. SendWhatsApp action: `engine/actions/whatsapp.go`

```go
// Package actions contiene los ejecutores de acciones del motor de reglas.
package actions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

// SendWhatsAppAction envía un mensaje de WhatsApp al cliente.
type SendWhatsAppAction struct {
	waClient domain.WAClient
}

// NewSendWhatsAppAction crea la acción de envío de WhatsApp.
func NewSendWhatsAppAction(waClient domain.WAClient) *SendWhatsAppAction {
	return &SendWhatsAppAction{waClient: waClient}
}

func (a *SendWhatsAppAction) Type() string {
	return "send_whatsapp"
}

func (a *SendWhatsAppAction) Execute(ctx context.Context, params engine.ActionParams) error {
	if a.waClient == nil {
		slog.Warn("SendWhatsAppAction: waClient is nil, skipping")
		return nil
	}

	// Obtener el teléfono del cliente desde el contexto
	phone, ok := params.Context["customer_phone"].(string)
	if !ok || phone == "" {
		return fmt.Errorf("SendWhatsAppAction.Execute: customer_phone no disponible en contexto")
	}

	// Verificar que la instancia WA está conectada
	connected, err := a.waClient.IsConnected(ctx, params.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("SendWhatsAppAction.Execute: instancia '%s' no conectada", params.TenantSlug)
	}

	// Renderizar el template con las variables del contexto
	text := renderTemplate(params.Template, params.Context)
	if text == "" {
		return fmt.Errorf("SendWhatsAppAction.Execute: template vacío tras renderizar")
	}

	_, err = a.waClient.SendText(ctx, params.TenantSlug, phone, text)
	if err != nil {
		return fmt.Errorf("SendWhatsAppAction.Execute: %w", err)
	}

	slog.Info("SendWhatsAppAction: mensaje enviado", "tenant", params.TenantSlug, "phone", phone)
	return nil
}

// renderTemplate reemplaza variables {{key}} en el template con valores del contexto.
// Soporta notación plana: {{customer_name}}, {{service_name}}, etc.
func renderTemplate(tmpl string, ctx map[string]any) string {
	result := tmpl
	for key, val := range ctx {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}
	return result
}
```

### 3C. CreateTask action: `engine/actions/task.go`

```go
package actions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

// CreateTaskAction crea una tarea en el sistema.
type CreateTaskAction struct {
	taskRepo domain.TaskRepository
}

// NewCreateTaskAction crea la acción de creación de tarea.
func NewCreateTaskAction(taskRepo domain.TaskRepository) *CreateTaskAction {
	return &CreateTaskAction{taskRepo: taskRepo}
}

func (a *CreateTaskAction) Type() string {
	return "create_task"
}

func (a *CreateTaskAction) Execute(ctx context.Context, params engine.ActionParams) error {
	title := renderTemplate(params.Template, params.Context)
	if title == "" {
		title = "Tarea automática"
	}

	// Descripción opcional desde params
	description := ""
	if desc, ok := params.Params["description"].(string); ok {
		description = renderTemplate(desc, params.Context)
	}

	// Assigned to (opcional)
	var assignedTo *uuid.UUID
	if assignedStr, ok := params.Params["assigned_to"].(string); ok && assignedStr != "" {
		id, err := uuid.Parse(assignedStr)
		if err == nil {
			assignedTo = &id
		}
	}

	// Due in days (opcional, por defecto 7 días)
	dueInDays := 7
	if d, ok := params.Params["due_in_days"].(float64); ok && d > 0 {
		dueInDays = int(d)
	}
	dueAt := time.Now().Add(time.Duration(dueInDays) * 24 * time.Hour)

	// Rule ID para trazabilidad
	var ruleID *uuid.UUID
	if ruleStr, ok := params.Params["rule_id"].(string); ok && ruleStr != "" {
		id, err := uuid.Parse(ruleStr)
		if err == nil {
			ruleID = &id
		}
	}

	customerID := params.CustomerID
	task := &domain.Task{
		ID:            uuid.New(),
		TenantID:      params.TenantID,
		AssignedTo:    assignedTo,
		CustomerID:    &customerID,
		Title:         title,
		Description:   description,
		Status:        "pending",
		DueAt:         &dueAt,
		Source:        "rule",
		RuleID:        ruleID,
	}

	// Vincular a la entidad que disparó el evento si es appointment o treatment
	switch strings.ToLower(params.EntityType) {
	case "appointment":
		task.AppointmentID = &params.EntityID
	case "treatment":
		task.TreatmentID = &params.EntityID
	}

	if err := a.taskRepo.Create(ctx, task); err != nil {
		return fmt.Errorf("CreateTaskAction.Execute: %w", err)
	}

	slog.Info("CreateTaskAction: tarea creada", "task_id", task.ID, "tenant", params.TenantID)
	return nil
}
```

### 3D. MoveStage action: `engine/actions/stage.go`

```go
package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

// MoveStageAction mueve un cliente a otra etapa del pipeline.
type MoveStageAction struct {
	customerRepo domain.CustomerRepository
}

// NewMoveStageAction crea la acción de mover etapa.
func NewMoveStageAction(customerRepo domain.CustomerRepository) *MoveStageAction {
	return &MoveStageAction{customerRepo: customerRepo}
}

func (a *MoveStageAction) Type() string {
	return "move_stage"
}

func (a *MoveStageAction) Execute(ctx context.Context, params engine.ActionParams) error {
	// stage_id puede venir como string o como parámetro directo
	stageIDStr, ok := params.Params["stage_id"].(string)
	if !ok || stageIDStr == "" {
		return fmt.Errorf("MoveStageAction.Execute: stage_id no especificado en params")
	}

	stageID, err := uuid.Parse(stageIDStr)
	if err != nil {
		return fmt.Errorf("MoveStageAction.Execute: stage_id inválido '%s': %w", stageIDStr, err)
	}

	if err := a.customerRepo.UpdateStage(ctx, params.TenantID, params.CustomerID, stageID); err != nil {
		return fmt.Errorf("MoveStageAction.Execute: %w", err)
	}

	slog.Info("MoveStageAction: cliente movido de etapa",
		"customer_id", params.CustomerID,
		"stage_id", stageID,
		"tenant", params.TenantID,
	)
	return nil
}
```

### 3E. UpdateField action: `engine/actions/field.go`

```go
package actions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

// UpdateFieldAction actualiza un campo del cliente.
type UpdateFieldAction struct {
	customerRepo domain.CustomerRepository
}

// NewUpdateFieldAction crea la acción de actualización de campo.
func NewUpdateFieldAction(customerRepo domain.CustomerRepository) *UpdateFieldAction {
	return &UpdateFieldAction{customerRepo: customerRepo}
}

func (a *UpdateFieldAction) Type() string {
	return "update_field"
}

func (a *UpdateFieldAction) Execute(ctx context.Context, params engine.ActionParams) error {
	field, ok := params.Params["field"].(string)
	if !ok || field == "" {
		return fmt.Errorf("UpdateFieldAction.Execute: 'field' no especificado en params")
	}

	value, ok := params.Params["value"]
	if !ok {
		return fmt.Errorf("UpdateFieldAction.Execute: 'value' no especificado en params")
	}

	// Para campos de fecha, calcular valor relativo si es "now+Nd"
	if field == "next_recall_at" || field == "last_visit_at" {
		value = resolveTimeValue(value)
	}

	if err := a.customerRepo.UpdateField(ctx, params.TenantID, params.CustomerID, field, value); err != nil {
		return fmt.Errorf("UpdateFieldAction.Execute: %w", err)
	}

	slog.Info("UpdateFieldAction: campo actualizado",
		"customer_id", params.CustomerID,
		"field", field,
		"tenant", params.TenantID,
	)
	return nil
}

// resolveTimeValue resuelve valores de tiempo relativos.
// Soporta:
//   - "now"         → time.Now()
//   - float64 (N)   → NOW() + N days (ej: 180 → en 6 meses)
//   - string ISO    → se pasa tal cual a la DB
func resolveTimeValue(value any) any {
	switch v := value.(type) {
	case string:
		if v == "now" {
			return time.Now()
		}
		// Intentar parsear como ISO
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t
		}
		return v
	case float64:
		// Interpretar como días desde ahora
		return time.Now().Add(time.Duration(v) * 24 * time.Hour)
	case int:
		return time.Now().Add(time.Duration(v) * 24 * time.Hour)
	default:
		return value
	}
}
```

### Verification

- `go build ./internal/engine/...` compiles
- Each action implements `engine.ActionExecutor` interface
- WhatsApp action checks connection before sending
- Task action links to the triggering entity (appointment/treatment)
- Stage and field actions use the new `CustomerRepository` methods
- Template rendering uses simple `{{key}}` replacement

---

## Task 4: Engine - Executor (Orchestrator)

**File to create:** `apps/api/internal/engine/executor.go`

This is the core orchestrator that takes a rule + event context, evaluates conditions, checks cooldown, executes all actions, and logs the result.

```go
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// RuleExecutor es el orquestador del motor de reglas.
// Coordina evaluación de condiciones, cooldown y ejecución de acciones.
type RuleExecutor struct {
	ruleRepo     domain.RuleRepository
	execRepo     domain.RuleExecutionRepository
	authRepo     domain.AuthRepository
	registry     *ActionRegistry
}

// NewRuleExecutor crea el ejecutor de reglas.
func NewRuleExecutor(
	ruleRepo domain.RuleRepository,
	execRepo domain.RuleExecutionRepository,
	authRepo domain.AuthRepository,
	registry *ActionRegistry,
) *RuleExecutor {
	return &RuleExecutor{
		ruleRepo: ruleRepo,
		execRepo: execRepo,
		authRepo: authRepo,
		registry: registry,
	}
}

// ExecuteEventRules busca y ejecuta todas las reglas que coinciden con un evento.
func (e *RuleExecutor) ExecuteEventRules(ctx context.Context, event domain.RuleEvent) {
	rules, err := e.ruleRepo.ListActiveByTriggerEvent(ctx, event.TenantID, event.EventType)
	if err != nil {
		slog.Error("RuleExecutor.ExecuteEventRules: error listando reglas",
			"tenant_id", event.TenantID,
			"event", event.EventType,
			"error", err,
		)
		return
	}

	if len(rules) == 0 {
		return
	}

	// Resolver slug del tenant (necesario para acciones de WhatsApp)
	tenantSlug := e.resolveTenantSlug(ctx, event.TenantID)

	// Construir contexto para evaluación de condiciones
	evalCtx := e.buildEventContext(event)

	slog.Info("RuleExecutor: evaluando reglas para evento",
		"tenant_id", event.TenantID,
		"event", event.EventType,
		"rules_count", len(rules),
	)

	for _, rule := range rules {
		e.executeRule(ctx, rule, event.CustomerID, event.EntityID, event.EntityType, tenantSlug, evalCtx)
	}
}

// ExecuteTemporalRule ejecuta una regla temporal para un cliente específico.
func (e *RuleExecutor) ExecuteTemporalRule(ctx context.Context, rule *domain.Rule, customerID uuid.UUID, evalCtx map[string]any) {
	tenantSlug := e.resolveTenantSlug(ctx, rule.TenantID)
	entityID := customerID // para reglas temporales, la entidad es el propio cliente
	e.executeRule(ctx, rule, customerID, entityID, "customer", tenantSlug, evalCtx)
}

// executeRule ejecuta una regla individual: evalúa condiciones, verifica cooldown, ejecuta acciones.
func (e *RuleExecutor) executeRule(
	ctx context.Context,
	rule *domain.Rule,
	customerID, entityID uuid.UUID,
	entityType, tenantSlug string,
	evalCtx map[string]any,
) {
	// 1. Evaluar condiciones
	if !EvaluateConditions(rule.Conditions, evalCtx) {
		slog.Debug("RuleExecutor: condiciones no cumplidas",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
		)
		return
	}

	// 2. Verificar cooldown
	if rule.CooldownHours > 0 && customerID != uuid.Nil {
		recent, err := e.execRepo.HasRecentExecution(ctx, rule.TenantID, rule.ID, customerID, rule.CooldownHours)
		if err != nil {
			slog.Error("RuleExecutor: error verificando cooldown",
				"rule_id", rule.ID,
				"error", err,
			)
			return
		}
		if recent {
			slog.Debug("RuleExecutor: cooldown activo, saltando",
				"rule_id", rule.ID,
				"customer_id", customerID,
				"cooldown_hours", rule.CooldownHours,
			)
			return
		}
	}

	// 3. Ejecutar acciones
	var execErr error
	for _, action := range rule.Actions {
		executor := e.registry.Get(action.Type)
		if executor == nil {
			slog.Warn("RuleExecutor: tipo de acción no registrado",
				"action_type", action.Type,
				"rule_id", rule.ID,
			)
			continue
		}

		// Inyectar rule_id en params para trazabilidad
		actionParams := action.Params
		if actionParams == nil {
			actionParams = make(map[string]any)
		}
		actionParams["rule_id"] = rule.ID.String()

		params := ActionParams{
			TenantID:   rule.TenantID,
			TenantSlug: tenantSlug,
			CustomerID: customerID,
			EntityID:   entityID,
			EntityType: entityType,
			Template:   action.Template,
			Params:     actionParams,
			Context:    evalCtx,
		}

		if err := executor.Execute(ctx, params); err != nil {
			slog.Error("RuleExecutor: error ejecutando acción",
				"action_type", action.Type,
				"rule_id", rule.ID,
				"error", err,
			)
			execErr = err
			// Continuar con las demás acciones — no romper por una falla
		}
	}

	// 4. Registrar ejecución
	execution := &domain.RuleExecution{
		ID:                 uuid.New(),
		TenantID:           rule.TenantID,
		RuleID:             rule.ID,
		CustomerID:         &customerID,
		TriggeredAt:        time.Now(),
		TriggerEvent:       rule.TriggerEvent,
		ConditionsSnapshot: rule.Conditions,
		ActionsResult:      rule.Actions,
		Status:             "success",
	}

	if execErr != nil {
		execution.Status = "error"
		execution.ErrorMessage = execErr.Error()
	}

	if err := e.execRepo.Create(ctx, execution); err != nil {
		slog.Error("RuleExecutor: error registrando ejecución",
			"rule_id", rule.ID,
			"error", err,
		)
	}

	slog.Info("RuleExecutor: regla ejecutada",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"status", execution.Status,
		"customer_id", customerID,
	)
}

// buildEventContext construye el mapa de contexto a partir de un RuleEvent.
// Aplana el payload para que las condiciones puedan evaluarse con notación de punto.
func (e *RuleExecutor) buildEventContext(event domain.RuleEvent) map[string]any {
	ctx := map[string]any{
		"event_type":  event.EventType,
		"tenant_id":   event.TenantID.String(),
		"customer_id": event.CustomerID.String(),
		"entity_id":   event.EntityID.String(),
		"entity_type": event.EntityType,
	}

	// Copiar payload al contexto de nivel superior
	for k, v := range event.Payload {
		ctx[k] = v
	}

	// También crear subobjetos para notación de punto en condiciones
	// ej: event.Payload["customer"] = map[string]any{"total_visits": 5}
	// permite condición: field="customer.total_visits" op="gt" value=3
	return ctx
}

// resolveTenantSlug obtiene el slug del tenant (necesario para WhatsApp).
func (e *RuleExecutor) resolveTenantSlug(ctx context.Context, tenantID uuid.UUID) string {
	tenant, err := e.authRepo.FindTenantByID(ctx, tenantID)
	if err != nil {
		slog.Error("RuleExecutor: error resolviendo tenant slug",
			"tenant_id", tenantID,
			"error", err,
		)
		return ""
	}
	return tenant.Slug
}
```

### Verification

- `go build ./internal/engine/...` compiles
- Executor evaluates conditions, checks cooldown, runs actions, logs execution
- Actions continue even if one fails (best-effort per action)
- Execution is always logged (success or error)
- Tenant slug is resolved once per event batch, not per action

---

## Task 5: Event Publishing from Services

**Files to modify:**
- `apps/api/internal/service/appointments.go`
- `apps/api/internal/service/treatment.go`

### 5A. Add publisher to `appointmentSvc`

Modify the `appointmentSvc` struct and constructor to accept a publisher:

```go
type appointmentSvc struct {
	apptRepo     domain.AppointmentRepository
	serviceRepo  domain.ServiceRepository
	customerRepo domain.CustomerRepository
	authRepo     domain.AuthRepository
	waClient     domain.WAClient
	notifRepo    domain.NotificationRepository
	publisher    domain.MessagePublisher // NEW: para publicar eventos de reglas
}

// NewAppointmentSvc crea el servicio de citas.
func NewAppointmentSvc(
	apptRepo domain.AppointmentRepository,
	serviceRepo domain.ServiceRepository,
	customerRepo domain.CustomerRepository,
	authRepo domain.AuthRepository,
	waClient domain.WAClient,
	notifRepo domain.NotificationRepository,
	publisher domain.MessagePublisher, // NEW
) domain.AppointmentSvc {
	return &appointmentSvc{
		apptRepo:     apptRepo,
		serviceRepo:  serviceRepo,
		customerRepo: customerRepo,
		authRepo:     authRepo,
		waClient:     waClient,
		notifRepo:    notifRepo,
		publisher:    publisher,
	}
}
```

Add the event publishing helper method:

```go
// publishRuleEvent publica un evento en la cola rules.events (best-effort).
// Si el publisher es nil o falla, loguea y continúa — nunca bloquea al caller.
func (s *appointmentSvc) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("appointmentSvc.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("appointmentSvc.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
```

Add `"encoding/json"` and `"log/slog"` to the imports (replace `"log"` with `"log/slog"`).

Modify the `Update` method to emit events on status changes:

```go
// Update actualiza el estado y/o notas de una cita.
func (s *appointmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, req); err != nil {
		return err
	}
	if req.Status != "" {
		s.notifyAppointmentStatus(ctx, tenantID, id, req.Status)

		// Publicar evento para el motor de reglas
		s.emitAppointmentEvent(ctx, tenantID, id, req.Status)
	}
	return nil
}
```

Add the emit helper:

```go
// emitAppointmentEvent publica un evento de cita al motor de reglas.
func (s *appointmentSvc) emitAppointmentEvent(ctx context.Context, tenantID, apptID uuid.UUID, status string) {
	// Solo emitir para estados que tienen reglas
	eventType := ""
	switch status {
	case "completed":
		eventType = "appointment.completed"
	case "cancelled":
		eventType = "appointment.cancelled"
	case "no_show":
		eventType = "appointment.no_show"
	default:
		return
	}

	// Obtener detalles de la cita para enriquecer el payload
	appt, err := s.apptRepo.GetByID(ctx, tenantID, apptID)
	if err != nil {
		slog.Warn("appointmentSvc.emitAppointmentEvent: get appointment error", "error", err)
		return
	}

	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  eventType,
		CustomerID: appt.CustomerID,
		EntityID:   apptID,
		EntityType: "appointment",
		Payload: map[string]any{
			"appointment_id":    apptID.String(),
			"customer_id":       appt.CustomerID.String(),
			"professional_id":   appt.ProfessionalID.String(),
			"service_id":        appt.ServiceID.String(),
			"status":            status,
			"customer_name":     appt.CustomerName,
			"customer_phone":    appt.CustomerPhone,
			"professional_name": appt.ProfessionalName,
			"service_name":      appt.ServiceName,
			"starts_at":         appt.StartsAt.Format(time.RFC3339),
			"customer": map[string]any{
				"name":  appt.CustomerName,
				"phone": appt.CustomerPhone,
			},
		},
		Timestamp: time.Now(),
	})
}
```

Also add event emission to `Cancel`:

```go
// Cancel cancela una cita con motivo opcional.
func (s *appointmentSvc) Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	if err := s.apptRepo.UpdateStatus(ctx, tenantID, id, &domain.UpdateAppointmentRequest{
		Status:             "cancelled",
		CancellationReason: reason,
	}); err != nil {
		return err
	}
	s.notifyAppointmentStatus(ctx, tenantID, id, "cancelled")
	s.emitAppointmentEvent(ctx, tenantID, id, "cancelled")
	return nil
}
```

### 5B. Add publisher to `treatmentSvc`

Modify the struct and constructor:

```go
type treatmentSvc struct {
	repo      domain.TreatmentRepository
	publisher domain.MessagePublisher // NEW
}

func NewTreatmentSvc(repo domain.TreatmentRepository, publisher domain.MessagePublisher) domain.TreatmentSvc {
	return &treatmentSvc{repo: repo, publisher: publisher}
}
```

Add the event helper:

```go
// publishRuleEvent publica un evento en la cola rules.events (best-effort).
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
```

Add imports: `"encoding/json"`, `"log/slog"`, `"time"`.

Modify `UpdateStatus` to emit events:

```go
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

	// Emitir evento para el motor de reglas
	s.emitTreatmentEvent(ctx, tenantID, t, input.Status)

	return s.repo.GetByID(ctx, tenantID, id)
}

// emitTreatmentEvent publica un evento de tratamiento al motor de reglas.
func (s *treatmentSvc) emitTreatmentEvent(ctx context.Context, tenantID uuid.UUID, t *domain.Treatment, newStatus string) {
	eventType := ""
	switch newStatus {
	case "accepted":
		eventType = "treatment.accepted"
	case "completed":
		eventType = "treatment.completed"
	default:
		return
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
			"previous_status": t.Status,
			"treatment": map[string]any{
				"type":   t.TreatmentType,
				"name":   t.Name,
				"status": newStatus,
			},
		},
		Timestamp: time.Now(),
	})
}
```

### 5C. Customer events from `FindOrCreateByPhone`

For `customer.created`, modify the WhatsApp service which is the primary entry point for new customers. In `service/whatsapp.go`, after the `FindOrCreateByPhone` call, check if the customer was just created:

```go
// In ProcessInbound, after FindOrCreateByPhone (around line 114):
customer, err := s.customerRepo.FindOrCreateByPhone(ctx, tenant.ID, customerName, phone)
if err != nil {
	slog.Warn("whatsAppSvc.ProcessInbound: customer create warning", "error", err)
} else if customer.TotalVisits == 0 && customer.CreatedAt.After(time.Now().Add(-5*time.Second)) {
	// Cliente recién creado (created_at dentro de los últimos 5 segundos)
	s.publishRuleEvent(ctx, domain.RuleEvent{
		TenantID:   tenant.ID,
		EventType:  "customer.created",
		CustomerID: customer.ID,
		EntityID:   customer.ID,
		EntityType: "customer",
		Payload: map[string]any{
			"customer_id":   customer.ID.String(),
			"customer_name": customer.Name,
			"customer_phone": customer.Phone,
			"customer": map[string]any{
				"name":  customer.Name,
				"phone": customer.Phone,
			},
		},
		Timestamp: time.Now(),
	})
}
```

Add the publish helper to `whatsAppSvc`:

```go
// publishRuleEvent publica un evento en la cola rules.events (best-effort).
func (s *whatsAppSvc) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("whatsAppSvc.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("whatsAppSvc.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
```

### 5D. `customer.stage_changed` event

This event will be emitted by the `MoveStageAction` itself after successfully updating the stage. Add to `engine/actions/stage.go` after the `UpdateStage` call:

The `MoveStageAction` needs access to the publisher. Alternatively, since the stage change is the ACTION itself (not a trigger), we do NOT emit `customer.stage_changed` from the MoveStageAction -- that would create an infinite loop. Instead, if we ever need a separate handler to update stage (e.g., from the CRM UI), we add the event there. For MVP, `customer.stage_changed` events are NOT needed as triggers (they're the result of rules, not the cause). Skip this event for now.

### Verification

- `go build ./...` compiles with updated signatures
- Appointment events: `completed`, `cancelled`, `no_show` are emitted after status change
- Treatment events: `accepted`, `completed` emitted from `UpdateStatus`
- Customer event: `created` emitted from WhatsApp inbound processing
- All event publishing is best-effort (warn log on failure, never blocks)
- Publisher nil check in every `publishRuleEvent` method

---

## Task 6: RulesEventWorker

**File to create:** `apps/api/internal/worker/rules_event.go`

Follows the `OutboundWorker` pattern exactly: connects to RabbitMQ, consumes from `rules.events`, processes each event through the engine.

```go
package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

// RulesEventWorker consume eventos de la cola rules.events y ejecuta las reglas coincidentes.
type RulesEventWorker struct {
	amqpURL  string
	executor *engine.RuleExecutor
}

// NewRulesEventWorker crea el worker de eventos de reglas.
func NewRulesEventWorker(amqpURL string, executor *engine.RuleExecutor) *RulesEventWorker {
	return &RulesEventWorker{
		amqpURL:  amqpURL,
		executor: executor,
	}
}

// Start inicia el consumer de eventos de reglas. Bloqueante -- llamar con go.
// Implementa reconexión automática con backoff exponencial.
func (w *RulesEventWorker) Start(ctx context.Context) {
	slog.Info("RulesEventWorker: iniciado")
	delays := []time.Duration{1, 2, 4, 8, 16}
	idx := 0

	for {
		select {
		case <-ctx.Done():
			slog.Info("RulesEventWorker: detenido")
			return
		default:
		}

		if err := w.consume(ctx); err != nil {
			slog.Error("RulesEventWorker: error en consumer", "error", err)
		}

		// Backoff exponencial antes de reconectar
		d := delays[idx]
		if idx < len(delays)-1 {
			idx++
		}
		slog.Info("RulesEventWorker: reconectando", "delay_s", d)

		select {
		case <-ctx.Done():
			return
		case <-time.After(d * time.Second):
		}
	}
}

// consume conecta a RabbitMQ y procesa eventos de rules.events.
func (w *RulesEventWorker) consume(ctx context.Context) error {
	conn, err := amqp.Dial(w.amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declarar la cola (idempotente)
	q, err := ch.QueueDeclare("rules.events", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Prefetch: procesar de uno en uno para evitar sobrecarga
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "core-api-rules-event", false, false, false, false, nil)
	if err != nil {
		return err
	}

	slog.Info("RulesEventWorker: consumiendo rules.events")

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil // canal cerrado
			}
			w.processEvent(ctx, msg)
		}
	}
}

// processEvent procesa un evento individual del motor de reglas.
func (w *RulesEventWorker) processEvent(ctx context.Context, msg amqp.Delivery) {
	var event domain.RuleEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("RulesEventWorker: unmarshal error", "error", err)
		msg.Nack(false, false) // descartar mensaje mal formateado
		return
	}

	slog.Info("RulesEventWorker: procesando evento",
		"event_type", event.EventType,
		"tenant_id", event.TenantID,
		"customer_id", event.CustomerID,
	)

	// Ejecutar reglas que coincidan con este evento.
	// ExecuteEventRules maneja sus propios errores internamente.
	w.executor.ExecuteEventRules(ctx, event)

	// Siempre ACK: la evaluación de reglas no debe reencolar mensajes.
	// Los errores individuales se registran en rule_executions.
	msg.Ack(false)
}
```

### Verification

- `go build ./internal/worker/...` compiles
- Follows `OutboundWorker` pattern exactly (backoff, QueueDeclare, Qos, consumer tag)
- Always ACKs after processing (errors logged in rule_executions, not requeued)
- Graceful shutdown via context cancellation

---

## Task 7: TemporalRulesWorker

**File to create:** `apps/api/internal/worker/rules_temporal.go`

Follows the `ReminderWorker` pattern: ticker-based, runs every 15 minutes.

```go
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

// TemporalRulesWorker evalúa reglas temporales cada 15 minutos.
// Las reglas temporales verifican condiciones basadas en tiempo
// (ej: "si el cliente no vuelve en 180 días, enviar recordatorio").
type TemporalRulesWorker struct {
	ruleRepo     domain.RuleRepository
	customerRepo domain.CustomerRepository
	executor     *engine.RuleExecutor
	interval     time.Duration
}

// NewTemporalRulesWorker crea el worker de reglas temporales.
func NewTemporalRulesWorker(
	ruleRepo domain.RuleRepository,
	customerRepo domain.CustomerRepository,
	executor *engine.RuleExecutor,
) *TemporalRulesWorker {
	return &TemporalRulesWorker{
		ruleRepo:     ruleRepo,
		customerRepo: customerRepo,
		executor:     executor,
		interval:     15 * time.Minute,
	}
}

// Start inicia el cron de reglas temporales. Bloqueante -- llamar con go.
func (w *TemporalRulesWorker) Start(ctx context.Context) {
	slog.Info("TemporalRulesWorker: iniciado (cada 15 min)")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Ejecutar inmediatamente al iniciar
	w.run(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("TemporalRulesWorker: detenido")
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

// run ejecuta una evaluación de todas las reglas temporales.
func (w *TemporalRulesWorker) run(ctx context.Context) {
	rules, err := w.ruleRepo.ListActiveTemporal(ctx)
	if err != nil {
		slog.Error("TemporalRulesWorker: error listando reglas temporales", "error", err)
		return
	}

	if len(rules) == 0 {
		return
	}

	slog.Info("TemporalRulesWorker: evaluando reglas temporales", "count", len(rules))

	totalExecuted := 0
	for _, rule := range rules {
		n := w.evaluateTemporalRule(ctx, rule)
		totalExecuted += n
	}

	if totalExecuted > 0 {
		slog.Info("TemporalRulesWorker: reglas ejecutadas", "total", totalExecuted)
	}
}

// evaluateTemporalRule evalúa una regla temporal contra todos los clientes del tenant.
// Retorna el número de ejecuciones realizadas.
func (w *TemporalRulesWorker) evaluateTemporalRule(ctx context.Context, rule *domain.Rule) int {
	if rule.TriggerSchedule == nil {
		slog.Warn("TemporalRulesWorker: regla temporal sin schedule", "rule_id", rule.ID)
		return 0
	}

	// Obtener clientes del tenant que coinciden con la condición temporal
	// Usamos el reference_field y interval_days del schedule para buscar candidatos
	customers := w.findTemporalCandidates(ctx, rule)
	if len(customers) == 0 {
		return 0
	}

	executed := 0
	for _, customer := range customers {
		evalCtx := w.buildTemporalContext(customer, rule)
		w.executor.ExecuteTemporalRule(ctx, rule, customer.ID, evalCtx)
		executed++
	}
	return executed
}

// findTemporalCandidates busca clientes que cumplen la condición temporal.
// Ej: reference_field="last_visit_at", interval_days=180 → clientes cuya
// última visita fue hace >= 180 días.
func (w *TemporalRulesWorker) findTemporalCandidates(ctx context.Context, rule *domain.Rule) []*domain.Customer {
	sched := rule.TriggerSchedule

	// Obtener todos los clientes del tenant (paginado, límite razonable)
	// En un sistema grande esto se haría con una query SQL directa,
	// pero para MVP listamos con paginación.
	customers, err := w.customerRepo.List(ctx, rule.TenantID, "", 500, 0)
	if err != nil {
		slog.Error("TemporalRulesWorker: error listando clientes",
			"tenant_id", rule.TenantID,
			"error", err,
		)
		return nil
	}

	cutoff := time.Now().Add(-time.Duration(sched.IntervalDays) * 24 * time.Hour)
	var candidates []*domain.Customer

	for _, c := range customers {
		if w.matchesTemporalCondition(c, sched.ReferenceField, cutoff) {
			candidates = append(candidates, c)
		}
	}
	return candidates
}

// matchesTemporalCondition verifica si un cliente cumple la condición temporal.
func (w *TemporalRulesWorker) matchesTemporalCondition(customer *domain.Customer, referenceField string, cutoff time.Time) bool {
	switch referenceField {
	case "last_visit_at":
		// Cliente que no visita desde hace N días
		if customer.LastVisitAt == nil {
			return false // sin visitas previas, no aplica
		}
		return customer.LastVisitAt.Before(cutoff)

	case "next_recall_at":
		// Cliente cuyo recall programado ya pasó
		if customer.NextRecallAt == nil {
			return false
		}
		return customer.NextRecallAt.Before(time.Now())

	case "created_at":
		// Cliente creado hace N días (ej: bienvenida de seguimiento)
		return customer.CreatedAt.Before(cutoff)

	default:
		slog.Warn("TemporalRulesWorker: reference_field desconocido",
			"field", referenceField,
			"customer_id", customer.ID,
		)
		return false
	}
}

// buildTemporalContext construye el contexto de evaluación para una regla temporal.
func (w *TemporalRulesWorker) buildTemporalContext(customer *domain.Customer, rule *domain.Rule) map[string]any {
	ctx := map[string]any{
		"event_type":     "temporal",
		"tenant_id":      rule.TenantID.String(),
		"customer_id":    customer.ID.String(),
		"customer_name":  customer.Name,
		"customer_phone": customer.Phone,
		"customer": map[string]any{
			"name":          customer.Name,
			"phone":         customer.Phone,
			"total_visits":  customer.TotalVisits,
			"lifetime_value": customer.LifetimeValue,
		},
	}

	// Agregar campos de fecha como strings para las condiciones
	if customer.LastVisitAt != nil {
		ctx["customer"].(map[string]any)["last_visit_at"] = customer.LastVisitAt.Format(time.RFC3339)
		// Calcular días desde última visita
		daysSince := int(time.Since(*customer.LastVisitAt).Hours() / 24)
		ctx["customer"].(map[string]any)["days_since_last_visit"] = daysSince
		ctx["days_since_last_visit"] = daysSince
	}
	if customer.NextRecallAt != nil {
		ctx["customer"].(map[string]any)["next_recall_at"] = customer.NextRecallAt.Format(time.RFC3339)
	}
	if customer.StageID != nil {
		ctx["customer"].(map[string]any)["stage_id"] = customer.StageID.String()
		ctx["stage_id"] = customer.StageID.String()
	} else {
		ctx["stage_id"] = uuid.Nil.String()
	}

	return ctx
}
```

### Verification

- `go build ./internal/worker/...` compiles
- Follows `ReminderWorker` pattern (ticker, immediate run, graceful shutdown)
- Evaluates temporal rules against customer data
- Supports `last_visit_at`, `next_recall_at`, `created_at` as reference fields
- Cooldown checking is handled by the `RuleExecutor`, not duplicated here

---

## Task 8: Wiring in main.go

**File to modify:** `apps/api/cmd/server/main.go`

### 8A. Add import for engine packages

```go
import (
	// ... existing imports ...
	"github.com/citaspot/api/internal/engine"
	"github.com/citaspot/api/internal/engine/actions"
)
```

### 8B. Update service constructors that changed signatures

In the services section (~line 123), update the constructors:

```go
// Update appointmentSvc to receive publisher
apptSvc := service.NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, authRepo, waClient, notifRepo, publisher)

// Update treatmentSvc to receive publisher
treatmentSvc := service.NewTreatmentSvc(treatmentRepo, publisher)
```

### 8C. Create engine and action registry

After the services section and before handlers (~line 143):

```go
// ── Motor de reglas ──────────────────────────────────────────────────────
// Registrar ejecutores de acciones
actionRegistry := engine.NewActionRegistry()
actionRegistry.Register(actions.NewSendWhatsAppAction(waClient))
actionRegistry.Register(actions.NewCreateTaskAction(taskRepo))
actionRegistry.Register(actions.NewMoveStageAction(customerRepo))
actionRegistry.Register(actions.NewUpdateFieldAction(customerRepo))

// Crear el ejecutor central de reglas
ruleExecutor := engine.NewRuleExecutor(ruleRepo, ruleExecRepo, authRepo, actionRegistry)
```

### 8D. Start both rule workers

After the existing workers section (~line 165), add:

```go
// Workers del motor de reglas
if cfg.RabbitMQURL != "" {
	rulesEventWorker := worker.NewRulesEventWorker(cfg.RabbitMQURL, ruleExecutor)
	go rulesEventWorker.Start(ctx)
}
temporalRulesWorker := worker.NewTemporalRulesWorker(ruleRepo, customerRepo, ruleExecutor)
go temporalRulesWorker.Start(ctx)
```

The `RulesEventWorker` only starts if RabbitMQ URL is configured (it needs a connection). The `TemporalRulesWorker` always starts (it queries the DB directly).

### 8E. Full wiring summary

The complete dependency graph for Phase 6B:

```
main.go
  ├── ActionRegistry
  │     ├── SendWhatsAppAction(waClient)
  │     ├── CreateTaskAction(taskRepo)
  │     ├── MoveStageAction(customerRepo)
  │     └── UpdateFieldAction(customerRepo)
  ├── RuleExecutor(ruleRepo, ruleExecRepo, authRepo, actionRegistry)
  ├── RulesEventWorker(cfg.RabbitMQURL, ruleExecutor) → go Start(ctx)
  ├── TemporalRulesWorker(ruleRepo, customerRepo, ruleExecutor) → go Start(ctx)
  ├── appointmentSvc(..., publisher) → emits to "rules.events"
  └── treatmentSvc(repo, publisher) → emits to "rules.events"
```

### Verification

- `go build ./cmd/server/...` compiles
- All workers start as goroutines with the shared context
- Engine has access to all needed repositories and clients
- Publisher nil-safety maintained throughout
- No circular dependencies

---

## File Summary

### New files (5):
| File | Purpose |
|------|---------|
| `apps/api/internal/engine/evaluator.go` | Condition evaluator (pure functions) |
| `apps/api/internal/engine/action.go` | ActionExecutor interface + ActionRegistry |
| `apps/api/internal/engine/actions/whatsapp.go` | Send WhatsApp message action |
| `apps/api/internal/engine/actions/task.go` | Create task action |
| `apps/api/internal/engine/actions/stage.go` | Move pipeline stage action |
| `apps/api/internal/engine/actions/field.go` | Update customer field action |
| `apps/api/internal/engine/executor.go` | Rule executor orchestrator |
| `apps/api/internal/worker/rules_event.go` | RabbitMQ consumer for rules.events |
| `apps/api/internal/worker/rules_temporal.go` | Cron worker for temporal rules |

### Modified files (6):
| File | Change |
|------|--------|
| `apps/api/internal/domain/types.go` | Add `RuleEvent` struct |
| `apps/api/internal/domain/interfaces.go` | Add `UpdateStage`, `UpdateField` to CustomerRepository; `HasRecentExecution` to RuleExecutionRepository; `ListActiveByTriggerEvent`, `ListActiveTemporal` to RuleRepository |
| `apps/api/internal/client/rabbitmq/publisher.go` | Add `"rules.events"` queue |
| `apps/api/internal/repository/customers.go` | Implement `UpdateStage`, `UpdateField` |
| `apps/api/internal/repository/rules.go` | Implement `ListActiveByTriggerEvent`, `ListActiveTemporal` |
| `apps/api/internal/repository/rule_executions.go` | Implement `HasRecentExecution` |
| `apps/api/internal/service/appointments.go` | Add publisher field, emit events on status change |
| `apps/api/internal/service/treatment.go` | Add publisher field, emit events on `UpdateStatus` |
| `apps/api/internal/service/whatsapp.go` | Emit `customer.created` event |
| `apps/api/cmd/server/main.go` | Wire engine, registry, workers; update service constructors |

---

## Event Reference

| Event | Emitted from | Payload keys |
|-------|-------------|--------------|
| `appointment.completed` | `appointmentSvc.Update` | appointment_id, customer_id, professional_id, service_id, status, customer_name, customer_phone, professional_name, service_name, starts_at |
| `appointment.cancelled` | `appointmentSvc.Update`, `Cancel` | same as above |
| `appointment.no_show` | `appointmentSvc.Update` | same as above |
| `treatment.accepted` | `treatmentSvc.UpdateStatus` | treatment_id, customer_id, professional_id, treatment_type, name, status, previous_status |
| `treatment.completed` | `treatmentSvc.UpdateStatus` | same as above |
| `customer.created` | `whatsAppSvc.ProcessInbound` | customer_id, customer_name, customer_phone |

## Rule Action Reference

| Action Type | Required Params | Template Usage |
|-------------|----------------|----------------|
| `send_whatsapp` | (none, phone from context) | Message text with `{{variable}}` placeholders |
| `create_task` | `description`, `assigned_to`, `due_in_days` (all optional) | Task title with `{{variable}}` placeholders |
| `move_stage` | `stage_id` (required) | Not used |
| `update_field` | `field`, `value` (both required) | Not used |

## Temporal Rule Reference

| Reference Field | Meaning | Example |
|----------------|---------|---------|
| `last_visit_at` | Customer's last appointment date | "If no visit in 180 days, send recall" |
| `next_recall_at` | Scheduled recall date | "If recall date passed, create task" |
| `created_at` | Customer creation date | "7 days after signup, send follow-up" |
