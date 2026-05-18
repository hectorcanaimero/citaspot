-- Añade columna phone opcional a professionals.
-- Formato E.164 sin '+': prefijo país + número local (ej: '584243222332').
-- Validación de formato la hace el frontend (PhoneInput) y la capa de aplicación;
-- la DB solo persiste el string normalizado.
ALTER TABLE professionals
  ADD COLUMN phone VARCHAR(20);
