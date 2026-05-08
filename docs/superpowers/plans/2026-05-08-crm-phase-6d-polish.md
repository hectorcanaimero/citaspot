# CRM Phase 6D: Polish & Templates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans.

**Goal:** Polish the CRM feature with auto-seeding on onboarding, message template improvements, and basic metrics.

**Architecture:** Backend improvements (seed on registration, template system) + frontend metrics widget.

**Tech Stack:** Go 1.24, Next.js 14, TypeScript

---

## Task 1: Auto-seed dental pipeline on tenant registration

**Goal:** When a tenant with `business_type='dental'` completes onboarding, automatically seed the default pipeline stages and rule templates.

**Why:** Dental tenants should get a ready-to-use CRM pipeline without manual configuration.

### Step 1.1: Pass `*pgxpool.Pool` to the onboarding completion inline handler

The onboarding endpoint lives as an inline handler in `apps/api/cmd/server/main.go` (line ~390). It currently calls `authRepo.CompleteOnboarding(c.Context(), tenantID)` directly. We need to also call the seed functions, which require a `*pgxpool.Pool`.

**File:** `apps/api/cmd/server/main.go`

Replace the inline onboarding handler (around line 390-399):

```go
// BEFORE:
protected.Post("/onboarding/complete", func(c *fiber.Ctx) error {
    tenantID := middleware.TenantIDFromContext(c)
    if tenantID == uuid.Nil {
        return fiber.NewError(403, "tenant no identificado")
    }
    if err := authRepo.CompleteOnboarding(c.Context(), tenantID); err != nil {
        return fiber.NewError(500, "error interno")
    }
    return c.JSON(fiber.Map{"ok": true})
})
```

```go
// AFTER:
protected.Post("/onboarding/complete", func(c *fiber.Ctx) error {
    tenantID := middleware.TenantIDFromContext(c)
    if tenantID == uuid.Nil {
        return fiber.NewError(403, "tenant no identificado")
    }
    if err := authRepo.CompleteOnboarding(c.Context(), tenantID); err != nil {
        return fiber.NewError(500, "error interno")
    }

    // Auto-seed CRM pipeline para tenants dentales
    tenant := middleware.TenantFromContext(c)
    if tenant != nil && tenant.BusinessType == "dental" {
        go func() {
            bgCtx := context.Background()
            if err := seed.SeedDentalPipeline(bgCtx, pool, tenantID); err != nil {
                slog.Warn("onboarding: error seeding dental pipeline", "tenant_id", tenantID, "error", err)
            } else {
                slog.Info("onboarding: dental pipeline seeded", "tenant_id", tenantID)
            }
            // Rule templates son globales (idempotent) — safe to call multiple times
            if err := seed.SeedDentalRuleTemplates(bgCtx, pool); err != nil {
                slog.Warn("onboarding: error seeding dental rule templates", "error", err)
            }
        }()
    }

    return c.JSON(fiber.Map{"ok": true})
})
```

**Add import** at the top of `main.go`:

```go
"github.com/citaspot/api/internal/seed"
```

Note: The `pool` variable and `context` package are already available in scope. The seed runs in a goroutine so it doesn't block the onboarding response. `SeedDentalPipeline` uses `ON CONFLICT DO NOTHING` and `SeedDentalRuleTemplates` checks `EXISTS` — both are idempotent.

### Verification

- Register a tenant with `business_type: "dental"`
- Complete onboarding
- Check `pipeline_stages` table has 7 rows for the tenant
- Check `rules` table has template rows with `is_template = TRUE`
- Repeat — should be idempotent (no errors, no duplicates)

---

## Task 2: Enrich rule templates with actual WhatsApp message text

**Goal:** Update the seed templates in `seed/dental.go` so the `actions` JSON includes actual, natural WhatsApp messages in Spanish LATAM (voseo) with `{{variable}}` placeholders.

**Why:** Currently the action templates reference placeholder keys like `"template":"recall_reminder"` but don't contain the actual message text. The `renderTemplate()` function in `engine/actions/whatsapp.go` replaces `{{key}}` with context values — so the templates need real text.

**File:** `apps/api/internal/seed/dental.go`

Replace each rule template's `ActionsJSON` with actual message text. The `template` field inside each `send_whatsapp` action should contain the full message:

```go
var DefaultDentalRuleTemplates = []struct {
	Name         string
	Description  string
	TriggerType  string
	TriggerEvent string
	ActionsJSON  string
	TemplateKey  string
}{
	{
		Name:        "Recall semestral",
		Description: "Enviar WhatsApp cuando el paciente no ha visitado en 6 meses",
		TriggerType: "temporal",
		TemplateKey: "dental_recall_6m",
		ActionsJSON: `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Ya pasaron 6 meses desde tu ultima visita con nosotros. Te recomendamos agendar una cita de control para mantener tu salud dental al dia. Responde SI para que te agendemos.","params":{"interval_days":180}},{"type":"create_task","template":"Contactar a {{customer_name}} para recall semestral","params":{"due_days":1}}]`,
	},
	{
		Name:         "Post-extraccion 48h",
		Description:  "Seguimiento post-operatorio a las 48 horas de una extraccion",
		TriggerType:  "temporal",
		TemplateKey:  "dental_post_extraction_48h",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, como te sentis despues de la extraccion? Es normal algo de molestia las primeras 48 horas. Recorda: no escupir, no usar sorbete, y aplicar hielo por fuera 15 min cada hora. Si tenes sangrado que no para o dolor fuerte, escribinos.","params":{"interval_hours":48}}]`,
	},
	{
		Name:         "Post-endodoncia 7d",
		Description:  "Seguimiento una semana despues de endodoncia",
		TriggerType:  "temporal",
		TemplateKey:  "dental_post_endodoncia_7d",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya paso una semana desde tu endodoncia. Como te sentis? Recorda que es importante colocar la corona definitiva lo antes posible para proteger el diente. Queres que te agendemos? Responde SI.","params":{"interval_days":7}},{"type":"create_task","template":"Verificar evolucion post-endodoncia de {{customer_name}}","params":{"due_days":7}}]`,
	},
	{
		Name:         "Follow-up presupuesto",
		Description:  "Seguimiento 7 dias despues de enviar presupuesto",
		TriggerType:  "temporal",
		TemplateKey:  "dental_followup_budget",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Te escribimos para saber si pudiste revisar el presupuesto que te enviamos para {{service_name}}. Tenes alguna duda? Con gusto te ayudamos. Tambien podemos ver opciones de financiamiento si te interesa.","params":{"interval_days":7}},{"type":"create_task","template":"Seguimiento presupuesto de {{customer_name}} para {{service_name}}","params":{"due_days":7}}]`,
	},
	{
		Name:         "Bienvenida nuevo paciente",
		Description:  "Mensaje de bienvenida cuando un paciente nuevo llega por WhatsApp",
		TriggerType:  "event",
		TriggerEvent: "customer.created",
		TemplateKey:  "dental_welcome",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Bienvenido/a {{customer_name}}! Gracias por contactarnos. Somos {{tenant_name}} y estamos para ayudarte con tu salud dental. Podes agendar tu cita escribiendo la fecha y hora que te quede mejor, o responde CITA para ver horarios disponibles.","params":{}}]`,
	},
	{
		Name:         "Paciente no asistio",
		Description:  "Seguimiento cuando un paciente no asiste a su cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.no_show",
		TemplateKey:  "dental_no_show",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, notamos que no pudiste asistir a tu cita de {{service_name}}. Esperamos que todo este bien! Queres que te reagendemos? Responde SI y te buscamos un horario que te quede comodo.","params":{}},{"type":"create_task","template":"Reagendar cita de {{customer_name}} (no show)","params":{"due_days":1}}]`,
	},
	{
		Name:         "Retiro de puntos",
		Description:  "Recordatorio para agendar retiro de puntos 7 dias despues de cirugia",
		TriggerType:  "temporal",
		TemplateKey:  "dental_suture_removal",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya se cumplen 7 dias desde tu cirugia y es momento de retirar los puntos. Es un procedimiento rapido y sin dolor. Responde SI para agendar tu cita de retiro de puntos.","params":{"interval_days":7}},{"type":"create_task","template":"Agendar retiro de puntos de {{customer_name}}","params":{"due_days":7}}]`,
	},
	{
		Name:         "Seguimiento post-tratamiento",
		Description:  "Crear tarea de seguimiento cuando un tratamiento se completa",
		TriggerType:  "event",
		TriggerEvent: "treatment.completed",
		TemplateKey:  "dental_post_treatment",
		ActionsJSON:  `[{"type":"create_task","template":"Seguimiento post-tratamiento de {{customer_name}} - verificar satisfaccion y agendar control","params":{"due_days":7}}]`,
	},
	{
		Name:         "Inactividad 12 meses",
		Description:  "Mover a etapa Inactivo cuando el paciente no visita en 12 meses",
		TriggerType:  "temporal",
		TemplateKey:  "dental_inactivity_12m",
		ActionsJSON:  `[{"type":"move_stage","template":"move_to_inactive","params":{"interval_days":365}},{"type":"create_task","template":"Reactivar paciente {{customer_name}} - 12 meses sin visita","params":{"due_days":1}}]`,
	},
}
```

**Key changes:**
- `send_whatsapp` actions now have full Spanish LATAM messages with voseo ("sentis", "recorda", "tenes", "queres", "podes")
- `create_task` templates now have descriptive task names with `{{customer_name}}` and `{{service_name}}`
- `move_stage` action keeps the internal key (it doesn't render as a message)
- Removed accents from strings stored in Go source for DB compatibility (PostgreSQL handles UTF-8 fine, but keeping it simple)

**Important:** Since `SeedDentalRuleTemplates` uses `EXISTS` check, existing templates will NOT be overwritten. To update existing templates:
- Either delete the old ones first: `DELETE FROM rules WHERE is_template = TRUE AND tenant_id = '00000000-0000-0000-0000-000000000000'`
- Or add a new migration that updates the `actions` column for existing templates by `template_key`

### Step 2.1 (optional): Migration to update existing template text

If templates were already seeded, create a migration to update them:

**File:** `apps/api/db/migrations/0XX_update_dental_rule_templates_text.sql` (use next available number)

```sql
-- Actualizar templates de reglas dentales con texto real de WhatsApp.
-- Solo afecta reglas globales (tenant template UUID).

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Ya pasaron 6 meses desde tu ultima visita con nosotros. Te recomendamos agendar una cita de control para mantener tu salud dental al dia. Responde SI para que te agendemos.","params":{"interval_days":180}},{"type":"create_task","template":"Contactar a {{customer_name}} para recall semestral","params":{"due_days":1}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_recall_6m';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, como te sentis despues de la extraccion? Es normal algo de molestia las primeras 48 horas. Recorda: no escupir, no usar sorbete, y aplicar hielo por fuera 15 min cada hora. Si tenes sangrado que no para o dolor fuerte, escribinos.","params":{"interval_hours":48}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_post_extraction_48h';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya paso una semana desde tu endodoncia. Como te sentis? Recorda que es importante colocar la corona definitiva lo antes posible para proteger el diente. Queres que te agendemos? Responde SI.","params":{"interval_days":7}},{"type":"create_task","template":"Verificar evolucion post-endodoncia de {{customer_name}}","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_post_endodoncia_7d';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Te escribimos para saber si pudiste revisar el presupuesto que te enviamos para {{service_name}}. Tenes alguna duda? Con gusto te ayudamos. Tambien podemos ver opciones de financiamiento si te interesa.","params":{"interval_days":7}},{"type":"create_task","template":"Seguimiento presupuesto de {{customer_name}} para {{service_name}}","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_followup_budget';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Bienvenido/a {{customer_name}}! Gracias por contactarnos. Somos {{tenant_name}} y estamos para ayudarte con tu salud dental. Podes agendar tu cita escribiendo la fecha y hora que te quede mejor, o responde CITA para ver horarios disponibles.","params":{}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_welcome';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, notamos que no pudiste asistir a tu cita de {{service_name}}. Esperamos que todo este bien! Queres que te reagendemos? Responde SI y te buscamos un horario que te quede comodo.","params":{}},{"type":"create_task","template":"Reagendar cita de {{customer_name}} (no show)","params":{"due_days":1}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_no_show';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya se cumplen 7 dias desde tu cirugia y es momento de retirar los puntos. Es un procedimiento rapido y sin dolor. Responde SI para agendar tu cita de retiro de puntos.","params":{"interval_days":7}},{"type":"create_task","template":"Agendar retiro de puntos de {{customer_name}}","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_suture_removal';

UPDATE rules SET actions = '[{"type":"create_task","template":"Seguimiento post-tratamiento de {{customer_name}} - verificar satisfaccion y agendar control","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_post_treatment';

UPDATE rules SET actions = '[{"type":"move_stage","template":"move_to_inactive","params":{"interval_days":365}},{"type":"create_task","template":"Reactivar paciente {{customer_name}} - 12 meses sin visita","params":{"due_days":1}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_inactivity_12m';
```

### Verification

- After migration or fresh seed, query:
  ```sql
  SELECT template_key, actions->0->>'template' AS msg_preview
  FROM rules WHERE is_template = TRUE;
  ```
- Each should show a full Spanish message with `{{customer_name}}` placeholders
- Render manually: replace `{{customer_name}}` with "Juan" — message should read naturally

---

## Task 3: CRM Metrics API endpoint

**Goal:** Add `GET /api/v1/crm/metrics` returning aggregated CRM data.

### Step 3.1: Add domain types

**File:** `apps/api/internal/domain/types.go` — add at the end:

```go
// ── CRM Metrics ─────────────────────────────────────────────────────────────

// CRMMetrics agrega metricas del CRM para el dashboard.
type CRMMetrics struct {
	RulesFired30d     int                `json:"rules_fired_30d"`
	RulesSuccessRate  float64            `json:"rules_success_rate"`
	ActiveTreatments  int                `json:"active_treatments"`
	PendingTasks      int                `json:"pending_tasks"`
	CustomersPerStage []StageCustomerCount `json:"customers_per_stage"`
	TopRules          []TopRuleMetric    `json:"top_rules"`
}

// StageCustomerCount cantidad de clientes en una etapa del pipeline.
type StageCustomerCount struct {
	StageID   uuid.UUID `json:"stage_id"`
	StageName string    `json:"stage_name"`
	Color     string    `json:"color"`
	Count     int       `json:"count"`
}

// TopRuleMetric regla con mas ejecuciones en los ultimos 30 dias.
type TopRuleMetric struct {
	RuleID     uuid.UUID `json:"rule_id"`
	RuleName   string    `json:"rule_name"`
	Executions int       `json:"executions"`
}
```

### Step 3.2: Add repository interface

**File:** `apps/api/internal/domain/interfaces.go` — add at the end, before the closing of the file:

```go
// ── CRM Metrics ─────────────────────────────────────────────────────────────

// CRMMetricsRepository operaciones DB para metricas agregadas del CRM.
type CRMMetricsRepository interface {
	GetMetrics(ctx context.Context, tenantID uuid.UUID) (*CRMMetrics, error)
}
```

### Step 3.3: Implement CRM metrics repository

**File:** `apps/api/internal/repository/crm_metrics.go` (NEW)

```go
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type crmMetricsRepository struct {
	db *pgxpool.Pool
}

func NewCRMMetricsRepository(db *pgxpool.Pool) domain.CRMMetricsRepository {
	return &crmMetricsRepository{db: db}
}

func (r *crmMetricsRepository) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*domain.CRMMetrics, error) {
	m := &domain.CRMMetrics{}

	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// 1. Rules fired en los ultimos 30 dias + success rate
		err := tx.QueryRow(ctx, `
			SELECT
				COUNT(*) AS total,
				COALESCE(
					COUNT(*) FILTER (WHERE status = 'success')::float /
					NULLIF(COUNT(*), 0),
					0
				) AS success_rate
			FROM rule_executions
			WHERE tenant_id = $1
			  AND triggered_at > NOW() - INTERVAL '30 days'
		`, tenantID).Scan(&m.RulesFired30d, &m.RulesSuccessRate)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: rules_fired: %w", err)
		}

		// 2. Tratamientos activos
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM treatments
			WHERE tenant_id = $1 AND status = 'in_progress'
		`, tenantID).Scan(&m.ActiveTreatments)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: active_treatments: %w", err)
		}

		// 3. Tareas pendientes
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM tasks
			WHERE tenant_id = $1 AND status = 'pending'
		`, tenantID).Scan(&m.PendingTasks)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: pending_tasks: %w", err)
		}

		// 4. Clientes por etapa del pipeline
		rows, err := tx.Query(ctx, `
			SELECT ps.id, ps.name, ps.color, COUNT(c.id) AS cnt
			FROM pipeline_stages ps
			LEFT JOIN customers c ON c.stage_id = ps.id AND c.tenant_id = ps.tenant_id
			WHERE ps.tenant_id = $1
			GROUP BY ps.id, ps.name, ps.color, ps.position
			ORDER BY ps.position ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: customers_per_stage: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var sc domain.StageCustomerCount
			if err := rows.Scan(&sc.StageID, &sc.StageName, &sc.Color, &sc.Count); err != nil {
				return fmt.Errorf("crmMetricsRepository.GetMetrics: scan stage: %w", err)
			}
			m.CustomersPerStage = append(m.CustomersPerStage, sc)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: stage rows: %w", err)
		}

		// 5. Top 5 reglas mas ejecutadas en 30 dias
		topRows, err := tx.Query(ctx, `
			SELECT re.rule_id, r.name, COUNT(*) AS cnt
			FROM rule_executions re
			JOIN rules r ON r.id = re.rule_id AND r.tenant_id = re.tenant_id
			WHERE re.tenant_id = $1
			  AND re.triggered_at > NOW() - INTERVAL '30 days'
			GROUP BY re.rule_id, r.name
			ORDER BY cnt DESC
			LIMIT 5
		`, tenantID)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: top_rules: %w", err)
		}
		defer topRows.Close()

		for topRows.Next() {
			var tr domain.TopRuleMetric
			if err := topRows.Scan(&tr.RuleID, &tr.RuleName, &tr.Executions); err != nil {
				return fmt.Errorf("crmMetricsRepository.GetMetrics: scan top: %w", err)
			}
			m.TopRules = append(m.TopRules, tr)
		}
		if err := topRows.Err(); err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: top rows: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Garantizar slices no-nil para JSON
	if m.CustomersPerStage == nil {
		m.CustomersPerStage = []domain.StageCustomerCount{}
	}
	if m.TopRules == nil {
		m.TopRules = []domain.TopRuleMetric{}
	}

	return m, nil
}
```

### Step 3.4: Create CRM handler

**File:** `apps/api/internal/handler/crm.go` (NEW)

```go
package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// CRMHandler maneja endpoints de metricas CRM.
type CRMHandler struct {
	repo domain.CRMMetricsRepository
}

// NewCRMHandler crea un handler de metricas CRM.
// Usa el repository directamente — no hay logica de negocio extra para metricas de lectura.
func NewCRMHandler(repo domain.CRMMetricsRepository) *CRMHandler {
	return &CRMHandler{repo: repo}
}

// Metrics retorna metricas agregadas del CRM para el tenant actual.
func (h *CRMHandler) Metrics(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(403, "tenant no identificado")
	}

	metrics, err := h.repo.GetMetrics(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(metrics)
}
```

### Step 3.5: Wire everything in main.go

**File:** `apps/api/cmd/server/main.go`

In the repositories section (after `ruleExecRepo`):

```go
crmMetricsRepo := repository.NewCRMMetricsRepository(pool)
```

In the handlers section (after `ruleHandler`):

```go
crmHandler := handler.NewCRMHandler(crmMetricsRepo)
```

In the routes section (after the rules group, before the server startup):

```go
// CRM Metrics
protected.Get("/crm/metrics", crmHandler.Metrics)
```

### Verification

```bash
# Con un token valido:
curl -H "Authorization: Bearer $TOKEN" http://localhost:3001/api/v1/crm/metrics
```

Expected response:
```json
{
  "rules_fired_30d": 0,
  "rules_success_rate": 0,
  "active_treatments": 0,
  "pending_tasks": 0,
  "customers_per_stage": [],
  "top_rules": []
}
```

With data, `customers_per_stage` should list stages with counts, `top_rules` should show the most fired rules.

---

## Task 4: Frontend CRM metrics widget on dashboard

**Goal:** Add a CRM summary card in the dashboard's right-side widgets area.

### Step 4.1: Add API client method

**File:** `apps/web/lib/api.ts`

Add to the appropriate section of the API client (e.g., near the end, or in a new `crm` namespace):

```typescript
// ── CRM ──────────────────────────────────────────────────────────────────────

export interface StageCustomerCount {
  stage_id: string;
  stage_name: string;
  color: string;
  count: number;
}

export interface TopRuleMetric {
  rule_id: string;
  rule_name: string;
  executions: number;
}

export interface CRMMetrics {
  rules_fired_30d: number;
  rules_success_rate: number;
  active_treatments: number;
  pending_tasks: number;
  customers_per_stage: StageCustomerCount[];
  top_rules: TopRuleMetric[];
}

export const crm = {
  async getMetrics(): Promise<CRMMetrics> {
    const res = await apiFetch('/crm/metrics');
    return res.json();
  },
};
```

### Step 4.2: Add i18n keys

**File:** `apps/web/lib/i18n/es.ts` (and equivalents for `en.ts`, `pt.ts`)

Add to the `dashboard` section:

```typescript
// CRM metrics widget
crmTitle: 'CRM',
activeTreatments: 'Tratamientos activos',
pendingTasks: 'Tareas pendientes',
rulesFired: 'Reglas ejecutadas (30d)',
successRate: 'Tasa de exito',
pipelineDistribution: 'Pipeline',
noMetricsYet: 'Sin datos de CRM aun',
viewCRM: 'Ver CRM',
```

### Step 4.3: Add CRM widget component to dashboard page

**File:** `apps/web/app/(dashboard)/page.tsx`

Add imports at the top:

```typescript
import { Activity, Zap } from 'lucide-react';
import { crm, CRMMetrics } from '@/lib/api';
```

Add state in the component:

```typescript
const [crmMetrics, setCrmMetrics] = useState<CRMMetrics | null>(null);
const [crmLoading, setCrmLoading] = useState(true);
```

Add useEffect:

```typescript
useEffect(() => {
  crm.getMetrics()
    .then(setCrmMetrics)
    .catch(() => {})
    .finally(() => setCrmLoading(false));
}, []);
```

Add widget in the right column (`col-span-2` div), after the team widget (before the closing `</div>` of the widgets column):

```tsx
{/* Widget CRM */}
<div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
  <div className="mb-3 flex items-center justify-between">
    <h3 className="text-sm font-semibold text-neutral-900">{t.dashboard.crmTitle}</h3>
    <Activity className="h-4 w-4 text-neutral-400" />
  </div>

  {crmLoading ? (
    <div className="space-y-2">
      <div className="h-8 w-16 animate-pulse rounded-lg bg-neutral-100" />
      <div className="h-4 w-28 animate-pulse rounded bg-neutral-100" />
    </div>
  ) : !crmMetrics ? (
    <p className="text-xs text-neutral-400">{t.dashboard.noMetricsYet}</p>
  ) : (
    <>
      {/* KPIs en grid 2x2 */}
      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-lg bg-primary-50 p-2.5">
          <p className="text-xs text-primary-600">{t.dashboard.activeTreatments}</p>
          <p className="text-lg font-black text-primary-700">{crmMetrics.active_treatments}</p>
        </div>
        <div className="rounded-lg bg-amber-50 p-2.5">
          <p className="text-xs text-amber-600">{t.dashboard.pendingTasks}</p>
          <p className="text-lg font-black text-amber-700">{crmMetrics.pending_tasks}</p>
        </div>
        <div className="rounded-lg bg-sky-50 p-2.5">
          <p className="text-xs text-sky-600">{t.dashboard.rulesFired}</p>
          <p className="text-lg font-black text-sky-700">{crmMetrics.rules_fired_30d}</p>
        </div>
        <div className="rounded-lg bg-emerald-50 p-2.5">
          <p className="text-xs text-emerald-600">{t.dashboard.successRate}</p>
          <p className="text-lg font-black text-emerald-700">
            {Math.round(crmMetrics.rules_success_rate * 100)}%
          </p>
        </div>
      </div>

      {/* Pipeline distribution - simple horizontal bar */}
      {crmMetrics.customers_per_stage.length > 0 && (
        <div className="mt-3">
          <p className="mb-2 text-xs font-medium text-neutral-500">{t.dashboard.pipelineDistribution}</p>
          <div className="space-y-1.5">
            {crmMetrics.customers_per_stage.map((stage) => {
              const maxCount = Math.max(...crmMetrics.customers_per_stage.map(s => s.count), 1);
              const pct = (stage.count / maxCount) * 100;
              return (
                <div key={stage.stage_id} className="flex items-center gap-2">
                  <span className="w-24 truncate text-xs text-neutral-600">{stage.stage_name}</span>
                  <div className="flex-1">
                    <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{ width: `${pct}%`, backgroundColor: stage.color }}
                      />
                    </div>
                  </div>
                  <span className="w-6 text-right text-xs font-semibold text-neutral-700">{stage.count}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Top rules */}
      {crmMetrics.top_rules.length > 0 && (
        <div className="mt-3 border-t border-neutral-100 pt-3">
          <div className="space-y-1.5">
            {crmMetrics.top_rules.slice(0, 3).map((rule) => (
              <div key={rule.rule_id} className="flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-xs text-neutral-600">
                  <Zap className="h-3 w-3 text-amber-500" />
                  {rule.rule_name}
                </span>
                <span className="text-xs font-semibold text-neutral-500">{rule.executions}x</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </>
  )}

  <Link
    href="/dashboard/crm"
    className="mt-3 flex items-center justify-between rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
  >
    {t.dashboard.viewCRM}
    <ArrowUpRight className="h-3.5 w-3.5" />
  </Link>
</div>
```

### Verification

- Load the dashboard page
- The CRM widget should appear in the right column
- With no CRM data: shows "Sin datos de CRM aun"
- With pipeline stages: shows a horizontal bar chart
- With rule executions: shows top 3 rules by execution count
- Handles API errors gracefully (widget shows empty state, page doesn't break)

---

## Summary of files changed

### Modified files

| File | Changes |
|------|---------|
| `apps/api/cmd/server/main.go` | Import `seed` package; add CRM repo+handler wiring; modify onboarding handler to seed dental pipeline |
| `apps/api/internal/seed/dental.go` | Replace placeholder template keys with full WhatsApp messages |
| `apps/api/internal/domain/types.go` | Add `CRMMetrics`, `StageCustomerCount`, `TopRuleMetric` types |
| `apps/api/internal/domain/interfaces.go` | Add `CRMMetricsRepository` interface |
| `apps/web/app/(dashboard)/page.tsx` | Add CRM widget with metrics display |
| `apps/web/lib/api.ts` | Add `crm.getMetrics()` client method and types |
| `apps/web/lib/i18n/es.ts` | Add CRM-related translation keys |
| `apps/web/lib/i18n/en.ts` | Add CRM-related translation keys |
| `apps/web/lib/i18n/pt.ts` | Add CRM-related translation keys |

### New files

| File | Purpose |
|------|---------|
| `apps/api/internal/repository/crm_metrics.go` | CRM metrics repository with aggregation queries |
| `apps/api/internal/handler/crm.go` | CRM metrics HTTP handler |
| `apps/api/db/migrations/0XX_update_dental_rule_templates_text.sql` | (Optional) Update existing seeded templates with real message text |

### Dependencies between tasks

```
Task 1 (auto-seed) ──── independent
Task 2 (templates) ──── independent (but pairs with Task 1 for fresh installs)
Task 3 (metrics API) ── independent
Task 4 (frontend) ───── depends on Task 3 (needs the API endpoint)
```

Tasks 1, 2, and 3 can be implemented in parallel. Task 4 requires Task 3 to be complete.
