-- 008_appointments.sql
-- Tabla central: las citas del negocio

CREATE TABLE appointments (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  customer_id     UUID NOT NULL REFERENCES customers(id),
  professional_id UUID NOT NULL REFERENCES professionals(id),
  service_id      UUID NOT NULL REFERENCES services(id),
  starts_at       TIMESTAMPTZ NOT NULL,
  ends_at         TIMESTAMPTZ NOT NULL,    -- starts_at + service.duration_min

  status          VARCHAR(20) NOT NULL DEFAULT 'pending',
                  -- 'pending','confirmed','cancelled','completed','no_show'

  source          VARCHAR(20) DEFAULT 'whatsapp',
                  -- 'whatsapp','web','dashboard','api'

  price           DECIMAL(10,2),           -- snapshot del precio al agendar
  notes           TEXT,                    -- Notas del cliente para la cita
  internal_notes  TEXT,                    -- Notas internas del negocio

  confirmed_at        TIMESTAMPTZ,
  cancelled_at        TIMESTAMPTZ,
  cancellation_reason TEXT,

  -- Control de notificaciones
  reminder_24h_sent   BOOLEAN DEFAULT FALSE,
  reminder_2h_sent    BOOLEAN DEFAULT FALSE,
  review_requested    BOOLEAN DEFAULT FALSE,

  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Índices críticos para el engine de disponibilidad
CREATE INDEX idx_appointments_tenant_time    ON appointments(tenant_id, starts_at);
CREATE INDEX idx_appointments_professional   ON appointments(professional_id, starts_at);
CREATE INDEX idx_appointments_customer       ON appointments(customer_id);
CREATE INDEX idx_appointments_status        ON appointments(tenant_id, status);

-- Para el cron de recordatorios
CREATE INDEX idx_appointments_reminders ON appointments(tenant_id, starts_at, status)
  WHERE reminder_24h_sent = FALSE OR reminder_2h_sent = FALSE;

ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;

CREATE POLICY appointments_tenant_isolation ON appointments
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
