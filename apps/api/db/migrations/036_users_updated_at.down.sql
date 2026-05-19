-- 036_users_updated_at.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS updated_at;
