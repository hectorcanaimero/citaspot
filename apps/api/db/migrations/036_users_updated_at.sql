-- 036_users_updated_at.sql
-- Añade columna updated_at a users para audit trail y consistencia con el resto del schema.
-- Backfill con created_at en filas existentes.

ALTER TABLE users
  ADD COLUMN updated_at TIMESTAMPTZ DEFAULT NOW();

UPDATE users SET updated_at = created_at WHERE updated_at IS NULL;
