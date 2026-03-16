# CLAUDE.md — CitaSpot Monorepo
> Lee este archivo COMPLETO antes de escribir cualquier línea de código.
> Este es el contrato de trabajo entre tú (agente) y el proyecto.

---

## 🏢 IDENTIDAD DEL PROYECTO

**CitaSpot** es un SaaS multi-tenant de gestión de citas con asistente IA
conversacional vía WhatsApp para negocios de salud y belleza en LATAM.

**Mercados iniciales:** República Dominicana y Venezuela  
**Modelo:** Suscripción mensual $10–$25 USD  
**Stack:** Go (core API) · Python (AI service) · Next.js (frontend)  
**Infra:** PostgreSQL + pgvector · Redis · RabbitMQ · Evolution API  

---

## 🚨 REGLAS ABSOLUTAS — NUNCA VIOLAR

### 1. Multi-tenancy es sagrado
```
✅ SIEMPRE filtrar por tenant_id en TODA query a la DB
✅ SIEMPRE resolver el tenant antes de cualquier operación
✅ Row Level Security (RLS) está activo — NUNCA desactivarlo
❌ NUNCA hacer SELECT * sin WHERE tenant_id = $1
❌ NUNCA cruzar datos entre tenants
```

Patrón obligatorio en Go:
```go
tenantID := middleware.TenantFromContext(ctx)
if tenantID == uuid.Nil {
    return ErrUnauthorized
}
// Setear en sesión DB antes de cualquier query
db.Exec(ctx, "SET app.tenant_id = $1", tenantID)
```

### 2. Idioma por contexto
```
Código (vars, funciones, tipos) → inglés
Comentarios en código           → español
Mensajes de error al usuario    → español LATAM (tono amable)
Logs del sistema                → inglés
Git commits                     → Conventional Commits en inglés
```

Ejemplos de commits válidos:
```
feat(appointments): add conflict detection for overlapping slots
fix(whatsapp): handle reconnection after session timeout
chore(db): add index on appointments.starts_at
```

### 3. Arquitectura en capas — sin excepciones
```
Handler  → solo recibe HTTP, valida input, delega al service
Service  → lógica de negocio, orquesta, sin queries directas a DB
Repository → solo acceso a DB, sin lógica de negocio
Domain   → tipos, interfaces, errores — sin dependencias externas
```

❌ NUNCA poner queries SQL en handlers o services  
❌ NUNCA poner lógica de negocio en repositories  
❌ NUNCA llamar a servicios externos directamente desde handlers  

### 4. Seguridad no negociable
```
❌ NUNCA loggear tokens, passwords, API keys, ni datos PII
❌ NUNCA hardcodear credenciales — siempre desde variables de entorno
❌ NUNCA skipear validación de JWT
❌ NUNCA exponer stack traces al cliente — solo en logs internos
✅ SIEMPRE sanitizar input de WhatsApp antes de pasarlo al LLM
✅ SIEMPRE validar el webhook secret de Evolution API
```

### 5. Tests antes de completar
```bash
# Antes de marcar cualquier tarea como completa:
make test        # Todos los tests deben pasar
make lint        # Sin warnings
```

Cobertura mínima en funciones críticas: **80%**  
Mocks obligatorios para: Evolution API, LLM providers, servicios externos

### 6. Base de datos
```
✅ Toda migración en /apps/api/db/migrations/ formato: NNN_description.sql
✅ SIEMPRE usar transacciones para operaciones multi-tabla
✅ Índices obligatorios en: tenant_id + columnas de búsqueda frecuente
❌ NUNCA alterar migraciones ya aplicadas — crear nueva migración
❌ NUNCA hacer DROP sin migración de reversión
```

### 7. WhatsApp — proteger la sesión
```
✅ Rate limit: máximo 1 mensaje por segundo por sesión de tenant
✅ Validar que status === 'CONNECTED' antes de enviar
✅ Manejar reconexión con backoff exponencial (1s, 2s, 4s, 8s, 16s)
❌ NUNCA enviar más de 3 mensajes consecutivos sin input del usuario
❌ NUNCA enviar a números que no están en la conversación activa
```

### 8. LLM / IA
```
✅ Siempre procesar mensajes WA de forma asíncrona vía RabbitMQ
✅ Contexto máximo: últimos 10 mensajes de la conversación
✅ Si el LLM tarda > 8 segundos: enviar "Un momento..." al usuario
✅ Usar LLM Router (Gemini Flash primero, GPT-4o-mini como fallback)
❌ NUNCA llamar al LLM de forma síncrona en el handler HTTP
❌ NUNCA guardar estado de conversación en memoria del proceso (usar Redis)
```

---

## 📁 ESTRUCTURA DEL MONOREPO

```
citaspot/
├── CLAUDE.md                     ← Este archivo (reglas globales)
├── .env.example                  ← Variables de entorno documentadas
├── docker-compose.yml            ← Ambiente de desarrollo
├── Makefile                      ← Comandos del proyecto
│
├── apps/
│   ├── api/                      ← Go — Core API
│   │   └── CLAUDE.md             ← Reglas específicas de Go
│   ├── ai/                       ← Python — AI Service
│   │   └── CLAUDE.md             ← Reglas específicas de Python
│   └── web/                      ← Next.js — Dashboard + Booking
│       └── CLAUDE.md             ← Reglas específicas de Next.js
│
├── packages/
│   ├── shared-types/             ← TypeScript types compartidos
│   └── config/                   ← Configuración compartida
│
├── infra/
│   ├── coolify/                  ← Configuración de deploy
│   └── traefik/                  ← Proxy inverso
│
└── docs/
    ├── PRD_business.md           ← PRD de negocio
    └── PRD_technical.md          ← PRD técnico (este proyecto)
```

---

## ⚙️ COMANDOS DEL PROYECTO

```bash
make dev          # Levantar todos los servicios con docker-compose
make test         # Correr todos los tests (Go + Python + Next.js)
make test-api     # Solo tests del Core API (Go)
make test-ai      # Solo tests del AI Service (Python)
make test-web     # Solo tests del frontend (Next.js)
make migrate      # Aplicar migraciones pendientes
make migrate-down # Revertir última migración
make lint         # golangci-lint + ruff + eslint
make gen          # Generar código (sqlc + mocks + TypeScript types)
make seed         # Insertar datos de prueba en desarrollo
make logs         # Ver logs de todos los servicios
make clean        # Limpiar containers y volúmenes de desarrollo
```

---

## 🌍 VARIABLES DE ENTORNO

> Ver `.env.example` en la raíz para la lista completa.  
> NUNCA commitear valores reales — solo `.env.example` va al repo.

Variables críticas por servicio:

| Servicio | Variables clave |
|---|---|
| Core API | `DATABASE_URL`, `JWT_SECRET`, `EVOLUTION_API_KEY` |
| AI Service | `GEMINI_API_KEY`, `OPENAI_API_KEY`, `CORE_API_URL` |
| Web | `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY` |
| Payments | `STRIPE_SECRET_KEY`, `MERCADOPAGO_ACCESS_TOKEN` |

---

## 🔄 FLUJO DE MENSAJE WHATSAPP (Visión global)

```
1. Cliente envía mensaje por WhatsApp
        ↓
2. Evolution API → POST /api/v1/whatsapp/webhook (Core API)
        ↓
3. Core API valida, identifica tenant, publica en RabbitMQ
        ↓
4. AI Service consumer recibe del queue wa.messages.inbound
        ↓
5. Asistente detecta intención → genera respuesta con RAG + LLM
        ↓
6. AI Service publica respuesta en wa.messages.outbound
        ↓
7. Core API consumer → Evolution API → WhatsApp al cliente
```

---

## 🗄️ COLAS RABBITMQ

| Cola | Producer | Consumer |
|---|---|---|
| `wa.messages.inbound` | Core API | AI Service |
| `wa.messages.outbound` | AI Service | Core API |
| `notifications.reminders` | Core API (cron) | Core API Worker |
| `knowledge.vectorize` | Core API | AI Service |
| `notifications.review` | Core API (cron) | Core API Worker |

---

## ✅ ORDEN DE DESARROLLO (19 tareas en 5 fases)

Ver `docs/PRD_technical.md` sección 10 para el roadmap completo.

**Fase 1** — Infraestructura base (tareas 1–4)  
**Fase 2** — Motor de agenda (tareas 5–8)  
**Fase 3** — WhatsApp + Recordatorios (tareas 9–11)  
**Fase 4** — Asistente IA + RAG (tareas 12–15)  
**Fase 5** — Pagos + Onboarding + Launch (tareas 16–19)  

> ⚠️ No avanzar a la siguiente fase sin que todos los criterios de aceptación
> de la fase actual estén cumplidos y los tests pasen.
