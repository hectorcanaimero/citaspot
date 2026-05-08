-- 024_default_reminder_48h.sql
-- CITAS-13: agregar 48h (2880 min) al default de reminder_minutes.
-- Pasamos de [1440, 120] (24h, 2h) a [2880, 1440, 120] (48h, 24h, 2h).

-- 1) Nuevo default para tenants nuevos
ALTER TABLE tenants ALTER COLUMN settings SET DEFAULT jsonb_build_object(
    'reminder_minutes', '[2880, 1440, 120]'::jsonb,
    'booking_intro_text', '',
    'booking_success_text', '',
    'bot_name', '',
    'bot_greeting', ''
);

-- 2) Backfill SOLO tenants que tienen el default viejo exacto.
-- No tocar tenants que personalizaron (otros valores, mas o menos entries, etc.).
UPDATE tenants
SET settings = jsonb_set(settings, '{reminder_minutes}', '[2880, 1440, 120]'::jsonb)
WHERE settings->'reminder_minutes' = '[1440, 120]'::jsonb;
