-- 044_dental_assistant_mode.sql
-- Configuración del modo del asistente conversacional para tenants dentales.
-- Aplica gate por business_type='dental' en runtime (apps/ai/app/agent/orchestrator.py).
--
-- Modos:
--   'in_chat'   → flujo determinista actual (state machine completa) — default seguro
--   'send_link' → bot prefiere mandar link de booking en lugar de agendar in-chat
--   'hybrid'    → primero ofrece link, si rechaza usa state machine

ALTER TABLE tenants
  ADD COLUMN dental_assistant_mode VARCHAR(20) NOT NULL DEFAULT 'in_chat'
    CHECK (dental_assistant_mode IN ('in_chat', 'send_link', 'hybrid')),
  ADD COLUMN urgency_phone   VARCHAR(30),  -- número de emergencia que se da en triaje
  ADD COLUMN urgency_message TEXT;         -- mensaje custom de urgencia (opcional)
