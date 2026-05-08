-- 020_tasks.sql
-- Tareas asignables a profesionales (manuales o generadas por reglas)

CREATE TABLE tasks (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  assigned_to     UUID REFERENCES professionals(id),
  customer_id     UUID REFERENCES customers(id),
  appointment_id  UUID REFERENCES appointments(id),
  treatment_id    UUID REFERENCES treatments(id),
  title           TEXT NOT NULL,
  description     TEXT,
  status          TEXT NOT NULL DEFAULT 'pending',
  due_at          TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  source          TEXT NOT NULL DEFAULT 'manual',
  rule_id         UUID REFERENCES rules(id),
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tasks_tenant       ON tasks(tenant_id);
CREATE INDEX idx_tasks_assigned     ON tasks(tenant_id, assigned_to) WHERE status IN ('pending', 'in_progress');
CREATE INDEX idx_tasks_customer     ON tasks(customer_id);
CREATE INDEX idx_tasks_status       ON tasks(tenant_id, status);
CREATE INDEX idx_tasks_due          ON tasks(tenant_id, due_at) WHERE status = 'pending';

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;

CREATE POLICY tasks_tenant_isolation ON tasks
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
