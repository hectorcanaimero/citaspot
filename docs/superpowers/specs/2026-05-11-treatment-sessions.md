# Spec: Treatment Sessions

**Date:** 2026-05-11  
**Status:** approved  

---

## Objetivo

Permitir que los usuarios registren y gestionen sesiones individuales dentro de un tratamiento. Cada sesión puede agendarse en el futuro o registrarse manualmente como completada. El contador `completed_sessions` en `treatments` se recalcula automáticamente desde la tabla de sesiones, eliminando el drift del contador manual.

---

## Modelo de datos

### Nueva tabla: `treatment_sessions` (migración `019_treatment_sessions.sql`)

```sql
CREATE TABLE treatment_sessions (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id),
  treatment_id        UUID NOT NULL REFERENCES treatments(id) ON DELETE CASCADE,
  professional_id     UUID NOT NULL REFERENCES professionals(id),

  status              TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'completed', 'cancelled')),

  scheduled_at        TIMESTAMPTZ NOT NULL,
  duration_minutes    INT,
  completed_at        TIMESTAMPTZ,

  procedures_done     TEXT,
  notes               TEXT,

  paid_in_session     NUMERIC(10,2),
  currency            TEXT,

  next_session_at     TIMESTAMPTZ,

  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE treatment_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON treatment_sessions
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE INDEX idx_treatment_sessions_treatment_id ON treatment_sessions(treatment_id);
CREATE INDEX idx_treatment_sessions_tenant_scheduled ON treatment_sessions(tenant_id, scheduled_at);
```

### Cambio en `treatments`

`completed_sessions` deja de ser un contador manual. El service lo recalcula como:

```sql
SELECT COUNT(*) FROM treatment_sessions
WHERE treatment_id = $1 AND status = 'completed'
```

Este recalculo ocurre dentro de la misma transacción cada vez que una sesión cambia de estado a `completed` o `cancelled`.

---

## Backend

### Domain (`apps/api/internal/domain/types.go`)

```go
type TreatmentSession struct {
    ID              uuid.UUID  `json:"id"`
    TenantID        uuid.UUID  `json:"tenant_id"`
    TreatmentID     uuid.UUID  `json:"treatment_id"`
    ProfessionalID  uuid.UUID  `json:"professional_id"`
    Status          string     `json:"status"`
    ScheduledAt     time.Time  `json:"scheduled_at"`
    DurationMinutes *int       `json:"duration_minutes,omitempty"`
    CompletedAt     *time.Time `json:"completed_at,omitempty"`
    ProceduresDone  string     `json:"procedures_done,omitempty"`
    Notes           string     `json:"notes,omitempty"`
    PaidInSession   *float64   `json:"paid_in_session,omitempty"`
    Currency        string     `json:"currency,omitempty"`
    NextSessionAt   *time.Time `json:"next_session_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`

    // JOIN expandido
    ProfessionalName string `json:"professional_name,omitempty"`
}

type TreatmentSessionInput struct {
    ProfessionalID  uuid.UUID  `json:"professional_id"  validate:"required"`
    Status          string     `json:"status"           validate:"required,oneof=pending completed"`
    ScheduledAt     time.Time  `json:"scheduled_at"     validate:"required"`
    DurationMinutes *int       `json:"duration_minutes" validate:"omitempty,min=1"`
    ProceduresDone  string     `json:"procedures_done"`
    Notes           string     `json:"notes"`
    PaidInSession   *float64   `json:"paid_in_session"  validate:"omitempty,min=0"`
    Currency        string     `json:"currency"         validate:"omitempty,len=3"`
    NextSessionAt   *time.Time `json:"next_session_at"`
}

type UpdateTreatmentSessionInput struct {
    Status          string     `json:"status"           validate:"required,oneof=pending completed cancelled"`
    DurationMinutes *int       `json:"duration_minutes" validate:"omitempty,min=1"`
    ProceduresDone  string     `json:"procedures_done"`
    Notes           string     `json:"notes"`
    PaidInSession   *float64   `json:"paid_in_session"  validate:"omitempty,min=0"`
    NextSessionAt   *time.Time `json:"next_session_at"`
}
```

### Interfaz (`domain/interfaces.go`)

```go
type TreatmentSessionRepository interface {
    Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input TreatmentSessionInput) (*TreatmentSession, error)
    List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*TreatmentSession, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*TreatmentSession, error)
    Update(ctx context.Context, tenantID, id uuid.UUID, input UpdateTreatmentSessionInput) (*TreatmentSession, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    RecalcCompleted(ctx context.Context, tx pgx.Tx, treatmentID uuid.UUID) error
}

type TreatmentSessionService interface {
    Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input TreatmentSessionInput) (*TreatmentSession, error)
    List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*TreatmentSession, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*TreatmentSession, error)
    Update(ctx context.Context, tenantID, treatmentID, id uuid.UUID, input UpdateTreatmentSessionInput) (*TreatmentSession, error)
    Delete(ctx context.Context, tenantID, treatmentID, id uuid.UUID) error
}
```

### Repository (`apps/api/internal/repository/treatment_sessions.go`)

- `Create`: inserta sesión. Si `status=completed`, llama `RecalcCompleted` en transacción.
- `List`: query con JOIN a `professionals` para `professional_name`. Ordenado por `scheduled_at DESC`.
- `GetByID`: fetch único.
- `Update`: actualiza campos. Si `status` cambia a `completed` o `cancelled`, llama `RecalcCompleted` en transacción.
- `Delete`: soft-delete no requerido, DELETE real. Llama `RecalcCompleted` en transacción si la sesión era `completed`.
- `RecalcCompleted`: `UPDATE treatments SET completed_sessions = (SELECT COUNT(*) ...) WHERE id = $1`.

### Service (`apps/api/internal/service/treatment_sessions.go`)

Delega al repository. Valida que el `treatment_id` pertenece al tenant antes de operar. No contiene queries SQL directas.

### Handler (`apps/api/internal/handler/treatment_sessions.go`)

```
GET    /api/v1/treatments/:id/sessions           → List
POST   /api/v1/treatments/:id/sessions           → Create
GET    /api/v1/treatments/:id/sessions/:sid      → GetByID
PATCH  /api/v1/treatments/:id/sessions/:sid      → Update
DELETE /api/v1/treatments/:id/sessions/:sid      → Delete
```

Todos los endpoints requieren JWT + tenant middleware.

### Router (`apps/api/cmd/server/main.go`)

Registrar las nuevas rutas bajo el grupo `/api/v1` existente.

---

## Frontend

### Tipos (`apps/web/lib/api.ts`)

```typescript
export interface TreatmentSession {
  id: string;
  treatment_id: string;
  professional_id: string;
  professional_name?: string;
  status: 'pending' | 'completed' | 'cancelled';
  scheduled_at: string;
  duration_minutes: number | null;
  completed_at: string | null;
  procedures_done: string;
  notes: string;
  paid_in_session: number | null;
  currency: string;
  next_session_at: string | null;
  created_at: string;
  updated_at: string;
}

export const treatmentSessions = {
  async list(treatmentId: string): Promise<{ data: TreatmentSession[] }>,
  async create(treatmentId: string, data: Partial<TreatmentSession>): Promise<TreatmentSession>,
  async update(treatmentId: string, sessionId: string, data: Partial<TreatmentSession>): Promise<TreatmentSession>,
  async remove(treatmentId: string, sessionId: string): Promise<void>,
}
```

### Página del tratamiento (`apps/web/app/dashboard/treatments/[id]/page.tsx`) — NUEVA

- Header con nombre, cliente, profesional, progreso `N/M sesiones`
- Botón `+ Nueva sesión` → abre `SessionModal`
- Lista de sesiones ordenadas por `scheduled_at DESC`
- Cada sesión: número, estado visual (color por status), procedimientos, fecha, duración, monto cobrado
- Acciones inline: Completar (solo `pending`), Cancelar, Eliminar

### Card del tratamiento en cliente (`apps/web/app/dashboard/clients/[id]/page.tsx`) — MODIFICAR

- Añadir bajo la barra de progreso: lista compacta de las últimas 3 sesiones
- Cada sesión: dot de color por status, número, descripción breve, fecha
- Botón `+ Sesión` en el header de la card → abre `SessionModal`
- Link `Ver todas las sesiones →` navega a `/dashboard/treatments/[id]`

### Modal (`apps/web/components/sessions/SessionModal.tsx`) — NUEVO

Un modal con toggle de dos modos:

**Modo "Agendar" (crea sesión `pending`)**
- Profesional (select)
- Fecha + Hora
- Duración estimada (select: 15/30/45/60/90 min)
- Notas previas (textarea, opcional)
- Próxima sesión sugerida (date, opcional)

**Modo "Registrar completada" (crea sesión `completed`)**
- Profesional (select)
- Fecha realizada + Duración real
- Procedimientos realizados (textarea)
- Notas clínicas (textarea, opcional)
- Cobrado en sesión (número) + Moneda (select)
- Próxima sesión sugerida (date, opcional)

El toggle cambia los campos visibles. Al confirmar en modo "Registrar", el backend recalcula `completed_sessions` automáticamente.

---

## Flujo de datos

```
Usuario → "+ Nueva sesión" (modal)
  → POST /api/v1/treatments/:id/sessions
    → TreatmentSessionHandler.Create
      → TreatmentSessionService.Create
        → TreatmentSessionRepository.Create
          → INSERT treatment_sessions
          → si status=completed: RecalcCompleted (misma tx)
            → UPDATE treatments SET completed_sessions = COUNT(...)
  ← TreatmentSession response
← Refresca lista de sesiones + progreso del tratamiento
```

---

## Archivos afectados

| Archivo | Acción |
|---|---|
| `apps/api/db/migrations/019_treatment_sessions.sql` | Crear |
| `apps/api/internal/domain/types.go` | Modificar — añadir structs |
| `apps/api/internal/domain/interfaces.go` | Modificar — añadir interfaces |
| `apps/api/internal/repository/treatment_sessions.go` | Crear |
| `apps/api/internal/service/treatment_sessions.go` | Crear |
| `apps/api/internal/handler/treatment_sessions.go` | Crear |
| `apps/api/cmd/server/main.go` | Modificar — registrar rutas |
| `apps/web/lib/api.ts` | Modificar — añadir `treatmentSessions` |
| `apps/web/app/dashboard/treatments/[id]/page.tsx` | Crear |
| `apps/web/components/sessions/SessionModal.tsx` | Crear |
| `apps/web/app/dashboard/clients/[id]/page.tsx` | Modificar — añadir sesiones a la card |

---

## i18n

Añadir claves en `apps/web/lib/i18n/locales/{en,es,pt}.ts` para:

- Labels del modal (Agendar, Registrar completada, Profesional, Procedimientos, etc.)
- Estados de sesión (pending → "Pendiente", completed → "Completada", cancelled → "Cancelada")
- Textos de la página del tratamiento (título, columnas de la lista, acciones)
- Strings de la card compacta en el cliente ("Ver todas las sesiones", "Últimas sesiones")

---

## Criterios de aceptación

- [ ] Crear sesión `pending` (agendada) desde la card del cliente y desde la página del tratamiento
- [ ] Crear sesión `completed` (registro manual) con procedimientos y monto cobrado
- [ ] Marcar sesión `pending` → `completed` desde la lista
- [ ] Cancelar sesión desde la lista
- [ ] `completed_sessions` en el tratamiento se actualiza automáticamente al completar/cancelar una sesión
- [ ] Las últimas 3 sesiones se muestran en la card del cliente con estado visual
- [ ] Link "Ver todas las sesiones →" navega a la página del tratamiento
- [ ] RLS activo: un tenant no puede ver sesiones de otro tenant
- [ ] Todos los endpoints requieren JWT válido
