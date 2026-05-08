package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type ruleRepository struct {
	db *pgxpool.Pool
}

func NewRuleRepository(db *pgxpool.Pool) domain.RuleRepository {
	return &ruleRepository{db: db}
}

const ruleColumns = `
	id, tenant_id, name, description, trigger_type, trigger_event, trigger_schedule,
	conditions, actions, is_active, is_template, template_key,
	cooldown_hours, priority, created_at, updated_at
`

func scanRule(row pgx.Row, r *domain.Rule) error {
	var (
		description         *string
		triggerEvent        *string
		templateKey         *string
		triggerScheduleJSON []byte
		conditionsJSON      []byte
		actionsJSON         []byte
	)

	err := row.Scan(
		&r.ID, &r.TenantID, &r.Name, &description,
		&r.TriggerType, &triggerEvent, &triggerScheduleJSON,
		&conditionsJSON, &actionsJSON,
		&r.IsActive, &r.IsTemplate, &templateKey,
		&r.CooldownHours, &r.Priority, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if description != nil {
		r.Description = *description
	}
	if triggerEvent != nil {
		r.TriggerEvent = *triggerEvent
	}
	if templateKey != nil {
		r.TemplateKey = *templateKey
	}

	if len(triggerScheduleJSON) > 0 && string(triggerScheduleJSON) != "null" {
		r.TriggerSchedule = &domain.RuleTriggerSchedule{}
		if err := json.Unmarshal(triggerScheduleJSON, r.TriggerSchedule); err != nil {
			return fmt.Errorf("scanRule: unmarshal trigger_schedule: %w", err)
		}
	}
	if len(conditionsJSON) > 0 {
		if err := json.Unmarshal(conditionsJSON, &r.Conditions); err != nil {
			return fmt.Errorf("scanRule: unmarshal conditions: %w", err)
		}
	}
	if len(actionsJSON) > 0 {
		if err := json.Unmarshal(actionsJSON, &r.Actions); err != nil {
			return fmt.Errorf("scanRule: unmarshal actions: %w", err)
		}
	}
	return nil
}

func (r *ruleRepository) Create(ctx context.Context, rule *domain.Rule) error {
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Create: marshal conditions: %w", err)
	}
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Create: marshal actions: %w", err)
	}
	var triggerScheduleJSON []byte
	if rule.TriggerSchedule != nil {
		triggerScheduleJSON, err = json.Marshal(rule.TriggerSchedule)
		if err != nil {
			return fmt.Errorf("ruleRepository.Create: marshal trigger_schedule: %w", err)
		}
	}

	return withTenant(ctx, r.db, rule.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO rules
				(id, tenant_id, name, description, trigger_type, trigger_event, trigger_schedule,
				 conditions, actions, is_active, is_template, template_key,
				 cooldown_hours, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		`, rule.ID, rule.TenantID, rule.Name, rule.Description,
			rule.TriggerType, rule.TriggerEvent, triggerScheduleJSON,
			conditionsJSON, actionsJSON,
			rule.IsActive, rule.IsTemplate, rule.TemplateKey,
			rule.CooldownHours, rule.Priority)
		if err != nil {
			return fmt.Errorf("ruleRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *ruleRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	var result []*domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1
			ORDER BY priority DESC, name ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("ruleRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.List: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}

func (r *ruleRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Rule, error) {
	var rule *domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rule = &domain.Rule{}
		err := scanRule(tx.QueryRow(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), rule)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("ruleRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *ruleRepository) Update(ctx context.Context, rule *domain.Rule) error {
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Update: marshal conditions: %w", err)
	}
	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return fmt.Errorf("ruleRepository.Update: marshal actions: %w", err)
	}
	var triggerScheduleJSON []byte
	if rule.TriggerSchedule != nil {
		triggerScheduleJSON, err = json.Marshal(rule.TriggerSchedule)
		if err != nil {
			return fmt.Errorf("ruleRepository.Update: marshal trigger_schedule: %w", err)
		}
	}

	return withTenant(ctx, r.db, rule.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE rules
			SET name = $3, description = $4, trigger_type = $5, trigger_event = $6,
			    trigger_schedule = $7, conditions = $8, actions = $9,
			    is_active = $10, cooldown_hours = $11, priority = $12, updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, rule.TenantID, rule.ID, rule.Name, rule.Description,
			rule.TriggerType, rule.TriggerEvent, triggerScheduleJSON,
			conditionsJSON, actionsJSON,
			rule.IsActive, rule.CooldownHours, rule.Priority)
		if err != nil {
			return fmt.Errorf("ruleRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *ruleRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM rules
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("ruleRepository.Delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *ruleRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	var result []*domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE tenant_id = $1 AND is_active = TRUE AND is_template = FALSE
			ORDER BY priority DESC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("ruleRepository.ListActive: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.ListActive: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}

// ListActiveByTriggerEvent retorna reglas activas que se disparan por un evento específico.
func (r *ruleRepository) ListActiveByTriggerEvent(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*domain.Rule, error) {
	var result []*domain.Rule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+` FROM rules
			WHERE tenant_id = $1 AND is_active = TRUE AND is_template = FALSE
			  AND trigger_type = 'event' AND trigger_event = $2
			ORDER BY priority DESC
		`, tenantID, triggerEvent)
		if err != nil {
			return fmt.Errorf("ruleRepository.ListActiveByTriggerEvent: query: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.ListActiveByTriggerEvent: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}

// ListActiveTemporal retorna todas las reglas temporales activas (cross-tenant, para cron).
func (r *ruleRepository) ListActiveTemporal(ctx context.Context) ([]*domain.Rule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+ruleColumns+` FROM rules
		WHERE is_active = TRUE AND is_template = FALSE AND trigger_type = 'temporal'
		ORDER BY tenant_id, priority DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("ruleRepository.ListActiveTemporal: query: %w", err)
	}
	defer rows.Close()
	var result []*domain.Rule
	for rows.Next() {
		rule := &domain.Rule{}
		if err := scanRule(rows, rule); err != nil {
			return nil, fmt.Errorf("ruleRepository.ListActiveTemporal: scan: %w", err)
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (r *ruleRepository) ListTemplates(ctx context.Context) ([]*domain.Rule, error) {
	var result []*domain.Rule
	// Templates usan tenant_id nil — configurar RLS context para acceder
	templateTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	err := withTenant(ctx, r.db, templateTenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+ruleColumns+`
			FROM rules
			WHERE is_template = TRUE
			ORDER BY priority DESC, name ASC
		`)
		if err != nil {
			return fmt.Errorf("ruleRepository.ListTemplates: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule := &domain.Rule{}
			if err := scanRule(rows, rule); err != nil {
				return fmt.Errorf("ruleRepository.ListTemplates: scan: %w", err)
			}
			result = append(result, rule)
		}
		return rows.Err()
	})
	return result, err
}
