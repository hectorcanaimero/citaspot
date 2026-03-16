-- 009_conversations.sql
-- Conversaciones del asistente IA vía WhatsApp

CREATE TABLE conversations (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  customer_id     UUID REFERENCES customers(id),
  wa_phone        VARCHAR(20) NOT NULL,       -- Número que inició la conversación
  status          VARCHAR(20) DEFAULT 'active', -- 'active','handed_off','closed'
  context         JSONB DEFAULT '{}',          -- Estado actual del agente IA
  last_message_at TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Mensajes individuales de la conversación
CREATE TABLE messages (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  tenant_id       UUID NOT NULL,
  role            VARCHAR(10) NOT NULL,  -- 'user','assistant','system'
  content         TEXT NOT NULL,
  wa_message_id   VARCHAR(255),          -- ID del mensaje en WhatsApp
  tokens_used     INTEGER,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversations_tenant_phone ON conversations(tenant_id, wa_phone);
CREATE INDEX idx_conversations_status      ON conversations(tenant_id, status);
CREATE INDEX idx_messages_conversation     ON messages(conversation_id, created_at);

ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages      ENABLE ROW LEVEL SECURITY;

CREATE POLICY conversations_tenant_isolation ON conversations
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY messages_tenant_isolation ON messages
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
