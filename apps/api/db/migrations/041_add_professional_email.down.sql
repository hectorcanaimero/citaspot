DROP INDEX IF EXISTS idx_professionals_tenant_email;
ALTER TABLE professionals DROP COLUMN IF EXISTS email;
