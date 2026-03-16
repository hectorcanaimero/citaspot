-- 007_customers.sql
-- Clientes del negocio (los que reservan citas)

CREATE TABLE customers (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name         VARCHAR(255) NOT NULL,
  phone        VARCHAR(20),                -- Número WhatsApp normalizado: +1XXXXXXXXXX
  email        VARCHAR(255),
  notes        TEXT,                       -- Notas privadas del negocio sobre el cliente
  tags         TEXT[] DEFAULT '{}',        -- Ej: ['vip', 'alergia_latex']
  wa_opt_in    BOOLEAN DEFAULT TRUE,       -- Aceptó recibir mensajes WA
  last_visit   DATE,
  total_visits INTEGER DEFAULT 0,
  created_at   TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, phone)
);

CREATE INDEX idx_customers_tenant ON customers(tenant_id);
CREATE INDEX idx_customers_phone  ON customers(tenant_id, phone);

-- GIN index para búsqueda de texto en nombre
CREATE INDEX idx_customers_name_trgm ON customers USING GIN (name gin_trgm_ops);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;

CREATE POLICY customers_tenant_isolation ON customers
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
