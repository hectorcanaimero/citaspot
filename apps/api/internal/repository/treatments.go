package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type treatmentRepository struct {
	db *pgxpool.Pool
}

func NewTreatmentRepository(db *pgxpool.Pool) domain.TreatmentRepository {
	return &treatmentRepository{db: db}
}

const treatmentColumns = `
	id, tenant_id, customer_id, professional_id, name, treatment_type,
	status, total_sessions, completed_sessions, estimated_cost, paid_amount,
	currency, tooth_numbers, notes, started_at, completed_at, next_session_at,
	created_at, updated_at
`

func scanTreatment(row pgx.Row, t *domain.Treatment) error {
	var notes *string
	err := row.Scan(
		&t.ID, &t.TenantID, &t.CustomerID, &t.ProfessionalID,
		&t.Name, &t.TreatmentType, &t.Status,
		&t.TotalSessions, &t.CompletedSessions,
		&t.EstimatedCost, &t.PaidAmount, &t.Currency,
		&t.ToothNumbers, &notes,
		&t.StartedAt, &t.CompletedAt, &t.NextSessionAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if notes != nil {
		t.Notes = *notes
	}
	return nil
}

func (r *treatmentRepository) Create(ctx context.Context, t *domain.Treatment) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO treatments
				(id, tenant_id, customer_id, professional_id, name, treatment_type,
				 status, total_sessions, estimated_cost, currency, tooth_numbers, notes,
				 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		`, t.ID, t.TenantID, t.CustomerID, t.ProfessionalID,
			t.Name, t.TreatmentType, t.Status,
			t.TotalSessions, t.EstimatedCost, t.Currency,
			t.ToothNumbers, t.Notes)
		if err != nil {
			return fmt.Errorf("treatmentRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *treatmentRepository) List(ctx context.Context, tenantID uuid.UUID, q *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
	var result []*domain.Treatment
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := `SELECT ` + treatmentColumns + ` FROM treatments WHERE tenant_id = $1`
		args := []any{tenantID}
		argN := 2

		if q != nil {
			if q.CustomerID != nil {
				query += fmt.Sprintf(" AND customer_id = $%d", argN)
				args = append(args, *q.CustomerID)
				argN++
			}
			if q.ProfessionalID != nil {
				query += fmt.Sprintf(" AND professional_id = $%d", argN)
				args = append(args, *q.ProfessionalID)
				argN++
			}
			if q.Status != "" {
				query += fmt.Sprintf(" AND status = $%d", argN)
				args = append(args, q.Status)
				argN++
			}
		}
		query += " ORDER BY created_at DESC"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("treatmentRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			t := &domain.Treatment{}
			if err := scanTreatment(rows, t); err != nil {
				return fmt.Errorf("treatmentRepository.List: scan: %w", err)
			}
			result = append(result, t)
		}
		return rows.Err()
	})
	return result, err
}

func (r *treatmentRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Treatment, error) {
	var t *domain.Treatment
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		t = &domain.Treatment{}
		err := scanTreatment(tx.QueryRow(ctx, `
			SELECT `+treatmentColumns+`
			FROM treatments
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), t)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("treatmentRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *treatmentRepository) Update(ctx context.Context, t *domain.Treatment) error {
	return withTenant(ctx, r.db, t.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE treatments
			SET name = $3, treatment_type = $4, total_sessions = $5,
			    estimated_cost = $6, currency = $7, tooth_numbers = $8,
			    notes = $9, updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, t.TenantID, t.ID, t.Name, t.TreatmentType,
			t.TotalSessions, t.EstimatedCost, t.Currency,
			t.ToothNumbers, t.Notes)
		if err != nil {
			return fmt.Errorf("treatmentRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *treatmentRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		setClause := "status = $3, updated_at = NOW()"
		switch status {
		case "in_progress":
			setClause += ", started_at = COALESCE(started_at, NOW())"
		case "completed":
			setClause += ", completed_at = NOW()"
		}

		tag, err := tx.Exec(ctx, `
			UPDATE treatments SET `+setClause+`
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id, status)
		if err != nil {
			return fmt.Errorf("treatmentRepository.UpdateStatus: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
