-- 003_users.sql
-- Usuarios con acceso al dashboard (dueños y empleados del negocio)

CREATE TABLE users (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email       VARCHAR(255) NOT NULL,
  name        VARCHAR(255) NOT NULL,
  role        VARCHAR(20)  NOT NULL DEFAULT 'owner',  -- 'owner','admin','staff'
  avatar_url  TEXT,
  auth_id     VARCHAR(255) UNIQUE,                    -- Supabase Auth UID
  last_login  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant  ON users(tenant_id);
CREATE INDEX idx_users_auth_id ON users(auth_id);

-- RLS: cada usuario solo ve usuarios de su tenant
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_tenant_isolation ON users
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
