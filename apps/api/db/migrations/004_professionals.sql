-- 004_professionals.sql
-- Profesionales del negocio (peluqueros, dentistas, psicólogos, etc.)

CREATE TABLE professionals (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id     UUID REFERENCES users(id),    -- NULL si no tiene acceso al dashboard
  name        VARCHAR(255) NOT NULL,
  specialty   VARCHAR(100),                 -- 'Colorista', 'Ortodoncista', etc.
  bio         TEXT,
  avatar_url  TEXT,
  color       VARCHAR(7) DEFAULT '#3B82F6', -- Color en el calendario (hex)
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_professionals_tenant ON professionals(tenant_id);

ALTER TABLE professionals ENABLE ROW LEVEL SECURITY;

CREATE POLICY professionals_tenant_isolation ON professionals
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
