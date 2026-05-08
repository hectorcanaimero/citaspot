package actions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type CreateTaskAction struct {
	taskRepo domain.TaskRepository
}

func NewCreateTaskAction(taskRepo domain.TaskRepository) *CreateTaskAction {
	return &CreateTaskAction{taskRepo: taskRepo}
}

func (a *CreateTaskAction) Type() string { return "create_task" }

func (a *CreateTaskAction) Execute(ctx context.Context, params engine.ActionParams) error {
	title := renderTemplate(params.Template, params.Context)
	if title == "" {
		title = "Tarea automática"
	}

	description := ""
	if desc, ok := params.Params["description"].(string); ok {
		description = renderTemplate(desc, params.Context)
	}

	var assignedTo *uuid.UUID
	if assignedStr, ok := params.Params["assigned_to"].(string); ok && assignedStr != "" {
		id, err := uuid.Parse(assignedStr)
		if err == nil {
			assignedTo = &id
		}
	}

	dueInDays := 7
	if d, ok := params.Params["due_in_days"].(float64); ok && d > 0 {
		dueInDays = int(d)
	}
	if d, ok := params.Params["due_days"].(float64); ok && d > 0 {
		dueInDays = int(d)
	}
	dueAt := time.Now().Add(time.Duration(dueInDays) * 24 * time.Hour)

	var ruleID *uuid.UUID
	if ruleStr, ok := params.Params["rule_id"].(string); ok && ruleStr != "" {
		id, err := uuid.Parse(ruleStr)
		if err == nil {
			ruleID = &id
		}
	}

	customerID := params.CustomerID
	task := &domain.Task{
		ID:          uuid.New(),
		TenantID:    params.TenantID,
		AssignedTo:  assignedTo,
		CustomerID:  &customerID,
		Title:       title,
		Description: description,
		Status:      "pending",
		DueAt:       &dueAt,
		Source:      "rule",
		RuleID:      ruleID,
	}

	switch strings.ToLower(params.EntityType) {
	case "appointment":
		task.AppointmentID = &params.EntityID
	case "treatment":
		task.TreatmentID = &params.EntityID
	}

	if err := a.taskRepo.Create(ctx, task); err != nil {
		return fmt.Errorf("CreateTaskAction.Execute: %w", err)
	}

	slog.Info("CreateTaskAction: tarea creada", "task_id", task.ID, "tenant", params.TenantID)
	return nil
}
