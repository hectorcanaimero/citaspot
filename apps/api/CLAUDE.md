# CLAUDE.md — apps/api (Go Core Service)
> Reglas específicas para el servicio Go. Lee también el CLAUDE.md raíz.

---

## 🎯 RESPONSABILIDADES DE ESTE SERVICIO

- API REST principal del SaaS (todos los endpoints de negocio)
- Gestión de autenticación y multi-tenancy con RLS
- Cliente de Evolution API (sesiones de WhatsApp)
- Productor/consumidor de colas RabbitMQ
- Integración con Stripe
- Cron jobs de recordatorios y notificaciones

**NO es responsabilidad de este servicio:**
- Lógica del LLM o RAG (→ apps/ai)
- Renderizado de UI (→ apps/web)
- Procesamiento de mensajes IA (→ apps/ai)

---

## 📁 ESTRUCTURA REAL

```
apps/api/
├── CLAUDE.md
├── cmd/
│   ├── server/
│   │   └── main.go              ← Entrypoint — setup, inyección de dependencias, start
│   └── migrate/
│       └── main.go              ← Runner de migraciones (custom, sin frameworks externos)
├── internal/
│   ├── handler/                 ← HTTP handlers (1 archivo por dominio)
│   │   ├── appointments.go
│   │   ├── auth.go
│   │   ├── billing.go
│   │   ├── customers.go
│   │   ├── errors.go            ← handleServiceError() centralizado
│   │   ├── knowledge.go
│   │   ├── professionals.go
│   │   ├── public.go            ← Endpoints públicos (booking sin auth)
│   │   ├── services.go
│   │   └── whatsapp.go
│   ├── service/                 ← Lógica de negocio (1 archivo por dominio)
│   │   ├── appointments.go
│   │   ├── auth.go
│   │   ├── availability.go      ← Engine de disponibilidad (crítico)
│   │   ├── knowledge.go
│   │   ├── professionals.go
│   │   ├── public.go
│   │   ├── services.go
│   │   └── whatsapp.go
│   ├── repository/              ← Acceso a DB — SOLO pgx + SQL directo
│   │   ├── appointments.go
│   │   ├── auth.go
│   │   ├── conversations.go
│   │   ├── customers.go
│   │   ├── knowledge.go
│   │   ├── notifications.go
│   │   ├── professionals.go
│   │   ├── reminders.go
│   │   ├── schedules.go
│   │   └── services.go
│   ├── middleware/
│   │   ├── context.go           ← Getters para extraer valores del contexto Fiber
│   │   ├── jwt.go               ← JWTMiddleware — valida HS256 + ES256 (Supabase)
│   │   └── tenant.go            ← TenantMiddleware + SetTenantID helper
│   ├── domain/                  ← Tipos, interfaces, errores — sin dependencias externas
│   │   ├── errors.go            ← Errores de dominio tipados
│   │   ├── interfaces.go        ← Interfaces de repositories y servicios
│   │   └── types.go             ← Tipos de dominio (Tenant, User, Appointment, etc.)
│   ├── worker/
│   │   ├── outbound.go          ← Consumer de wa.messages.outbound → Evolution API
│   │   └── reminder.go          ← Worker de cron de recordatorios
│   ├── client/
│   │   ├── evolution/
│   │   │   └── client.go        ← Cliente HTTP para Evolution API
│   │   └── rabbitmq/
│   │       └── publisher.go     ← Publisher RabbitMQ (amqp091-go)
│   ├── config/
│   │   └── config.go            ← Carga de variables de entorno
│   └── docs/
│       └── embed.go             ← Embed de OpenAPI spec (swagger)
├── db/
│   ├── migrations/              ← Archivos SQL numerados: NNN_description.sql
│   │   ├── 001_extensions.sql   ← pgvector, uuid-ossp
│   │   ├── 002_tenants.sql      ← SIN RLS (tabla global)
│   │   ├── 003_users.sql
│   │   ├── 004_professionals.sql
│   │   ├── 005_services.sql
│   │   ├── 006_schedules.sql
│   │   ├── 007_customers.sql
│   │   ├── 008_appointments.sql
│   │   ├── 009_conversations.sql
│   │   ├── 010_knowledge_base.sql
│   │   ├── 011_notification_logs.sql
│   │   └── 012_fix_notification_logs_rls.sql
│   └── queries/                 ← Queries SQL para sqlc (configurado en sqlc.yaml)
├── go.mod                       ← Go 1.25
├── go.sum
├── sqlc.yaml
├── Dockerfile.dev
└── .air.toml                    ← Hot reload (air-verse/air v1.52.3)
```

---

## 🔧 CONVENCIONES GO

### Naming
```go
// Exports: CamelCase
type AppointmentService interface { ... }
func NewAppointmentHandler(...) *AppointmentHandler { ... }

// Privados: camelCase
func (s *appointmentService) validateSlot(...) error { ... }

// Constantes: SCREAMING_SNAKE o CamelCase según scope
const MaxRetries = 3
const defaultBufferMinutes = 10
```

### Errores — siempre wrappear con contexto
```go
// ✅ Correcto
if err := s.repo.Create(ctx, appt); err != nil {
    return fmt.Errorf("appointmentService.Create: %w", err)
}

// ❌ Incorrecto — pierde contexto
if err := s.repo.Create(ctx, appt); err != nil {
    return err
}
```

### Errores de dominio tipados
```go
// En internal/domain/errors.go
var (
    ErrNotFound           = errors.New("recurso no encontrado")
    ErrUnauthorized       = errors.New("no autorizado")
    ErrForbidden          = errors.New("acceso denegado")
    ErrSlotUnavailable    = errors.New("el horario solicitado no está disponible")
    ErrConflict           = errors.New("conflicto con un recurso existente")
    ErrValidation         = errors.New("datos inválidos")
    ErrInternal           = errors.New("error interno")
    ErrEmailAlreadyExists = errors.New("el email ya está registrado")
    ErrSlugAlreadyExists  = errors.New("el slug ya está en uso")
    ErrInvalidCredentials = errors.New("credenciales inválidas")
    ErrInvalidToken       = errors.New("token inválido")
)

// Mapeo en handler (internal/handler/errors.go)
func handleServiceError(err error) error {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return fiber.NewError(404, err.Error())
    case errors.Is(err, domain.ErrSlotUnavailable):
        return fiber.NewError(409, err.Error())
    case errors.Is(err, domain.ErrConflict):
        return fiber.NewError(409, err.Error())
    case errors.Is(err, domain.ErrUnauthorized):
        return fiber.NewError(401, err.Error())
    case errors.Is(err, domain.ErrForbidden):
        return fiber.NewError(403, err.Error())
    case errors.Is(err, domain.ErrValidation):
        return fiber.NewError(400, err.Error())
    default:
        return fiber.NewError(500, "error interno")
    }
}
```

### Context — siempre el primer parámetro
```go
// ✅ Correcto
func (s *AppointmentService) Create(ctx context.Context, req CreateRequest) (*Appointment, error)

// ❌ Incorrecto
func (s *AppointmentService) Create(req CreateRequest) (*Appointment, error)
```

### Interfaces para dependencias (facilita testing)
```go
// En domain/interfaces.go — el service recibe la interface, no la implementación
type AppointmentRepository interface {
    Create(ctx context.Context, appt *Appointment) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Appointment, error)
    ListByDate(ctx context.Context, tenantID uuid.UUID, date time.Time) ([]*Appointment, error)
    UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
    CheckConflict(ctx context.Context, tenantID uuid.UUID, professionalID uuid.UUID, start, end time.Time) (bool, error)
}
```

### Logging — log estándar de Go

```go
// El proyecto usa log estándar, NO slog, NO zap
log.Printf("handler.Create: tenant=%s err=%v", tenantID, err)
log.Printf("worker.outbound: mensaje enviado a %s", phone)
```

---

## 🏗️ PATRÓN DE HANDLER (copiar este template)

```go
package handler

type AppointmentHandler struct {
    service domain.AppointmentSvc
}

func NewAppointmentHandler(s domain.AppointmentSvc) *AppointmentHandler {
    return &AppointmentHandler{service: s}
}

func (h *AppointmentHandler) Create(c *fiber.Ctx) error {
    // 1. Parse y validar input
    var req CreateAppointmentRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "formato de datos inválido")
    }

    // 2. Tenant ya resuelto por middleware — solo obtener
    tenantID := middleware.TenantIDFromContext(c)
    if tenantID == uuid.Nil {
        return fiber.NewError(403, "tenant no identificado")
    }

    // 3. Llamar al service
    result, err := h.service.Create(c.Context(), tenantID, req)
    if err != nil {
        return handleServiceError(err)
    }

    // 4. Responder
    return c.Status(201).JSON(result)
}
```

---

## 🔐 MIDDLEWARE — ORDEN Y USO

### Orden obligatorio en rutas protegidas

```text
JWTMiddleware → TenantMiddleware → Handler
```

### Funciones disponibles (internal/middleware/context.go)

```go
AuthIDFromContext(c *fiber.Ctx) string         // sub del JWT (Supabase user UUID)
UserFromContext(c *fiber.Ctx) *domain.User     // usuario autenticado
TenantFromContext(c *fiber.Ctx) *domain.Tenant // tenant actual (con slug, timezone, etc.)
TenantIDFromContext(c *fiber.Ctx) uuid.UUID    // UUID del tenant (más común)
```

### JWT — soporta HS256 y ES256

```go
// JWTMiddleware valida tokens Supabase (ES256) Y tokens legacy (HS256)
// Carga claves EC desde {SUPABASE_URL}/auth/v1/.well-known/jwks.json al startup
app.Use(middleware.JWTMiddleware(cfg.JWTSecret, cfg.SupabaseURL))
```

### SetTenantID — helper para repositories

```go
// En repository — patrón para activar RLS en la conexión
func (r *appointmentRepository) Create(c *fiber.Ctx, appt *domain.Appointment) error {
    conn, err := middleware.SetTenantID(c, r.db)
    if err != nil {
        return fmt.Errorf("SetTenantID: %w", err)
    }
    defer conn.Release()

    _, err = conn.Exec(c.Context(), `INSERT INTO appointments ...`, ...)
    return err
}
```

---

## 🗄️ QUERIES SQL (db/queries/)

Usar **sqlc** para generar código Go desde SQL.
Configuración en `sqlc.yaml` en la raíz de apps/api.

```sql
-- db/queries/appointments.sql

-- name: CreateAppointment :one
-- Crea una nueva cita — tenant_id viene del RLS context
INSERT INTO appointments (
    tenant_id, customer_id, professional_id, service_id,
    starts_at, ends_at, status, source, price, notes
) VALUES (
    current_setting('app.tenant_id')::UUID, $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetAppointmentsByDate :many
-- Obtiene citas de un día para el tenant actual
SELECT a.*,
       c.name as customer_name, c.phone as customer_phone,
       p.name as professional_name,
       s.name as service_name, s.duration_min
FROM appointments a
JOIN customers c ON c.id = a.customer_id
JOIN professionals p ON p.id = a.professional_id
JOIN services s ON s.id = a.service_id
WHERE a.tenant_id = current_setting('app.tenant_id')::UUID
  AND DATE(a.starts_at AT TIME ZONE $1) = $2
ORDER BY a.starts_at ASC;
```

---

## ⏰ ENGINE DE DISPONIBILIDAD (service/availability.go)

Este es el algoritmo más crítico del sistema. Debe:

```
1. Obtener horario del profesional para el día solicitado (schedules)
2. Obtener bloqueos del día (schedule_blocks)
3. Obtener citas existentes del día (appointments con status != 'cancelled')
4. Calcular slots libres considerando:
   - Duración del servicio
   - Buffer entre citas (service.buffer_min)
   - Intervalos de slot: cada 30 minutos por defecto
5. Retornar lista de {starts_at, ends_at} disponibles

Restricciones:
- No ofrecer slots que ya pasaron (comparar con NOW() en timezone del tenant)
- No ofrecer slots que solapen con bloqueos
- No ofrecer slots que solapen con citas existentes (incluir buffer)
- Máximo slots a futuro: 60 días
```

---

## 🐇 RABBITMQ — PUBLICAR EVENTOS

```go
// Ejemplo de publicar en cola wa.messages.inbound
func (s *WhatsAppService) PublishInboundMessage(ctx context.Context, msg domain.WAInboundPayload) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("marshal message: %w", err)
    }

    return s.publisher.Publish(ctx, "wa.messages.inbound", amqp.Publishing{
        ContentType:  "application/json",
        DeliveryMode: amqp.Persistent,   // ← siempre Persistent para no perder mensajes
        Body:         body,
    })
}
```

**Colas activas:**

| Cola | Rol de este servicio |
| --- | --- |
| `wa.messages.inbound` | Producer (desde webhook de Evolution API) |
| `wa.messages.outbound` | Consumer (worker/outbound.go → Evolution API) |
| `notifications.reminders` | Producer (cron) + Consumer (worker/reminder.go) |
| `knowledge.vectorize` | Producer (al crear/actualizar documentos) |

---

## 🧪 TESTING

### Estructura de tests

```text
handler_test.go      → usa httptest, mockea service interface
service_test.go      → usa mocks de interfaces, testea lógica pura
integration_test.go  → tests end-to-end con DB real (build tag: integration)
```

### Ejemplo de test de service
```go
func TestAppointmentService_Create_SlotConflict(t *testing.T) {
    // Arrange — mock de la interface de repository
    mockRepo := &mockAppointmentRepository{}
    mockRepo.checkConflictFn = func(...) (bool, error) { return true, nil }

    svc := service.NewAppointmentService(mockRepo, ...)

    // Act
    _, err := svc.Create(context.Background(), tenantID, req)

    // Assert
    assert.ErrorIs(t, err, domain.ErrSlotUnavailable)
}
```

---

## 📦 DEPENDENCIAS REALES (go.mod — Go 1.25)

```go
github.com/gofiber/fiber/v2             v2.52.6    // HTTP framework
github.com/gofiber/contrib/swagger      v1.3.0     // Swagger UI embebido
github.com/jackc/pgx/v5                v5.6.0     // PostgreSQL driver
github.com/google/uuid                  v1.6.0     // UUID
github.com/golang-jwt/jwt/v5           v5.3.1     // JWT (HS256 + ES256)
github.com/rabbitmq/amqp091-go         v1.10.0    // RabbitMQ
github.com/stripe/stripe-go/v76        v76.25.0   // Stripe
github.com/go-playground/validator/v10  v10.30.1   // Validación de structs
```

**Herramientas (no en go.mod, instalar con go install):**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest        # Generar queries
go install github.com/vektra/mockery/v2@latest             # Generar mocks
go install github.com/air-verse/air@v1.52.3                # Hot reload (requiere Go 1.22+)
```
