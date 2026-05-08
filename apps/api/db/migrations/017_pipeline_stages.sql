-- 017_pipeline_stages.sql
-- Etapas del pipeline CRM (nuevo lead, en tratamiento, completado, etc.)

CREATE TABLE pipeline_stages (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name                TEXT NOT NULL,
  position            INT NOT NULL,
  color               TEXT DEFAULT '#6366f1',
  is_default          BOOLEAN DEFAULT FALSE,
  auto_rules_enabled  BOOLEAN DEFAULT TRUE,
  created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_pipeline_stages_tenant_position ON pipeline_stages(tenant_id, position);
CREATE INDEX idx_pipeline_stages_tenant ON pipeline_stages(tenant_id);

ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;

CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
