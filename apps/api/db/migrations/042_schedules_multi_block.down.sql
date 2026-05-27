-- 042_schedules_multi_block.down.sql
-- Reversión: restaurar UNIQUE y eliminar CHECK + índice nuevo.
-- NOTA: si hay filas con múltiples bloques/día, este down FALLA — limpiar antes.

DROP INDEX IF EXISTS idx_schedules_professional_day;
CREATE INDEX idx_schedules_professional ON schedules(professional_id);

ALTER TABLE schedules
  DROP CONSTRAINT IF EXISTS schedules_time_range_check;

ALTER TABLE schedules
  ADD CONSTRAINT schedules_professional_id_day_of_week_key
  UNIQUE (professional_id, day_of_week);
