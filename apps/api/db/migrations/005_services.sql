-- 005_services.sql
-- Servicios que ofrece el negocio

CREATE TABLE services (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name          VARCHAR(255) NOT NULL,
  description   TEXT,
  duration_min  INTEGER NOT NULL DEFAULT 60,   -- duración en minutos
  price         DECIMAL(10,2),
  currency      CHAR(3) DEFAULT 'USD',
  buffer_min    INTEGER DEFAULT 0,             -- minutos de buffer entre citas
  is_active     BOOLEAN DEFAULT TRUE,
  sort_order    INTEGER DEFAULT 0,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Qué profesionales realizan qué servicios
CREATE TABLE professional_services (
  professional_id UUID REFERENCES professionals(id) ON DELETE CASCADE,
  service_id      UUID REFERENCES services(id) ON DELETE CASCADE,
  PRIMARY KEY (professional_id, service_id)
);

CREATE INDEX idx_services_tenant ON services(tenant_id);

ALTER TABLE services ENABLE ROW LEVEL SECURITY;

CREATE POLICY services_tenant_isolation ON services
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
