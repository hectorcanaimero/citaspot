-- 006_schedules.sql
-- Disponibilidad semanal por profesional + bloqueos específicos

CREATE TABLE schedules (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  professional_id UUID NOT NULL REFERENCES professionals(id) ON DELETE CASCADE,
  day_of_week     SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0=Dom, 1=Lun, ..., 6=Sáb
  start_time      TIME NOT NULL,
  end_time        TIME NOT NULL,
  is_active       BOOLEAN DEFAULT TRUE,
  UNIQUE(professional_id, day_of_week)
);

-- Bloqueos específicos: vacaciones, festivos, pausas
CREATE TABLE schedule_blocks (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  professional_id UUID REFERENCES professionals(id),   -- NULL = bloqueo para todo el negocio
  starts_at       TIMESTAMPTZ NOT NULL,
  ends_at         TIMESTAMPTZ NOT NULL,
  reason          VARCHAR(255),
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_schedules_professional ON schedules(professional_id);
CREATE INDEX idx_blocks_professional_time ON schedule_blocks(professional_id, starts_at, ends_at);

ALTER TABLE schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE schedule_blocks ENABLE ROW LEVEL SECURITY;

CREATE POLICY schedules_tenant_isolation ON schedules
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY blocks_tenant_isolation ON schedule_blocks
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
