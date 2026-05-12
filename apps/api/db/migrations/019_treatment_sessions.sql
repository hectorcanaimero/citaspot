-- 019_treatment_sessions.sql
-- Sesiones individuales de un tratamiento (cada cita dentro de un plan de tratamiento)

CREATE TABLE treatment_sessions (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  treatment_id      UUID NOT NULL REFERENCES treatments(id) ON DELETE CASCADE,
  professional_id   UUID NOT NULL REFERENCES professionals(id),

  status            TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'completed', 'cancelled')),

  scheduled_at      TIMESTAMPTZ NOT NULL,
  duration_minutes  INT,
  completed_at      TIMESTAMPTZ,

  procedures_done   TEXT,
  notes             TEXT,

  paid_in_session   NUMERIC(10,2),
  currency          TEXT,

  next_session_at   TIMESTAMPTZ,

  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE treatment_sessions ENABLE ROW LEVEL SECURITY;

CREATE POLICY treatment_sessions_tenant_isolation ON treatment_sessions
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE INDEX idx_treatment_sessions_treatment ON treatment_sessions(treatment_id);
CREATE INDEX idx_treatment_sessions_tenant_scheduled ON treatment_sessions(tenant_id, scheduled_at);
