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
		Name:         "Cita creada — pendiente de aprobación",
		Description:  "Notifica al paciente que su cita fue registrada y está pendiente de confirmación",
		TriggerType:  "event",
		TriggerEvent: "appointment.created",
		TemplateKey:  "generic_appointment_created",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"¡Hola {{customer_name}}! 📋 Tu cita para *{{service_name}}* con {{professional_name}} ha sido registrada y está *pendiente de aprobación*. Te avisaremos cuando sea confirmada.","params":{}}]`,
	},
	{
		Name:         "Cita confirmada",
		Description:  "Notifica al paciente que su cita fue confirmada",
		TriggerType:  "event",
		TriggerEvent: "appointment.confirmed",
		TemplateKey:  "generic_appointment_confirmed",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"¡Hola {{customer_name}}! ✅ Tu cita en *{{tenant_name}}* ha sido *confirmada*:\n\n📌 *{{service_name}}*\n👩‍⚕️ {{professional_name}}\n📅 {{starts_at}}\n\n¡Te esperamos!","params":{}}]`,
	},
	{
		Name:         "Cita cancelada",
		Description:  "Notifica al paciente que su cita fue cancelada",
		TriggerType:  "event",
		TriggerEvent: "appointment.cancelled",
		TemplateKey:  "generic_appointment_cancelled",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}, lamentamos informarte que tu cita de *{{service_name}}* ha sido *cancelada*. Puedes reservar nuevamente cuando lo desees.","params":{}}]`,
	},
	{
		Name:         "Cita reagendada",
		Description:  "Notifica al paciente que su cita fue reagendada",
		TriggerType:  "event",
		TriggerEvent: "appointment.rescheduled",
		TemplateKey:  "generic_appointment_rescheduled",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"¡Hola {{customer_name}}! 🔄 Tu cita en *{{tenant_name}}* ha sido *reagendada*:\n\n📌 *{{service_name}}*\n👩‍⚕️ {{professional_name}}\n📅 {{starts_at}}\n\n¡Te esperamos!","params":{}}]`,
	},
	{
		Name:         "Seguimiento post-cita",
		Description:  "Mensaje de agradecimiento al completar una cita",
		TriggerType:  "event",
		TriggerEvent: "appointment.completed",
		TemplateKey:  "generic_post_appointment_24h",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"Hola {{customer_name}}! Gracias por venir hoy a {{tenant_name}}. Como te fue? Responde del 1 al 5 para calificarnos. Para agendar tu proxima cita responde CITA.","params":{}}]`,
	},
	// Reglas para notificar al PROFESIONAL — mismos templates que migración 040.
	// El action resuelve el teléfono vía recipient=professional → context["professional_phone"].
	{
		Name:         "Profesional — nueva reserva pendiente",
		Description:  "Notifica al profesional cuando un cliente reserva una cita (status pending)",
		TriggerType:  "event",
		TriggerEvent: "appointment.created",
		TemplateKey:  "prof_appointment_created",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"📋 *Nueva reserva pendiente*\n\n👤 {{customer_name}}\n📌 {{service_name}}\n📅 {{starts_at}}\n\nConfirma o ajusta desde tu panel.","params":{"recipient":"professional"}}]`,
	},
	{
		Name:         "Profesional — cita confirmada",
		Description:  "Notifica al profesional cuando una cita es confirmada",
		TriggerType:  "event",
		TriggerEvent: "appointment.confirmed",
		TemplateKey:  "prof_appointment_confirmed",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"✅ *Cita confirmada*\n\n👤 {{customer_name}}\n📌 {{service_name}}\n📅 {{starts_at}}\n\nTe esperamos en tu agenda.","params":{"recipient":"professional"}}]`,
	},
	{
		Name:         "Profesional — cita cancelada",
		Description:  "Notifica al profesional cuando una cita es cancelada",
		TriggerType:  "event",
		TriggerEvent: "appointment.cancelled",
		TemplateKey:  "prof_appointment_cancelled",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"❌ *Cita cancelada*\n\n👤 {{customer_name}} canceló su cita.\n📌 {{service_name}}\n📅 {{starts_at}}\n\nEse horario queda libre.","params":{"recipient":"professional"}}]`,
	},
	{
		Name:         "Profesional — cita reagendada",
		Description:  "Notifica al profesional cuando una cita es reagendada",
		TriggerType:  "event",
		TriggerEvent: "appointment.rescheduled",
		TemplateKey:  "prof_appointment_rescheduled",
		ActionsJSON:  `[{"type":"send_whatsapp","template":"🔄 *Cita reagendada*\n\n👤 {{customer_name}}\n📌 {{service_name}}\n📅 Nuevo horario: {{starts_at}}","params":{"recipient":"professional"}}]`,
	},
}

// DefaultAppointmentRuleKeys son los template_keys de reglas de notificación
// de citas que se copian automáticamente a cada tenant nuevo.
var DefaultAppointmentRuleKeys = []string{
	"generic_appointment_created",
	"generic_appointment_confirmed",
	"generic_appointment_cancelled",
	"generic_appointment_rescheduled",
	"prof_appointment_created",
	"prof_appointment_confirmed",
	"prof_appointment_cancelled",
	"prof_appointment_rescheduled",
}

// SeedAppointmentRulesForTenant copia las reglas de notificación de citas
// desde los templates globales al tenant indicado. Idempotente via template_key.
func SeedAppointmentRulesForTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed.SeedAppointmentRulesForTenant: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("seed.SeedAppointmentRulesForTenant: set_config: %w", err)
	}

	for _, tmpl := range DefaultGenericRuleTemplates {
		if !isAppointmentRule(tmpl.TemplateKey) {
			continue
		}

		var exists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM rules WHERE tenant_id = $1 AND template_key = $2)
		`, tenantID, tmpl.TemplateKey).Scan(&exists)
		if err != nil {
			return fmt.Errorf("seed.SeedAppointmentRulesForTenant: check '%s': %w", tmpl.TemplateKey, err)
		}
		if exists {
			continue
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO rules
				(id, tenant_id, name, description, trigger_type, trigger_event,
				 conditions, actions, is_active, is_template, template_key,
				 cooldown_hours, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, '[]'::jsonb, $7::jsonb, TRUE, FALSE, $8, 0, 0, NOW(), NOW())
		`, uuid.New(), tenantID, tmpl.Name, tmpl.Description,
			tmpl.TriggerType, tmpl.TriggerEvent, tmpl.ActionsJSON, tmpl.TemplateKey)
		if err != nil {
			return fmt.Errorf("seed.SeedAppointmentRulesForTenant: insert '%s': %w", tmpl.TemplateKey, err)
		}
	}

	return tx.Commit(ctx)
}

func isAppointmentRule(key string) bool {
	for _, k := range DefaultAppointmentRuleKeys {
		if k == key {
			return true
		}
	}
	return false
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
