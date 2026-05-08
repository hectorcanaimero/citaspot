-- 026_add_dental_recall_3m_template.sql
-- Inserta el template global de reactivacion a 3 meses (90 dias).
-- Idempotente: WHERE NOT EXISTS evita duplicar si la migracion corre dos veces.
-- Requiere que el tenant de templates 00000000-0000-0000-0000-000000000000 exista
-- en la tabla tenants (FK constraint en rules.tenant_id).

-- Configurar app.tenant_id para satisfacer RLS al consultar / insertar.
SELECT set_config('app.tenant_id', '00000000-0000-0000-0000-000000000000', true);

INSERT INTO rules
  (id, tenant_id, name, description, trigger_type, trigger_event,
   conditions, actions, is_active, is_template, template_key,
   cooldown_hours, priority, created_at, updated_at)
SELECT
  gen_random_uuid(),
  '00000000-0000-0000-0000-000000000000'::uuid,
  'Recall trimestral',
  'Enviar WhatsApp para reactivar paciente inactivo hace 3 meses',
  'temporal',
  NULL,
  '[]'::jsonb,
  '[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Vimos que hace un tiempo que no nos visitas. Para mantener tu salud bucal y prevenir problemas mas serios, te recomendamos agendar un control. Responde SI y te buscamos un horario que te quede comodo.","params":{"interval_days":90}},{"type":"create_task","template":"Contactar a {{customer_name}} para reactivacion 3m","params":{"due_days":1}}]'::jsonb,
  TRUE, TRUE, 'dental_recall_3m',
  24, 0, NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM rules WHERE is_template = TRUE AND template_key = 'dental_recall_3m'
);
