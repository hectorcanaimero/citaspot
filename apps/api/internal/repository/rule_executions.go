package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type ruleExecutionRepository struct {
	db *pgxpool.Pool
}

func NewRuleExecutionRepository(db *pgxpool.Pool) domain.RuleExecutionRepository {
	return &ruleExecutionRepository{db: db}
}

func scanRuleExecution(row pgx.Row, e *domain.RuleExecution) error {
	var (
		triggerEvent      *string
		errorMessage      *string
		conditionsJSON    []byte
		actionsResultJSON []byte
	)

	err := row.Scan(
		&e.ID, &e.TenantID, &e.RuleID, &e.CustomerID,
		&e.TriggeredAt, &triggerEvent,
		&conditionsJSON, &actionsResultJSON,
		&e.Status, &errorMessage,
	)
	if err != nil {
		return err
	}
	if triggerEvent != nil {
		e.TriggerEvent = *triggerEvent
	}
	if errorMessage != nil {
		e.ErrorMessage = *errorMessage
	}
	if len(conditionsJSON) > 0 {
		_ = json.Unmarshal(conditionsJSON, &e.ConditionsSnapshot)
	}
	if len(actionsResultJSON) > 0 {
		_ = json.Unmarshal(actionsResultJSON, &e.ActionsResult)
	}
	return nil
}

func (r *ruleExecutionRepository) Create(ctx context.Context, e *domain.RuleExecution) error {
	conditionsJSON, _ := json.Marshal(e.ConditionsSnapshot)
	actionsJSON, _ := json.Marshal(e.ActionsResult)

	return withTenant(ctx, r.db, e.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO rule_executions
				(id, tenant_id, rule_id, customer_id, triggered_at, trigger_event,
				 conditions_snapshot, actions_result, status, error_message)
			VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7, $8, $9)
		`, e.ID, e.TenantID, e.RuleID, e.CustomerID,
			e.TriggerEvent, conditionsJSON, actionsJSON,
			e.Status, e.ErrorMessage)
		if err != nil {
			return fmt.Errorf("ruleExecutionRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *ruleExecutionRepository) ListByRule(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var result []*domain.RuleExecution
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, rule_id, customer_id, triggered_at, trigger_event,
			       conditions_snapshot, actions_result, status, error_message
			FROM rule_executions
			WHERE tenant_id = $1 AND rule_id = $2
			ORDER BY triggered_at DESC
			LIMIT $3
		`, tenantID, ruleID, limit)
		if err != nil {
			return fmt.Errorf("ruleExecutionRepository.ListByRule: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			e := &domain.RuleExecution{}
			if err := scanRuleExecution(rows, e); err != nil {
				return fmt.Errorf("ruleExecutionRepository.ListByRule: scan: %w", err)
			}
			result = append(result, e)
		}
		return rows.Err()
	})
	return result, err
}

func (r *ruleExecutionRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var result []*domain.RuleExecution
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, rule_id, customer_id, triggered_at, trigger_event,
			       conditions_snapshot, actions_result, status, error_message
			FROM rule_executions
			WHERE tenant_id = $1 AND customer_id = $2
			ORDER BY triggered_at DESC
			LIMIT $3
		`, tenantID, customerID, limit)
		if err != nil {
			return fmt.Errorf("ruleExecutionRepository.ListByCustomer: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			e := &domain.RuleExecution{}
			if err := scanRuleExecution(rows, e); err != nil {
				return fmt.Errorf("ruleExecutionRepository.ListByCustomer: scan: %w", err)
			}
			result = append(result, e)
		}
		return rows.Err()
	})
	return result, err
}
