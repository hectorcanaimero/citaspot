package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type MoveStageAction struct {
	customerRepo domain.CustomerRepository
	publisher    domain.MessagePublisher // opcional — nil deshabilita la emisión del evento
}

func NewMoveStageAction(customerRepo domain.CustomerRepository, publisher domain.MessagePublisher) *MoveStageAction {
	return &MoveStageAction{customerRepo: customerRepo, publisher: publisher}
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

	a.emitStageChangedEvent(ctx, params.TenantID, params.CustomerID, stageID)
	return nil
}

// emitStageChangedEvent publica customer.stage_changed en la cola rules.events.
// Best effort — un fallo aquí no debe revertir el cambio de etapa.
func (a *MoveStageAction) emitStageChangedEvent(ctx context.Context, tenantID, customerID, stageID uuid.UUID) {
	if a.publisher == nil {
		return
	}
	body, err := json.Marshal(domain.RuleEvent{
		TenantID:   tenantID,
		EventType:  "customer.stage_changed",
		CustomerID: customerID,
		EntityID:   customerID,
		EntityType: "customer",
		Payload: map[string]any{
			"customer_id": customerID.String(),
			"stage_id":    stageID.String(),
		},
		Timestamp: time.Now(),
	})
	if err != nil {
		slog.Warn("MoveStageAction.emitStageChangedEvent: marshal error", "error", err)
		return
	}
	if err := a.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("MoveStageAction.emitStageChangedEvent: publish error", "error", err)
	}
}
