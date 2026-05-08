package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DefaultDentalPipelineStages = []struct {
	Name     string
	Position int
	Color    string
}{
	{Name: "Nuevo contacto", Position: 0, Color: "#6366f1"},
	{Name: "Primera consulta agendada", Position: 1, Color: "#8b5cf6"},
	{Name: "Presupuesto enviado", Position: 2, Color: "#a855f7"},
	{Name: "Presupuesto aceptado", Position: 3, Color: "#f59e0b"},
	{Name: "En tratamiento", Position: 4, Color: "#10b981"},
	{Name: "Mantenimiento", Position: 5, Color: "#22c55e"},
	{Name: "Inactivo", Position: 6, Color: "#6b7280"},
}

func SeedDentalPipeline(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed.SeedDentalPipeline: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("seed.SeedDentalPipeline: set_config: %w", err)
	}

	for i, s := range DefaultDentalPipelineStages {
		isDefault := i == 0
		_, err := tx.Exec(ctx, `
			INSERT INTO pipeline_stages (id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW())
			ON CONFLICT (tenant_id, position) DO NOTHING
		`, uuid.New(), tenantID, s.Name, s.Position, s.Color, isDefault)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalPipeline: stage '%s': %w", s.Name, err)
		}
	}

	return tx.Commit(ctx)
}

var DefaultDentalRuleTemplates = []struct {
	Name         string
	Description  string
	TriggerType  string
	TriggerEvent string
	ActionsJSON  string
	TemplateKey  string
}{
	{
		Name:        "Recall trimestral",
		Description: "Enviar WhatsApp para reactivar paciente inactivo hace 3 meses",
		TriggerType: "temporal",
		TemplateKey: "dental_recall_3m",
		ActionsJSON: `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Vimos que hace un tiempo que no nos visitas. Para mantener tu salud bucal y prevenir problemas mas serios, te recomendamos agendar un control. Responde SI y te buscamos un horario que te quede comodo.","params":{"interval_days":90}},{"type":"create_task","template":"Contactar a {{customer_name}} para reactivacion 3m","params":{"due_days":1}}]`,
	},
	{
		Name:        "Recall semestral",
		Description: "Enviar WhatsApp cuando el paciente no ha visitado en 6 meses",
		TriggerType: "temporal",
		TemplateKey: "dental_recall_6m",
		ActionsJSON: `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Ya pasaron 6 meses desde tu ultima visita con nosotros. Te recomendamos agendar una cita de control para mantener tu salud dental al dia. Responde SI para que te agendemos.","params":{"interval_days":180}},{"type":"create_task","template":"Contactar a {{customer_name}} para recall semestral","params":{"due_days":1}}]`,
	},
	{
		Name:         "Post-extraccion 48h",
		Description:  "Seguimiento post-operatorio a las 48 horas de una extraccion",
		TriggerType:  "temporal",
		TemplateKey:  "dental_post_extraction_48h",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, como te sentis despues de la extraccion? Es normal algo de molestia las primeras 48 horas. Recorda: no escupir, no usar sorbete, y aplicar hielo por fuera 15 min cada hora. Si tenes sangrado que no para o dolor fuerte, escribinos.","params":{"interval_hours":48}}]`,
	},
	{
		Name:         "Post-endodoncia 7d",
		Description:  "Seguimiento una semana despues de endodoncia",
		TriggerType:  "temporal",
		TemplateKey:  "dental_post_endodoncia_7d",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya paso una semana desde tu endodoncia. Como te sentis? Recorda que es importante colocar la corona definitiva lo antes posible para proteger el diente. Queres que te agendemos? Responde SI.","params":{"interval_days":7}},{"type":"create_task","template":"Verificar evolucion post-endodoncia de {{customer_name}}","params":{"due_days":7}}]`,
	},
	{
		Name:         "Follow-up presupuesto",
		Description:  "Seguimiento 7 dias despues de enviar presupuesto",
		TriggerType:  "temporal",
		TemplateKey:  "dental_followup_budget",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Te escribimos para saber si pudiste revisar el presupuesto que te enviamos para {{service_name}}. Tenes alguna duda? Con gusto te ayudamos. Tambien podemos ver opciones de financiamiento si te interesa.","params":{"interval_days":7}},{"type":"create_task","template":"Seguimiento presupuesto de {{customer_name}} para {{service_name}}","params":{"due_days":7}}]`,
	},
	{
		Name:         "Bienvenida nuevo paciente",
		Description:  "Mensaje de bienvenida cuando un paciente nuevo llega por WhatsApp",
		TriggerType:  "event",
		TriggerEvent: "customer.created",
		TemplateKey:  "dental_welcome",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Bienvenido/a {{customer_name}}! Gracias por contactarnos. Somos {{tenant_name}} y estamos para ayudarte con tu salud dental. Podes agendar tu cita escribiendo la fecha y hora que te quede mejor, o responde CITA para ver horarios disponibles.","params":{}}]`,
	},
	{
		Name:         "Paciente no asistio",
		Description:  "Seguimiento cuando un paciente no asiste a su cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.no_show",
		TemplateKey:  "dental_no_show",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, notamos que no pudiste asistir a tu cita de {{service_name}}. Esperamos que todo este bien! Queres que te reagendemos? Responde SI y te buscamos un horario que te quede comodo.","params":{}},{"type":"create_task","template":"Reagendar cita de {{customer_name}} (no show)","params":{"due_days":1}}]`,
	},
	{
		Name:         "Retiro de puntos",
		Description:  "Recordatorio para agendar retiro de puntos 7 dias despues de cirugia",
		TriggerType:  "temporal",
		TemplateKey:  "dental_suture_removal",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, ya se cumplen 7 dias desde tu cirugia y es momento de retirar los puntos. Es un procedimiento rapido y sin dolor. Responde SI para agendar tu cita de retiro de puntos.","params":{"interval_days":7}},{"type":"create_task","template":"Agendar retiro de puntos de {{customer_name}}","params":{"due_days":7}}]`,
	},
	{
		Name:         "Seguimiento post-tratamiento",
		Description:  "Crear tarea de seguimiento cuando un tratamiento se completa",
		TriggerType:  "event",
		TriggerEvent: "treatment.completed",
		TemplateKey:  "dental_post_treatment",
		ActionsJSON:  `[{"type":"create_task","template":"Seguimiento post-tratamiento de {{customer_name}} - verificar satisfaccion y agendar control","params":{"due_days":7}}]`,
	},
	{
		Name:         "Inactividad 12 meses",
		Description:  "Mover a etapa Inactivo cuando el paciente no visita en 12 meses",
		TriggerType:  "temporal",
		TemplateKey:  "dental_inactivity_12m",
		ActionsJSON:  `[{"type":"move_stage","template":"move_to_inactive","params":{"interval_days":365}},{"type":"create_task","template":"Reactivar paciente {{customer_name}} - 12 meses sin visita","params":{"due_days":1}}]`,
	},
}

func SeedDentalRuleTemplates(ctx context.Context, pool *pgxpool.Pool) error {
	templateTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed.SeedDentalRuleTemplates: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Configurar RLS context para el tenant de templates
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", templateTenantID.String()); err != nil {
		return fmt.Errorf("seed.SeedDentalRuleTemplates: set_config: %w", err)
	}

	for _, r := range DefaultDentalRuleTemplates {
		var exists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM rules WHERE is_template = TRUE AND template_key = $1)
		`, r.TemplateKey).Scan(&exists)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalRuleTemplates: check '%s': %w", r.TemplateKey, err)
		}
		if exists {
			continue
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO rules
				(id, tenant_id, name, description, trigger_type, trigger_event,
				 conditions, actions, is_active, is_template, template_key,
				 cooldown_hours, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, '[]'::jsonb, $7::jsonb, TRUE, TRUE, $8, 24, 0, NOW(), NOW())
		`, uuid.New(), templateTenantID, r.Name, r.Description,
			r.TriggerType, r.TriggerEvent, r.ActionsJSON, r.TemplateKey)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalRuleTemplates: insert '%s': %w", r.TemplateKey, err)
		}
	}

	return tx.Commit(ctx)
}

// Documentos ficticios de Knowledge para clinica dental demo.
// Email/telefono/direccion son ficticios para evitar exponer datos reales en el demo.
var DefaultDentalKnowledgeDocs = []struct {
	Category string
	Title    string
	Content  string
}{
	{
		Category: "team",
		Title:    "Sobre la clinica",
		Content:  "Clinica Dental Sonrisa es una clinica familiar especializada en odontologia general y estetica. Atendemos pacientes de todas las edades con tecnologia moderna y un equipo profesional. Email: contacto@clinicasonrisa.demo - Telefono: +1-809-555-0100",
	},
	{
		Category: "location",
		Title:    "Direccion y horarios",
		Content:  "Direccion: Av. Ficticia 123, Santo Domingo, Republica Dominicana. Horarios: lunes a viernes de 9:00 a 18:00, sabados de 9:00 a 13:00. Cerrado domingos y feriados.",
	},
	{
		Category: "services",
		Title:    "Servicios disponibles",
		Content:  "Limpieza dental, blanqueamiento, ortodoncia, endodoncia, extracciones, implantes, protesis, odontopediatria, cirugia oral. Consulta inicial gratuita para nuevos pacientes.",
	},
	{
		Category: "pricing",
		Title:    "Precios referenciales",
		Content:  "Limpieza dental: USD 50. Blanqueamiento: USD 200. Consulta inicial: gratuita. Endodoncia: desde USD 300. Implante unitario: desde USD 800. Los precios finales pueden variar segun el caso clinico.",
	},
	{
		Category: "faq",
		Title:    "Preguntas frecuentes",
		Content:  "Aceptamos efectivo, tarjeta de credito/debito y transferencia. Para cancelar o reagendar una cita avisanos con al menos 24 horas de anticipacion. Atendemos urgencias el mismo dia llamando al telefono de la clinica. No requerimos referencia medica para la primera consulta.",
	},
}

// SeedDentalKnowledge inserta documentos ficticios de Knowledge para un tenant dental.
// Idempotente: salta documentos ya existentes con mismo title para el tenant.
func SeedDentalKnowledge(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed.SeedDentalKnowledge: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("seed.SeedDentalKnowledge: set_config: %w", err)
	}

	for _, d := range DefaultDentalKnowledgeDocs {
		_, err := tx.Exec(ctx, `
			INSERT INTO knowledge_documents
				(id, tenant_id, category, title, content, is_active, source_type, status, created_at, updated_at)
			SELECT $1, $2, $3, $4, $5, TRUE, 'text', 'ready', NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM knowledge_documents
				WHERE tenant_id = $2 AND title = $4
			)
		`, uuid.New(), tenantID, d.Category, d.Title, d.Content)
		if err != nil {
			return fmt.Errorf("seed.SeedDentalKnowledge: insert '%s': %w", d.Title, err)
		}
	}

	return tx.Commit(ctx)
}
