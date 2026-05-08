-- 019a_seed_templates_tenant.sql
-- Crea el tenant especial 00000000-... usado para alojar reglas globales (templates).
-- Idempotente: ON CONFLICT (id) DO NOTHING.
--
-- Posicionado alfabéticamente entre 019_rules.sql (que define el FK rules.tenant_id -> tenants.id)
-- y cualquier migración subsiguiente que inserte filas en `rules` referenciando este tenant
-- (por ejemplo 025_generic_rule_templates.sql, 026_add_dental_recall_3m_template.sql).
--
-- Este tenant es invisible al frontend: middleware/tenant.go:83 rechaza requests HTTP
-- que intenten usarlo. Solo se usa internamente para almacenar templates de reglas.

INSERT INTO tenants (
  id, slug, name, business_type, email,
  plan, plan_status, onboarding_done, settings
)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  '__templates__',
  'Templates Globales',
  'dental',
  'templates@citaspot.internal',
  'basic',
  'active',
  TRUE,
  '{"internal":true,"role":"templates_repository"}'::jsonb
)
ON CONFLICT (id) DO NOTHING;
