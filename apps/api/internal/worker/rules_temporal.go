package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type TemporalRulesWorker struct {
	ruleRepo     domain.RuleRepository
	customerRepo domain.CustomerRepository
	executor     *engine.RuleExecutor
	interval     time.Duration
}

func NewTemporalRulesWorker(
	ruleRepo domain.RuleRepository,
	customerRepo domain.CustomerRepository,
	executor *engine.RuleExecutor,
) *TemporalRulesWorker {
	return &TemporalRulesWorker{
		ruleRepo:     ruleRepo,
		customerRepo: customerRepo,
		executor:     executor,
		interval:     15 * time.Minute,
	}
}

func (w *TemporalRulesWorker) Start(ctx context.Context) {
	slog.Info("TemporalRulesWorker: iniciado (cada 15 min)")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.run(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("TemporalRulesWorker: detenido")
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *TemporalRulesWorker) run(ctx context.Context) {
	rules, err := w.ruleRepo.ListActiveTemporal(ctx)
	if err != nil {
		slog.Error("TemporalRulesWorker: error listando reglas temporales", "error", err)
		return
	}
	if len(rules) == 0 {
		return
	}

	slog.Info("TemporalRulesWorker: evaluando reglas temporales", "count", len(rules))

	totalExecuted := 0
	for _, rule := range rules {
		n := w.evaluateTemporalRule(ctx, rule)
		totalExecuted += n
	}

	if totalExecuted > 0 {
		slog.Info("TemporalRulesWorker: reglas ejecutadas", "total", totalExecuted)
	}
}

func (w *TemporalRulesWorker) evaluateTemporalRule(ctx context.Context, rule *domain.Rule) int {
	if rule.TriggerSchedule == nil {
		slog.Warn("TemporalRulesWorker: regla temporal sin schedule", "rule_id", rule.ID)
		return 0
	}

	customers := w.findTemporalCandidates(ctx, rule)
	if len(customers) == 0 {
		return 0
	}

	executed := 0
	for _, customer := range customers {
		evalCtx := w.buildTemporalContext(customer, rule)
		w.executor.ExecuteTemporalRule(ctx, rule, customer.ID, evalCtx)
		executed++
	}
	return executed
}

func (w *TemporalRulesWorker) findTemporalCandidates(ctx context.Context, rule *domain.Rule) []*domain.Customer {
	sched := rule.TriggerSchedule

	customers, err := w.customerRepo.List(ctx, rule.TenantID, "", 500, 0)
	if err != nil {
		slog.Error("TemporalRulesWorker: error listando clientes", "tenant_id", rule.TenantID, "error", err)
		return nil
	}

	cutoff := time.Now().Add(-time.Duration(sched.IntervalDays) * 24 * time.Hour)
	var candidates []*domain.Customer

	for _, c := range customers {
		if w.matchesTemporalCondition(c, sched.ReferenceField, cutoff) {
			candidates = append(candidates, c)
		}
	}
	return candidates
}

func (w *TemporalRulesWorker) matchesTemporalCondition(customer *domain.Customer, referenceField string, cutoff time.Time) bool {
	switch referenceField {
	case "last_visit_at":
		if customer.LastVisitAt == nil {
			return false
		}
		return customer.LastVisitAt.Before(cutoff)
	case "next_recall_at":
		if customer.NextRecallAt == nil {
			return false
		}
		return customer.NextRecallAt.Before(time.Now())
	case "created_at":
		return customer.CreatedAt.Before(cutoff)
	default:
		slog.Warn("TemporalRulesWorker: reference_field desconocido", "field", referenceField)
		return false
	}
}

func (w *TemporalRulesWorker) buildTemporalContext(customer *domain.Customer, rule *domain.Rule) map[string]any {
	ctx := map[string]any{
		"event_type":     "temporal",
		"tenant_id":      rule.TenantID.String(),
		"customer_id":    customer.ID.String(),
		"customer_name":  customer.Name,
		"customer_phone": customer.Phone,
		"customer": map[string]any{
			"name":           customer.Name,
			"phone":          customer.Phone,
			"total_visits":   customer.TotalVisits,
			"lifetime_value": customer.LifetimeValue,
		},
	}

	if customer.LastVisitAt != nil {
		ctx["customer"].(map[string]any)["last_visit_at"] = customer.LastVisitAt.Format(time.RFC3339)
		daysSince := int(time.Since(*customer.LastVisitAt).Hours() / 24)
		ctx["customer"].(map[string]any)["days_since_last_visit"] = daysSince
		ctx["days_since_last_visit"] = daysSince
	}
	if customer.NextRecallAt != nil {
		ctx["customer"].(map[string]any)["next_recall_at"] = customer.NextRecallAt.Format(time.RFC3339)
	}
	if customer.StageID != nil {
		ctx["customer"].(map[string]any)["stage_id"] = customer.StageID.String()
		ctx["stage_id"] = customer.StageID.String()
	} else {
		ctx["stage_id"] = uuid.Nil.String()
	}

	return ctx
}
