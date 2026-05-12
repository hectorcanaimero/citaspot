# Treatment Sessions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agregar la entidad `TreatmentSession` que permite agendar sesiones futuras o registrar sesiones completadas para un tratamiento, con recálculo automático de `completed_sessions` en la misma transacción.

**Architecture:** Tabla `treatment_sessions` independiente con RLS. El repo maneja transacciones internamente al cambiar status. El service valida lógica de negocio. Handler expone rutas REST anidadas bajo `/treatments/:id/sessions`. Frontend: nuevo componente `SessionModal`, nueva página `/treatments/[id]`, y card compacta en el detalle del cliente.

**Tech Stack:** Go 1.22 · pgx/v5 · Fiber v2 · go-playground/validator · Next.js 14 · TypeScript · Tailwind CSS

---

## File Map

| Archivo | Acción |
|---|---|
| `apps/api/db/migrations/019_treatment_sessions.sql` | Crear |
| `apps/api/internal/domain/types.go` | Modificar — añadir structs TreatmentSession |
| `apps/api/internal/domain/interfaces.go` | Modificar — añadir interfaces repo y service |
| `apps/api/internal/repository/treatment_sessions.go` | Crear |
| `apps/api/internal/service/treatment_sessions.go` | Crear |
| `apps/api/internal/handler/treatment_sessions.go` | Crear |
| `apps/api/cmd/server/main.go` | Modificar — instanciar y registrar rutas |
| `apps/web/lib/api.ts` | Modificar — añadir `treatmentSessions` |
| `apps/web/lib/i18n/locales/en.ts` | Modificar — añadir keys de sesiones |
| `apps/web/lib/i18n/locales/es.ts` | Modificar — añadir keys de sesiones |
| `apps/web/lib/i18n/locales/pt.ts` | Modificar — añadir keys de sesiones |
| `apps/web/components/sessions/SessionModal.tsx` | Crear |
| `apps/web/app/dashboard/treatments/[id]/page.tsx` | Crear |
| `apps/web/app/dashboard/clients/[id]/page.tsx` | Modificar — añadir sesiones a la card |

---

## Task 1: Migración DB

**Files:**
- Create: `apps/api/db/migrations/019_treatment_sessions.sql`

- [ ] **Step 1: Crear el archivo de migración**

```sql
-- apps/api/db/migrations/019_treatment_sessions.sql

CREATE TABLE treatment_sessions (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  treatment_id      UUID NOT NULL REFERENCES treatments(id) ON DELETE CASCADE,
  professional_id   UUID NOT NULL REFERENCES professionals(id),

  status            TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'completed', 'cancelled')),

  scheduled_at      TIMESTAMPTZ NOT NULL,
  duration_minutes  INT,
  completed_at      TIMESTAMPTZ,

  procedures_done   TEXT,
  notes             TEXT,

  paid_in_session   NUMERIC(10,2),
  currency          TEXT,

  next_session_at   TIMESTAMPTZ,

  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE treatment_sessions ENABLE ROW LEVEL SECURITY;

CREATE POLICY treatment_sessions_tenant_isolation ON treatment_sessions
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE INDEX idx_treatment_sessions_treatment ON treatment_sessions(treatment_id);
CREATE INDEX idx_treatment_sessions_tenant_scheduled ON treatment_sessions(tenant_id, scheduled_at);
```

- [ ] **Step 2: Aplicar la migración**

```bash
cd /Users/al3jandro/project/agendAI
make migrate
```

Salida esperada: `Applying migration 019_treatment_sessions.sql... done`

- [ ] **Step 3: Verificar la tabla en la DB**

```bash
docker compose exec postgres psql -U citaspot -d citaspot -c "\d treatment_sessions"
```

Esperado: tabla con columnas `id, tenant_id, treatment_id, professional_id, status, scheduled_at, ...`

- [ ] **Step 4: Commit**

```bash
git add apps/api/db/migrations/019_treatment_sessions.sql
git commit -m "feat(db): add treatment_sessions table with RLS"
```

---

## Task 2: Domain types

**Files:**
- Modify: `apps/api/internal/domain/types.go`

- [ ] **Step 1: Añadir structs al final del bloque de tipos de tratamientos**

Buscar el bloque `UpdateTreatmentStatusInput` en `types.go` y añadir después:

```go
// TreatmentSession representa una sesión individual dentro de un tratamiento.
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

// TreatmentSessionInput datos para crear una sesión.
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

// UpdateTreatmentSessionInput datos para actualizar una sesión.
type UpdateTreatmentSessionInput struct {
	Status          string     `json:"status"           validate:"required,oneof=pending completed cancelled"`
	DurationMinutes *int       `json:"duration_minutes" validate:"omitempty,min=1"`
	ProceduresDone  string     `json:"procedures_done"`
	Notes           string     `json:"notes"`
	PaidInSession   *float64   `json:"paid_in_session"  validate:"omitempty,min=0"`
	Currency        string     `json:"currency"         validate:"omitempty,len=3"`
	NextSessionAt   *time.Time `json:"next_session_at"`
}
```

- [ ] **Step 2: Verificar que compila**

```bash
cd apps/api && go build ./...
```

Esperado: sin errores.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/domain/types.go
git commit -m "feat(domain): add TreatmentSession types"
```

---

## Task 3: Domain interfaces

**Files:**
- Modify: `apps/api/internal/domain/interfaces.go`

- [ ] **Step 1: Añadir interfaces después del bloque TreatmentSvc**

```go
// TreatmentSessionRepository operaciones DB para sesiones de tratamiento.
type TreatmentSessionRepository interface {
	Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *TreatmentSessionInput) (*TreatmentSession, error)
	List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*TreatmentSession, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*TreatmentSession, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *UpdateTreatmentSessionInput) (*TreatmentSession, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// TreatmentSessionSvc lógica de negocio para sesiones de tratamiento.
type TreatmentSessionSvc interface {
	Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *TreatmentSessionInput) (*TreatmentSession, error)
	List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*TreatmentSession, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*TreatmentSession, error)
	Update(ctx context.Context, tenantID, treatmentID, id uuid.UUID, input *UpdateTreatmentSessionInput) (*TreatmentSession, error)
	Delete(ctx context.Context, tenantID, treatmentID, id uuid.UUID) error
}
```

- [ ] **Step 2: Verificar que compila**

```bash
cd apps/api && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/domain/interfaces.go
git commit -m "feat(domain): add TreatmentSession repository and service interfaces"
```

---

## Task 4: Repository

**Files:**
- Create: `apps/api/internal/repository/treatment_sessions.go`

- [ ] **Step 1: Crear el archivo**

```go
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pideai/citaspot/apps/api/internal/domain"
)

const sessionCols = `
	ts.id, ts.tenant_id, ts.treatment_id, ts.professional_id,
	ts.status, ts.scheduled_at, ts.duration_minutes, ts.completed_at,
	ts.procedures_done, ts.notes, ts.paid_in_session, ts.currency,
	ts.next_session_at, ts.created_at, ts.updated_at,
	p.name AS professional_name`

type treatmentSessionRepo struct {
	db *pgxpool.Pool
}

// NewTreatmentSessionRepository crea el repositorio de sesiones de tratamiento.
func NewTreatmentSessionRepository(db *pgxpool.Pool) domain.TreatmentSessionRepository {
	return &treatmentSessionRepo{db: db}
}

func scanSession(row pgx.Row, s *domain.TreatmentSession) error {
	var (
		proceduresDone   *string
		notes            *string
		currency         *string
		professionalName *string
	)
	err := row.Scan(
		&s.ID, &s.TenantID, &s.TreatmentID, &s.ProfessionalID,
		&s.Status, &s.ScheduledAt, &s.DurationMinutes, &s.CompletedAt,
		&proceduresDone, &notes, &s.PaidInSession, &currency,
		&s.NextSessionAt, &s.CreatedAt, &s.UpdatedAt,
		&professionalName,
	)
	if err != nil {
		return err
	}
	if proceduresDone != nil {
		s.ProceduresDone = *proceduresDone
	}
	if notes != nil {
		s.Notes = *notes
	}
	if currency != nil {
		s.Currency = *currency
	}
	if professionalName != nil {
		s.ProfessionalName = *professionalName
	}
	return nil
}

// recalcCompleted actualiza completed_sessions en treatments dentro de la tx.
func recalcCompleted(ctx context.Context, tx pgx.Tx, treatmentID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE treatments
		SET completed_sessions = (
			SELECT COUNT(*) FROM treatment_sessions
			WHERE treatment_id = $1 AND status = 'completed'
		),
		updated_at = NOW()
		WHERE id = $1
	`, treatmentID)
	return err
}

func (r *treatmentSessionRepo) withTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
	return err
}

func (r *treatmentSessionRepo) Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *domain.TreatmentSessionInput) (*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	id := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO treatment_sessions
			(id, tenant_id, treatment_id, professional_id, status,
			 scheduled_at, duration_minutes, procedures_done, notes,
			 paid_in_session, currency, next_session_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())
	`,
		id, tenantID, treatmentID, input.ProfessionalID, input.Status,
		input.ScheduledAt, input.DurationMinutes,
		nullString(input.ProceduresDone), nullString(input.Notes),
		input.PaidInSession, nullString(input.Currency), input.NextSessionAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	if input.Status == "completed" {
		if err := recalcCompleted(ctx, tx, treatmentID); err != nil {
			return nil, fmt.Errorf("recalc completed: %w", err)
		}
	}

	s := &domain.TreatmentSession{}
	row2 := tx.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.id = $1
	`, id)
	if err := scanSession(row2, s); err != nil {
		return nil, fmt.Errorf("fetch created session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s, nil
}

func (r *treatmentSessionRepo) List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.treatment_id = $1
		ORDER BY ts.scheduled_at DESC
	`, treatmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.TreatmentSession
	for rows.Next() {
		s := &domain.TreatmentSession{}
		if err := scanSession(rows, s); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	tx.Commit(ctx)
	return sessions, nil
}

func (r *treatmentSessionRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	s := &domain.TreatmentSession{}
	row := tx.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.id = $1
	`, id)
	if err := scanSession(row, s); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	tx.Commit(ctx)
	return s, nil
}

func (r *treatmentSessionRepo) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	// Leer treatment_id antes de actualizar
	var treatmentID uuid.UUID
	var oldStatus string
	err = tx.QueryRow(ctx, `SELECT treatment_id, status FROM treatment_sessions WHERE id = $1`, id).
		Scan(&treatmentID, &oldStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	completedAt := "NULL"
	if input.Status == "completed" {
		completedAt = "NOW()"
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		UPDATE treatment_sessions SET
			status           = $2,
			duration_minutes = $3,
			procedures_done  = $4,
			notes            = $5,
			paid_in_session  = $6,
			currency         = $7,
			next_session_at  = $8,
			completed_at     = %s,
			updated_at       = NOW()
		WHERE id = $1
	`, completedAt),
		id, input.Status, input.DurationMinutes,
		nullString(input.ProceduresDone), nullString(input.Notes),
		input.PaidInSession, nullString(input.Currency), input.NextSessionAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Recalcular si el status cambia y afecta el conteo
	if oldStatus != input.Status && (input.Status == "completed" || input.Status == "cancelled" || oldStatus == "completed") {
		if err := recalcCompleted(ctx, tx, treatmentID); err != nil {
			return nil, fmt.Errorf("recalc: %w", err)
		}
	}

	s := &domain.TreatmentSession{}
	row := tx.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.id = $1
	`, id)
	if err := scanSession(row, s); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s, nil
}

func (r *treatmentSessionRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return err
	}

	var treatmentID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `DELETE FROM treatment_sessions WHERE id = $1 RETURNING treatment_id, status`, id).
		Scan(&treatmentID, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}

	// Solo recalcular si era una sesión completada (afecta el conteo)
	if status == "completed" {
		if err := recalcCompleted(ctx, tx, treatmentID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// nullString convierte string vacío en nil para columnas TEXT nullables.
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

- [ ] **Step 2: Verificar que compila**

```bash
cd apps/api && go build ./internal/repository/...
```

Esperado: sin errores.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/repository/treatment_sessions.go
git commit -m "feat(repository): add TreatmentSessionRepository with transaction-based recalc"
```

---

## Task 5: Service

**Files:**
- Create: `apps/api/internal/service/treatment_sessions.go`

- [ ] **Step 1: Crear el archivo**

```go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/pideai/citaspot/apps/api/internal/domain"
)

type treatmentSessionSvc struct {
	repo          domain.TreatmentSessionRepository
	treatmentRepo domain.TreatmentRepository
}

// NewTreatmentSessionSvc crea el servicio de sesiones de tratamiento.
func NewTreatmentSessionSvc(repo domain.TreatmentSessionRepository, treatmentRepo domain.TreatmentRepository) domain.TreatmentSessionSvc {
	return &treatmentSessionSvc{repo: repo, treatmentRepo: treatmentRepo}
}

func (s *treatmentSessionSvc) Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *domain.TreatmentSessionInput) (*domain.TreatmentSession, error) {
	// Validar que el treatment pertenece al tenant
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return nil, fmt.Errorf("treatment not found: %w", err)
	}
	return s.repo.Create(ctx, tenantID, treatmentID, input)
}

func (s *treatmentSessionSvc) List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*domain.TreatmentSession, error) {
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return nil, fmt.Errorf("treatment not found: %w", err)
	}
	return s.repo.List(ctx, tenantID, treatmentID)
}

func (s *treatmentSessionSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TreatmentSession, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *treatmentSessionSvc) Update(ctx context.Context, tenantID, treatmentID, id uuid.UUID, input *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error) {
	// Validar que el treatment pertenece al tenant
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return nil, fmt.Errorf("treatment not found: %w", err)
	}
	return s.repo.Update(ctx, tenantID, id, input)
}

func (s *treatmentSessionSvc) Delete(ctx context.Context, tenantID, treatmentID, id uuid.UUID) error {
	if _, err := s.treatmentRepo.GetByID(ctx, tenantID, treatmentID); err != nil {
		return fmt.Errorf("treatment not found: %w", err)
	}
	return s.repo.Delete(ctx, tenantID, id)
}
```

- [ ] **Step 2: Verificar que compila**

```bash
cd apps/api && go build ./internal/service/...
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/service/treatment_sessions.go
git commit -m "feat(service): add TreatmentSessionSvc"
```

---

## Task 6: Handler

**Files:**
- Create: `apps/api/internal/handler/treatment_sessions.go`

- [ ] **Step 1: Crear el archivo**

```go
package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/pideai/citaspot/apps/api/internal/domain"
	"github.com/pideai/citaspot/apps/api/internal/middleware"
)

// TreatmentSessionHandler maneja los endpoints REST de sesiones de tratamiento.
type TreatmentSessionHandler struct {
	svc      domain.TreatmentSessionSvc
	validate *validator.Validate
}

// NewTreatmentSessionHandler crea el handler de sesiones.
func NewTreatmentSessionHandler(svc domain.TreatmentSessionSvc) *TreatmentSessionHandler {
	return &TreatmentSessionHandler{svc: svc, validate: validator.New()}
}

// List GET /api/v1/treatments/:id/sessions
func (h *TreatmentSessionHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	sessions, err := h.svc.List(c.UserContext(), tenantID, treatmentID)
	if err != nil {
		return handleServiceError(c, err)
	}

	if sessions == nil {
		sessions = []*domain.TreatmentSession{}
	}
	return c.JSON(fiber.Map{"data": sessions})
}

// Create POST /api/v1/treatments/:id/sessions
func (h *TreatmentSessionHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	var input domain.TreatmentSessionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	session, err := h.svc.Create(c.UserContext(), tenantID, treatmentID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(session)
}

// GetByID GET /api/v1/treatments/:id/sessions/:sid
func (h *TreatmentSessionHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	sessionID, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de sesión inválido"})
	}

	session, err := h.svc.GetByID(c.UserContext(), tenantID, sessionID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(session)
}

// Update PATCH /api/v1/treatments/:id/sessions/:sid
func (h *TreatmentSessionHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	sessionID, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de sesión inválido"})
	}

	var input domain.UpdateTreatmentSessionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	session, err := h.svc.Update(c.UserContext(), tenantID, treatmentID, sessionID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(session)
}

// Delete DELETE /api/v1/treatments/:id/sessions/:sid
func (h *TreatmentSessionHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no autorizado"})
	}

	treatmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de tratamiento inválido"})
	}

	sessionID, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id de sesión inválido"})
	}

	if err := h.svc.Delete(c.UserContext(), tenantID, treatmentID, sessionID); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 2: Verificar que compila**

```bash
cd apps/api && go build ./internal/handler/...
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/handler/treatment_sessions.go
git commit -m "feat(handler): add TreatmentSessionHandler"
```

---

## Task 7: Rutas en main.go

**Files:**
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Instanciar repositorio, servicio y handler**

Buscar la sección donde se instancia `treatmentRepo` y `treatmentSvc` (alrededor de línea 150-200). Añadir después:

```go
treatmentSessionRepo := repository.NewTreatmentSessionRepository(pool)
treatmentSessionSvc  := service.NewTreatmentSessionSvc(treatmentSessionRepo, treatmentRepo)
treatmentSessionHandler := handler.NewTreatmentSessionHandler(treatmentSessionSvc)
```

- [ ] **Step 2: Registrar rutas**

Buscar el bloque donde se registran las rutas de `treatments` (alrededor de línea 571). Añadir después:

```go
// Sesiones de tratamiento
treatmentSessions := treatments.Group("/:id/sessions")
treatmentSessions.Get("/", treatmentSessionHandler.List)
treatmentSessions.Post("/", treatmentSessionHandler.Create)
treatmentSessions.Get("/:sid", treatmentSessionHandler.GetByID)
treatmentSessions.Patch("/:sid", treatmentSessionHandler.Update)
treatmentSessions.Delete("/:sid", treatmentSessionHandler.Delete)
```

- [ ] **Step 3: Verificar que compila**

```bash
cd apps/api && go build ./cmd/server/...
```

- [ ] **Step 4: Smoke test con curl**

```bash
# Levantar la API
make dev

# Listar sesiones de un tratamiento (reemplazar UUIDs con valores reales de tu DB)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:3001/api/v1/treatments/<TREATMENT_ID>/sessions | jq .
```

Esperado: `{"data": []}` (lista vacía si no hay sesiones aún).

- [ ] **Step 5: Commit**

```bash
git add apps/api/cmd/server/main.go
git commit -m "feat(api): register treatment session routes"
```

---

## Task 8: Frontend API client

**Files:**
- Modify: `apps/web/lib/api.ts`

- [ ] **Step 1: Añadir tipo e interface**

Al final del bloque de tipos (después de `Treatment`):

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

export interface TreatmentSessionInput {
  professional_id: string;
  status: 'pending' | 'completed';
  scheduled_at: string;
  duration_minutes?: number;
  procedures_done?: string;
  notes?: string;
  paid_in_session?: number;
  currency?: string;
  next_session_at?: string;
}

export interface UpdateTreatmentSessionInput {
  status: 'pending' | 'completed' | 'cancelled';
  duration_minutes?: number;
  procedures_done?: string;
  notes?: string;
  paid_in_session?: number;
  currency?: string;
  next_session_at?: string;
}
```

- [ ] **Step 2: Añadir objeto `treatmentSessions`**

Después del objeto `treatments`:

```typescript
export const treatmentSessions = {
  async list(treatmentId: string): Promise<{ data: TreatmentSession[] }> {
    return request(`/api/v1/treatments/${treatmentId}/sessions`);
  },

  async create(treatmentId: string, data: TreatmentSessionInput): Promise<TreatmentSession> {
    return request(`/api/v1/treatments/${treatmentId}/sessions`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(treatmentId: string, sessionId: string, data: UpdateTreatmentSessionInput): Promise<TreatmentSession> {
    return request(`/api/v1/treatments/${treatmentId}/sessions/${sessionId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async remove(treatmentId: string, sessionId: string): Promise<void> {
    return request(`/api/v1/treatments/${treatmentId}/sessions/${sessionId}`, {
      method: 'DELETE',
    });
  },
};
```

- [ ] **Step 3: Verificar que TypeScript compila**

```bash
cd apps/web && npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/lib/api.ts
git commit -m "feat(web): add TreatmentSession types and API client"
```

---

## Task 9: i18n — añadir keys de sesiones

**Files:**
- Modify: `apps/web/lib/i18n/locales/en.ts`
- Modify: `apps/web/lib/i18n/locales/es.ts`
- Modify: `apps/web/lib/i18n/locales/pt.ts`

- [ ] **Step 1: Añadir keys en en.ts**

Dentro del objeto de translations, añadir una sección `sessions`:

```typescript
sessions: {
  title: 'Sessions',
  newSession: '+ Session',
  schedule: 'Schedule',
  logCompleted: 'Log completed',
  scheduleSession: 'Schedule session',
  logCompletedSession: 'Log completed session',
  professional: 'Professional',
  date: 'Date',
  time: 'Time',
  estimatedDuration: 'Estimated duration',
  actualDuration: 'Actual duration',
  priorNotes: 'Prior notes (optional)',
  proceduresDone: 'Procedures performed',
  clinicalNotes: 'Clinical notes (optional)',
  paidInSession: 'Amount collected',
  nextSession: 'Next session (optional)',
  scheduleBtn: 'Schedule session',
  logBtn: 'Log completed session',
  viewAll: 'View all sessions →',
  lastSessions: 'Last sessions',
  noSessions: 'No sessions yet',
  status: {
    pending: 'Pending',
    completed: 'Completed',
    cancelled: 'Cancelled',
  },
  actions: {
    complete: 'Mark as completed',
    cancel: 'Cancel',
    delete: 'Delete',
  },
  duration: {
    min15: '15 min',
    min30: '30 min',
    min45: '45 min',
    min60: '1 hour',
    min90: '1.5 hours',
  },
},
```

- [ ] **Step 2: Añadir keys en es.ts**

```typescript
sessions: {
  title: 'Sesiones',
  newSession: '+ Sesión',
  schedule: 'Agendar',
  logCompleted: 'Registrar completada',
  scheduleSession: 'Agendar sesión',
  logCompletedSession: 'Registrar sesión completada',
  professional: 'Profesional',
  date: 'Fecha',
  time: 'Hora',
  estimatedDuration: 'Duración estimada',
  actualDuration: 'Duración real',
  priorNotes: 'Notas previas (opcional)',
  proceduresDone: 'Procedimientos realizados',
  clinicalNotes: 'Notas clínicas (opcional)',
  paidInSession: 'Cobrado en sesión',
  nextSession: 'Próxima sesión (opcional)',
  scheduleBtn: 'Agendar sesión',
  logBtn: 'Registrar sesión completada',
  viewAll: 'Ver todas las sesiones →',
  lastSessions: 'Últimas sesiones',
  noSessions: 'Sin sesiones aún',
  status: {
    pending: 'Pendiente',
    completed: 'Completada',
    cancelled: 'Cancelada',
  },
  actions: {
    complete: 'Marcar como completada',
    cancel: 'Cancelar',
    delete: 'Eliminar',
  },
  duration: {
    min15: '15 min',
    min30: '30 min',
    min45: '45 min',
    min60: '1 hora',
    min90: '1.5 horas',
  },
},
```

- [ ] **Step 3: Añadir keys en pt.ts**

```typescript
sessions: {
  title: 'Sessões',
  newSession: '+ Sessão',
  schedule: 'Agendar',
  logCompleted: 'Registrar concluída',
  scheduleSession: 'Agendar sessão',
  logCompletedSession: 'Registrar sessão concluída',
  professional: 'Profissional',
  date: 'Data',
  time: 'Hora',
  estimatedDuration: 'Duração estimada',
  actualDuration: 'Duração real',
  priorNotes: 'Notas prévias (opcional)',
  proceduresDone: 'Procedimentos realizados',
  clinicalNotes: 'Notas clínicas (opcional)',
  paidInSession: 'Cobrado na sessão',
  nextSession: 'Próxima sessão (opcional)',
  scheduleBtn: 'Agendar sessão',
  logBtn: 'Registrar sessão concluída',
  viewAll: 'Ver todas as sessões →',
  lastSessions: 'Últimas sessões',
  noSessions: 'Sem sessões ainda',
  status: {
    pending: 'Pendente',
    completed: 'Concluída',
    cancelled: 'Cancelada',
  },
  actions: {
    complete: 'Marcar como concluída',
    cancel: 'Cancelar',
    delete: 'Excluir',
  },
  duration: {
    min15: '15 min',
    min30: '30 min',
    min45: '45 min',
    min60: '1 hora',
    min90: '1,5 horas',
  },
},
```

- [ ] **Step 4: Verificar TypeScript**

```bash
cd apps/web && npx tsc --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/i18n/locales/
git commit -m "feat(i18n): add session keys to en/es/pt"
```

---

## Task 10: SessionModal component

**Files:**
- Create: `apps/web/components/sessions/SessionModal.tsx`

- [ ] **Step 1: Crear el componente**

```tsx
'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { treatmentSessions, TreatmentSessionInput, professionals as profApi } from '@/lib/api';

interface Professional {
  id: string;
  name: string;
}

interface SessionModalProps {
  isOpen: boolean;
  onClose: () => void;
  treatmentId: string;
  onSuccess: () => void;
}

type Mode = 'schedule' | 'log';

const DURATION_OPTIONS = [15, 30, 45, 60, 90] as const;

export function SessionModal({ isOpen, onClose, treatmentId, onSuccess }: SessionModalProps) {
  const t = useTranslations('sessions');
  const [mode, setMode] = useState<Mode>('schedule');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [professionals, setProfessionals] = useState<Professional[]>([]);

  useEffect(() => {
    if (!isOpen) return;
    profApi.list().then(res => setProfessionals(res.data.map(p => ({ id: p.id, name: p.name }))));
  }, [isOpen]);

  const [form, setForm] = useState({
    professional_id: professionals[0]?.id ?? '',
    scheduled_at: '',
    time: '10:00',
    duration_minutes: 45,
    notes: '',
    procedures_done: '',
    paid_in_session: '',
    currency: 'USD',
    next_session_at: '',
  });

  if (!isOpen) return null;

  const set = (key: string, value: string | number) =>
    setForm(prev => ({ ...prev, [key]: value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const scheduledAt = mode === 'schedule'
        ? new Date(`${form.scheduled_at}T${form.time}:00`).toISOString()
        : new Date(form.scheduled_at).toISOString();

      const input: TreatmentSessionInput = {
        professional_id: form.professional_id,
        status: mode === 'schedule' ? 'pending' : 'completed',
        scheduled_at: scheduledAt,
        duration_minutes: form.duration_minutes || undefined,
        notes: form.notes || undefined,
        procedures_done: mode === 'log' ? form.procedures_done || undefined : undefined,
        paid_in_session: mode === 'log' && form.paid_in_session ? Number(form.paid_in_session) : undefined,
        currency: mode === 'log' && form.paid_in_session ? form.currency : undefined,
        next_session_at: form.next_session_at ? new Date(form.next_session_at).toISOString() : undefined,
      };

      await treatmentSessions.create(treatmentId, input);
      onSuccess();
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error al guardar la sesión');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-slate-900">
              {mode === 'schedule' ? t('scheduleSession') : t('logCompletedSession')}
            </h2>
            <button onClick={onClose} className="text-slate-400 hover:text-slate-600">✕</button>
          </div>

          {/* Toggle */}
          <div className="flex gap-1 bg-slate-100 rounded-lg p-1 mb-5">
            <button
              type="button"
              onClick={() => setMode('schedule')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-all ${
                mode === 'schedule'
                  ? 'bg-white text-indigo-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              }`}
            >
              📅 {t('schedule')}
            </button>
            <button
              type="button"
              onClick={() => setMode('log')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-all ${
                mode === 'log'
                  ? 'bg-white text-indigo-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              }`}
            >
              ✓ {t('logCompleted')}
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Profesional */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {t('professional')}
              </label>
              <select
                value={form.professional_id}
                onChange={e => set('professional_id', e.target.value)}
                required
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {professionals.map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            {/* Fecha + Hora (schedule) o solo fecha (log) */}
            <div className={`grid gap-3 ${mode === 'schedule' ? 'grid-cols-2' : 'grid-cols-1'}`}>
              <div>
                <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                  {t('date')}
                </label>
                <input
                  type="date"
                  value={form.scheduled_at}
                  onChange={e => set('scheduled_at', e.target.value)}
                  required
                  className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>
              {mode === 'schedule' && (
                <div>
                  <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                    {t('time')}
                  </label>
                  <input
                    type="time"
                    value={form.time}
                    onChange={e => set('time', e.target.value)}
                    required
                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </div>
              )}
            </div>

            {/* Duración */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {mode === 'schedule' ? t('estimatedDuration') : t('actualDuration')}
              </label>
              <select
                value={form.duration_minutes}
                onChange={e => set('duration_minutes', Number(e.target.value))}
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {DURATION_OPTIONS.map(min => (
                  <option key={min} value={min}>{t(`duration.min${min}`)}</option>
                ))}
              </select>
            </div>

            {/* Procedimientos (solo log) */}
            {mode === 'log' && (
              <div>
                <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                  {t('proceduresDone')}
                </label>
                <textarea
                  value={form.procedures_done}
                  onChange={e => set('procedures_done', e.target.value)}
                  rows={3}
                  className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
                  placeholder="Ej: Ajuste de brackets superiores, radiografía..."
                />
              </div>
            )}

            {/* Notas */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {mode === 'log' ? t('clinicalNotes') : t('priorNotes')}
              </label>
              <textarea
                value={form.notes}
                onChange={e => set('notes', e.target.value)}
                rows={2}
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
              />
            </div>

            {/* Cobrado (solo log) */}
            {mode === 'log' && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                    {t('paidInSession')}
                  </label>
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    value={form.paid_in_session}
                    onChange={e => set('paid_in_session', e.target.value)}
                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    placeholder="0.00"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                    Moneda
                  </label>
                  <select
                    value={form.currency}
                    onChange={e => set('currency', e.target.value)}
                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="USD">USD</option>
                    <option value="DOP">DOP</option>
                    <option value="VES">VES</option>
                    <option value="EUR">EUR</option>
                  </select>
                </div>
              </div>
            )}

            {/* Próxima sesión sugerida */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {t('nextSession')}
              </label>
              <input
                type="date"
                value={form.next_session_at}
                onChange={e => set('next_session_at', e.target.value)}
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>

            {error && (
              <p className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className={`w-full py-2.5 px-4 rounded-lg text-sm font-semibold text-white transition-colors ${
                mode === 'schedule'
                  ? 'bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-300'
                  : 'bg-green-600 hover:bg-green-700 disabled:bg-green-300'
              }`}
            >
              {loading ? '...' : mode === 'schedule' ? t('scheduleBtn') : t('logBtn')}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verificar TypeScript**

```bash
cd apps/web && npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add apps/web/components/sessions/
git commit -m "feat(web): add SessionModal component"
```

---

## Task 11: Página de detalle del tratamiento

**Files:**
- Create: `apps/web/app/dashboard/treatments/[id]/page.tsx`

- [ ] **Step 1: Crear la página**

```tsx
'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { treatments, treatmentSessions, Treatment, TreatmentSession } from '@/lib/api';
import { SessionModal } from '@/components/sessions/SessionModal';

const STATUS_COLORS: Record<string, string> = {
  pending:   'bg-violet-100 text-violet-700',
  completed: 'bg-green-100 text-green-700',
  cancelled: 'bg-red-100 text-red-600',
};

const SESSION_DOT: Record<string, string> = {
  pending:   'bg-violet-400',
  completed: 'bg-green-500',
  cancelled: 'bg-red-400',
};

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('es', { day: 'numeric', month: 'short', year: 'numeric' });
}

export default function TreatmentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const t = useTranslations('sessions');

  const [treatment, setTreatment] = useState<Treatment | null>(null);
  const [sessions, setSessions] = useState<TreatmentSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [completing, setCompleting] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [tr, sessionsRes] = await Promise.all([
        treatments.getById(id),
        treatmentSessions.list(id),
      ]);
      setTreatment(tr);
      setSessions(sessionsRes.data);
    } catch {
      router.push('/dashboard/treatments');
    } finally {
      setLoading(false);
    }
  }, [id, router]);

  useEffect(() => { load(); }, [load]);

  const handleComplete = async (session: TreatmentSession) => {
    setCompleting(session.id);
    try {
      await treatmentSessions.update(id, session.id, {
        status: 'completed',
        duration_minutes: session.duration_minutes ?? undefined,
        procedures_done: session.procedures_done,
        notes: session.notes,
      });
      load();
    } finally {
      setCompleting(null);
    }
  };

  const handleCancel = async (session: TreatmentSession) => {
    await treatmentSessions.update(id, session.id, {
      status: 'cancelled',
      duration_minutes: session.duration_minutes ?? undefined,
    });
    load();
  };

  const handleDelete = async (session: TreatmentSession) => {
    if (!confirm('¿Eliminar esta sesión?')) return;
    await treatmentSessions.remove(id, session.id);
    load();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
      </div>
    );
  }

  if (!treatment) return null;

  const total = treatment.total_sessions;
  const completed = treatment.completed_sessions;

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-start justify-between mb-8">
        <div>
          <button
            onClick={() => router.back()}
            className="text-sm text-slate-500 hover:text-slate-700 mb-2 flex items-center gap-1"
          >
            ← Volver
          </button>
          <h1 className="text-2xl font-bold text-slate-900">{treatment.name}</h1>
          <p className="text-slate-500 text-sm mt-1">
            {treatment.customer_name && `${treatment.customer_name} · `}
            {completed}{total ? `/${total}` : ''} {t('title').toLowerCase()}
          </p>
        </div>
        <button
          onClick={() => setModalOpen(true)}
          className="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-semibold hover:bg-indigo-700 transition-colors"
        >
          {t('newSession')}
        </button>
      </div>

      {/* Sessions list */}
      <div>
        <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
          {t('title')} ({sessions.length})
        </p>

        {sessions.length === 0 ? (
          <div className="text-center py-16 text-slate-400">
            <p className="text-4xl mb-3">📋</p>
            <p>{t('noSessions')}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {sessions.map((session, index) => (
              <div
                key={session.id}
                className={`border rounded-xl p-4 flex items-center gap-4 ${
                  session.status === 'cancelled' ? 'opacity-60 bg-slate-50' : 'bg-white'
                }`}
              >
                {/* Número */}
                <div className={`w-9 h-9 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ${
                  session.status === 'completed' ? 'bg-green-100 text-green-700' :
                  session.status === 'cancelled' ? 'bg-red-100 text-red-500' :
                  'bg-indigo-100 text-indigo-700'
                }`}>
                  {session.status === 'cancelled' ? '✕' : sessions.length - index}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <p className={`text-sm font-medium text-slate-900 ${session.status === 'cancelled' ? 'line-through text-slate-400' : ''}`}>
                    {session.procedures_done || (session.status === 'pending' ? 'Sesión agendada' : 'Sesión completada')}
                  </p>
                  <p className="text-xs text-slate-400 mt-0.5">
                    {formatDate(session.scheduled_at)}
                    {session.duration_minutes && ` · ${session.duration_minutes} min`}
                    {session.paid_in_session && ` · $${session.paid_in_session} cobrados`}
                    {session.professional_name && ` · ${session.professional_name}`}
                  </p>
                </div>

                {/* Status + Actions */}
                <div className="flex items-center gap-2 shrink-0">
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_COLORS[session.status]}`}>
                    {t(`status.${session.status}`)}
                  </span>
                  {session.status === 'pending' && (
                    <>
                      <button
                        onClick={() => handleComplete(session)}
                        disabled={completing === session.id}
                        className="text-xs border border-slate-200 rounded-lg px-2 py-1 hover:bg-slate-50 disabled:opacity-50"
                      >
                        {completing === session.id ? '...' : t('actions.complete')}
                      </button>
                      <button
                        onClick={() => handleCancel(session)}
                        className="text-xs text-slate-400 hover:text-slate-600"
                      >
                        {t('actions.cancel')}
                      </button>
                    </>
                  )}
                  {session.status !== 'cancelled' && (
                    <button
                      onClick={() => handleDelete(session)}
                      className="text-xs text-red-400 hover:text-red-600 ml-1"
                    >
                      {t('actions.delete')}
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <SessionModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        treatmentId={id}
        onSuccess={load}
      />
    </div>
  );
}
```

- [ ] **Step 2: Verificar TypeScript**

```bash
cd apps/web && npx tsc --noEmit
```

- [ ] **Step 3: Navegar a la página en el browser**

```
http://localhost:3000/dashboard/treatments/<TREATMENT_ID>
```

Verificar: muestra nombre del tratamiento, botón `+ Sesión`, lista vacía con mensaje "Sin sesiones aún".

- [ ] **Step 4: Commit**

```bash
git add apps/web/app/dashboard/treatments/
git commit -m "feat(web): add treatment detail page with sessions list"
```

---

## Task 12: Modificar card del cliente

**Files:**
- Modify: `apps/web/app/dashboard/clients/[id]/page.tsx`

- [ ] **Step 1: Importar tipos y API**

Al inicio del archivo, añadir a los imports existentes:

```typescript
import { treatmentSessions, TreatmentSession } from '@/lib/api';
import { SessionModal } from '@/components/sessions/SessionModal';
```

- [ ] **Step 2: Añadir estado de sesiones en el componente principal**

Dentro del componente que muestra los tratamientos, añadir estado para sesiones y modal:

```typescript
const [sessionsByTreatment, setSessionsByTreatment] = useState<Record<string, TreatmentSession[]>>({});
const [sessionModalTreatment, setSessionModalTreatment] = useState<string | null>(null);

// Cargar las últimas sesiones de cada tratamiento al montar
useEffect(() => {
  if (!customerTreatments.length) return;
  Promise.all(
    customerTreatments.map(t =>
      treatmentSessions.list(t.id).then(res => ({ id: t.id, sessions: res.data.slice(0, 3) }))
    )
  ).then(results => {
    const map: Record<string, TreatmentSession[]> = {};
    results.forEach(r => { map[r.id] = r.sessions; });
    setSessionsByTreatment(map);
  });
}, [customerTreatments]);
```

- [ ] **Step 3: Actualizar la TreatmentsSection**

Dentro del componente `TreatmentsSection` (o donde se renderizan las cards de tratamiento), añadir después de la barra de progreso:

```tsx
{/* Últimas sesiones */}
{(() => {
  const sessions = sessionsByTreatment[treatment.id] ?? [];
  if (sessions.length === 0) return null;
  return (
    <div className="border-t border-slate-100 pt-3 mt-3">
      <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2">
        {t('sessions.lastSessions')}
      </p>
      <div className="space-y-1">
        {sessions.map(s => (
          <div key={s.id} className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                s.status === 'completed' ? 'bg-green-500' :
                s.status === 'cancelled' ? 'bg-red-400' : 'bg-violet-400'
              }`} />
              <span className="text-slate-700 truncate max-w-[180px]">
                {s.procedures_done || (s.status === 'pending' ? 'Sesión agendada' : 'Sesión completada')}
              </span>
            </div>
            <span className="text-slate-400 shrink-0 ml-2">
              {new Date(s.scheduled_at).toLocaleDateString('es', { day: 'numeric', month: 'short' })}
            </span>
          </div>
        ))}
      </div>
      <a
        href={`/dashboard/treatments/${treatment.id}`}
        className="text-xs text-indigo-600 hover:text-indigo-700 mt-2 inline-block"
      >
        {t('sessions.viewAll')}
      </a>
    </div>
  );
})()}
```

- [ ] **Step 4: Añadir botón `+ Sesión` en el header de la card**

En el header de cada card de tratamiento (donde está el badge de status), añadir:

```tsx
<button
  onClick={() => setSessionModalTreatment(treatment.id)}
  className="text-xs bg-indigo-600 text-white px-2.5 py-1 rounded-md hover:bg-indigo-700 transition-colors"
>
  {t('sessions.newSession')}
</button>
```

- [ ] **Step 5: Añadir el SessionModal al final del componente**

```tsx
{sessionModalTreatment && (
  <SessionModal
    isOpen={!!sessionModalTreatment}
    onClose={() => setSessionModalTreatment(null)}
    treatmentId={sessionModalTreatment}
    onSuccess={() => {
      setSessionModalTreatment(null);
      // Recargar sesiones del tratamiento afectado
      treatmentSessions.list(sessionModalTreatment).then(res => {
        setSessionsByTreatment(prev => ({
          ...prev,
          [sessionModalTreatment]: res.data.slice(0, 3),
        }));
      });
    }}
  />
)}
```

- [ ] **Step 6: Verificar TypeScript**

```bash
cd apps/web && npx tsc --noEmit
```

- [ ] **Step 7: Verificar en browser**

```
http://localhost:3000/dashboard/clients/<CLIENT_ID>
```

Verificar: cada tratamiento muestra sección "Últimas sesiones" (vacía al inicio), botón `+ Sesión` funcional, y link "Ver todas las sesiones →" navega a `/dashboard/treatments/<TREATMENT_ID>`.

- [ ] **Step 8: Commit final**

```bash
git add apps/web/app/dashboard/clients/
git commit -m "feat(web): add sessions summary and modal to client treatment cards"
```
