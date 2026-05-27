-- Añade columna email opcional a professionals.
-- Usado en la vista 360 del profesional (perfil completo) y para futuras
-- integraciones de notificaciones por email.
ALTER TABLE professionals ADD COLUMN email VARCHAR(255);

-- Índice parcial: solo indexa filas con email no NULL para acelerar lookups
-- por tenant + email sin penalizar inserciones de profesionales sin email.
CREATE INDEX idx_professionals_tenant_email
  ON professionals(tenant_id, email)
  WHERE email IS NOT NULL;
