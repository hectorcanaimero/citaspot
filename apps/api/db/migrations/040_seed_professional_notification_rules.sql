-- 040_seed_professional_notification_rules.sql
-- Inserta reglas de notificacion al PROFESIONAL para cada evento del ciclo de vida de una cita.
-- Complementa la 031 (que solo notifica al cliente).
-- Idempotente via template_key + tenant_id (mismo patron que 031).
--
-- Las acciones usan "recipient":"professional" en params, lo que hace que
-- SendWhatsAppAction resuelva el telefono desde context["professional_phone"]
-- (ya inyectado por appointmentSvc.emitAppointmentEvent).
-- Si el profesional no tiene telefono, el action loggea warn y skipea (best-effort).

DO $$
DECLARE
    t RECORD;
    rule_templates JSONB := '[
        {
            "name": "Profesional — nueva reserva pendiente",
            "description": "Notifica al profesional cuando un cliente reserva una cita (status pending)",
            "trigger_event": "appointment.created",
            "template_key": "prof_appointment_created",
            "actions": [{"type":"send_whatsapp","template":"📋 *Nueva reserva pendiente*\n\n👤 {{customer_name}}\n📌 {{service_name}}\n📅 {{starts_at}}\n\nConfirma o ajusta desde tu panel.","params":{"recipient":"professional"}}]
        },
        {
            "name": "Profesional — cita confirmada",
            "description": "Notifica al profesional cuando una cita es confirmada",
            "trigger_event": "appointment.confirmed",
            "template_key": "prof_appointment_confirmed",
            "actions": [{"type":"send_whatsapp","template":"✅ *Cita confirmada*\n\n👤 {{customer_name}}\n📌 {{service_name}}\n📅 {{starts_at}}\n\nTe esperamos en tu agenda.","params":{"recipient":"professional"}}]
        },
        {
            "name": "Profesional — cita cancelada",
            "description": "Notifica al profesional cuando una cita es cancelada",
            "trigger_event": "appointment.cancelled",
            "template_key": "prof_appointment_cancelled",
            "actions": [{"type":"send_whatsapp","template":"❌ *Cita cancelada*\n\n👤 {{customer_name}} canceló su cita.\n📌 {{service_name}}\n📅 {{starts_at}}\n\nEse horario queda libre.","params":{"recipient":"professional"}}]
        },
        {
            "name": "Profesional — cita reagendada",
            "description": "Notifica al profesional cuando una cita es reagendada",
            "trigger_event": "appointment.rescheduled",
            "template_key": "prof_appointment_rescheduled",
            "actions": [{"type":"send_whatsapp","template":"🔄 *Cita reagendada*\n\n👤 {{customer_name}}\n📌 {{service_name}}\n📅 Nuevo horario: {{starts_at}}","params":{"recipient":"professional"}}]
        }
    ]';
    tmpl JSONB;
BEGIN
    FOR t IN SELECT id FROM tenants LOOP
        -- Setear tenant para RLS (rules tiene tenant_isolation policy)
        PERFORM set_config('app.tenant_id', t.id::text, true);

        FOR tmpl IN SELECT * FROM jsonb_array_elements(rule_templates) LOOP
            -- Idempotencia: solo insertar si no existe para este tenant
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
