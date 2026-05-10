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

type professionalRepository struct {
	db *pgxpool.Pool
}

// scanProfessional escanea una fila de la tabla professionals manejando campos nullable.
func scanProfessional(row interface {
	Scan(dest ...any) error
}, p *domain.Professional) error {
	var specialty, bio, avatarURL *string
	if err := row.Scan(
		&p.ID, &p.TenantID, &p.UserID, &p.Name, &specialty,
		&bio, &avatarURL, &p.Color, &p.IsActive, &p.IsArchived, &p.CreatedAt,
	); err != nil {
		return err
	}
	if specialty != nil {
		p.Specialty = *specialty
	}
	if bio != nil {
		p.Bio = *bio
	}
	if avatarURL != nil {
		p.AvatarURL = *avatarURL
	}
	return nil
}

// NewProfessionalRepository crea el repositorio de profesionales.
func NewProfessionalRepository(db *pgxpool.Pool) domain.ProfessionalRepository {
	return &professionalRepository{db: db}
}

// withTenant ejecuta fn dentro de una transacción con app.tenant_id seteado (RLS).
func withTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("withTenant: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("withTenant: set_config: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// List retorna todos los profesionales activos del tenant.
func (r *professionalRepository) List(ctx context.Context, tenantID uuid.UUID, includeArchived bool) ([]*domain.Professional, error) {
	var result []*domain.Professional
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, user_id, name, specialty, bio, avatar_url, color, is_active, is_archived, created_at
			FROM professionals
			WHERE tenant_id = $1 AND (is_archived = FALSE OR $2 = TRUE)
			ORDER BY name ASC
		`, tenantID, includeArchived)
		if err != nil {
			return fmt.Errorf("professionalRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			p := &domain.Professional{}
			if err := scanProfessional(rows, p); err != nil {
				return fmt.Errorf("professionalRepository.List: scan: %w", err)
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	return result, err
}

// Create inserta un nuevo profesional.
func (r *professionalRepository) Create(ctx context.Context, p *domain.Professional) error {
	return withTenant(ctx, r.db, p.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO professionals (id, tenant_id, name, specialty, bio, avatar_url, color, is_active, is_archived, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE, NOW())
		`, p.ID, p.TenantID, p.Name, p.Specialty, p.Bio, p.AvatarURL, p.Color, p.IsActive)
		if err != nil {
			return fmt.Errorf("professionalRepository.Create: %w", err)
		}
		return nil
	})
}

// GetByID retorna un profesional por ID, validando que pertenece al tenant.
func (r *professionalRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Professional, error) {
	var p *domain.Professional
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		p = &domain.Professional{}
		row := tx.QueryRow(ctx, `
			SELECT id, tenant_id, user_id, name, specialty, bio, avatar_url, color, is_active, is_archived, created_at
			FROM professionals
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err := scanProfessional(row, p); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("professionalRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Update actualiza los datos de un profesional.
func (r *professionalRepository) Update(ctx context.Context, p *domain.Professional) error {
	return withTenant(ctx, r.db, p.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE professionals
			SET name = $3, specialty = $4, bio = $5, avatar_url = $6, color = $7, is_active = $8, is_archived = $9
			WHERE tenant_id = $1 AND id = $2
		`, p.TenantID, p.ID, p.Name, p.Specialty, p.Bio, p.AvatarURL, p.Color, p.IsActive, p.IsArchived)
		if err != nil {
			return fmt.Errorf("professionalRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// ListByTenantPublic retorna profesionales activos para la página pública de reservas.
// Usa withTenant para satisfacer la política RLS aunque sea un endpoint sin auth.
func (r *professionalRepository) ListByTenantPublic(ctx context.Context, tenantID uuid.UUID) ([]*domain.Professional, error) {
	var result []*domain.Professional
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, user_id, name, specialty, bio, avatar_url, color, is_active, is_archived, created_at
			FROM professionals
			WHERE tenant_id = $1 AND is_active = TRUE AND is_archived = FALSE
			ORDER BY name ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("professionalRepository.ListByTenantPublic: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			p := &domain.Professional{}
			if err := scanProfessional(rows, p); err != nil {
				return fmt.Errorf("professionalRepository.ListByTenantPublic: scan: %w", err)
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	return result, err
}
