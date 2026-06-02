# Chatbot CitaSpot — Flujo end-to-end

> Documento de referencia técnica del asistente conversacional de CitaSpot.
> Cubre desde que llega un mensaje de WhatsApp hasta que el cliente recibe la respuesta.

---

## 1. Visión general

CitaSpot opera un asistente conversacional vía WhatsApp para negocios de salud y belleza en LATAM (mercados iniciales: RD y VE). El chatbot atiende **3 tipos de necesidades**:

1. **Reservar una cita** (BOOKING) — flujo determinista con state machine
2. **Consultar información** (QUERY) — agente con tool-calling sobre el catálogo del negocio + RAG
3. **Gestionar citas existentes** (CANCEL / RESCHEDULE / MY_APPOINTMENTS)

El sistema es **multi-tenant**. Cada mensaje se aísla por `tenant_id` y respeta Row Level Security en PostgreSQL.

### Diagrama end-to-end

```
┌──────────┐         ┌───────────────┐         ┌─────────────────┐
│ Cliente  │ ──────▶ │ Evolution API │ ──HTTP─▶│ Core API (Go)   │
│ WhatsApp │         │  (WhatsApp)   │         │  /webhook       │
└──────────┘         └───────────────┘         └────────┬────────┘
     ▲                                                  │
     │                                                  ▼
     │                                          ┌───────────────┐
     │                                          │   RabbitMQ    │
     │                                          │ wa.messages.  │
     │                                          │   inbound     │
     │                                          └───────┬───────┘
     │                                                  │
     │                                                  ▼
     │                                          ┌───────────────┐
     │                                          │ AI Service    │
     │                                          │   (Python)    │
     │                                          │  Orquestador  │
     │                                          └───────┬───────┘
     │                                                  │
     │            ┌─────────────────────────────────────┤
     │            │                                     │
     │            ▼                                     ▼
     │     ┌──────────────┐                  ┌────────────────────┐
     │     │ Redis        │                  │ PostgreSQL +       │
     │     │ • history    │                  │ pgvector (RAG)     │
     │     │ • state      │                  │ + Core API (HTTP)  │
     │     └──────────────┘                  └────────────────────┘
     │                                                  │
     │                                                  ▼
     │                                          ┌───────────────┐
     │                                          │   RabbitMQ    │
     │                                          │ wa.messages.  │
     │                                          │   outbound    │
     │                                          └───────┬───────┘
     │                                                  │
     │                                                  ▼
     │                                          ┌───────────────┐
     │                                          │ Core API Go   │
     │                                          │ outbound      │
     │                                          │ worker        │
     │                                          └───────┬───────┘
     │                                                  │
     │                                          ┌───────▼───────┐
     └──────────────────────────────────────────│ Evolution API │
                                                └───────────────┘
```

### Stack por capa

| Capa | Tecnología | Responsabilidad |
|---|---|---|
| Bridge WhatsApp | Evolution API (externo) | Conecta a WhatsApp, expone webhook y `sendText` |
| Ingreso / Egreso | Go (Fiber) | Webhook, tenant resolution, rate limit, persistencia conversaciones |
| Mensajería | RabbitMQ | Desacopla ingestión, procesamiento y envío |
| Orquestación IA | Python (FastAPI) | Intent, state machine, tool-calling, RAG |
| Estado conversacional | Redis | Historial (24h TTL) + state machine |
| Datos | PostgreSQL + pgvector | Tenants, citas, conocimiento vectorizado |
| LLM | DeepSeek + Gemini + GPT-4o-mini | Cascada con fallback |

---

## 2. Flujo inbound — mensaje entrante

### 2.1 Webhook Evolution → Core API

Evolution API llama al webhook del Core API cuando llega un mensaje nuevo de WhatsApp.

**Endpoint:** `POST /api/v1/whatsapp/webhook`
**Handler:** [apps/api/internal/handler/whatsapp.go](../apps/api/internal/handler/whatsapp.go) — `WhatsAppHandler.Webhook()`

Eventos manejados:
- `messages.upsert` — mensaje nuevo entrante
- `qrcode.updated` — QR de conexión actualizado
- `connection.update` — cambios de estado de sesión

Validaciones:
- Si `WEBHOOK_SECRET` está configurado, valida header `apikey`
- En dev queda vacío (red Docker privada); en prod obligatorio

El handler responde HTTP 200 inmediatamente y dispara `ProcessInbound()` de forma asíncrona.

### 2.2 Resolución de tenant

Evolution envía el campo `instanceName` que coincide con el `slug` del tenant en CitaSpot.

```
instanceName → tenants.slug → tenant_id (UUID)
```

**Lookup:** [apps/api/internal/repository/auth.go](../apps/api/internal/repository/auth.go) — `FindTenantBySlug()`
La tabla `tenants` NO tiene RLS (es la única excepción global, necesaria antes de tener contexto).

### 2.3 Procesamiento e ingestión

**Servicio:** [apps/api/internal/service/whatsapp.go](../apps/api/internal/service/whatsapp.go) — `ProcessInbound()`

Pasos:

1. Valida que `fromMe = false` (ignora mensajes enviados desde el panel)
2. Normaliza phone: `18091234567@s.whatsapp.net` → `+18091234567`
3. Sanitiza contenido: max 2000 chars, remueve caracteres de control
4. Crea/recupera `conversation` y `customer` (atómico por `tenant_id + phone`)
5. Publica el mensaje a RabbitMQ
6. Publica eventos al rule engine (cola `rules.events`) para automatizaciones

### 2.4 Payload publicado a RabbitMQ

**Cola:** `wa.messages.inbound`

```json
{
  "tenant_id": "uuid",
  "tenant_slug": "clinica-acme",
  "conversation_id": "uuid",
  "customer_id": "uuid",
  "wa_phone": "+18091234567",
  "message_text": "Hola, quiero agendar una limpieza",
  "received_at": "2026-06-02T15:30:00Z"
}
```

Nota: el campo `history` ya **no se incluye**. Desde 2026-05-19 el historial vive en Redis y es responsabilidad del AI Service cargarlo.

---

## 3. AI Service — Orquestador

### 3.1 Consumer RabbitMQ

**Archivo:** [apps/ai/app/worker/inbound.py](../apps/ai/app/worker/inbound.py) — `start_consumer()` + `_process()`

El consumer arranca como `asyncio.Task` durante el startup de FastAPI. Por cada mensaje:

1. Deserializa el payload
2. Carga historial desde Redis
3. Llama al orquestador
4. Persiste el turno (user + assistant) en Redis
5. ACK al broker

### 3.2 Carga de historial desde Redis

**Archivo:** [apps/ai/app/agent/history.py](../apps/ai/app/agent/history.py)

| Aspecto | Valor |
|---|---|
| Key | `conv:hist:{tenant_id}:{conversation_id}` |
| Estructura Redis | LIST (LPUSH + LTRIM 20) |
| Formato entrada | `{"role": "user\|assistant", "content": "..."}` |
| TTL | 24 horas, refrescado en cada `append_message()` |
| Tope | 20 mensajes (~10 turnos) |

Funciones:
- `append_message()` — agrega user/assistant
- `get_recent()` — devuelve los últimos N en orden cronológico

### 3.3 Orquestador principal

**Archivo:** [apps/ai/app/agent/orchestrator.py](../apps/ai/app/agent/orchestrator.py)

Entry point: `process_message()` → `_handle()` (state machine).

Responsabilidades:
- Recuperar el estado conversacional desde Redis
- Llamar al clasificador de intent
- Rutear por estado actual + intent detectado
- Generar respuesta (determinista o vía LLM)
- Publicar a la cola outbound

---

## 4. Clasificación de intent

**Archivo:** [apps/ai/app/agent/intent.py](../apps/ai/app/agent/intent.py) — `detect()`

Pipeline en **dos capas** para optimizar costo y latencia:

### Layer 1 — Pattern matching (regex)

Captura cases obvios sin llamar al LLM:
- Confirmaciones: `sí`, `dale`, `ok`, `confirmo`, `correcto`
- Cancelaciones: `no`, `cancelar`, `mejor no`
- BOOKING imperativo: `agéndame`, `agenda por mi`, `hazlo tú`, `quiero agendar`, `me gustaría agendar`

Esta capa es **state-aware**: en `AWAITING_CONFIRM` un "sí" se interpreta como CONFIRM directo sin LLM.

### Layer 2 — LLM (Gemini 2.5 Flash Lite)

Cuando regex no resuelve. Modelo barato (~$0.10/M tokens). Fallback: GPT-4o-mini.

### Intents soportados

| Intent | Descripción |
|---|---|
| `BOOKING` | El usuario quiere reservar una cita |
| `QUERY` | Pregunta de información (precios, servicios, horarios, ubicación) |
| `CONFIRM` | Afirmación / aceptación |
| `CANCEL` | Cancelar cita o salir del flujo actual |
| `MY_APPOINTMENTS` | Consultar citas existentes |
| `RESCHEDULE` | Mover una cita |
| `HANDOFF` | Escalar a humano |
| `SEND_BOOKING_LINK` | Pedir el link de reserva web |
| `UNKNOWN` | No determinable |

---

## 5. State machine

**Archivo:** [apps/ai/app/agent/state.py](../apps/ai/app/agent/state.py)

### Estados

| Estado | Cuándo se entra |
|---|---|
| `IDLE` | Conversación nueva o terminada |
| `AWAITING_SLOT` | Recolectando servicio/profesional/fecha/hora |
| `AWAITING_CONFIRM` | Mostrando resumen antes de crear cita |
| `AWAITING_NAME` | Falta nombre del cliente |
| `AWAITING_CANCEL_SELECT` | Eligiendo qué cita cancelar |
| `AWAITING_CANCEL_CONFIRM` | Confirmando cancelación |
| `AWAITING_RESCHEDULE_SELECT` | Eligiendo qué cita reagendar |
| `AWAITING_RESCHEDULE_DATE` | Eligiendo nueva fecha |
| `AWAITING_RESCHEDULE_SLOT` | Eligiendo nuevo horario |
| `AWAITING_RESCHEDULE_CONFIRM` | Confirmando reagenda |
| `HANDED_OFF` | Escalado a humano (bot mute) |

### Storage

| Aspecto | Valor |
|---|---|
| Key Redis | `conv:{tenant_id}:{conversation_id}` |
| TTL | 24 horas |
| Campos | `state`, `pending_service_id`, `pending_professional_id`, `pending_date`, `pending_selected_slot`, `customer_name`, `booking_turns_without_service` |

### Diagrama de transiciones (flujo BOOKING)

```
       ┌────────┐
       │  IDLE  │◀───────────────────────┐
       └───┬────┘                        │
           │ BOOKING                     │
           ▼                             │
  ┌─────────────────┐                    │
  │ AWAITING_SLOT   │                    │
  │ (servicio →     │                    │
  │  profesional →  │                    │
  │  fecha → hora)  │                    │
  └────────┬────────┘                    │
           │ slot completo               │
           ▼                             │
  ┌─────────────────┐    sin nombre      │
  │ AWAITING_CONFIRM│────────────▶┌──────────────┐
  └────────┬────────┘             │ AWAITING_NAME│
           │ CONFIRM              └──────┬───────┘
           ▼                             │
  ┌─────────────────┐                    │
  │ book_appointment│◀───────────────────┘
  │     (tool)      │
  └────────┬────────┘
           │ éxito
           ▼
       ┌────────┐
       │  IDLE  │
       └────────┘
```

Escape hatches: desde cualquier `AWAITING_*` un intent `CANCEL` o `HANDOFF` resetea a `IDLE` / `HANDED_OFF`.

---

## 6. LLM Router

**Archivo:** [apps/ai/app/llm/router.py](../apps/ai/app/llm/router.py)

Tres modos según necesidad:

| Función | Uso | Provider primario | Fallback |
|---|---|---|---|
| `chat_lite()` | Clasificación intent | Gemini 2.5 Flash Lite | GPT-4o-mini |
| `chat()` | Respuestas conversacionales | DeepSeek V4 Flash | GPT-4o-mini |
| `chat_with_tools()` | Function-calling | DeepSeek (OpenAI-compat) | GPT-4o-mini |

**Por qué esta cascada:**
- **Gemini Flash Lite** es el más barato (~$0.10/M tokens) — perfecto para clasificación binaria/multi-clase
- **DeepSeek V4 Flash** ofrece la mejor relación calidad/precio para respuestas y soporta tool-calling vía API OpenAI-compatible
- **GPT-4o-mini** como fallback universal por disponibilidad y estabilidad
- Gemini queda **fuera del tool-calling** por incompatibilidades de schema

Tool-calling loop interno: `_openai_chat_with_tools()` ejecuta hasta **3 iteraciones** (LLM → tools → LLM → tools → LLM).

---

## 7. Tool-calling — el "cerebro" del intent QUERY

**Archivo principal:** [apps/ai/app/agent/orchestrator.py](../apps/ai/app/agent/orchestrator.py) — `_handle_query()`

### Flujo

```
1. Pre-cargar RAG: search_knowledge(tenant_id, message_text)
       ↓
2. Construir system prompt:
   - Tono + idioma del tenant
   - Información del negocio (nombre, ciudad, timezone)
   - Top 5 chunks de RAG (KB blocks)
   - 4 reglas anti-pregunta-redundante
       ↓
3. chat_with_tools(messages, tools=OPENAI_TOOLS, ctx={...})
   - LLM decide qué tool llamar (o ninguna)
   - Hasta 3 iteraciones
       ↓
4. format_for_whatsapp(respuesta) → chunks de 400 chars
       ↓
5. publish_reply() → RabbitMQ wa.messages.outbound
```

### Tools disponibles

**Archivos:** [apps/ai/app/agent/tools.py](../apps/ai/app/agent/tools.py) (registry + schemas) y [apps/ai/app/agent/actions.py](../apps/ai/app/agent/actions.py) (HTTP wrappers).

| Tool | Qué hace | Params LLM | Context inyectado |
|---|---|---|---|
| `list_services` | Catálogo: id, nombre, precio, duración, descripción | — | `tenant_slug` |
| `list_professionals` | Staff: id, nombre, especialidad, bio | — | `tenant_slug` |
| `get_service_professionals` | Profesionales que dan un servicio | `service_id` | `tenant_slug` |
| `check_availability` | Slots libres por fecha + servicio + profesional | `service_id`, `professional_id`, `date`, `timezone` | `tenant_slug` |
| `search_knowledge` | Búsqueda semántica RAG | `query` | `tenant_id` |
| `get_business_info` | Nombre, tipo, ciudad, contacto del negocio | — | `tenant_slug` |
| `list_my_treatments` | Planes activos (solo dental, 403 si no) | — | `tenant_slug`, `customer_phone` |
| `book_appointment` | Crear cita confirmada | `service_id`, `professional_id`, `slot`, `customer_name` | `tenant_slug`, `customer_phone` |

### Patrón **context injection**

Los `context_args` (tenant_slug, tenant_id, customer_phone) **NO los ve el LLM**. Se inyectan server-side en `execute_tool()` antes de invocar la función real. Esto evita que el modelo invente o filtre datos sensibles entre tenants.

```python
# Lo que ve el LLM
{"name": "book_appointment", "arguments": {"service_id": "...", "slot": "..."}}

# Lo que se ejecuta
book_appointment(
    service_id="...",
    slot="...",
    tenant_slug="clinica-acme",        # ← inyectado
    customer_phone="+18091234567"      # ← inyectado
)
```

Esta convención es **obligatoria** para toda tool que muta estado.

### Manejo de errores

`execute_tool()` mapea errores a códigos legibles por el LLM:
- `no_profile`, `missing_service_id`, `slot_no_longer_available`, `invalid_professional_id`, `booking_failed_http_409`, etc.

El LLM razona sobre el código y responde apropiadamente al usuario (ej: "ese horario ya no está disponible, ¿querés ver otra opción?").

---

## 8. Endpoints públicos del Core API

Las tools del AI Service consumen estos endpoints vía HTTP (`CORE_API_URL = http://api:3001/api/v1`).

**Handler:** [apps/api/internal/handler/public.go](../apps/api/internal/handler/public.go)

| Método | Endpoint | Acción |
|---|---|---|
| GET | `/public/:slug` | `GetProfile` — info pública del tenant |
| GET | `/public/:slug/availability` | `GetAvailability` — slots libres |
| POST | `/public/:slug/book` | `Book` — crear cita |
| GET | `/public/:slug/my-appointments?phone=` | `ListMyAppointments` |
| GET | `/public/:slug/my-treatments?phone=` | `ListMyTreatments` (403 si no dental) |
| POST | `/public/:slug/appointments/:id/cancel` | `CancelAppointment` |
| POST | `/public/:slug/appointments/:id/reschedule` | `RescheduleAppointment` |
| GET | `/public/:slug/search-knowledge` | `RagQuery` — top-K chunks |

Estos endpoints **no requieren auth** (son públicos por slug) pero respetan RLS internamente: cada query setea `app.tenant_id` en la sesión de Postgres.

Timeout cliente HTTP: **8 segundos** (consistente con el delay "Un momento..." del orquestador).

---

## 9. RAG / Knowledge base

**Archivo:** [apps/ai/app/rag/store.py](../apps/ai/app/rag/store.py)

### Ingesta

1. Documento subido vía `POST /api/v1/knowledge/upload`
2. Parser ([apps/ai/app/rag/parser.py](../apps/ai/app/rag/parser.py)) chunkea en ~300 tokens
3. Cada chunk genera embedding con OpenAI `text-embedding-3-small`
4. Insertado en tabla `knowledge_chunks` (columna `embedding vector(1536)`)
5. Best-effort: si el publisher RabbitMQ falla, el documento se guarda igual

### Query

```sql
SELECT content, 1 - (embedding <=> $query_vec) AS similarity
FROM knowledge_chunks
WHERE tenant_id = $1
  AND 1 - (embedding <=> $query_vec) > 0.55
ORDER BY similarity DESC
LIMIT 5;
```

- Threshold por defecto: **0.55**
- Top K: **5 chunks**
- Filtrado por `tenant_id` (RLS activa)

### Pre-carga en QUERY

Los chunks se inyectan en el system prompt **antes** de la llamada al LLM. Esto reduce alucinaciones y permite respuestas tipo "según la información que tenemos…".

---

## 10. Flujo outbound — respuesta al usuario

### 10.1 Formateo para WhatsApp

**Archivo:** [apps/ai/app/agent/output.py](../apps/ai/app/agent/output.py)

Reglas:
- **Max 400 caracteres por mensaje** (chunking automático)
- Headers Markdown (`#`, `##`) → removidos
- `**bold**` → `*bold*` (un solo asterisco, formato WhatsApp)
- `[texto](url)` → `texto (url)`
- Code fences ` ``` ` → texto plano
- 3+ saltos de línea → 2

Estrategia de chunking:
1. Si ≤400 chars → mensaje único
2. Si no → split por párrafos → por oraciones → hard-split preservando palabras
3. Delay 1s entre chunks (respeta rate limit WhatsApp)

### 10.2 Publicación a RabbitMQ

**Cola:** `wa.messages.outbound`

```json
{
  "tenant_id": "uuid",
  "tenant_slug": "clinica-acme",
  "conversation_id": "uuid",
  "wa_phone": "+18091234567",
  "content": "Listo! Tu cita quedó confirmada para el lunes 15 a las 10:00 ✅"
}
```

### 10.3 Worker outbound del Core API

**Archivo:** [apps/api/internal/worker/outbound.go](../apps/api/internal/worker/outbound.go)

| Aspecto | Valor |
|---|---|
| Modo consumo | Prefetch=1 (procesa secuencial) |
| Pre-envío | Verifica `IsConnected()` contra Evolution |
| Rate limit | `time.Sleep(1 * time.Second)` entre envíos |
| Error permanente | NACK sin requeue |
| Error transitorio | NACK con requeue |
| Reconexión | Backoff exponencial (1s, 2s, 4s, 8s, 16s) |
| Post-envío | Guarda mensaje assistant en `conversations.messages` |

### 10.4 Cliente Evolution API

**Archivo:** [apps/api/internal/client/evolution/client.go](../apps/api/internal/client/evolution/client.go)

| Método | Endpoint Evolution |
|---|---|
| `SendText()` | `POST /message/sendText/{instanceName}` |
| `IsConnected()` | `GET /instance/connectionState/{instanceName}` |
| `Connect()` | `POST /instance/create` + `GET /instance/connect/{instanceName}` |
| `SetWebhook()` | `POST /webhook/set/{instanceName}` |
| `FetchQR()` | `GET /instance/connect/{instanceName}` |

Todas las requests incluyen header `apikey: $EVOLUTION_API_KEY`.

---

## 11. Diagrama completo end-to-end

```
┌──────────────┐
│   Cliente    │
│  WhatsApp    │
└──────┬───────┘
       │ mensaje
       ▼
┌──────────────────────────┐
│      Evolution API       │
│   (instanceName=slug)    │
└──────────┬───────────────┘
           │ webhook POST
           ▼
┌──────────────────────────────────────────────┐
│  Core API (Go) — /webhook                    │
│  • valida apikey (si WEBHOOK_SECRET)         │
│  • resuelve tenant por instanceName          │
│  • normaliza phone + sanitiza                │
│  • crea/actualiza conversation + customer    │
└────┬─────────────────────────────────┬───────┘
     │                                 │
     ▼                                 ▼
┌─────────┐                  ┌──────────────────┐
│RabbitMQ │                  │ PostgreSQL       │
│wa.msgs. │                  │ tenants,         │
│inbound  │                  │ conversations,   │
└────┬────┘                  │ customers        │
     │                       └──────────────────┘
     ▼
┌──────────────────────────────────────────────┐
│  AI Service (Python) — worker/inbound.py     │
└────┬─────────────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│  Orchestrator — process_message()            │
│                                              │
│  1. history.get_recent() ──────▶ ┌────────┐  │
│  2. state.get()         ────────▶│ Redis  │  │
│                                  └────────┘  │
│  3. intent.detect()                          │
│     ├─ Layer 1: regex                        │
│     └─ Layer 2: Gemini Flash Lite            │
│                  │                           │
│                  ▼                           │
│              ┌──────────────────┐            │
│              │  _handle(state,  │            │
│              │   intent)        │            │
│              └─┬──────────────┬─┘            │
│                │              │              │
│         BOOKING│              │QUERY         │
│                ▼              ▼              │
│      ┌───────────────┐  ┌──────────────┐     │
│      │ state machine │  │_handle_query │     │
│      │ determinista  │  │              │     │
│      └───────┬───────┘  │  RAG search ─┼──┐  │
│              │          │  + tools loop│  │  │
│              │          └──────┬───────┘  │  │
│              │                 │          ▼  │
│              │                 │  ┌──────────┐
│              │                 │  │PostgreSQL│
│              │                 │  │+pgvector │
│              │                 │  └──────────┘
│              │                 │              │
│              ▼                 ▼              │
│        ┌─────────────────────────────┐        │
│        │  chat_with_tools()          │        │
│        │  • DeepSeek → GPT-4o-mini   │        │
│        │  • tools.execute_tool()     │        │
│        │    └─ HTTP a Core API:      │        │
│        │       /public/:slug/...     │        │
│        └─────────────┬───────────────┘        │
│                      │                        │
│                      ▼                        │
│        ┌─────────────────────────────┐        │
│        │ format_for_whatsapp()       │        │
│        │ • max 400 chars             │        │
│        │ • chunking + delay 1s       │        │
│        └─────────────┬───────────────┘        │
│                      │                        │
│  4. history.append() ┼─▶ Redis                │
└──────────────────────┼────────────────────────┘
                       │
                       ▼
                ┌─────────┐
                │RabbitMQ │
                │wa.msgs. │
                │outbound │
                └────┬────┘
                     │
                     ▼
┌──────────────────────────────────────────────┐
│  Core API (Go) — worker/outbound.go          │
│  • prefetch=1 (secuencial)                   │
│  • IsConnected() check                       │
│  • rate limit 1s                             │
│  • SendText() → Evolution API                │
└──────────────────────┬───────────────────────┘
                       │
                       ▼
                ┌──────────────┐
                │ Evolution    │
                │     API      │
                └──────┬───────┘
                       │
                       ▼
                ┌──────────────┐
                │   Cliente    │
                │  WhatsApp    │
                └──────────────┘
```

---

## 12. Restricciones y reglas de oro

### Multi-tenancy

- **TODA query** filtra por `tenant_id` (Row Level Security activa)
- Excepciones globales: `tenants` (lookup inicial), `professional_services` (join sin tenant_id, protegido vía FK)
- Antes de cualquier query: `SET app.tenant_id = $1` en la sesión Postgres

### Output WhatsApp

- Max **400 caracteres** por mensaje
- Sin headers Markdown (`#`, `##`)
- Bold con **un solo** asterisco: `*texto*`
- Links: `texto (url)`
- Delay 1 segundo entre chunks

### LLM

- **NUNCA inventa datos** — siempre via tools (servicios, precios, horarios)
- Tool-calling: **max 3 iteraciones**
- Tools que mutan estado: **obligatoriamente** usan `context_args` para tenant/phone
- Context máximo: últimos 20 mensajes de Redis
- Si tarda > 8s: enviar "Un momento…"

### Historia conversacional

- **Redis es fuente de verdad** — el campo `history` del payload se ignora (compat)
- TTL 24 horas, auto-refresh en cada append
- Cap 20 mensajes (LTRIM)

### Rate limiting

- 1 mensaje/segundo por sesión de tenant (worker outbound)
- Prefetch=1 garantiza orden y respeto del límite
- Backoff exponencial en reconexión: 1s → 2s → 4s → 8s → 16s

### Seguridad

- NUNCA loggear tokens, passwords, PII
- Sanitizar input WA antes de pasarlo al LLM (max 2000 chars, sin caracteres de control)
- Validar `WEBHOOK_SECRET` en prod
- `EVOLUTION_API_KEY` y `WEBHOOK_SECRET` son distintos (uno para llamar, otro para recibir)

---

## 13. Referencias

### Documentación interna del orquestador

- `architecture_ai_tool_calling.md` — diseño detallado del tool-calling
- `bugfix_llm_permission_questions.md` (Plane #40) — anti-pregunta-redundante
- `bugfix_booking_classifier_imperative.md` (Plane #38) — regex BOOKING en voseo/tú
- `bugfix_booking_classifier_conditional.md` (Plane #39) — "me gustaría/quisiera agendar"
- `pattern_tool_context_injection.md` — convención context_args
- `pattern_rule_seed_parity.md` — sincronización de reglas (migración SQL + seed Go)

### Documentación del proyecto

- [docs/PRD_technical.md](PRD_technical.md) — PRD técnico completo
- [docs/runbook.md](runbook.md) — operación en producción
- [docs/API_DOCS.md](API_DOCS.md) — referencia de endpoints

### Colas RabbitMQ

| Cola | Producer | Consumer |
|---|---|---|
| `wa.messages.inbound` | Core API | AI Service |
| `wa.messages.outbound` | AI Service | Core API |
| `notifications.reminders` | Core API (cron) | Core API Worker |
| `knowledge.vectorize` | Core API | AI Service |
| `notifications.review` | Core API (cron) | Core API Worker |
| `rules.events` | Core API | Core API (rule engine) |

---

> **Última actualización:** 2026-06-02
> **Mantenedor:** Equipo CitaSpot
