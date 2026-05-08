-- 016_flexible_reminders.sql
-- Generic reminders tracking: stores which reminder times have been sent.
-- Keys are minutes-before-appointment as strings, values are booleans.

ALTER TABLE appointments ADD COLUMN reminders_sent JSONB NOT NULL DEFAULT '{}';

-- Backfill from existing boolean columns
UPDATE appointments SET reminders_sent = jsonb_build_object(
    '1440', reminder_24h_sent,
    '120', reminder_2h_sent
);

-- Index for reminder worker queries
CREATE INDEX idx_appointments_reminders_jsonb ON appointments(tenant_id, starts_at, status)
    WHERE reminders_sent != '{"1440":true,"120":true}';
