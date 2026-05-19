DROP INDEX IF EXISTS idx_treatments_recall_pending;
ALTER TABLE treatments DROP COLUMN IF EXISTS recall_sent_at;

DROP INDEX IF EXISTS idx_treatment_sessions_post_op_pending;
ALTER TABLE treatment_sessions DROP COLUMN IF EXISTS post_op_sent_at;
