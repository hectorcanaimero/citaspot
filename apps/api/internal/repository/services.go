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

type serviceRepository struct {
	db *pgxpool.Pool
}

// NewServiceRepository crea el repositorio de servicios.
func NewServiceRepository(db *pgxpool.Pool) domain.ServiceRepository {
	return &serviceRepository{db: db}
}

const serviceColumns = `
	id, tenant_id, name, description, duration_min, price, currency,
	buffer_min, is_active, sort_order, created_at
`

func scanService(row pgx.Row, s *domain.Service) error {
	return row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.DurationMin,
		&s.Price, &s.Currency, &s.BufferMin, &s.IsActive, &s.SortOrder, &s.CreatedAt,
	)
}

// List retorna todos los servicios del tenant ordenados por sort_order.
func (r *serviceRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Service, error) {
	var result []*domain.Service
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+serviceColumns+`
			FROM services
			WHERE tenant_id = $1
			ORDER BY sort_order ASC, name ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("serviceRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.Service{}
			if err := scanService(rows, s); err != nil {
				return fmt.Errorf("serviceRepository.List: scan: %w", err)
			}
			result = append(result, s)
		}
		return rows.Err()
	})
	return result, err
}

// ListActive retorna solo servicios activos (para booking público).
// Usa withTenant para satisfacer la política RLS aunque sea un endpoint sin auth.
func (r *serviceRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*domain.Service, error) {
	var result []*domain.Service
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+serviceColumns+`
			FROM services
			WHERE tenant_id = $1 AND is_active = TRUE
			ORDER BY sort_order ASC, name ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("serviceRepository.ListActive: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.Service{}
			if err := scanService(rows, s); err != nil {
				return fmt.Errorf("serviceRepository.ListActive: scan: %w", err)
			}
			result = append(result, s)
		}
		return rows.Err()
	})
	return result, err
}

// Create inserta un nuevo servicio.
func (r *serviceRepository) Create(ctx context.Context, s *domain.Service) error {
	return withTenant(ctx, r.db, s.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO services
				(id, tenant_id, name, description, duration_min, price, currency, buffer_min, is_active, sort_order, created_at)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		`, s.ID, s.TenantID, s.Name, s.Description, s.DurationMin,
			s.Price, s.Currency, s.BufferMin, s.IsActive, s.SortOrder)
		if err != nil {
			return fmt.Errorf("serviceRepository.Create: %w", err)
		}
		return nil
	})
}

// GetByID retorna un servicio por ID validando tenant.
func (r *serviceRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Service, error) {
	var svc *domain.Service
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		svc = &domain.Service{}
		err := scanService(tx.QueryRow(ctx, `
			SELECT `+serviceColumns+`
			FROM services
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id), svc)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("serviceRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// Delete elimina permanentemente un servicio del tenant.
func (r *serviceRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM services
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("serviceRepository.Delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Update actualiza un servicio.
func (r *serviceRepository) Update(ctx context.Context, s *domain.Service) error {
	return withTenant(ctx, r.db, s.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE services
			SET name = $3, description = $4, duration_min = $5, price = $6,
			    currency = $7, buffer_min = $8, is_active = $9, sort_order = $10
			WHERE tenant_id = $1 AND id = $2
		`, s.TenantID, s.ID, s.Name, s.Description, s.DurationMin,
			s.Price, s.Currency, s.BufferMin, s.IsActive, s.SortOrder)
		if err != nil {
			return fmt.Errorf("serviceRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
