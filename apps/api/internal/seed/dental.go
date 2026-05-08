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
		Name:        "Recall semestral",
		Description: "Enviar WhatsApp cuando el paciente no ha visitado en 6 meses",
		TriggerType: "temporal",
		TemplateKey: "dental_recall_6m",
		ActionsJSON: `[{"type":"send_whatsapp","template":"recall_reminder","params":{"interval_days":180}},{"type":"create_task","template":"contact_for_recall","params":{"due_days":1}}]`,
	},
	{
		Name:         "Post-extracción 48h",
		Description:  "Seguimiento post-operatorio a las 48 horas de una extracción",
		TriggerType:  "temporal",
		TemplateKey:  "dental_post_extraction_48h",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"post_extraction_48h","params":{"interval_hours":48}}]`,
	},
	{
		Name:         "Post-endodoncia 7d",
		Description:  "Seguimiento una semana después de endodoncia",
		TriggerType:  "temporal",
		TemplateKey:  "dental_post_endodoncia_7d",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"post_endodoncia_7d","params":{"interval_days":7}},{"type":"create_task","template":"verify_evolution","params":{"due_days":7}}]`,
	},
	{
		Name:         "Follow-up presupuesto",
		Description:  "Seguimiento 7 días después de enviar presupuesto",
		TriggerType:  "temporal",
		TemplateKey:  "dental_followup_budget",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"followup_budget","params":{"interval_days":7}},{"type":"create_task","template":"followup_budget","params":{"due_days":7}}]`,
	},
	{
		Name:         "Bienvenida nuevo paciente",
		Description:  "Mensaje de bienvenida cuando un paciente nuevo llega por WhatsApp",
		TriggerType:  "event",
		TriggerEvent: "customer.created",
		TemplateKey:  "dental_welcome",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"welcome_new_patient","params":{}}]`,
	},
	{
		Name:         "Paciente no asistió",
		Description:  "Seguimiento cuando un paciente no asiste a su cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.no_show",
		TemplateKey:  "dental_no_show",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"no_show_followup","params":{}},{"type":"create_task","template":"reschedule","params":{"due_days":1}}]`,
	},
	{
		Name:         "Retiro de puntos",
		Description:  "Recordatorio para agendar retiro de puntos 7 días después de cirugía",
		TriggerType:  "temporal",
		TemplateKey:  "dental_suture_removal",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"suture_removal_reminder","params":{"interval_days":7}},{"type":"create_task","template":"schedule_suture_removal","params":{"due_days":7}}]`,
	},
	{
		Name:         "Seguimiento post-tratamiento",
		Description:  "Crear tarea de seguimiento cuando un tratamiento se completa",
		TriggerType:  "event",
		TriggerEvent: "treatment.completed",
		TemplateKey:  "dental_post_treatment",
		ActionsJSON:  `[{"type":"create_task","template":"post_treatment_followup","params":{"due_days":7}}]`,
	},
	{
		Name:         "Inactividad 12 meses",
		Description:  "Mover a etapa Inactivo cuando el paciente no visita en 12 meses",
		TriggerType:  "temporal",
		TemplateKey:  "dental_inactivity_12m",
		ActionsJSON:  `[{"type":"move_stage","template":"move_to_inactive","params":{"interval_days":365}},{"type":"create_task","template":"reactivate_patient","params":{"due_days":1}}]`,
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
