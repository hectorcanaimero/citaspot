-- Queries para la tabla professionals
-- Las queries se implementan en Fase 2, Tarea 5

-- name: ListProfessionals :many
SELECT * FROM professionals
WHERE tenant_id = current_setting('app.tenant_id')::UUID
  AND is_active = TRUE
ORDER BY name ASC;
