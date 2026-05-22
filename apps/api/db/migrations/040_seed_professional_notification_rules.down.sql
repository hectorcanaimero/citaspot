-- 040_seed_professional_notification_rules.down.sql
-- Revierte el seed de reglas de notificacion al profesional.
-- Borra solo las reglas con template_key conocido para no afectar reglas custom.
-- RLS requiere tenant context, pero como borramos por template_key especifico
-- a nivel global, desactivamos RLS temporalmente via SECURITY DEFINER no es opcion;
-- en su lugar iteramos tenants igual que el up.

DO $$
DECLARE
    t RECORD;
BEGIN
    FOR t IN SELECT id FROM tenants LOOP
        PERFORM set_config('app.tenant_id', t.id::text, true);

        DELETE FROM rules
        WHERE tenant_id = t.id
          AND template_key IN (
              'prof_appointment_created',
              'prof_appointment_confirmed',
              'prof_appointment_cancelled',
              'prof_appointment_rescheduled'
          );
    END LOOP;
END $$;
