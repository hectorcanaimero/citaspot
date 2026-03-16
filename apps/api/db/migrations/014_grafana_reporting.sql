-- 014_grafana_reporting.sql
-- Rol de solo lectura para Grafana + vistas de reporting cross-tenant
-- BYPASSRLS es necesario para que Grafana vea datos de todos los tenants
-- El usuario grafana_user SOLO debe usarse desde Grafana — nunca desde la app

-- ── Rol de solo lectura ─────────────────────────────────────────────────
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'grafana_ro') THEN
    CREATE ROLE grafana_ro NOLOGIN;
  END IF;
END
$$;

GRANT CONNECT  ON DATABASE citaspot TO grafana_ro;
GRANT USAGE    ON SCHEMA public    TO grafana_ro;
GRANT SELECT   ON ALL TABLES IN SCHEMA public TO grafana_ro;

-- Permisos para tablas futuras que se creen después de esta migración
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO grafana_ro;

-- BYPASSRLS permite cruzar datos de múltiples tenants (solo para reporting)
ALTER ROLE grafana_ro BYPASSRLS;

-- ── Usuario de login para Grafana ───────────────────────────────────────
-- Contraseña por defecto en desarrollo. En producción cambiar con:
--   ALTER USER grafana_user PASSWORD '<GRAFANA_DB_PASSWORD>';
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'grafana_user') THEN
    CREATE USER grafana_user WITH PASSWORD 'grafana_dev';
  END IF;
END
$$;
GRANT grafana_ro TO grafana_user;

-- ── Vista: citas diarias cross-tenant ───────────────────────────────────
CREATE OR REPLACE VIEW rpt_appointments_daily AS
SELECT
  DATE_TRUNC('day', starts_at)                       AS day,
  tenant_id,
  COUNT(*)                                            AS total,
  COUNT(*) FILTER (WHERE status = 'completed')        AS completed,
  COUNT(*) FILTER (WHERE status = 'confirmed')        AS confirmed,
  COUNT(*) FILTER (WHERE status = 'cancelled')        AS cancelled,
  COUNT(*) FILTER (WHERE status = 'no_show')          AS no_show,
  COUNT(*) FILTER (WHERE source  = 'whatsapp')        AS from_whatsapp,
  COUNT(*) FILTER (WHERE source  = 'web')             AS from_web,
  COUNT(*) FILTER (WHERE source  = 'dashboard')       AS from_dashboard,
  COALESCE(SUM(price), 0)                             AS revenue
FROM appointments
GROUP BY 1, 2;

GRANT SELECT ON rpt_appointments_daily TO grafana_ro;

-- ── Vista: KPIs de tenants ───────────────────────────────────────────────
CREATE OR REPLACE VIEW rpt_tenant_kpis AS
SELECT
  t.id,
  t.slug,
  t.name,
  t.business_type,
  t.country,
  t.plan,
  t.plan_status,
  t.wa_status,
  t.created_at,
  COALESCE(a.total_appointments, 0) AS total_appointments,
  COALESCE(c.total_customers, 0)    AS total_customers,
  -- MRR estimado por tenant según plan
  CASE t.plan
    WHEN 'basic'  THEN 10
    WHEN 'pro'    THEN 20
    WHEN 'clinic' THEN 25
    ELSE 0
  END AS mrr_usd
FROM tenants t
LEFT JOIN (
  SELECT tenant_id, COUNT(*) AS total_appointments
  FROM appointments
  GROUP BY tenant_id
) a ON a.tenant_id = t.id
LEFT JOIN (
  SELECT tenant_id, COUNT(*) AS total_customers
  FROM customers
  GROUP BY tenant_id
) c ON c.tenant_id = t.id;

GRANT SELECT ON rpt_tenant_kpis TO grafana_ro;

-- ── Vista: tokens IA por día ─────────────────────────────────────────────
CREATE OR REPLACE VIEW rpt_ai_tokens_daily AS
SELECT
  DATE_TRUNC('day', m.created_at)         AS day,
  m.tenant_id,
  COALESCE(SUM(m.tokens_used), 0)         AS tokens,
  COUNT(DISTINCT m.conversation_id)       AS conversations,
  COUNT(*)                                AS messages,
  -- Costo estimado: Gemini 2.5 Flash ≈ $0.075 / 1M tokens de input
  ROUND(COALESCE(SUM(m.tokens_used), 0) * 0.000000075 ::NUMERIC, 6) AS cost_usd
FROM messages m
GROUP BY 1, 2;

GRANT SELECT ON rpt_ai_tokens_daily TO grafana_ro;

-- ── Vista: notificaciones WhatsApp ───────────────────────────────────────
CREATE OR REPLACE VIEW rpt_notifications_daily AS
SELECT
  DATE_TRUNC('day', sent_at)              AS day,
  tenant_id,
  type,
  status,
  COUNT(*)                                AS total
FROM notification_logs
GROUP BY 1, 2, 3, 4;

GRANT SELECT ON rpt_notifications_daily TO grafana_ro;
