-- 027_fix_service_typos.sql
-- Corrige typos en nombres de servicios introducidos manualmente en datos demo.
-- Idempotente: solo afecta filas cuyo name coincide exactamente con el typo.
-- Nota: la tabla services no tiene columna updated_at (solo created_at).

UPDATE services SET name = 'Extracción'              WHERE name = 'Extracion';
UPDATE services SET name = 'Ortodoncia'              WHERE name = 'OIrtodoncia';
UPDATE services SET name = 'Tratamiento de conducto' WHERE name = 'Tratamiento de conduct';
