package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type MoveStageAction struct {
	customerRepo domain.CustomerRepository
}

func NewMoveStageAction(customerRepo domain.CustomerRepository) *MoveStageAction {
	return &MoveStageAction{customerRepo: customerRepo}
}

func (a *MoveStageAction) Type() string { return "move_stage" }

func (a *MoveStageAction) Execute(ctx context.Context, params engine.ActionParams) error {
	stageIDStr, ok := params.Params["stage_id"].(string)
	if !ok || stageIDStr == "" {
		return fmt.Errorf("MoveStageAction.Execute: stage_id no especificado en params")
	}

	stageID, err := uuid.Parse(stageIDStr)
	if err != nil {
		return fmt.Errorf("MoveStageAction.Execute: stage_id inválido '%s': %w", stageIDStr, err)
	}

	if err := a.customerRepo.UpdateStage(ctx, params.TenantID, params.CustomerID, stageID); err != nil {
		return fmt.Errorf("MoveStageAction.Execute: %w", err)
	}

	slog.Info("MoveStageAction: cliente movido de etapa", "customer_id", params.CustomerID, "stage_id", stageID, "tenant", params.TenantID)
	return nil
}
