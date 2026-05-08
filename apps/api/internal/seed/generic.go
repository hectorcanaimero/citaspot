package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Templates de reglas globales que aplican a cualquier vertical (no solo dental).
// Se insertan bajo el tenant 00000000-0000-0000-0000-000000000000 (templates globales).
var DefaultGenericRuleTemplates = []struct {
	Name         string
	Description  string
	TriggerType  string
	TriggerEvent string
	ActionsJSON  string
	TemplateKey  string
}{
	{
		Name:         "Seguimiento post-cita",
		Description:  "Mensaje de agradecimiento al completar una cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.completed",
		TemplateKey:  "generic_post_appointment_24h",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Gracias por venir hoy a {{tenant_name}}. Como te fue? Responde del 1 al 5 para calificarnos. Para agendar tu proxima cita responde CITA.","params":{}}]`,
	},
}

// SeedGenericRuleTemplates inserta templates de reglas vertical-agnosticas.
// Idempotente via EXISTS por template_key. Vertical-agnostico = corre para todos los tenants.
func SeedGenericRuleTemplates(ctx context.Context, pool *pgxpool.Pool) error {
	templateTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed.SeedGenericRuleTemplates: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", templateTenantID.String()); err != nil {
		return fmt.Errorf("seed.SeedGenericRuleTemplates: set_config: %w", err)
	}

	for _, r := range DefaultGenericRuleTemplates {
		var exists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM rules WHERE is_template = TRUE AND template_key = $1)
		`, r.TemplateKey).Scan(&exists)
		if err != nil {
			return fmt.Errorf("seed.SeedGenericRuleTemplates: check '%s': %w", r.TemplateKey, err)
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
			return fmt.Errorf("seed.SeedGenericRuleTemplates: insert '%s': %w", r.TemplateKey, err)
		}
	}

	return tx.Commit(ctx)
}
