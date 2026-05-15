# Diseno Tecnico: Sistema de Event Analytics

> **Estado:** Aprobado
> **Fecha:** 2026-05-14
> **Proyecto:** CitaSpot — SaaS multi-tenant de gestion de citas con IA via WhatsApp
> **Alcance:** Persistencia y consulta de eventos operacionales por tenant

---

## 1. Contexto y Motivacion

CitaSpot ya genera eventos a traves de RabbitMQ (citas, clientes, tratamientos, mensajes) como parte del flujo de reglas de negocio. Sin embargo, estos eventos son **transitorios** — se consumen y desaparecen. No existe forma de consultar patrones historicos, tasas de cancelacion, rendimiento del asistente IA, ni metricas operacionales por tenant.

Este diseno introduce una capa de **persistencia de eventos** que:

1. Captura todos los eventos operacionales relevantes en PostgreSQL
2. Permite consultas analiticas por tenant con filtros de tiempo y tipo
3. Implementa retencion por tier de suscripcion
4. Mantiene el principio de multi-tenancy con RLS

**Principio de diseno:** best-effort. La persistencia de eventos NUNCA debe bloquear ni romper una operacion de negocio. Si falla el INSERT, se loguea el error y la vida sigue. Mismo patron que el publisher nil check de RabbitMQ existente.

---

## 2. Esquema de Base de Datos

### 2.1 Tabla particionada por mes

La particion por `occurred_at` (rango mensual) permite:

- Queries eficientes acotados por tiempo (la mayoria de consultas son "ultimos N dias")
- DROP de particiones antiguas es O(1) vs DELETE masivo
- Vacuum y mantenimiento granular por particion

```sql
CREATE TABLE events (
    id          UUID DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    event_type  TEXT NOT NULL,
    actor_type  TEXT NOT NULL,          -- 'system', 'customer', 'professional', 'admin'
    actor_id    UUID,
    entity_type TEXT NOT NULL,          -- 'appointment', 'customer', 'treatment', 'message'
    entity_id   UUID NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);
```

**Decisiones clave:**

- `id` es UUID v4 para evitar colisiones entre particiones y facilitar idempotencia
- `occurred_at` es parte de la PK porque PostgreSQL lo requiere en tablas particionadas
- `payload` es JSONB para flexibilidad — cada tipo de evento tiene datos distintos sin necesidad de alterar el esquema
- `actor_id` es nullable porque eventos de tipo `system` no tienen un actor humano identificable

### 2.2 Indices

```sql
-- Consultas temporales por tenant (el mas usado)
CREATE INDEX idx_events_tenant_time
    ON events (tenant_id, occurred_at DESC);

-- Filtro por tipo de evento en un rango de tiempo
CREATE INDEX idx_events_tenant_type_time
    ON events (tenant_id, event_type, occurred_at DESC);

-- Buscar todos los eventos de una entidad especifica
CREATE INDEX idx_events_tenant_entity
    ON events (tenant_id, entity_type, entity_id);

-- Consultas sobre campos del payload (uso futuro, dashboards)
CREATE INDEX idx_events_payload
    ON events USING GIN (payload);
```

**Nota:** Los indices se crean sobre la tabla padre y PostgreSQL los propaga automaticamente a cada particion. El indice GIN sobre `payload` es mas costoso en escritura pero habilita queries como `payload->>'source' = 'whatsapp'` sin full scan.

### 2.3 Row Level Security

Misma politica que el resto de las tablas del sistema:

```sql
ALTER TABLE events ENABLE ROW LEVEL SECURITY;

CREATE POLICY events_tenant_isolation ON events
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

Esto garantiza que un tenant NUNCA puede ver eventos de otro tenant, incluso si hay un bug en la capa de aplicacion.

---

## 3. Catalogo de Eventos (12 eventos iniciales)

El catalogo define los eventos que se persisten desde el dia uno. Esta disenado para cubrir las metricas operacionales mas criticas sin sobrecargar la tabla.

### 3.1 Eventos de Citas (6)

| event_type | entity_type | actor_type | Claves en payload |
|---|---|---|---|
| `appointment.created` | appointment | customer / admin | `service_id`, `professional_id`, `starts_at`, `source` (whatsapp/web) |
| `appointment.confirmed` | appointment | customer / admin | `confirmed_by` |
| `appointment.completed` | appointment | professional / system | `duration_minutes`, `notes` |
| `appointment.cancelled` | appointment | customer / admin | `reason`, `cancelled_by`, `hours_before` |
| `appointment.no_show` | appointment | system | — |
| `appointment.rescheduled` | appointment | customer / admin | `old_starts_at`, `new_starts_at` |

**Justificacion:** Estos 6 eventos cubren el ciclo de vida completo de una cita. `source` en `appointment.created` es clave para medir la adopcion del canal WhatsApp vs Web. `hours_before` en cancelaciones permite detectar patrones de cancelacion tardia.

### 3.2 Eventos de Clientes (1)

| event_type | entity_type | actor_type | Claves en payload |
|---|---|---|---|
| `customer.created` | customer | system | `acquisition_source`, `phone` |

**Justificacion:** Medir crecimiento de base de clientes y canal de adquisicion.

### 3.3 Eventos de Tratamientos (1)

| event_type | entity_type | actor_type | Claves en payload |
|---|---|---|---|
| `treatment.created` | treatment | professional | `customer_id`, `plan_type` |

**Justificacion:** Correlacionar tratamientos con citas completadas.

### 3.4 Eventos de Mensajeria (2)

| event_type | entity_type | actor_type | Claves en payload |
|---|---|---|---|
| `message.inbound` | conversation | customer | `channel` (whatsapp), `intent_detected` |
| `message.outbound` | conversation | system | `channel`, `response_time_ms`, `llm_provider` |

**Justificacion:** `response_time_ms` y `llm_provider` permiten monitorear la performance del asistente IA y comparar entre proveedores (Gemini vs GPT-4o-mini). `intent_detected` habilita analisis de que piden los clientes.

### 3.5 Eventos de Notificaciones (2)

| event_type | entity_type | actor_type | Claves en payload |
|---|---|---|---|
| `notification.sent` | notification | system | `type` (reminder/review), `channel`, `status` |
| `notification.failed` | notification | system | `type`, `channel`, `error` |

**Justificacion:** Medir la efectividad del sistema de recordatorios y detectar fallos en el envio.

---

## 4. Capa de Servicio — Integracion en Go

### 4.1 Tipo de dominio

```go
// domain/event.go

type Event struct {
    TenantID   uuid.UUID
    EventType  string
    ActorType  string
    ActorID    *uuid.UUID
    EntityType string
    EntityID   uuid.UUID
    Payload    map[string]any
    OccurredAt time.Time
}

type EventRepository interface {
    Insert(ctx context.Context, event Event) error
    InsertBatch(ctx context.Context, events []Event) error
}
```

**Decisiones:**

- `ActorID` es puntero porque puede ser nil (eventos de sistema)
- `Payload` es `map[string]any` que se serializa a JSONB — flexible y type-safe en el caller
- `InsertBatch` existe para casos donde una operacion genera multiples eventos (ej: cancelar una cita puede generar `appointment.cancelled` + `notification.sent`)

### 4.2 Patron de persistencia: best-effort

```go
// Ejemplo en el service de appointments

func (s *AppointmentService) Cancel(ctx context.Context, id uuid.UUID, reason string) error {
    // 1. Logica de negocio (critica — si falla, retornamos error)
    apt, err := s.repo.Cancel(ctx, id, reason)
    if err != nil {
        return err
    }

    // 2. Evento analitico (best-effort — si falla, solo logueamos)
    if err := s.events.Insert(ctx, domain.Event{
        TenantID:   apt.TenantID,
        EventType:  "appointment.cancelled",
        ActorType:  "customer",
        ActorID:    apt.CustomerID,
        EntityType: "appointment",
        EntityID:   apt.ID,
        Payload: map[string]any{
            "reason":       reason,
            "cancelled_by": "customer",
            "hours_before": time.Until(apt.StartsAt).Hours(),
        },
    }); err != nil {
        log.Printf("ERROR: failed to persist event appointment.cancelled: %v", err)
    }

    // 3. Regla de negocio via RabbitMQ (sigue igual, sin cambios)
    s.publishRuleEvent("appointment.cancelled", apt)

    return nil
}
```

**Principio clave:** El INSERT al events table y el publish a RabbitMQ son **independientes y complementarios**:

- **Events table** → persistencia para analytics (lectura futura)
- **RabbitMQ** → trigger para reglas de negocio en tiempo real (procesamiento inmediato)

Ambos son fire-and-forget desde la perspectiva del flujo de negocio. Ninguno bloquea la operacion principal.

---

## 5. Retencion por Tier de Suscripcion

### 5.1 Politica de retencion

| Plan | Precio | Retencion de eventos |
|---|---|---|
| Basic | $10/mes | 90 dias |
| Pro | $15/mes | 365 dias |
| Enterprise | $25/mes | Sin limite |

**Justificacion:** La retencion diferenciada es un incentivo de upgrade natural. Un tenant en plan Basic puede ver tendencias de 3 meses; para analisis anual necesita Pro. No se requiere tabla de configuracion nueva — la logica usa el campo `plan` existente en la tabla `tenants`.

### 5.2 Cron de limpieza diaria

Ejecuta a las 3:00 AM (hora del servidor) para minimizar impacto en operaciones:

```sql
-- Para tenants en plan Basic (90 dias)
DELETE FROM events
WHERE tenant_id IN (
    SELECT id FROM tenants WHERE plan = 'basic'
)
AND occurred_at < NOW() - INTERVAL '90 days';

-- Para tenants en plan Pro (365 dias)
DELETE FROM events
WHERE tenant_id IN (
    SELECT id FROM tenants WHERE plan = 'pro'
)
AND occurred_at < NOW() - INTERVAL '365 days';

-- Enterprise: no se borra nada
```

**Nota:** Gracias a la particion por mes, estos DELETE son eficientes. Para tenants Basic, las particiones de mas de 3 meses completos se pueden dropear directamente en vez de hacer DELETE fila por fila.

### 5.3 Cron de mantenimiento de particiones (mensual)

```sql
-- Crear particion del mes siguiente (ejecutar el dia 1 de cada mes)
CREATE TABLE events_y2026m07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

-- Dropear particiones vacias antiguas (despues del cron de limpieza)
-- Solo si la particion esta completamente fuera del rango de retencion mas largo activo
```

---

## 6. Queries Operacionales de Ejemplo

Estas queries estan disenadas para el dashboard interno (platform-facing). En el futuro se exponen al tenant con filtros adicionales.

### 6.1 Citas canceladas este mes por tenant

Agrupadas por razon y quien cancelo — detecta patrones de cancelacion:

```sql
SELECT
    payload->>'reason'       AS cancel_reason,
    payload->>'cancelled_by' AS cancelled_by,
    COUNT(*)                 AS total
FROM events
WHERE tenant_id = $1
  AND event_type = 'appointment.cancelled'
  AND occurred_at >= date_trunc('month', NOW())
GROUP BY
    payload->>'reason',
    payload->>'cancelled_by'
ORDER BY total DESC;
```

### 6.2 Tasa de no-show por profesional (ultimos 30 dias)

Identifica profesionales con alta tasa de ausencias para tomar accion:

```sql
WITH stats AS (
    SELECT
        payload->>'professional_id' AS professional_id,
        event_type,
        COUNT(*) AS cnt
    FROM events
    WHERE tenant_id = $1
      AND entity_type = 'appointment'
      AND event_type IN ('appointment.completed', 'appointment.no_show')
      AND occurred_at >= NOW() - INTERVAL '30 days'
    GROUP BY payload->>'professional_id', event_type
)
SELECT
    professional_id,
    COALESCE(SUM(cnt) FILTER (WHERE event_type = 'appointment.no_show'), 0) AS no_shows,
    SUM(cnt) AS total,
    ROUND(
        100.0 * COALESCE(SUM(cnt) FILTER (WHERE event_type = 'appointment.no_show'), 0)
        / NULLIF(SUM(cnt), 0),
        1
    ) AS no_show_rate_pct
FROM stats
GROUP BY professional_id
ORDER BY no_show_rate_pct DESC;
```

### 6.3 Desglose de origen de reservas: WhatsApp vs Web (ultimos 30 dias)

Mide la adopcion del canal conversacional:

```sql
SELECT
    payload->>'source' AS booking_source,
    COUNT(*)           AS total,
    ROUND(
        100.0 * COUNT(*) / SUM(COUNT(*)) OVER (),
        1
    ) AS percentage
FROM events
WHERE tenant_id = $1
  AND event_type = 'appointment.created'
  AND occurred_at >= NOW() - INTERVAL '30 days'
GROUP BY payload->>'source'
ORDER BY total DESC;
```

### 6.4 Tiempo de respuesta del asistente IA: promedio y p95 (ultimos 7 dias)

Monitorea la performance del LLM Router:

```sql
SELECT
    COUNT(*)                                                          AS total_responses,
    ROUND(AVG((payload->>'response_time_ms')::numeric), 0)           AS avg_response_ms,
    ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (
        ORDER BY (payload->>'response_time_ms')::numeric
    ), 0)                                                             AS p95_response_ms,
    payload->>'llm_provider'                                          AS provider
FROM events
WHERE tenant_id = $1
  AND event_type = 'message.outbound'
  AND occurred_at >= NOW() - INTERVAL '7 days'
  AND payload ? 'response_time_ms'
GROUP BY payload->>'llm_provider'
ORDER BY provider;
```

---

## 7. Plan de Migracion

### 7.1 Migracion SQL

Archivo: `apps/api/db/migrations/NNN_create_events_table.sql`

Contenido:

1. Crear tabla particionada `events`
2. Crear particion inicial para el mes actual y el siguiente
3. Crear los 4 indices
4. Habilitar RLS y crear politica

### 7.2 Codigo Go

Archivos nuevos:

- `apps/api/internal/domain/event.go` — tipo `Event` e interfaz `EventRepository`
- `apps/api/internal/repository/events.go` — implementacion PostgreSQL con `Insert` y `InsertBatch`
- `apps/api/internal/repository/events_test.go` — tests del repositorio

Archivos modificados:

- Services existentes (appointments, customers, treatments, messaging) — agregar llamada a `events.Insert` post-operacion, con el patron best-effort descrito en la seccion 4.2

### 7.3 Cron jobs

- Registrar job diario de limpieza (3:00 AM) en el cron scheduler existente
- Registrar job mensual de creacion de particiones

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigacion |
|---|---|---|---|
| INSERT al events table agrega latencia a operaciones | Baja | Medio | Best-effort, fire-and-forget. Si la DB esta lenta, el INSERT falla rapido y se loguea |
| Particion del mes no existe | Baja | Alto | Cron mensual crea particiones con anticipacion. Fallback: la tabla padre rechaza el insert, se loguea, no se pierde la operacion de negocio |
| Payload JSONB crece sin control | Media | Bajo | Documentar claves permitidas por event_type en el catalogo. Code review en PRs que agregan eventos |
| GIN index en payload ralentiza writes | Baja | Bajo | Monitorear. Si es problema, convertir a indice parcial o eliminar |

---

## 9. Decisiones Descartadas

### 9.1 Tabla separada por tipo de evento

**Descartada porque:** Multiples tablas incrementan complejidad de mantenimiento, migraciones, y queries cross-event. Una tabla unica con JSONB payload es mas simple y suficiente para el volumen esperado (<100K eventos/mes en los primeros 12 meses).

### 9.2 Solo RabbitMQ + consumer dedicado

**Descartada porque:** Agrega un consumer mas al sistema, introduce latencia y complejidad de retry/dead-letter. El INSERT directo es mas simple, predecible, y suficiente dado que es best-effort.

### 9.3 TimescaleDB o ClickHouse

**Descartada porque:** Agrega infraestructura. PostgreSQL con particionamiento nativo es suficiente para el volumen actual y proyectado. Se puede migrar a TimescaleDB si el volumen lo justifica (>10M eventos/mes).

---

## 10. Criterios de Aceptacion

- [ ] Tabla `events` creada con particionamiento mensual y RLS activo
- [ ] Los 12 eventos del catalogo se persisten correctamente desde sus respectivos services
- [ ] Fallo en INSERT no rompe ninguna operacion de negocio
- [ ] Las 4 queries operacionales retornan resultados correctos
- [ ] Cron de limpieza respeta los tiers de retencion
- [ ] Cron de particiones crea la particion del mes siguiente
- [ ] Tests unitarios para el repositorio con cobertura >= 80%
- [ ] Tests de integracion para el patron best-effort (simular fallo de INSERT)
