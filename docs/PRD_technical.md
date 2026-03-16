# CitaSpot — Technical PRD
> Versión 1.0 | Monorepo | Claude Code | Stack: Go · Python · Next.js

---

## Índice

1. [Reglas Globales (CLAUDE.md)](#1-reglas-globales)
2. [Estructura del Monorepo](#2-estructura-del-monorepo)
3. [Schema de Base de Datos](#3-schema-de-base-de-datos)
4. [Contratos de API](#4-contratos-de-api)
5. [Servicio de IA — Flujos](#5-servicio-de-ia)
6. [Prompts del Asistente](#6-prompts-del-asistente)
7. [Colas RabbitMQ](#7-colas-rabbitmq)
8. [Variables de Entorno](#8-variables-de-entorno)
9. [Orden de Desarrollo — 19 Tareas](#9-orden-de-desarrollo)

---

## 1. Reglas Globales

> Ver `CLAUDE.md` en la raíz del monorepo.

Reglas absolutas:
- **Multi-tenancy**: TODA query debe incluir `WHERE tenant_id = $1`
- **Arquitectura**: Handler → Service → Repository (sin saltarse capas)
- **Seguridad**: NUNCA loggear PII, tokens ni credenciales
- **Tests**: `make test` debe pasar antes de completar cualquier tarea
- **WhatsApp**: Rate limit de 1 msg/seg, validar sesión antes de enviar

---

## 2. Estructura del Monorepo

```
citaspot/
├── CLAUDE.md               ← Reglas globales para Claude Code
├── .env.example            ← Variables de entorno documentadas
├── docker-compose.yml      ← Ambiente de desarrollo
├── Makefile                ← Comandos del proyecto
├── apps/
│   ├── api/                ← Go — Core API
│   ├── ai/                 ← Python — AI Service
│   └── web/                ← Next.js — Dashboard + Booking
├── packages/
│   └── shared-types/       ← TypeScript types compartidos
└── infra/                  ← Configuración de infraestructura
```

---

## 3. Schema de Base de Datos

11 tablas con RLS activo. Migraciones en `apps/api/db/migrations/`.

| Migración | Tabla(s) | Descripción |
|---|---|---|
| 001 | — | Extensiones: uuid-ossp, vector, pg_trgm |
| 002 | `tenants` | Negocios clientes del SaaS |
| 003 | `users` | Usuarios con acceso al dashboard |
| 004 | `professionals` | Profesionales del negocio |
| 005 | `services`, `professional_services` | Servicios y precios |
| 006 | `schedules`, `schedule_blocks` | Disponibilidad y bloqueos |
| 007 | `customers` | Clientes del negocio |
| 008 | `appointments` | Citas (tabla central) |
| 009 | `conversations`, `messages` | Conversaciones IA vía WA |
| 010 | `knowledge_documents`, `knowledge_chunks` | RAG base de conocimiento |
| 011 | `notification_logs` | Log de mensajes enviados |

---

## 4. Contratos de API

Base URL: `GET /api/v1` | Auth: `Bearer JWT` | Header: `X-Tenant-ID`

### Auth
| Method | Path | Response |
|---|---|---|
| POST | `/auth/register` | `201: {tenant_id, user_id, token}` |
| POST | `/auth/login` | `200: {token, tenant, user}` |
| POST | `/auth/refresh` | `200: {token}` |

### Appointments
| Method | Path | Descripción |
|---|---|---|
| GET | `/appointments` | Lista con filtros de fecha/profesional |
| POST | `/appointments` | Crear cita (409 si hay conflicto) |
| PATCH | `/appointments/:id` | Actualizar estado/notas |
| DELETE | `/appointments/:id/cancel` | Cancelar cita |
| GET | `/appointments/availability` | Slots disponibles |

### Professionals
| Method | Path | Descripción |
|---|---|---|
| GET | `/professionals` | Lista de profesionales |
| POST | `/professionals` | Crear profesional |
| PATCH | `/professionals/:id` | Actualizar |
| GET | `/professionals/:id/schedule` | Ver horarios |
| PUT | `/professionals/:id/schedule` | Actualizar horarios |

### Services / Customers / Knowledge
> Ver `apps/api/CLAUDE.md` para la lista completa de endpoints.

### Booking Público (sin auth)
| Method | Path | Descripción |
|---|---|---|
| GET | `/public/:slug` | Perfil del negocio + servicios |
| GET | `/public/:slug/availability` | Disponibilidad pública |
| POST | `/public/:slug/book` | Crear reserva |
| GET | `/public/confirm/:token` | Confirmar cita |
| GET | `/public/cancel/:token` | Cancelar cita |

---

## 5. Servicio de IA

### Flujo de mensaje WhatsApp

```
Cliente WA → Evolution API → POST /webhook → Core API
                                                  ↓
                                          Publica en RabbitMQ
                                          wa.messages.inbound
                                                  ↓
                                          AI Service Consumer
                                                  ↓
                                    ┌─────────────────────────┐
                                    │   Detectar intención    │
                                    │   BOOKING / QUERY /     │
                                    │   CANCEL / HANDOFF      │
                                    └─────────┬───────────────┘
                                              ↓
                               ┌──────────────────────────────┐
                               │  RAG search (si aplica)      │
                               │  Availability (si booking)   │
                               │  LLM generate response       │
                               └──────────────┬───────────────┘
                                              ↓
                                    Publica en RabbitMQ
                                    wa.messages.outbound
                                              ↓
                                    Core API → Evolution API
                                              ↓
                                    Mensaje a cliente WhatsApp
```

### Endpoints AI Service (FastAPI :8001)

| Method | Path | Descripción |
|---|---|---|
| POST | `/process` | Procesar mensaje y generar respuesta |
| POST | `/vectorize` | Vectorizar documento de conocimiento |
| POST | `/search` | Búsqueda semántica en RAG |
| GET | `/health` | Estado del servicio + LLM |

---

## 6. Prompts del Asistente

### Intenciones detectadas

```python
class Intent(str, Enum):
    BOOKING  = "booking"   # Agendar cita
    QUERY    = "query"     # Consulta de precios/servicios
    CONFIRM  = "confirm"   # Confirmar cita
    CANCEL   = "cancel"    # Cancelar cita
    HANDOFF  = "handoff"   # Escalar a humano
    UNKNOWN  = "unknown"   # No determinado
```

### Estados de conversación (Redis)

```python
class ConversationState(str, Enum):
    IDLE             = "idle"
    AWAITING_SLOT    = "awaiting_slot"
    AWAITING_CONFIRM = "awaiting_confirm"
    AWAITING_NAME    = "awaiting_name"
    HANDED_OFF       = "handed_off"
```

### Formato de respuesta (siempre JSON)

```json
{
  "message": "texto para el cliente",
  "intent": "booking|query|confirm|cancel|handoff|unknown",
  "action": null,
  "action_data": {},
  "new_state": "idle|awaiting_slot|awaiting_confirm|awaiting_name|handed_off"
}
```

---

## 7. Colas RabbitMQ

| Cola | Producer | Consumer | Propósito |
|---|---|---|---|
| `wa.messages.inbound` | Core API | AI Service | Mensajes WA → IA |
| `wa.messages.outbound` | AI Service | Core API | Respuestas IA → WA |
| `notifications.reminders` | Core API (cron) | Core API Worker | Recordatorios |
| `knowledge.vectorize` | Core API | AI Service | Vectorizar documentos |
| `notifications.review` | Core API (cron) | Core API Worker | Pedir reseñas |

### Payload: `wa.messages.inbound`

```json
{
  "tenant_id": "uuid",
  "conversation_id": "uuid",
  "wa_phone": "+18091234567",
  "wa_message_id": "3EB0...",
  "content": "hola quiero una cita",
  "message_type": "text",
  "timestamp": "2026-03-01T10:00:00Z"
}
```

---

## 8. Variables de Entorno

> Ver `.env.example` en la raíz para la lista completa y documentada.

Variables críticas por servicio:

| Servicio | Variables esenciales |
|---|---|
| Core API (Go) | `DATABASE_URL`, `JWT_SECRET`, `EVOLUTION_API_KEY`, `RABBITMQ_URL` |
| AI Service (Python) | `GEMINI_API_KEY`, `OPENAI_API_KEY`, `CORE_API_URL`, `REDIS_URL` |
| Web (Next.js) | `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY` |
| Payments | `STRIPE_SECRET_KEY`, `MERCADOPAGO_ACCESS_TOKEN` |

---

## 9. Orden de Desarrollo

19 tareas en 5 fases. Criterios de aceptación obligatorios antes de avanzar.

### Fase 1 — Infraestructura Base
| # | Tarea | Criterios |
|---|---|---|
| 1 | Setup monorepo + docker-compose | `make dev` levanta sin errores |
| 2 | Migraciones DB + RLS | `make migrate` aplica 001-011 con RLS activo |
| 3 | Auth + registro de tenants | `POST /auth/register` crea tenant+user con JWT válido |
| 4 | Middleware tenant resolution | Sin X-Tenant-ID → 401; ID inválido → 403 |

### Fase 2 — Motor de Agenda
| # | Tarea | Criterios |
|---|---|---|
| 5 | CRUD Professionals + Services | CRUD completo con tests y RLS validado |
| 6 | Engine de disponibilidad | `GET /availability` retorna slots correctos |
| 7 | Crear/cancelar appointments | 409 en conflicto, transacción atómica |
| 8 | Página pública de reservas | `/book/:slug` funciona sin auth |

### Fase 3 — WhatsApp + Recordatorios
| # | Tarea | Criterios |
|---|---|---|
| 9 | Integración Evolution API | Conectar número, recibir webhook, publicar en MQ |
| 10 | Worker de recordatorios | Cron cada 5 min, envío correcto, logs de entrega |
| 11 | Dashboard básico | Vista de agenda del día + cambio de estado de citas |

### Fase 4 — Asistente IA + RAG
| # | Tarea | Criterios |
|---|---|---|
| 12 | AI Service FastAPI + LLM Router | `POST /process` < 5s, fallback funcional |
| 13 | RAG vectorización y búsqueda | Vectorizar docs, búsqueda retorna resultados relevantes |
| 14 | Orquestador del asistente | Intención correcta, agenda citas, handoff funcional |
| 15 | Panel de base de conocimiento | Editor por categoría, vectorización en background |

### Fase 5 — Pagos + Onboarding + Launch
| # | Tarea | Criterios |
|---|---|---|
| 16 | Stripe subscriptions | Trial 14d → checkout → webhook activa plan |
| 17 | Onboarding flow | 7 pasos en < 15 min para usuario nuevo |
| 18 | Analytics dashboard | KPIs: ocupación, no-show rate, citas por período |
| 19 | Deploy en Hetzner + Coolify | CI/CD con git push, SSL, health checks activos |

---

*CitaSpot Technical PRD v1.0 — Confidencial*
