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
	{Name: "Nuevo Paciente", Position: 0, Color: "#6366f1"},
	{Name: "Consulta Inicial", Position: 1, Color: "#8b5cf6"},
	{Name: "Plan de Tratamiento", Position: 2, Color: "#a855f7"},
	{Name: "En Tratamiento", Position: 3, Color: "#f59e0b"},
	{Name: "Seguimiento", Position: 4, Color: "#10b981"},
	{Name: "Completado", Position: 5, Color: "#22c55e"},
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
		Name:        "Recordatorio de recall (6 meses)",
		Description: "Enviar WhatsApp cuando el paciente no ha visitado en 6 meses",
		TriggerType: "temporal",
		TemplateKey: "dental_recall_6m",
		ActionsJSON: `[{"type":"send_whatsapp","template":"recall_reminder","params":{"interval_days":180}}]`,
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
		Name:         "Paciente no asistió",
		Description:  "Crear tarea de contacto cuando un paciente no asiste a su cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.no_show",
		TemplateKey:  "dental_no_show",
		ActionsJSON:  `[{"type":"create_task","template":"no_show_followup","params":{"due_days":1}}]`,
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
