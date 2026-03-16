-- Queries para la tabla appointments
-- Las queries se implementan en Fase 2, Tarea 7

-- name: GetAppointmentsByDate :many
-- Obtiene citas de un día para el tenant actual (RLS aplica automáticamente)
SELECT a.*,
       c.name AS customer_name, c.phone AS customer_phone,
       p.name AS professional_name,
       s.name AS service_name, s.duration_min
FROM appointments a
JOIN customers c ON c.id = a.customer_id
JOIN professionals p ON p.id = a.professional_id
JOIN services s ON s.id = a.service_id
WHERE a.tenant_id = current_setting('app.tenant_id')::UUID
  AND DATE(a.starts_at AT TIME ZONE $1) = $2::DATE
ORDER BY a.starts_at ASC;
