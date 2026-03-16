-- 012_fix_notification_logs_rls.sql
-- notification_logs tenía tenant_id pero le faltaba RLS — violación de la regla de multi-tenancy.
-- Este fix agrega la política de aislamiento obligatoria.

ALTER TABLE notification_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY notification_logs_tenant_isolation ON notification_logs
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
