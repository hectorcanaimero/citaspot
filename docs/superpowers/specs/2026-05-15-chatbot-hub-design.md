# Chatbot Hub — Design Spec

> **Date:** 2026-05-15
> **Status:** Draft
> **Scope:** Replace "Conocimiento" section with unified "Chatbot" hub

---

## Context

Today CitaSpot users configure their AI assistant across scattered locations: knowledge documents in `/dashboard/knowledge`, bot name and greeting buried in Settings > Customization, and tone/personality hardcoded in the AI service with no user control. There's zero way to test the chatbot from the dashboard — users must wait for a real WhatsApp customer to discover if their setup works.

This creates a painful feedback loop: configure blindly → deploy → hope it works. The goal is to unify everything into a single "Chatbot" section with guided setup, live testing, and automated validation.

---

## Layout

Dashboard with **three columns**:

```
┌──────────────────────────────────────────────────────────────┐
│  🤖 Chatbot                                                 │
├─────────────┬──────────────────────────────┬─────────────────┤
│  PROGRESS   │     MAIN CONTENT             │  TEST CHAT      │
│  CHECKLIST  │                              │  PANEL          │
│             │  [Personalidad] [Conocim.]   │                 │
│  ✅ Name    │  [Validación]                │  [messages]     │
│  ✅ Greeting│                              │                 │
│  ⬜ Docs    │  (content changes per tab)   │  [input]        │
│  ⬜ Tested  │                              │  [Send]         │
│  ⬜ Validated│                             │                 │
│             │                              │  [Clear chat]   │
│  ──────     │                              │                 │
│  2 of 5     │                              │  "Test mode —   │
│             │                              │   not sent to   │
│             │                              │   customers"    │
└─────────────┴──────────────────────────────┴─────────────────┘
```

**Left sidebar (~200px):** Setup progress checklist. Each item links to its relevant tab. Shows completion count (e.g., "3 de 5"). Items:
1. Bot name configured
2. Greeting configured
3. At least 1 active knowledge document
4. Test chat used at least once
5. AI validation passed (or skipped)

**Center (~flexible):** Tabbed content area with three tabs.

**Right panel (~350px):** Persistent chat panel for live testing. Collapsible on mobile.

---

## Tab 1: Personalidad

### Template Selector (first-time setup only)

Cards displayed at top when `template_id` is null:

| Template | Tone | Bot name example | Pre-loaded docs |
|----------|------|-------------------|-----------------|
| Peluquería | Friendly casual | Luna | FAQ (cancelaciones, métodos de pago), Servicios comunes, Políticas |
| Dental | Professional | Dra. Asistente | FAQ (primera cita, seguros), Preparación pre-consulta, Políticas |
| Spa / Estética | Premium relaxed | Serenity | FAQ (contraindicaciones, paquetes), Servicios, Cuidados post |
| Barbería | Casual direct | Barber Bot | FAQ, Servicios, Horarios |
| Genérico | Professional | Asistente | FAQ básicas, Políticas genéricas |

**On template selection:**
1. Pre-fill `bot_name`, `bot_greeting`, `tone`
2. Create knowledge documents as **drafts** (`is_active: false`) — user activates after reviewing
3. Save `template_id` so the selector hides on future visits (show "Cambiar template" link instead)

### Configuration Fields

| Field | Type | Storage | Notes |
|-------|------|---------|-------|
| `bot_name` | text input, max 30 chars | `chatbot_configs.bot_name` | Required. Pre-filled from template |
| `bot_greeting` | textarea, max 500 chars | `chatbot_configs.bot_greeting` | Required. Supports `{business_name}` variable |
| `tone` | select: `friendly`, `professional`, `premium`, `casual` | `chatbot_configs.tone` | Maps to system prompt tone instructions |
| `custom_instructions` | textarea, max 1000 chars | `chatbot_configs.custom_instructions` | Optional. Appended to system prompt. E.g., "Siempre ofrecer combo corte+barba" |

**Auto-save** on field blur (PATCH endpoint), with visual feedback (checkmark flash).

### Tone → System Prompt mapping

| Tone | Prompt instructions (appended to system_intro) |
|------|------------------------------------------------|
| `friendly` | "Usá un tono amigable y cercano. Podés usar emojis con moderación. Tuteá al cliente." |
| `professional` | "Mantené un tono profesional y respetuoso. Evitá emojis. Usá usted." |
| `premium` | "Usá un tono cálido pero elegante. Transmití exclusividad y cuidado personalizado." |
| `casual` | "Sé directo y relajado. Podés usar expresiones coloquiales. Tuteá al cliente." |

These are combined with the existing `system_intro` template in `messages.py`. The `custom_instructions` field is appended after the tone instructions.

---

## Tab 2: Conocimiento

**Identical to current `/dashboard/knowledge` functionality:**
- CRUD of knowledge documents (text + file upload)
- Categories: services, pricing, faq, policies, team, location, promotions
- Activate/deactivate toggle per document
- File upload (PDF, DOCX, XLSX, CSV, max 8MB)
- Processing status indicators (ready/processing/error)

**New: Template-generated drafts**
- When a template is selected, draft documents appear with badge "Sugerido por template"
- Drafts are `is_active: false` until user reviews and activates
- User can edit content before activating
- User can delete drafts they don't want

**No other changes to knowledge functionality.**

---

## Tab 3: Validación

### Static Checklist (always visible, zero LLM cost)

Rules engine checks in real-time:

| Rule | Check | Suggestion on failure |
|------|-------|-----------------------|
| Bot name | `bot_name` is not empty | "Dale un nombre a tu asistente para que se presente con tus clientes" |
| Greeting | `bot_greeting` is not empty | "Configurá un saludo de bienvenida" |
| Active docs ≥ 1 | Count of `is_active` docs | "Activá al menos un documento de conocimiento" |
| FAQ category | At least 1 active doc with category `faq` | "Agregá preguntas frecuentes — es lo que más consultan tus clientes" |
| Pricing documented | At least 1 active doc with category `pricing` or `services` | "Documentá tus precios para que el bot pueda informar a tus clientes" |
| Cancellation policy | At least 1 active doc with category `policies` | "Agregá tu política de cancelación" |
| Chat tested | Session flag or persisted: user sent ≥1 test message | "Probá tu chatbot en el panel de la derecha" |

Each item shows ✅ or ⬜ with the suggestion text. Items link to the relevant tab/field.

### AI Validator (on-demand, button click)

**Button:** "Validar con IA" — disabled until at least 1 document is active.

**Flow:**
1. Frontend calls `POST /api/v1/chatbot/validate`
2. Backend generates 5-8 questions based on active document categories:
   - If has `services` docs → "¿Qué servicios ofrecen?", "¿Cuánto cuesta [servicio más mencionado]?"
   - If has `faq` docs → "¿Cuál es el horario de atención?"
   - If has `policies` docs → "¿Cuál es la política de cancelación?"
   - Generic: "¿Cómo puedo agendar una cita?", "¿Aceptan tarjeta?"
3. Each question is processed through the same AI pipeline (orchestrator → RAG → LLM)
4. Backend evaluates each response: did the bot answer with specific info (✅) or give a vague/no-info response (❌)?
5. Returns report to frontend

**Report UI:**

```
Validación IA — 6 de 8 preguntas respondidas correctamente

✅ "¿Qué servicios ofrecen?" → Respondió con lista de servicios
✅ "¿Cuánto cuesta un corte?" → Respondió con precio: $500
❌ "¿Aceptan tarjeta?" → No encontró información
   💡 Sugerencia: Agregá métodos de pago a tu base de conocimiento
❌ "¿Tienen estacionamiento?" → No encontró información
   💡 Sugerencia: Agregá información de ubicación y facilidades
```

Each ❌ item has a "Agregar documento →" shortcut that opens Tab 2 with the relevant category pre-selected.

**Cost control:** Validation is limited to 1 run per hour per tenant (rate limit on backend).

---

## Test Chat Panel

### Behavior
- Persistent right panel (~350px width)
- Collapsible via toggle button (collapsed by default on screens < 1280px)
- Chat bubbles styled like WhatsApp (user right, bot left)
- Timestamps on each message
- "Modo prueba" banner at top

### Technical Flow
1. User types message → `POST /api/v1/chatbot/test`
2. Backend creates a temporary in-memory conversation context (NOT persisted to DB)
3. Calls AI service `/process` with `test: true` flag
4. AI service processes normally (intent → RAG → LLM) but:
   - Does NOT publish to `wa.messages.outbound`
   - Does NOT create conversation records
   - Uses a dedicated Redis key prefix `test:{tenant_id}` for state (TTL 30 min)
5. Returns response synchronously to frontend
6. "Clear chat" button clears the Redis test state and UI

### Endpoint: `POST /api/v1/chatbot/test`

```json
// Request
{
  "message": "¿Cuánto cuesta un corte?"
}

// Response
{
  "response": "¡Hola! El corte de cabello tiene un precio de $500...",
  "intent_detected": "QUERY",
  "rag_sources_used": ["Servicios y precios"],
  "processing_time_ms": 1200
}
```

The response includes debug info (`intent_detected`, `rag_sources_used`, `processing_time_ms`) shown as collapsed metadata below the bot message. This helps the user understand WHY the bot answered that way.

---

## Backend Changes

### New table: `chatbot_configs`

```sql
CREATE TABLE chatbot_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    bot_name    VARCHAR(30) NOT NULL DEFAULT '',
    bot_greeting TEXT NOT NULL DEFAULT '',
    tone        VARCHAR(20) NOT NULL DEFAULT 'friendly',
    custom_instructions TEXT NOT NULL DEFAULT '',
    template_id VARCHAR(30),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

ALTER TABLE chatbot_configs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON chatbot_configs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

**Migration:** Also copies existing `bot_name` and `bot_greeting` from `tenants.settings` JSONB into the new table.

### New API endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/chatbot/config` | Get chatbot config (creates default if not exists) |
| `PATCH` | `/api/v1/chatbot/config` | Update chatbot config fields |
| `POST` | `/api/v1/chatbot/test` | Send test message, get response synchronously |
| `POST` | `/api/v1/chatbot/validate` | Run AI validation, return coverage report |

### AI Service changes

**`messages.py`:**
- `_build_system_prompt()` reads `tone` and `custom_instructions` from the tenant profile (new fields in the profile API response)
- Append tone instructions after existing `system_intro`
- Append `custom_instructions` at the end

**`orchestrator.py`:**
- Check for `test: true` flag in payload
- If test: skip outbound publishing, use `test:{tenant_id}` Redis prefix

**`process.py` router:**
- Accept `test: bool` field in request payload
- If test: return response synchronously instead of publishing to outbound queue

### Frontend changes

| File | Change |
|------|--------|
| Nav sidebar | Replace "Conocimiento" with "Chatbot" (icon: MessageSquare or Bot) |
| `/dashboard/knowledge/page.tsx` | Delete — replaced by new chatbot page |
| `/dashboard/chatbot/page.tsx` | **New** — main chatbot hub with 3-column layout |
| `/dashboard/chatbot/components/ProgressChecklist.tsx` | Left sidebar checklist component |
| `/dashboard/chatbot/components/PersonalityTab.tsx` | Template selector + config fields |
| `/dashboard/chatbot/components/KnowledgeTab.tsx` | Migrated from knowledge page (mostly reuse) |
| `/dashboard/chatbot/components/ValidationTab.tsx` | Static checklist + AI validator |
| `/dashboard/chatbot/components/TestChatPanel.tsx` | Right panel chat interface |
| `/dashboard/settings/customization/page.tsx` | Remove bot_name/bot_greeting fields, add "Ir a Chatbot →" link |
| `lib/api.ts` | Add `chatbot` API client (config, test, validate endpoints) |
| `lib/i18n/locales/*.ts` | Add `t.chatbot.*` keys, keep `t.knowledge.*` for KnowledgeTab reuse |

---

## Template Data

Templates are **static JSON** shipped with the frontend (no API needed). Each template contains:

```typescript
interface ChatbotTemplate {
  id: string;                    // "salon", "dental", "spa", "barbershop", "generic"
  name: string;                  // Display name
  icon: string;                  // Emoji or icon key
  defaults: {
    bot_name: string;
    bot_greeting: string;
    tone: 'friendly' | 'professional' | 'premium' | 'casual';
  };
  suggested_docs: Array<{
    category: string;
    title: string;
    content: string;             // Pre-written content in Spanish
  }>;
  validation_questions: string[]; // Used by AI validator for this vertical
}
```

Templates ship in Spanish as default. i18n for templates is out of scope for v1 — the user customizes the pre-filled content anyway.

---

## Scope Boundaries

**In scope:**
- New chatbot hub page with 3-column layout
- Personality tab with templates and config fields
- Knowledge tab (migrated, minimal changes)
- Validation tab (static checklist + AI validator)
- Test chat panel
- New `chatbot_configs` table + migration
- New API endpoints (config, test, validate)
- AI service changes (tone, custom_instructions, test mode)
- Remove bot config from Settings page
- Nav update

**Out of scope (future):**
- Multi-language templates (ship Spanish only for v1)
- Custom categories for knowledge documents
- Conversation analytics / chat history from real WhatsApp conversations
- A/B testing different bot configurations
- Voice/audio message support in test chat
- Streaming responses in test chat (v1 waits for full response)

---

## Verification Plan

1. **Smoke test:** Navigate to `/dashboard/chatbot` → see 3-column layout, empty state with template selector
2. **Template flow:** Select "Peluquería" template → verify fields pre-fill, draft docs appear in Knowledge tab
3. **Config save:** Change bot name → blur → verify PATCH call succeeds, progress checklist updates
4. **Knowledge CRUD:** Create/edit/delete documents → same behavior as before
5. **Test chat:** Send "¿Qué servicios ofrecen?" → receive bot response with debug metadata
6. **Test chat state:** Send booking-related messages → verify conversation state works across messages
7. **Clear chat:** Click clear → verify Redis state is reset, chat is empty
8. **Static validation:** Check all items update correctly as config/docs change
9. **AI validation:** Click "Validar con IA" → receive report with ✅/❌ per question
10. **Settings migration:** Verify bot_name/bot_greeting no longer appear in Settings > Customization
11. **Existing data:** Tenants with existing bot_name/bot_greeting → verify data migrated to chatbot_configs
