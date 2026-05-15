-- 031_seed_appointment_notification_rules.sql
-- Inserta reglas de notificación de citas para TODOS los tenants existentes.
-- Estas reglas reemplazan las notificaciones hardcodeadas que se eliminaron de appointmentSvc.
-- Idempotente via template_key + tenant_id.

DO $$
DECLARE
    t RECORD;
    rule_templates JSONB := '[
        {
            "name": "Cita creada — pendiente de aprobación",
            "description": "Notifica al paciente que su cita fue registrada y está pendiente de confirmación",
            "trigger_event": "appointment.created",
            "template_key": "generic_appointment_created",
            "actions": [{"type":"send_whatsapp","template":"¡Hola {{customer_name}}! 📋 Tu cita para *{{service_name}}* con {{professional_name}} ha sido registrada y está *pendiente de aprobación*. Te avisaremos cuando sea confirmada.","params":{}}]
        },
        {
            "name": "Cita confirmada",
            "description": "Notifica al paciente que su cita fue confirmada",
            "trigger_event": "appointment.confirmed",
            "template_key": "generic_appointment_confirmed",
            "actions": [{"type":"send_whatsapp","template":"¡Hola {{customer_name}}! ✅ Tu cita en *{{tenant_name}}* ha sido *confirmada*:\n\n📌 *{{service_name}}*\n👩‍⚕️ {{professional_name}}\n📅 {{starts_at}}\n\n¡Te esperamos!","params":{}}]
        },
        {
            "name": "Cita cancelada",
            "description": "Notifica al paciente que su cita fue cancelada",
            "trigger_event": "appointment.cancelled",
            "template_key": "generic_appointment_cancelled",
            "actions": [{"type":"send_whatsapp","template":"Hola {{customer_name}}, lamentamos informarte que tu cita de *{{service_name}}* ha sido *cancelada*. Puedes reservar nuevamente cuando lo desees.","params":{}}]
        },
        {
            "name": "Cita reagendada",
            "description": "Notifica al paciente que su cita fue reagendada",
            "trigger_event": "appointment.rescheduled",
            "template_key": "generic_appointment_rescheduled",
            "actions": [{"type":"send_whatsapp","template":"¡Hola {{customer_name}}! 🔄 Tu cita en *{{tenant_name}}* ha sido *reagendada*:\n\n📌 *{{service_name}}*\n👩‍⚕️ {{professional_name}}\n📅 {{starts_at}}\n\n¡Te esperamos!","params":{}}]
        }
    ]';
    tmpl JSONB;
BEGIN
    FOR t IN SELECT id FROM tenants LOOP
        -- Setear tenant para RLS
        PERFORM set_config('app.tenant_id', t.id::text, true);

        FOR tmpl IN SELECT * FROM jsonb_array_elements(rule_templates) LOOP
            -- Solo insertar si no existe para este tenant
            IF NOT EXISTS (
                SELECT 1 FROM rules
                WHERE tenant_id = t.id
                  AND template_key = tmpl->>'template_key'
            ) THEN
                INSERT INTO rules (
                    id, tenant_id, name, description,
                    trigger_type, trigger_event,
                    conditions, actions,
                    is_active, is_template, template_key,
                    cooldown_hours, priority,
                    created_at, updated_at
                ) VALUES (
                    uuid_generate_v4(), t.id,
                    tmpl->>'name', tmpl->>'description',
                    'event', tmpl->>'trigger_event',
                    '[]'::jsonb, (tmpl->'actions'),
                    TRUE, FALSE, tmpl->>'template_key',
                    0, 0,
                    NOW(), NOW()
                );
            END IF;
        END LOOP;
    END LOOP;
END $$;
