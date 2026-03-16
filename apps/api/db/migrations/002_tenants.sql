-- 002_tenants.sql
-- Tabla principal de tenants (negocios clientes de CitaSpot)
-- Un tenant = un negocio (peluquería, clínica, spa, etc.)

CREATE TABLE tenants (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  slug                VARCHAR(100) UNIQUE NOT NULL,           -- URL pública: citaspot.com/{slug}
  name                VARCHAR(255) NOT NULL,
  business_type       VARCHAR(50)  NOT NULL,                  -- 'beauty','dental','wellness','barbershop'
  phone               VARCHAR(20),
  email               VARCHAR(255) UNIQUE NOT NULL,
  city                VARCHAR(100),
  country             CHAR(2)      NOT NULL DEFAULT 'DO',     -- ISO 3166-1 alpha-2
  timezone            VARCHAR(50)  NOT NULL DEFAULT 'America/Santo_Domingo',
  logo_url            TEXT,
  address             TEXT,

  -- Plan y facturación
  plan                VARCHAR(20)  NOT NULL DEFAULT 'basic',  -- 'basic','pro','clinic'
  plan_status         VARCHAR(20)  NOT NULL DEFAULT 'trial',  -- 'trial','active','past_due','cancelled'
  trial_ends_at       TIMESTAMPTZ,
  stripe_customer_id  VARCHAR(100),
  stripe_sub_id       VARCHAR(100),

  -- WhatsApp
  whatsapp_number     VARCHAR(20),                            -- Número conectado a Evolution API
  wa_instance_id      VARCHAR(100),                          -- ID de instancia en Evolution API
  wa_status           VARCHAR(20) DEFAULT 'disconnected',    -- 'connected','disconnected','banned'

  -- Config
  onboarding_done     BOOLEAN DEFAULT FALSE,
  settings            JSONB   DEFAULT '{}',                   -- Configuraciones adicionales por tenant

  created_at          TIMESTAMPTZ DEFAULT NOW(),
  updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug  ON tenants(slug);
CREATE INDEX idx_tenants_email ON tenants(email);
