-- 033_chatbot_configs.sql
-- Tabla de configuración del chatbot por tenant.
-- Migra bot_name y bot_greeting desde tenants.settings JSONB.

CREATE TABLE IF NOT EXISTS chatbot_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_name            VARCHAR(30) NOT NULL DEFAULT '',
    bot_greeting        TEXT NOT NULL DEFAULT '',
    tone                VARCHAR(20) NOT NULL DEFAULT 'friendly',
    custom_instructions TEXT NOT NULL DEFAULT '',
    template_id         VARCHAR(30),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- RLS obligatorio — aislamiento por tenant
ALTER TABLE chatbot_configs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON chatbot_configs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Índice en tenant_id para lookups rápidos (la UK ya lo cubre, pero explícito para RLS)
CREATE INDEX idx_chatbot_configs_tenant ON chatbot_configs(tenant_id);

-- Migrar datos existentes de tenants.settings JSONB
INSERT INTO chatbot_configs (tenant_id, bot_name, bot_greeting)
SELECT
    id,
    COALESCE(settings->>'bot_name', ''),
    COALESCE(settings->>'bot_greeting', '')
FROM tenants
WHERE settings->>'bot_name' IS NOT NULL
   OR settings->>'bot_greeting' IS NOT NULL
ON CONFLICT (tenant_id) DO NOTHING;
