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

type tenantModuleRepo struct {
	db *pgxpool.Pool
}

// NewTenantModuleRepository crea el repositorio de activaciones de módulos.
// La tabla no tiene RLS — se consulta antes de tener contexto de tenant.
func NewTenantModuleRepository(db *pgxpool.Pool) domain.TenantModuleRepository {
	return &tenantModuleRepo{db: db}
}

func (r *tenantModuleRepo) IsActive(ctx context.Context, tenantID uuid.UUID, moduleKey string) (bool, error) {
	if r.db == nil {
		return false, errors.New("tenantModuleRepo: nil pool")
	}
	var enabled bool
	err := r.db.QueryRow(ctx, `
		SELECT enabled
		FROM tenant_modules
		WHERE tenant_id = $1 AND module_key = $2
	`, tenantID, moduleKey).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("tenantModuleRepo.IsActive: %w", err)
	}
	return enabled, nil
}

func (r *tenantModuleRepo) ListActiveKeys(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	if r.db == nil {
		return nil, errors.New("tenantModuleRepo: nil pool")
	}
	rows, err := r.db.Query(ctx, `
		SELECT module_key
		FROM tenant_modules
		WHERE tenant_id = $1 AND enabled = TRUE
		ORDER BY module_key
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenantModuleRepo.ListActiveKeys: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0, 4)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *tenantModuleRepo) Enable(ctx context.Context, tenantID uuid.UUID, moduleKey string) error {
	if r.db == nil {
		return errors.New("tenantModuleRepo: nil pool")
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenant_modules (tenant_id, module_key, enabled, activated_at, deactivated_at)
		VALUES ($1, $2, TRUE, NOW(), NULL)
		ON CONFLICT (tenant_id, module_key) DO UPDATE
		SET enabled = TRUE,
		    activated_at = NOW(),
		    deactivated_at = NULL,
		    updated_at = NOW()
	`, tenantID, moduleKey)
	if err != nil {
		return fmt.Errorf("tenantModuleRepo.Enable: %w", err)
	}
	return nil
}

func (r *tenantModuleRepo) Disable(ctx context.Context, tenantID uuid.UUID, moduleKey string) error {
	if r.db == nil {
		return errors.New("tenantModuleRepo: nil pool")
	}
	_, err := r.db.Exec(ctx, `
		UPDATE tenant_modules
		SET enabled = FALSE,
		    deactivated_at = NOW(),
		    updated_at = NOW()
		WHERE tenant_id = $1 AND module_key = $2 AND enabled = TRUE
	`, tenantID, moduleKey)
	if err != nil {
		return fmt.Errorf("tenantModuleRepo.Disable: %w", err)
	}
	return nil
}
