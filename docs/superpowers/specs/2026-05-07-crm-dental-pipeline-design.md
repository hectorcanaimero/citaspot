# CRM Dental con Pipeline de Reglas para CitaSpot

## Context

CitaSpot es un SaaS multi-tenant de gestión de citas con IA conversacional vía WhatsApp. Actualmente cubre: scheduling, WhatsApp AI booking, knowledge base RAG, y un dashboard básico de clientes. 

**Problema:** No existe capa CRM ni automatización por reglas. Los odontólogos necesitan gestionar el lifecycle completo del paciente (lead → tratamiento → mantenimiento), automatizar protocolos clínicos (recall 6 meses, post-op 48h), y tener un pipeline comercial (presupuesto → aceptación → cobro).

**Decisiones tomadas:**
- Vertical dental dentro de CitaSpot (no producto separado) — usa `business_type='dental'` existente
- Motor de reglas híbrido: templates predefinidos + builder custom simplificado
- Arquitectura híbrida (Enfoque C): Events para triggers inmediatos + Cron para triggers temporales
- Acciones MVP: WhatsApp + tareas internas (arquitectura extensible para email/webhooks después)

---

## Modelo de Dominio — Nuevas Entidades

### Treatment (Tratamiento dental)
```sql
CREATE TABLE treatments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  customer_id UUID NOT NULL REFERENCES customers(id),
  professional_id UUID NOT NULL REFERENCES professionals(id),
  name TEXT NOT NULL,  -- "Ortodoncia", "Implante #14"
  treatment_type TEXT NOT NULL,  -- ortodoncia, endodoncia, implante, protesis, cirugia, periodoncia, estetica, general
  status TEXT NOT NULL DEFAULT 'proposed',  -- proposed → accepted → in_progress → completed → abandoned
  total_sessions INT,
  completed_sessions INT DEFAULT 0,
  estimated_cost NUMERIC(10,2),
  paid_amount NUMERIC(10,2) DEFAULT 0,
  currency TEXT DEFAULT 'USD',
  tooth_numbers INT[],  -- notación FDI (11-48)
  notes TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  next_session_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- RLS + index on tenant_id, customer_id, status
```

### Pipeline Stages (Etapas del paciente)
```sql
CREATE TABLE pipeline_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  name TEXT NOT NULL,  -- "Nuevo contacto", "Presupuesto enviado", "En tratamiento", "Mantenimiento"
  position INT NOT NULL,
  color TEXT DEFAULT '#6366f1',
  is_default BOOLEAN DEFAULT FALSE,
  auto_rules_enabled BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
-- UNIQUE(tenant_id, position)
-- Default stages seeded on tenant creation when business_type='dental'
```

### Customer Pipeline Position
```sql
-- Extension a customers table:
ALTER TABLE customers ADD COLUMN stage_id UUID REFERENCES pipeline_stages(id);
ALTER TABLE customers ADD COLUMN last_visit_at TIMESTAMPTZ;
ALTER TABLE customers ADD COLUMN next_recall_at TIMESTAMPTZ;
ALTER TABLE customers ADD COLUMN lifetime_value NUMERIC(10,2) DEFAULT 0;
ALTER TABLE customers ADD COLUMN acquisition_source TEXT;  -- whatsapp, web, referral, walk_in
```

### Rules (Motor de automatización)
```sql
CREATE TABLE rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  name TEXT NOT NULL,
  description TEXT,
  trigger_type TEXT NOT NULL,  -- 'event' | 'temporal'
  trigger_event TEXT,  -- 'appointment.completed', 'customer.stage_changed', 'treatment.accepted'
  trigger_schedule JSONB,  -- { "type": "relative", "after": "last_visit", "duration": "6_months" }
  conditions JSONB DEFAULT '[]',  -- [{ "field": "service.type", "op": "eq", "value": "extraction" }]
  actions JSONB NOT NULL,  -- [{ "type": "send_whatsapp", "params": {...} }, { "type": "create_task", "params": {...} }]
  is_active BOOLEAN DEFAULT TRUE,
  is_template BOOLEAN DEFAULT FALSE,
  template_key TEXT,  -- 'dental_recall_6m', 'post_op_48h', etc.
  cooldown_hours INT DEFAULT 24,
  priority INT DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Tasks (Tareas internas del equipo)
```sql
CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  assigned_to UUID REFERENCES professionals(id),
  customer_id UUID REFERENCES customers(id),
  appointment_id UUID REFERENCES appointments(id),
  treatment_id UUID REFERENCES treatments(id),
  title TEXT NOT NULL,
  description TEXT,
  status TEXT NOT NULL DEFAULT 'pending',  -- pending → in_progress → completed → dismissed
  due_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  source TEXT NOT NULL DEFAULT 'manual',  -- manual | rule | system
  rule_id UUID REFERENCES rules(id),
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Rule Execution Log (Auditoría)
```sql
CREATE TABLE rule_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  rule_id UUID NOT NULL REFERENCES rules(id),
  customer_id UUID REFERENCES customers(id),
  triggered_at TIMESTAMPTZ DEFAULT NOW(),
  trigger_event TEXT,
  conditions_snapshot JSONB,
  actions_result JSONB,  -- [{ "type": "send_whatsapp", "status": "success" }, ...]
  status TEXT NOT NULL,  -- success | partial | failed
  error_message TEXT
);
-- Index on (tenant_id, rule_id, customer_id, triggered_at) for cooldown checks
```

### Extension: Appointments
```sql
ALTER TABLE appointments ADD COLUMN treatment_id UUID REFERENCES treatments(id);
```

---

## Arquitectura del Motor de Reglas (Enfoque C: Híbrido)

### Path 1: Event-Driven (triggers inmediatos)

```
[Cambio de estado] → Service emite evento → RabbitMQ queue: "rules.events"
                                                    ↓
                                          RulesEventWorker:
                                          1. Deserializa evento
                                          2. Query reglas activas del tenant donde trigger_event = evento
                                          3. Evalúa condiciones de cada regla
                                          4. Verifica cooldown (no re-disparar)
                                          5. Ejecuta acciones
                                          6. Loguea resultado en rule_executions
```

**Eventos soportados (MVP):**
- `appointment.created`
- `appointment.confirmed`
- `appointment.completed`
- `appointment.cancelled`
- `appointment.no_show`
- `customer.created`
- `customer.stage_changed`
- `treatment.proposed`
- `treatment.accepted`
- `treatment.completed`

### Path 2: Temporal Cron (triggers basados en tiempo)

```
Cada 15 min → TemporalRulesWorker:
  1. Query reglas temporales activas
  2. Para cada regla, evaluar condición temporal:
     - "6 meses desde last_visit_at" → SELECT customers WHERE last_visit_at < NOW() - 6 months
     - "48h después de appointment.completed" → SELECT appointments WHERE completed AND completed_at < NOW() - 48h
  3. Filtrar por cooldown (rule_executions)
  4. Ejecutar acciones para cada match
  5. Loguear resultados
```

### Action Executors (extensibles)

```go
type ActionExecutor interface {
    Type() string
    Execute(ctx context.Context, params ActionParams) error
}

// MVP Actions:
type SendWhatsAppAction struct{}    // Envía mensaje WA via Evolution API
type CreateTaskAction struct{}      // Crea tarea en tasks table
type MoveStageAction struct{}       // Mueve customer a otra pipeline stage
type UpdateFieldAction struct{}     // Actualiza campo del customer (tags, next_recall_at)
```

### Condition Evaluator

```go
type Condition struct {
    Field    string `json:"field"`    // "service.type", "customer.total_visits", "appointment.source"
    Operator string `json:"op"`       // eq, neq, gt, lt, gte, lte, contains, in
    Value    any    `json:"value"`
}

// Evalúa condiciones contra el contexto del evento
func EvaluateConditions(conditions []Condition, context EventContext) bool
```

---

## Templates Dentales Pre-configurados

Cuando un tenant con `business_type='dental'` se crea, se seedean:

### Pipeline Stages Default
1. Nuevo contacto
2. Primera consulta agendada
3. Presupuesto enviado
4. Presupuesto aceptado
5. En tratamiento
6. Mantenimiento/Recall
7. Inactivo (>12 meses)

### Rules Templates
| Template | Trigger | Condición | Acción |
|----------|---------|-----------|--------|
| Recall semestral | temporal: 6m desde last_visit | stage = "Mantenimiento" | WA: "Hola {name}, ya pasaron 6 meses..." + Tarea: "Contactar para recall" |
| Post-extracción 48h | temporal: 48h post-appointment | service.type = "extraction" | WA: "Hola {name}, ¿cómo te sentís? Recordá..." |
| Post-endodoncia 7d | temporal: 7d post-appointment | service.type = "endodoncia" | WA: check-up + Tarea: "Verificar evolución" |
| Follow-up presupuesto | temporal: 7d desde stage_changed | stage = "Presupuesto enviado" | WA: "Hola {name}, ¿pudiste revisar el presupuesto?" + Tarea |
| Bienvenida nuevo paciente | event: customer.created | source = "whatsapp" | WA: mensaje de bienvenida con info de la clínica |
| Cumpleaños | temporal: día del cumpleaños | customer.birthday IS NOT NULL | WA: felicitación |
| No-show follow-up | event: appointment.no_show | — | WA: "Notamos que no pudiste asistir..." + Tarea: "Reagendar" |
| Retiro de puntos | temporal: 7d post-appointment | service.type = "cirugia" | WA: recordatorio + Tarea: "Agendar retiro" |
| Inactividad 12m | temporal: 12m desde last_visit | stage != "Inactivo" | Mover a stage "Inactivo" + Tarea: "Intentar reactivar" |

---

## Estructura de Archivos (nuevos)

```
apps/api/
├── db/migrations/
│   ├── 013_pipeline_stages.sql
│   ├── 014_treatments.sql
│   ├── 015_tasks.sql
│   ├── 016_rules_engine.sql
│   └── 017_customer_crm_fields.sql
├── internal/
│   ├── domain/
│   │   ├── treatment.go        # tipos Treatment
│   │   ├── pipeline.go         # tipos PipelineStage, CustomerPipeline
│   │   ├── rule.go             # tipos Rule, Condition, Action, RuleExecution
│   │   └── task.go             # tipo Task
│   ├── repository/
│   │   ├── treatment.go
│   │   ├── pipeline.go
│   │   ├── rule.go
│   │   └── task.go
│   ├── service/
│   │   ├── treatment.go
│   │   ├── pipeline.go
│   │   ├── rule.go             # CRUD de reglas
│   │   └── task.go
│   ├── handler/
│   │   ├── treatment.go
│   │   ├── pipeline.go
│   │   ├── rule.go
│   │   └── task.go
│   ├── worker/
│   │   ├── rules_event.go      # Consumer de rules.events queue
│   │   └── rules_temporal.go   # Cron cada 15 min
│   └── engine/
│       ├── evaluator.go        # Evalúa condiciones
│       ├── executor.go         # Orquesta ejecución de acciones
│       ├── actions/
│       │   ├── whatsapp.go     # SendWhatsAppAction
│       │   ├── task.go         # CreateTaskAction
│       │   ├── stage.go        # MoveStageAction
│       │   └── field.go        # UpdateFieldAction
│       └── templates/
│           └── dental.go       # Templates dentales para seed

apps/web/app/(dashboard)/
├── pipeline/page.tsx           # Vista Kanban del pipeline de pacientes
├── treatments/page.tsx         # Lista de tratamientos activos
├── tasks/page.tsx              # Bandeja de tareas del equipo
├── automations/page.tsx        # Gestión de reglas (templates + custom)
└── automations/[id]/page.tsx   # Editor de regla individual
```

---

## Frontend — Páginas Clave

### 1. Pipeline View (Kanban)
- Vista drag-and-drop de pacientes por etapa
- Cards con: nombre, último servicio, días en etapa, próxima acción
- Filtros por profesional, tratamiento activo, tags

### 2. Automations Dashboard
- Lista de reglas activas/inactivas
- Toggle on/off por regla
- Sección "Templates disponibles" para activar nuevos
- Builder simplificado: trigger → condiciones → acciones
- Log de ejecuciones reciente

### 3. Tasks Inbox
- Bandeja de tareas agrupada por: hoy, esta semana, vencidas
- Filtro por profesional asignado
- Quick actions: completar, posponer, ver paciente
- Badge en sidebar con count de pendientes

### 4. Treatment Tracker
- Lista de tratamientos activos por paciente
- Progress bar (sesiones completadas / total)
- Timeline de citas vinculadas al tratamiento
- Costo vs pagado

---

## RabbitMQ — Nuevas Colas

| Cola | Producer | Consumer |
|------|----------|----------|
| `rules.events` | Services (appointment, customer, treatment) | RulesEventWorker |
| `rules.actions.whatsapp` | RulesEventWorker / TemporalWorker | Outbound WA Worker (existente) |
| `rules.actions.tasks` | RulesEventWorker / TemporalWorker | (procesado inline, no necesita cola) |

---

## Verificación / Testing

1. **Unit tests:** Evaluator de condiciones, cada ActionExecutor aislado
2. **Integration tests:** Worker procesa evento → regla se dispara → acción se ejecuta → log se crea
3. **Seed dental:** `make seed-dental` crea tenant dental con stages + rules templates + pacientes de ejemplo
4. **Manual E2E:** Completar cita en dashboard → verificar que WA se envía 48h después + tarea se crea
5. **Cooldown test:** Verificar que una regla no se re-dispara dentro del cooldown

---

## Fases de Implementación

### Fase 6A: Infraestructura CRM (foundation)
1. Migraciones DB (013-017)
2. Domain types + interfaces
3. Repositories CRUD
4. Services + handlers REST
5. Seed dental templates

### Fase 6B: Motor de Reglas
1. Engine: evaluator + executor + actions
2. Event publisher (emitir eventos desde services existentes)
3. RulesEventWorker (consumer)
4. TemporalRulesWorker (cron)
5. Rule execution logging

### Fase 6C: Frontend CRM
1. Pipeline Kanban view
2. Automations dashboard + rule editor
3. Tasks inbox
4. Treatment tracker
5. Extensiones a patient profile (timeline, treatments, stage)

### Fase 6D: Polish + Templates
1. Template seeding en onboarding
2. Message templates con variables ({name}, {service}, {date})
3. Métricas: reglas disparadas, conversion rate por stage, churn rate
4. Builder visual simplificado para reglas custom
