-- Activación de módulos por tenant (feature flags por suscripción).
-- Sin RLS: tabla de configuración consultada por el middleware ANTES de
-- tener contexto de tenant (similar al patrón de la tabla tenants).
-- Cada activación tiene auditoría (activated_at, deactivated_at).

CREATE TABLE tenant_modules (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  module_key      TEXT NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  activated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deactivated_at  TIMESTAMPTZ NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, module_key)
);

CREATE INDEX idx_tenant_modules_lookup
  ON tenant_modules(tenant_id, module_key)
  WHERE enabled = TRUE;

-- Backfill: activar módulo 'dental' para tenants con business_type='dental'
-- para no romper acceso tras desplegar el middleware.
INSERT INTO tenant_modules (tenant_id, module_key, enabled, activated_at)
SELECT id, 'dental', TRUE, NOW()
FROM tenants
WHERE business_type = 'dental'
ON CONFLICT (tenant_id, module_key) DO NOTHING;
