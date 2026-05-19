DROP INDEX IF EXISTS idx_treatment_sessions_appointment;
ALTER TABLE treatment_sessions DROP COLUMN IF EXISTS appointment_id;
