-- 042_schedules_multi_block.sql
-- CITAS-42 "Bloquear Horas" — soportar múltiples bloques de trabajo por día.
-- Ej: lunes 08:00-12:00 + 14:00-18:00 (pausa al mediodía).
--
-- Cambios:
--   1. Drop UNIQUE(professional_id, day_of_week) — permitía 1 bloque/día.
--   2. CHECK (start_time < end_time) — invariante a nivel DB.
--   3. Índice compuesto (professional_id, day_of_week, start_time) — orden estable.

ALTER TABLE schedules
  DROP CONSTRAINT IF EXISTS schedules_professional_id_day_of_week_key;

ALTER TABLE schedules
  ADD CONSTRAINT schedules_time_range_check CHECK (start_time < end_time);

DROP INDEX IF EXISTS idx_schedules_professional;
CREATE INDEX idx_schedules_professional_day
  ON schedules(professional_id, day_of_week, start_time);
