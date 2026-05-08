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

type pipelineStageRepository struct {
	db *pgxpool.Pool
}

func NewPipelineStageRepository(db *pgxpool.Pool) domain.PipelineStageRepository {
	return &pipelineStageRepository{db: db}
}

func scanPipelineStage(row pgx.Row, s *domain.PipelineStage) error {
	return row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Position, &s.Color,
		&s.IsDefault, &s.AutoRulesEnabled, &s.CreatedAt,
	)
}

const pipelineStageColumns = `id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at`

func (r *pipelineStageRepository) Create(ctx context.Context, s *domain.PipelineStage) error {
	return withTenant(ctx, r.db, s.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO pipeline_stages (id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`, s.ID, s.TenantID, s.Name, s.Position, s.Color, s.IsDefault, s.AutoRulesEnabled)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *pipelineStageRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.PipelineStage, error) {
	var result []*domain.PipelineStage
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+pipelineStageColumns+`
			FROM pipeline_stages
			WHERE tenant_id = $1
			ORDER BY position ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.PipelineStage{}
			if err := scanPipelineStage(rows, s); err != nil {
				return fmt.Errorf("pipelineStageRepository.List: scan: %w", err)
			}
			result = append(result, s)
		}
		return rows.Err()
	})
	return result, err
}

func (r *pipelineStageRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PipelineStage, error) {
	var s *domain.PipelineStage
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		s = &domain.PipelineStage{}
		err := scanPipelineStage(tx.QueryRow(ctx, `
			SELECT `+pipelineStageColumns+`
			FROM pipeline_stages
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), s)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("pipelineStageRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *pipelineStageRepository) Update(ctx context.Context, s *domain.PipelineStage) error {
	return withTenant(ctx, r.db, s.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE pipeline_stages
			SET name = $3, position = $4, color = $5, is_default = $6, auto_rules_enabled = $7
			WHERE tenant_id = $1 AND id = $2
		`, s.TenantID, s.ID, s.Name, s.Position, s.Color, s.IsDefault, s.AutoRulesEnabled)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *pipelineStageRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM pipeline_stages
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("pipelineStageRepository.Delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *pipelineStageRepository) Reorder(ctx context.Context, tenantID uuid.UUID, items []domain.ReorderStageInput) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		for _, item := range items {
			tag, err := tx.Exec(ctx, `
				UPDATE pipeline_stages
				SET position = $3
				WHERE tenant_id = $1 AND id = $2
			`, tenantID, item.StageID, item.Position)
			if err != nil {
				return fmt.Errorf("pipelineStageRepository.Reorder: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("pipelineStageRepository.Reorder: stage %s: %w", item.StageID, domain.ErrNotFound)
			}
		}
		return nil
	})
}
