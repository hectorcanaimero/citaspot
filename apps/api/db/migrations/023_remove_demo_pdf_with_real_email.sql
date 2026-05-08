-- 023_remove_demo_pdf_with_real_email.sql
-- Plane #3: el tenant demo tiene un PDF subido con email real (knaimero@gmail.com).
-- 1) Eliminar documentos ofensivos (chunks cascadean por FK).
-- 2) Reemplazar con knowledge ficticia de clinica dental para los tenants afectados.

CREATE TEMP TABLE _affected_tenants ON COMMIT DROP AS
SELECT DISTINCT tenant_id
FROM knowledge_documents
WHERE source_type = 'file'
  AND content ILIKE '%knaimero@gmail.com%';

DELETE FROM knowledge_documents
WHERE source_type = 'file'
  AND content ILIKE '%knaimero@gmail.com%';

-- Backfill ficticio para los tenants afectados.
-- Cada INSERT setea app.tenant_id para satisfacer RLS.
DO $$
DECLARE
  t RECORD;
BEGIN
  FOR t IN SELECT tenant_id FROM _affected_tenants LOOP
    PERFORM set_config('app.tenant_id', t.tenant_id::text, true);

    INSERT INTO knowledge_documents (tenant_id, category, title, content, is_active, source_type, status)
    SELECT t.tenant_id, x.category, x.title, x.content, TRUE, 'text', 'ready'
    FROM (VALUES
      ('team',     'Sobre la clinica',
        'Clinica Dental Sonrisa es una clinica familiar especializada en odontologia general y estetica. Atendemos pacientes de todas las edades con tecnologia moderna y un equipo profesional. Email: contacto@clinicasonrisa.demo - Telefono: +1-809-555-0100'),
      ('location', 'Direccion y horarios',
        'Direccion: Av. Ficticia 123, Santo Domingo, Republica Dominicana. Horarios: lunes a viernes de 9:00 a 18:00, sabados de 9:00 a 13:00. Cerrado domingos y feriados.'),
      ('services', 'Servicios disponibles',
        'Limpieza dental, blanqueamiento, ortodoncia, endodoncia, extracciones, implantes, protesis, odontopediatria, cirugia oral. Consulta inicial gratuita para nuevos pacientes.'),
      ('pricing',  'Precios referenciales',
        'Limpieza dental: USD 50. Blanqueamiento: USD 200. Consulta inicial: gratuita. Endodoncia: desde USD 300. Implante unitario: desde USD 800. Los precios finales pueden variar segun el caso clinico.'),
      ('faq',      'Preguntas frecuentes',
        'Aceptamos efectivo, tarjeta de credito/debito y transferencia. Para cancelar o reagendar una cita avisanos con al menos 24 horas de anticipacion. Atendemos urgencias el mismo dia llamando al telefono de la clinica. No requerimos referencia medica para la primera consulta.')
    ) AS x(category, title, content)
    WHERE NOT EXISTS (
      SELECT 1 FROM knowledge_documents kd
      WHERE kd.tenant_id = t.tenant_id AND kd.title = x.title
    );
  END LOOP;
END $$;
