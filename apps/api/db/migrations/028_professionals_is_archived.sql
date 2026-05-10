-- 028_professionals_is_archived.sql
-- Agregar campo is_archived para borrado lógico de profesionales
ALTER TABLE professionals ADD COLUMN is_archived BOOLEAN NOT NULL DEFAULT FALSE;

-- Índice para filtrar archivados eficientemente
CREATE INDEX idx_professionals_archived ON professionals (tenant_id, is_archived);
