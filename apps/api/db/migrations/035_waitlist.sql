-- 035_waitlist.sql
-- Lista de espera pre-launch para capturar leads interesados en CitaSpot.
--
-- Esta tabla es GLOBAL (sin RLS) porque se llena ANTES de que exista
-- un tenant: los visitantes del landing dejan su email/negocio sin
-- crear cuenta. Igual que `tenants`, no tiene tenant_id y debe ser
-- accesible desde un endpoint público (sin JWT, sin contexto de tenant).
--
-- Acceso: solo escritura vía endpoint público POST /public/waitlist.
-- Lectura de admin se hace por psql directo o un endpoint admin futuro.

CREATE TABLE waitlist_signups (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT        NOT NULL UNIQUE,
  business_name TEXT        NOT NULL,
  ip_address    INET        NULL,
  user_agent    TEXT        NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- El UNIQUE en email ya crea un índice implícito (waitlist_signups_email_key).
-- Índice por created_at DESC para listados de admin (más recientes primero).
CREATE INDEX idx_waitlist_signups_created_at ON waitlist_signups (created_at DESC);

COMMENT ON TABLE waitlist_signups IS
  'Lista de espera pre-launch. Tabla global SIN RLS (no hay tenant_id porque '
  'los leads dejan datos antes de tener cuenta). Solo INSERT vía endpoint público.';
