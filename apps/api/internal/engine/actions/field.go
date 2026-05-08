package actions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type UpdateFieldAction struct {
	customerRepo domain.CustomerRepository
}

func NewUpdateFieldAction(customerRepo domain.CustomerRepository) *UpdateFieldAction {
	return &UpdateFieldAction{customerRepo: customerRepo}
}

func (a *UpdateFieldAction) Type() string { return "update_field" }

func (a *UpdateFieldAction) Execute(ctx context.Context, params engine.ActionParams) error {
	field, ok := params.Params["field"].(string)
	if !ok || field == "" {
		return fmt.Errorf("UpdateFieldAction.Execute: 'field' no especificado en params")
	}

	value, ok := params.Params["value"]
	if !ok {
		return fmt.Errorf("UpdateFieldAction.Execute: 'value' no especificado en params")
	}

	if field == "next_recall_at" || field == "last_visit_at" {
		value = resolveTimeValue(value)
	}

	if err := a.customerRepo.UpdateField(ctx, params.TenantID, params.CustomerID, field, value); err != nil {
		return fmt.Errorf("UpdateFieldAction.Execute: %w", err)
	}

	slog.Info("UpdateFieldAction: campo actualizado", "customer_id", params.CustomerID, "field", field, "tenant", params.TenantID)
	return nil
}

func resolveTimeValue(value any) any {
	switch v := value.(type) {
	case string:
		if v == "now" {
			return time.Now()
		}
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t
		}
		return v
	case float64:
		return time.Now().Add(time.Duration(v) * 24 * time.Hour)
	case int:
		return time.Now().Add(time.Duration(v) * 24 * time.Hour)
	default:
		return value
	}
}
