package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type crmMetricsRepository struct {
	db *pgxpool.Pool
}

// NewCRMMetricsRepository crea un repositorio de metricas CRM.
func NewCRMMetricsRepository(db *pgxpool.Pool) domain.CRMMetricsRepository {
	return &crmMetricsRepository{db: db}
}

func (r *crmMetricsRepository) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*domain.CRMMetrics, error) {
	m := &domain.CRMMetrics{}

	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// 1. Rules fired en los ultimos 30 dias + success rate
		err := tx.QueryRow(ctx, `
			SELECT
				COUNT(*) AS total,
				COALESCE(
					COUNT(*) FILTER (WHERE status = 'success')::float /
					NULLIF(COUNT(*), 0),
					0
				) AS success_rate
			FROM rule_executions
			WHERE tenant_id = $1
			  AND triggered_at > NOW() - INTERVAL '30 days'
		`, tenantID).Scan(&m.RulesFired30d, &m.RulesSuccessRate)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: rules_fired: %w", err)
		}

		// 2. Tratamientos activos
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM treatments
			WHERE tenant_id = $1 AND status = 'in_progress'
		`, tenantID).Scan(&m.ActiveTreatments)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: active_treatments: %w", err)
		}

		// 3. Tareas pendientes
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM tasks
			WHERE tenant_id = $1 AND status = 'pending'
		`, tenantID).Scan(&m.PendingTasks)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: pending_tasks: %w", err)
		}

		// 4. Clientes por etapa del pipeline
		rows, err := tx.Query(ctx, `
			SELECT ps.id, ps.name, ps.color, COUNT(c.id) AS cnt
			FROM pipeline_stages ps
			LEFT JOIN customers c ON c.stage_id = ps.id AND c.tenant_id = ps.tenant_id
			WHERE ps.tenant_id = $1
			GROUP BY ps.id, ps.name, ps.color, ps.position
			ORDER BY ps.position ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: customers_per_stage: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var sc domain.StageCustomerCount
			if err := rows.Scan(&sc.StageID, &sc.StageName, &sc.Color, &sc.Count); err != nil {
				return fmt.Errorf("crmMetricsRepository.GetMetrics: scan stage: %w", err)
			}
			m.CustomersPerStage = append(m.CustomersPerStage, sc)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: stage rows: %w", err)
		}

		// 5. Top 5 reglas mas ejecutadas en 30 dias
		topRows, err := tx.Query(ctx, `
			SELECT re.rule_id, r.name, COUNT(*) AS cnt
			FROM rule_executions re
			JOIN rules r ON r.id = re.rule_id AND r.tenant_id = re.tenant_id
			WHERE re.tenant_id = $1
			  AND re.triggered_at > NOW() - INTERVAL '30 days'
			GROUP BY re.rule_id, r.name
			ORDER BY cnt DESC
			LIMIT 5
		`, tenantID)
		if err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: top_rules: %w", err)
		}
		defer topRows.Close()

		for topRows.Next() {
			var tr domain.TopRuleMetric
			if err := topRows.Scan(&tr.RuleID, &tr.RuleName, &tr.Executions); err != nil {
				return fmt.Errorf("crmMetricsRepository.GetMetrics: scan top: %w", err)
			}
			m.TopRules = append(m.TopRules, tr)
		}
		if err := topRows.Err(); err != nil {
			return fmt.Errorf("crmMetricsRepository.GetMetrics: top rows: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Garantizar slices no-nil para JSON
	if m.CustomersPerStage == nil {
		m.CustomersPerStage = []domain.StageCustomerCount{}
	}
	if m.TopRules == nil {
		m.TopRules = []domain.TopRuleMetric{}
	}

	return m, nil
}
