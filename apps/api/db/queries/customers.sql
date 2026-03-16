-- Queries para la tabla customers
-- Las queries se implementan en Fase 2, Tarea 5

-- name: FindOrCreateCustomerByPhone :one
SELECT * FROM customers
WHERE tenant_id = current_setting('app.tenant_id')::UUID
  AND phone = $1;
