-- Link inverso entre treatment_sessions y appointments.
-- Cuando una appointment con treatment_id se crea, se materializa una
-- treatment_session vinculada. ON DELETE CASCADE garantiza que al borrar
-- el appointment se borra también la session derivada.
-- Sessions creadas manualmente (registro retroactivo) tienen appointment_id NULL.

ALTER TABLE treatment_sessions
  ADD COLUMN appointment_id UUID NULL REFERENCES appointments(id) ON DELETE CASCADE;

CREATE INDEX idx_treatment_sessions_appointment
  ON treatment_sessions(appointment_id)
  WHERE appointment_id IS NOT NULL;
