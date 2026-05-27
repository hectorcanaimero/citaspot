-- Notificaciones in-app para usuarios del dashboard.
-- Distinta de notification_logs (que trackea mensajes WhatsApp salientes).
-- Esta tabla almacena el feed de novedades que ve el dueño/admin en el
-- bell-icon del dashboard: nuevas citas, cancelaciones, reagendamientos,
-- y mensajes entrantes de WhatsApp.

CREATE TABLE user_notifications (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  type        TEXT NOT NULL,
    -- 'appointment.created' | 'appointment.cancelled'
    -- 'appointment.rescheduled' | 'whatsapp.inbound'
  title       TEXT NOT NULL,
  body        TEXT,
  metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
  read_at     TIMESTAMPTZ NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Listing ordenado por fecha (paginación con cursor sobre created_at)
CREATE INDEX idx_user_notifications_tenant_created
  ON user_notifications (tenant_id, created_at DESC);

-- Conteo de no-leídas — partial index para queries baratas
CREATE INDEX idx_user_notifications_tenant_unread
  ON user_notifications (tenant_id)
  WHERE read_at IS NULL;

-- RLS: aislamiento por tenant (patrón canónico del proyecto)
ALTER TABLE user_notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_notifications_tenant_isolation ON user_notifications
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
