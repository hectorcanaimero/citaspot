-- 018_treatments.sql
-- Tratamientos dentales (o de cualquier vertical) asociados a un cliente

CREATE TABLE treatments (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  customer_id         UUID NOT NULL REFERENCES customers(id),
  professional_id     UUID NOT NULL REFERENCES professionals(id),
  name                TEXT NOT NULL,
  treatment_type      TEXT NOT NULL,

  status              TEXT NOT NULL DEFAULT 'proposed',

  total_sessions      INT,
  completed_sessions  INT DEFAULT 0,
  estimated_cost      NUMERIC(10,2),
  paid_amount         NUMERIC(10,2) DEFAULT 0,
  currency            TEXT DEFAULT 'USD',
  tooth_numbers       INT[],
  notes               TEXT,

  started_at          TIMESTAMPTZ,
  completed_at        TIMESTAMPTZ,
  next_session_at     TIMESTAMPTZ,

  created_at          TIMESTAMPTZ DEFAULT NOW(),
  updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_treatments_tenant         ON treatments(tenant_id);
CREATE INDEX idx_treatments_customer       ON treatments(tenant_id, customer_id);
CREATE INDEX idx_treatments_professional   ON treatments(tenant_id, professional_id);
CREATE INDEX idx_treatments_status         ON treatments(tenant_id, status);

ALTER TABLE treatments ENABLE ROW LEVEL SECURITY;

CREATE POLICY treatments_tenant_isolation ON treatments
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

-- Vincular citas con tratamientos
ALTER TABLE appointments ADD COLUMN treatment_id UUID REFERENCES treatments(id);
CREATE INDEX idx_appointments_treatment ON appointments(treatment_id);
