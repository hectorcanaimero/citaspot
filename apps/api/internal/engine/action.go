package engine

import (
	"context"

	"github.com/google/uuid"
)

type ActionParams struct {
	TenantID   uuid.UUID
	TenantSlug string
	CustomerID uuid.UUID
	EntityID   uuid.UUID
	EntityType string
	Template   string
	Params     map[string]any
	Context    map[string]any
}

type ActionExecutor interface {
	Type() string
	Execute(ctx context.Context, params ActionParams) error
}

type ActionRegistry struct {
	executors map[string]ActionExecutor
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{executors: make(map[string]ActionExecutor)}
}

func (r *ActionRegistry) Register(executor ActionExecutor) {
	r.executors[executor.Type()] = executor
}

func (r *ActionRegistry) Get(actionType string) ActionExecutor {
	return r.executors[actionType]
}
