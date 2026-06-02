ALTER TABLE tenants
  DROP COLUMN IF EXISTS dental_assistant_mode,
  DROP COLUMN IF EXISTS urgency_phone,
  DROP COLUMN IF EXISTS urgency_message;
