-- 019_rules.sql
-- Reglas de automatizacion CRM (CRUD only en Phase 6A, engine en Phase 6B)

CREATE TABLE rules (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name              TEXT NOT NULL,
  description       TEXT,
  trigger_type      TEXT NOT NULL,
  trigger_event     TEXT,
  trigger_schedule  JSONB,
  conditions        JSONB DEFAULT '[]',
  actions           JSONB NOT NULL,
  is_active         BOOLEAN DEFAULT TRUE,
  is_template       BOOLEAN DEFAULT FALSE,
  template_key      TEXT,
  cooldown_hours    INT DEFAULT 24,
  priority          INT DEFAULT 0,
  created_at        TIMESTAMPTZ DEFAULT NOW(),
  updated_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rules_tenant       ON rules(tenant_id);
CREATE INDEX idx_rules_active       ON rules(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_rules_template     ON rules(is_template) WHERE is_template = TRUE;

ALTER TABLE rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY rules_tenant_isolation ON rules
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE TABLE rule_executions (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rule_id             UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
  customer_id         UUID REFERENCES customers(id),
  triggered_at        TIMESTAMPTZ DEFAULT NOW(),
  trigger_event       TEXT,
  conditions_snapshot JSONB,
  actions_result      JSONB,
  status              TEXT NOT NULL,
  error_message       TEXT
);

CREATE INDEX idx_rule_executions_tenant    ON rule_executions(tenant_id);
CREATE INDEX idx_rule_executions_rule      ON rule_executions(rule_id, triggered_at DESC);
CREATE INDEX idx_rule_executions_customer  ON rule_executions(customer_id, triggered_at DESC);

ALTER TABLE rule_executions ENABLE ROW LEVEL SECURITY;

CREATE POLICY rule_executions_tenant_isolation ON rule_executions
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
