package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type RuleExecutor struct {
	ruleRepo domain.RuleRepository
	execRepo domain.RuleExecutionRepository
	authRepo domain.AuthRepository
	registry *ActionRegistry
}

func NewRuleExecutor(
	ruleRepo domain.RuleRepository,
	execRepo domain.RuleExecutionRepository,
	authRepo domain.AuthRepository,
	registry *ActionRegistry,
) *RuleExecutor {
	return &RuleExecutor{
		ruleRepo: ruleRepo,
		execRepo: execRepo,
		authRepo: authRepo,
		registry: registry,
	}
}

func (e *RuleExecutor) ExecuteEventRules(ctx context.Context, event domain.RuleEvent) {
	rules, err := e.ruleRepo.ListActiveByTriggerEvent(ctx, event.TenantID, event.EventType)
	if err != nil {
		slog.Error("RuleExecutor.ExecuteEventRules: error listando reglas",
			"tenant_id", event.TenantID, "event", event.EventType, "error", err)
		return
	}
	if len(rules) == 0 {
		return
	}

	tenantSlug := e.resolveTenantSlug(ctx, event.TenantID)
	evalCtx := e.buildEventContext(event)

	slog.Info("RuleExecutor: evaluando reglas para evento",
		"tenant_id", event.TenantID, "event", event.EventType, "rules_count", len(rules))

	for _, rule := range rules {
		e.executeRule(ctx, rule, event.CustomerID, event.EntityID, event.EntityType, tenantSlug, evalCtx)
	}
}

func (e *RuleExecutor) ExecuteTemporalRule(ctx context.Context, rule *domain.Rule, customerID uuid.UUID, evalCtx map[string]any) {
	tenantSlug := e.resolveTenantSlug(ctx, rule.TenantID)
	e.executeRule(ctx, rule, customerID, customerID, "customer", tenantSlug, evalCtx)
}

func (e *RuleExecutor) executeRule(
	ctx context.Context,
	rule *domain.Rule,
	customerID, entityID uuid.UUID,
	entityType, tenantSlug string,
	evalCtx map[string]any,
) {
	if !EvaluateConditions(rule.Conditions, evalCtx) {
		slog.Debug("RuleExecutor: condiciones no cumplidas", "rule_id", rule.ID, "rule_name", rule.Name)
		return
	}

	if rule.CooldownHours > 0 && customerID != uuid.Nil {
		recent, err := e.execRepo.HasRecentExecution(ctx, rule.TenantID, rule.ID, customerID, rule.CooldownHours)
		if err != nil {
			slog.Error("RuleExecutor: error verificando cooldown", "rule_id", rule.ID, "error", err)
			return
		}
		if recent {
			slog.Debug("RuleExecutor: cooldown activo", "rule_id", rule.ID, "customer_id", customerID)
			return
		}
	}

	var execErr error
	for _, action := range rule.Actions {
		executor := e.registry.Get(action.Type)
		if executor == nil {
			slog.Warn("RuleExecutor: tipo de accion no registrado", "action_type", action.Type, "rule_id", rule.ID)
			continue
		}

		actionParams := action.Params
		if actionParams == nil {
			actionParams = make(map[string]any)
		}
		actionParams["rule_id"] = rule.ID.String()

		params := ActionParams{
			TenantID:   rule.TenantID,
			TenantSlug: tenantSlug,
			CustomerID: customerID,
			EntityID:   entityID,
			EntityType: entityType,
			Template:   action.Template,
			Params:     actionParams,
			Context:    evalCtx,
		}

		if err := executor.Execute(ctx, params); err != nil {
			slog.Error("RuleExecutor: error ejecutando accion",
				"action_type", action.Type, "rule_id", rule.ID, "error", err)
			execErr = err
		}
	}

	execution := &domain.RuleExecution{
		ID:                 uuid.New(),
		TenantID:           rule.TenantID,
		RuleID:             rule.ID,
		CustomerID:         &customerID,
		TriggeredAt:        time.Now(),
		TriggerEvent:       rule.TriggerEvent,
		ConditionsSnapshot: rule.Conditions,
		ActionsResult:      rule.Actions,
		Status:             "success",
	}

	if execErr != nil {
		execution.Status = "error"
		execution.ErrorMessage = execErr.Error()
	}

	if err := e.execRepo.Create(ctx, execution); err != nil {
		slog.Error("RuleExecutor: error registrando ejecucion", "rule_id", rule.ID, "error", err)
	}

	slog.Info("RuleExecutor: regla ejecutada",
		"rule_id", rule.ID, "rule_name", rule.Name, "status", execution.Status, "customer_id", customerID)
}

func (e *RuleExecutor) buildEventContext(event domain.RuleEvent) map[string]any {
	ctx := map[string]any{
		"event_type":  event.EventType,
		"tenant_id":   event.TenantID.String(),
		"customer_id": event.CustomerID.String(),
		"entity_id":   event.EntityID.String(),
		"entity_type": event.EntityType,
	}
	for k, v := range event.Payload {
		ctx[k] = v
	}
	return ctx
}

func (e *RuleExecutor) resolveTenantSlug(ctx context.Context, tenantID uuid.UUID) string {
	tenant, err := e.authRepo.FindTenantByID(ctx, tenantID)
	if err != nil {
		slog.Error("RuleExecutor: error resolviendo tenant slug", "tenant_id", tenantID, "error", err)
		return ""
	}
	return tenant.Slug
}
