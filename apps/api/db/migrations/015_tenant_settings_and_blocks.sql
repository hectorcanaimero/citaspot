-- 015_tenant_settings_and_blocks.sql
-- Populate default settings for all existing tenants and add recurring block support.

-- Default tenant settings
UPDATE tenants SET settings = jsonb_build_object(
    'reminder_minutes', '[1440, 120]'::jsonb,
    'booking_intro_text', '',
    'booking_success_text', '',
    'bot_name', '',
    'bot_greeting', ''
) WHERE settings = '{}' OR settings IS NULL;

-- Set default for new tenants
ALTER TABLE tenants ALTER COLUMN settings SET DEFAULT jsonb_build_object(
    'reminder_minutes', '[1440, 120]'::jsonb,
    'booking_intro_text', '',
    'booking_success_text', '',
    'bot_name', '',
    'bot_greeting', ''
);

-- Recurring schedule blocks support
ALTER TABLE schedule_blocks ADD COLUMN is_recurring BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE schedule_blocks ADD COLUMN recurrence_days SMALLINT[];

COMMENT ON COLUMN schedule_blocks.is_recurring IS 'TRUE = repeats weekly on recurrence_days';
COMMENT ON COLUMN schedule_blocks.recurrence_days IS 'Days of week (0=Sun..6=Sat) when this block repeats';
