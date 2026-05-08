-- Actualizar templates de reglas dentales con texto real de WhatsApp.
-- Solo afecta reglas globales (tenant template UUID).

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Ya pasaron 6 meses desde tu ultima visita con nosotros. Te recomendamos agendar una cita de control para mantener tu salud dental al dia. Responde SI para que te agendemos.","params":{"interval_days":180}},{"type":"create_task","template":"Contactar a {{customer_name}} para recall semestral","params":{"due_days":1}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_recall_6m';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, como te sentis despues de la extraccion? Es normal algo de molestia las primeras 48 horas. Recorda: no escupir, no usar sorbete, y aplicar hielo por fuera 15 min cada hora. Si tenes sangrado que no para o dolor fuerte, escribinos.","params":{"interval_hours":48}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_post_extraction_48h';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya paso una semana desde tu endodoncia. Como te sentis? Recorda que es importante colocar la corona definitiva lo antes posible para proteger el diente. Queres que te agendemos? Responde SI.","params":{"interval_days":7}},{"type":"create_task","template":"Verificar evolucion post-endodoncia de {{customer_name}}","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_post_endodoncia_7d';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Te escribimos para saber si pudiste revisar el presupuesto que te enviamos para {{service_name}}. Tenes alguna duda? Con gusto te ayudamos. Tambien podemos ver opciones de financiamiento si te interesa.","params":{"interval_days":7}},{"type":"create_task","template":"Seguimiento presupuesto de {{customer_name}} para {{service_name}}","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_followup_budget';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Bienvenido/a {{customer_name}}! Gracias por contactarnos. Somos {{tenant_name}} y estamos para ayudarte con tu salud dental. Podes agendar tu cita escribiendo la fecha y hora que te quede mejor, o responde CITA para ver horarios disponibles.","params":{}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_welcome';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, notamos que no pudiste asistir a tu cita de {{service_name}}. Esperamos que todo este bien! Queres que te reagendemos? Responde SI y te buscamos un horario que te quede comodo.","params":{}},{"type":"create_task","template":"Reagendar cita de {{customer_name}} (no show)","params":{"due_days":1}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_no_show';

UPDATE rules SET actions = '[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya se cumplen 7 dias desde tu cirugia y es momento de retirar los puntos. Es un procedimiento rapido y sin dolor. Responde SI para agendar tu cita de retiro de puntos.","params":{"interval_days":7}},{"type":"create_task","template":"Agendar retiro de puntos de {{customer_name}}","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_suture_removal';

UPDATE rules SET actions = '[{"type":"create_task","template":"Seguimiento post-tratamiento de {{customer_name}} - verificar satisfaccion y agendar control","params":{"due_days":7}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_post_treatment';

UPDATE rules SET actions = '[{"type":"move_stage","template":"move_to_inactive","params":{"interval_days":365}},{"type":"create_task","template":"Reactivar paciente {{customer_name}} - 12 meses sin visita","params":{"due_days":1}}]'::jsonb, updated_at = NOW()
WHERE is_template = TRUE AND template_key = 'dental_inactivity_12m';
