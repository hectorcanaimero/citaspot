package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type taskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) domain.TaskRepository {
	return &taskRepository{db: db}
}

const taskColumns = `
	id, tenant_id, assigned_to, customer_id, appointment_id, treatment_id,
	title, description, status, due_at, completed_at, source, rule_id, created_at
`

func scanTask(row pgx.Row, t *domain.Task) error {
	var description *string
	err := row.Scan(
		&t.ID, &t.TenantID, &t.AssignedTo, &t.CustomerID,
		&t.AppointmentID, &t.TreatmentID,
		&t.Title, &description, &t.Status,
		&t.DueAt, &t.CompletedAt, &t.Source, &t.RuleID, &t.CreatedAt,
	)
	if err != nil {
		return err
	}
	if description != nil {
		t.Description = *description
	}
	return nil
}

func (r *taskRepository) Create(ctx context.Context, t *domain.Task) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO tasks
				(id, tenant_id, assigned_to, customer_id, appointment_id, treatment_id,
				 title, description, status, due_at, source, rule_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`, t.ID, t.TenantID, t.AssignedTo, t.CustomerID,
			t.AppointmentID, t.TreatmentID,
			t.Title, t.Description, t.Status,
			t.DueAt, t.Source, t.RuleID)
		if err != nil {
			return fmt.Errorf("taskRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *taskRepository) List(ctx context.Context, tenantID uuid.UUID, q *domain.TaskListQuery) ([]*domain.Task, error) {
	var result []*domain.Task
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := `SELECT ` + taskColumns + ` FROM tasks WHERE tenant_id = $1`
		args := []any{tenantID}
		argN := 2

		if q != nil {
			if q.AssignedTo != nil {
				query += fmt.Sprintf(" AND assigned_to = $%d", argN)
				args = append(args, *q.AssignedTo)
				argN++
			}
			if q.CustomerID != nil {
				query += fmt.Sprintf(" AND customer_id = $%d", argN)
				args = append(args, *q.CustomerID)
				argN++
			}
			if q.Status != "" {
				query += fmt.Sprintf(" AND status = $%d", argN)
				args = append(args, q.Status)
				argN++
			}
			if q.DueBefore != nil {
				query += fmt.Sprintf(" AND due_at <= $%d", argN)
				args = append(args, *q.DueBefore)
				argN++
			}
		}
		query += " ORDER BY COALESCE(due_at, '9999-12-31'::timestamptz) ASC, created_at DESC"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("taskRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			t := &domain.Task{}
			if err := scanTask(rows, t); err != nil {
				return fmt.Errorf("taskRepository.List: scan: %w", err)
			}
			result = append(result, t)
		}
		return rows.Err()
	})
	return result, err
}

func (r *taskRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	var t *domain.Task
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		t = &domain.Task{}
		err := scanTask(tx.QueryRow(ctx, `
			SELECT `+taskColumns+`
			FROM tasks
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), t)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("taskRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *taskRepository) Update(ctx context.Context, t *domain.Task) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET assigned_to = $3, customer_id = $4, appointment_id = $5, treatment_id = $6,
			    title = $7, description = $8, due_at = $9
			WHERE tenant_id = $1 AND id = $2
		`, t.TenantID, t.ID, t.AssignedTo, t.CustomerID,
			t.AppointmentID, t.TreatmentID, t.Title, t.Description, t.DueAt)
		if err != nil {
			return fmt.Errorf("taskRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *taskRepository) Complete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'completed', completed_at = $3
			WHERE tenant_id = $1 AND id = $2 AND status IN ('pending', 'in_progress')
		`, tenantID, id, time.Now())
		if err != nil {
			return fmt.Errorf("taskRepository.Complete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *taskRepository) Dismiss(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'dismissed'
			WHERE tenant_id = $1 AND id = $2 AND status IN ('pending', 'in_progress')
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("taskRepository.Dismiss: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
