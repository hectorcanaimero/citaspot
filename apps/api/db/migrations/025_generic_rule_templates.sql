-- 025_generic_rule_templates.sql
-- CITAS-15: insertar template generico vertical-agnostico de seguimiento post-cita.
-- Idempotente: solo inserta si no existe la fila con el template_key.
-- Corresponde a la fn Go seed.SeedGenericRuleTemplates — esta migration es backfill
-- inmediato para tenants ya onboarded antes del deploy.

-- Template tenant global. Configurar app.tenant_id para satisfacer RLS.
SELECT set_config('app.tenant_id', '00000000-0000-0000-0000-000000000000', true);

INSERT INTO rules
  (id, tenant_id, name, description, trigger_type, trigger_event,
   conditions, actions, is_active, is_template, template_key,
   cooldown_hours, priority, created_at, updated_at)
SELECT
  uuid_generate_v4(),
  '00000000-0000-0000-0000-000000000000'::uuid,
  'Seguimiento post-cita',
  'Mensaje de agradecimiento al completar una cita',
  'event',
  'appointment.completed',
  '[]'::jsonb,
  '[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Gracias por venir hoy a {{tenant_name}}. Como te fue? Responde del 1 al 5 para calificarnos. Para agendar tu proxima cita responde CITA.","params":{}}]'::jsonb,
  TRUE, TRUE, 'generic_post_appointment_24h',
  24, 0, NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM rules
  WHERE is_template = TRUE AND template_key = 'generic_post_appointment_24h'
);
