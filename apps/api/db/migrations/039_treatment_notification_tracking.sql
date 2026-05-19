-- Tracking de notificaciones automáticas para evitar reenvíos idempotentes.
-- - post_op_sent_at: marca cuándo se envió el WhatsApp post-op 48h después
--   de completar una sesión de tratamiento.
-- - recall_sent_at: marca cuándo se envió el recall 6 meses después de
--   completar el tratamiento.

ALTER TABLE treatment_sessions
  ADD COLUMN post_op_sent_at TIMESTAMPTZ NULL;

CREATE INDEX idx_treatment_sessions_post_op_pending
  ON treatment_sessions(completed_at)
  WHERE status = 'completed' AND post_op_sent_at IS NULL;

ALTER TABLE treatments
  ADD COLUMN recall_sent_at TIMESTAMPTZ NULL;

CREATE INDEX idx_treatments_recall_pending
  ON treatments(completed_at)
  WHERE status = 'completed' AND recall_sent_at IS NULL;
