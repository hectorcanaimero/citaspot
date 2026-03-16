# CLAUDE.md — apps/ai (Python AI Service)
> Reglas específicas para el servicio de IA. Lee también el CLAUDE.md raíz.

---

## 🎯 RESPONSABILIDADES DE ESTE SERVICIO

- Orquestar el asistente conversacional de WhatsApp
- Detectar intención del usuario (booking, query, cancel, handoff)
- Buscar contexto relevante en la base de conocimiento (RAG)
- Generar respuestas con LLM (Gemini Flash + GPT-4o-mini fallback)
- Vectorizar documentos de la base de conocimiento
- Consumir/publicar en colas RabbitMQ

**NO es responsabilidad de este servicio:**
- Enviar mensajes por WhatsApp (→ apps/api)
- Guardar citas en la DB (→ apps/api vía HTTP)
- Gestionar sesiones de WhatsApp (→ apps/api)

---

## 📁 ESTRUCTURA OBLIGATORIA

```
apps/ai/
├── CLAUDE.md                    ← Este archivo
├── main.py                      ← FastAPI entrypoint — solo setup
├── app/
│   ├── routers/
│   │   ├── process.py           ← POST /process — procesar mensaje
│   │   ├── knowledge.py         ← POST /vectorize, POST /search
│   │   └── health.py            ← GET /health
│   ├── services/
│   │   ├── llm.py               ← LLM Router (Gemini + GPT-4o-mini)
│   │   ├── rag.py               ← RAG engine (LlamaIndex + pgvector)
│   │   └── assistant.py         ← Orquestador — detecta intención y genera respuesta
│   ├── worker/
│   │   ├── consumer.py          ← Consumidor RabbitMQ
│   │   └── handlers.py          ← Handlers por tipo de evento
│   ├── models/
│   │   ├── messages.py          ← Pydantic schemas de mensajes
│   │   ├── assistant.py         ← Schemas de respuesta del asistente
│   │   └── knowledge.py         ← Schemas de base de conocimiento
│   └── core/
│       ├── config.py            ← Settings desde variables de entorno
│       ├── database.py          ← Conexión a PostgreSQL (solo para pgvector)
│       └── redis.py             ← Conexión a Redis (estado de conversación)
├── tests/
│   ├── test_assistant.py
│   ├── test_rag.py
│   └── test_llm_router.py
├── requirements.txt
├── requirements-dev.txt
└── pyproject.toml               ← ruff config
```

---

## 🧠 LLM ROUTER (services/llm.py)

### Orden de prioridad
```
1. Gemini 1.5 Flash   → principal (precio imbatible, velocidad)
2. GPT-4o-mini        → fallback automático si Flash falla o tarda > 5s
```

### Implementación obligatoria
```python
# services/llm.py

class LLMRouter:
    """
    Enruta llamadas LLM con fallback automático.
    Loggea cuando usa fallback para monitoreo.
    """
    
    async def complete(
        self, 
        messages: list[dict],
        tenant_id: str,
        max_tokens: int = 1000,
        response_format: dict | None = None
    ) -> LLMResponse:
        # Intentar con Gemini Flash primero
        try:
            return await self._call_gemini(messages, max_tokens, response_format)
        except (TimeoutError, APIError) as e:
            logger.warning(f"Gemini fallback triggered for tenant {tenant_id}: {e}")
            # Fallback a GPT-4o-mini
            return await self._call_openai(messages, max_tokens, response_format)

    async def _call_gemini(self, messages, max_tokens, response_format):
        # Timeout de 5 segundos — si pasa, usar fallback
        async with asyncio.timeout(5.0):
            # implementación...
            pass
```

---

## 🔍 RAG ENGINE (services/rag.py)

### Flujo de búsqueda
```python
async def search(self, tenant_id: str, query: str, top_k: int = 3) -> list[Chunk]:
    """
    1. Generar embedding del query
    2. Búsqueda cosine similarity en pgvector
    3. Filtrar por tenant_id (CRÍTICO — no mezclar tenants)
    4. Retornar top_k chunks más relevantes
    """

async def vectorize_document(self, tenant_id: str, document_id: str, content: str):
    """
    1. Dividir en chunks de ~300 tokens
    2. Generar embeddings con text-embedding-3-small
    3. Guardar en knowledge_chunks con tenant_id
    4. Loggear cantidad de chunks creados
    """
```

### Query pgvector (usar directamente, sin ORM)
```python
# services/rag.py
SEARCH_QUERY = """
SELECT content, 1 - (embedding <=> $1::vector) AS score
FROM knowledge_chunks
WHERE tenant_id = $2
ORDER BY embedding <=> $1::vector
LIMIT $3
"""

# CRÍTICO: siempre filtrar por tenant_id antes del ORDER BY
# El índice HNSW solo es eficiente con el filtro de tenant
```

---

## 🤖 ORQUESTADOR (services/assistant.py)

### Intenciones a detectar
```python
class Intent(str, Enum):
    BOOKING   = "booking"    # Quiere agendar una cita
    QUERY     = "query"      # Pregunta sobre precios, servicios, horarios
    CONFIRM   = "confirm"    # Confirmar cita existente
    CANCEL    = "cancel"     # Cancelar cita
    HANDOFF   = "handoff"    # Escalar a humano
    UNKNOWN   = "unknown"    # No se pudo determinar
```

### Estados de conversación (guardar en Redis)
```python
class ConversationState(str, Enum):
    IDLE              = "idle"
    AWAITING_SLOT     = "awaiting_slot"      # Esperando que elija horario
    AWAITING_CONFIRM  = "awaiting_confirm"   # Esperando confirmación de datos
    AWAITING_NAME     = "awaiting_name"      # Esperando nombre del cliente
    HANDED_OFF        = "handed_off"         # Pasado a humano
```

### Proceso obligatorio
```python
async def process(
    self,
    tenant_id: str,
    conversation_id: str,
    message: str,
    history: list[Message]
) -> AssistantResponse:
    
    # 1. Recuperar estado actual de Redis
    state = await self.redis.get_conversation_state(conversation_id)
    
    # 2. Detectar intención con LLM (structured output)
    intent = await self._detect_intent(message, history, state)
    
    # 3. Obtener contexto RAG (solo para QUERY y BOOKING)
    rag_context = ""
    if intent in (Intent.QUERY, Intent.BOOKING):
        chunks = await self.rag.search(tenant_id, message, top_k=3)
        rag_context = self._format_rag_context(chunks)
    
    # 4. Obtener disponibilidad si es booking
    availability = ""
    if intent == Intent.BOOKING:
        availability = await self._get_availability(tenant_id, message)
    
    # 5. Generar respuesta con el system prompt completo
    response = await self.llm.complete(
        messages=self._build_messages(message, history, rag_context, availability, state),
        tenant_id=tenant_id,
        response_format={"type": "json_object"}
    )
    
    # 6. Parsear respuesta JSON del LLM
    result = AssistantResponse.model_validate_json(response.content)
    
    # 7. Ejecutar acción si la hay (crear cita, cancelar, etc.)
    if result.action:
        await self._execute_action(tenant_id, conversation_id, result)
    
    # 8. Actualizar estado en Redis (TTL: 2 horas)
    await self.redis.set_conversation_state(
        conversation_id, result.new_state, ttl=7200
    )
    
    return result
```

---

## 📨 WORKER RABBITMQ (worker/consumer.py)

```python
# NUNCA procesar mensajes de forma síncrona
# SIEMPRE usar asyncio para procesar en paralelo

async def start_consumer():
    """Iniciar consumidor de wa.messages.inbound"""
    
    async def on_message(message: aio_pika.IncomingMessage):
        async with message.process():   # auto-ack/nack
            try:
                payload = InboundMessage.model_validate_json(message.body)
                await process_whatsapp_message(payload)
            except ValidationError as e:
                # Mensaje malformado — hacer nack sin requeue
                logger.error(f"Invalid message format: {e}")
                raise
            except Exception as e:
                # Error procesando — hacer nack con requeue (hasta 3 intentos)
                logger.error(f"Error processing message: {e}")
                raise
    
    # Prefetch: procesar max 5 mensajes simultáneamente
    await channel.set_qos(prefetch_count=5)
    await queue.consume(on_message)
```

---

## 📋 PROMPTS DEL ASISTENTE

### System Prompt (con variables dinámicas)
```python
SYSTEM_PROMPT_TEMPLATE = """
Eres el asistente virtual de {business_name}, un {business_type} en {city}.
Tu nombre es {assistant_name}.

## TU MISIÓN
Ayudar a los clientes a: agendar citas, responder preguntas sobre servicios
y precios, y brindar información general del negocio.

## REGLAS DE COMUNICACIÓN
- Responde SIEMPRE en español, tono amable pero profesional
- Mensajes CORTOS: máximo 3 oraciones por respuesta
- Usa emojis con moderación (máximo 2 por mensaje)
- Zona horaria: {timezone}
- NUNCA inventes precios, horarios o disponibilidad

## INFORMACIÓN DEL NEGOCIO
{rag_context}

## DISPONIBILIDAD
{availability_context}

## ESTADO DE LA CONVERSACIÓN
{conversation_state}

## FORMATO DE RESPUESTA — SIEMPRE JSON
{{
  "message": "texto para el cliente",
  "intent": "booking|query|confirm|cancel|handoff|unknown",
  "action": null | "create_appointment" | "cancel_appointment",
  "action_data": {{}},
  "new_state": "idle|awaiting_slot|awaiting_confirm|awaiting_name|handed_off"
}}

## CUÁNDO HACER HANDOFF
- Cliente molesto o con queja compleja
- Pregunta fuera de tu base de conocimiento
- Más de 3 intentos sin resolver la consulta
- El cliente pide explícitamente hablar con una persona
"""
```

---

## 🔐 SEGURIDAD Y DATOS

```python
# NUNCA loggear contenido de mensajes de usuarios (PII)
# ✅ Correcto
logger.info(f"Processing message for tenant={tenant_id} conv={conversation_id}")

# ❌ Incorrecto — expone datos del usuario
logger.info(f"Processing: '{message_content}' for tenant={tenant_id}")

# NUNCA incluir el system prompt completo en logs
# Solo loggear: intent detectado, tokens usados, tiempo de respuesta
```

---

## 🧪 TESTING

```python
# tests/test_assistant.py

@pytest.mark.asyncio
async def test_detect_intent_booking():
    """Test detección de intención de agendamiento"""
    assistant = AssistantService(
        llm=MockLLMRouter(),    # Siempre mockear LLM en tests
        rag=MockRAGService(),
        redis=MockRedis()
    )
    
    response = await assistant.process(
        tenant_id="test-tenant",
        conversation_id="test-conv",
        message="hola quiero una cita para corte",
        history=[]
    )
    
    assert response.intent == Intent.BOOKING
    assert response.new_state in (
        ConversationState.AWAITING_SLOT, 
        ConversationState.AWAITING_NAME
    )

@pytest.mark.asyncio  
async def test_llm_router_fallback():
    """Test que el fallback a GPT-4o-mini funciona"""
    router = LLMRouter()
    # Simular timeout de Gemini
    with patch.object(router, '_call_gemini', side_effect=TimeoutError):
        response = await router.complete([{"role": "user", "content": "test"}], "tenant-1")
        assert response.model.startswith("gpt")
```

---

## 📦 DEPENDENCIAS

```txt
# requirements.txt
fastapi==0.115.0
uvicorn[standard]==0.30.0
pydantic==2.7.0
pydantic-settings==2.3.0

# LLM
google-generativeai==0.7.0
openai==1.35.0
llama-index==0.10.0
llama-index-vector-stores-postgres==0.1.0

# DB
asyncpg==0.29.0
pgvector==0.3.0

# Queue / Cache
aio-pika==9.4.0         # RabbitMQ async
redis[asyncio]==5.0.0

# Utils
httpx==0.27.0
structlog==24.2.0

# requirements-dev.txt
pytest==8.2.0
pytest-asyncio==0.23.0
pytest-mock==3.14.0
ruff==0.4.0
```
