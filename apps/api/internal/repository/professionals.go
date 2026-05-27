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
	var specialty, bio, avatarURL, phone, email *string
	if err := row.Scan(
		&p.ID, &p.TenantID, &p.UserID, &p.Name, &specialty,
		&bio, &avatarURL, &phone, &email, &p.Color, &p.IsActive, &p.IsArchived, &p.CreatedAt,
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
	if phone != nil {
		p.Phone = *phone
	}
	if email != nil {
		p.Email = *email
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
			SELECT id, tenant_id, user_id, name, specialty, bio, avatar_url, phone, email, color, is_active, is_archived, created_at
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
			INSERT INTO professionals (id, tenant_id, name, specialty, bio, avatar_url, phone, email, color, is_active, is_archived, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, FALSE, NOW())
		`, p.ID, p.TenantID, p.Name, p.Specialty, p.Bio, p.AvatarURL, p.Phone, p.Email, p.Color, p.IsActive)
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
			SELECT id, tenant_id, user_id, name, specialty, bio, avatar_url, phone, email, color, is_active, is_archived, created_at
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
			SET name = $3, specialty = $4, bio = $5, avatar_url = $6, phone = $7, email = $8, color = $9, is_active = $10, is_archived = $11
			WHERE tenant_id = $1 AND id = $2
		`, p.TenantID, p.ID, p.Name, p.Specialty, p.Bio, p.AvatarURL, p.Phone, p.Email, p.Color, p.IsActive, p.IsArchived)
		if err != nil {
			return fmt.Errorf("professionalRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// ListServices retorna los servicios asignados a un profesional del tenant.
func (r *professionalRepository) ListServices(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Service, error) {
	var result []*domain.Service
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT s.id, s.tenant_id, s.name, s.description, s.duration_min, s.price,
			       s.currency, s.buffer_min, s.is_active, s.sort_order, s.created_at
			FROM services s
			JOIN professional_services ps ON ps.service_id = s.id
			WHERE ps.professional_id = $1 AND s.tenant_id = $2
			ORDER BY s.sort_order ASC, s.name ASC
		`, professionalID, tenantID)
		if err != nil {
			return fmt.Errorf("professionalRepository.ListServices: query: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			s := &domain.Service{}
			var description *string
			if err := rows.Scan(
				&s.ID, &s.TenantID, &s.Name, &description, &s.DurationMin, &s.Price,
				&s.Currency, &s.BufferMin, &s.IsActive, &s.SortOrder, &s.CreatedAt,
			); err != nil {
				return fmt.Errorf("professionalRepository.ListServices: scan: %w", err)
			}
			if description != nil {
				s.Description = *description
			}
			result = append(result, s)
		}
		return rows.Err()
	})
	return result, err
}

// ListServiceLinks retorna todos los pares servicio-profesional activos del tenant.
func (r *professionalRepository) ListServiceLinks(ctx context.Context, tenantID uuid.UUID) ([]domain.ServiceProfessionalLink, error) {
	var result []domain.ServiceProfessionalLink
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT ps.service_id, ps.professional_id
			FROM professional_services ps
			JOIN professionals p ON p.id = ps.professional_id
			JOIN services s ON s.id = ps.service_id
			WHERE p.tenant_id = $1
			  AND p.is_active = true
			  AND s.is_active = true
		`, tenantID)
		if err != nil {
			return fmt.Errorf("professionalRepository.ListServiceLinks: query: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var link domain.ServiceProfessionalLink
			if err := rows.Scan(&link.ServiceID, &link.ProfessionalID); err != nil {
				return fmt.Errorf("professionalRepository.ListServiceLinks: scan: %w", err)
			}
			result = append(result, link)
		}
		return rows.Err()
	})
	return result, err
}

// AssignService asigna un servicio a un profesional validando que ambos pertenecen al tenant.
func (r *professionalRepository) AssignService(ctx context.Context, tenantID, professionalID, serviceID uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var profExists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM professionals WHERE id=$1 AND tenant_id=$2)`,
			professionalID, tenantID,
		).Scan(&profExists); err != nil {
			return fmt.Errorf("professionalRepository.AssignService: check prof: %w", err)
		}
		if !profExists {
			return domain.ErrNotFound
		}

		var svcExists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM services WHERE id=$1 AND tenant_id=$2)`,
			serviceID, tenantID,
		).Scan(&svcExists); err != nil {
			return fmt.Errorf("professionalRepository.AssignService: check svc: %w", err)
		}
		if !svcExists {
			return domain.ErrNotFound
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO professional_services (professional_id, service_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, professionalID, serviceID); err != nil {
			return fmt.Errorf("professionalRepository.AssignService: insert: %w", err)
		}
		return nil
	})
}

// BulkAssignServicesToProfessional inserta múltiples vínculos profesional-servicio en una sola tx.
// Filtra por servicios activos del mismo tenant (ANY($2::uuid[])) y es idempotente.
func (r *professionalRepository) BulkAssignServicesToProfessional(
	ctx context.Context, tenantID, professionalID uuid.UUID, serviceIDs []uuid.UUID,
) (int, error) {
	if len(serviceIDs) == 0 {
		return 0, nil
	}
	var inserted int
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM professionals WHERE id=$1 AND tenant_id=$2)`,
			professionalID, tenantID,
		).Scan(&exists); err != nil {
			return fmt.Errorf("professionalRepository.BulkAssignServicesToProfessional: check prof: %w", err)
		}
		if !exists {
			return domain.ErrNotFound
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO professional_services (professional_id, service_id)
			SELECT $1, s.id FROM services s
			WHERE s.id = ANY($2::uuid[]) AND s.tenant_id = $3 AND s.is_active = TRUE
			ON CONFLICT DO NOTHING
		`, professionalID, serviceIDs, tenantID)
		if err != nil {
			return fmt.Errorf("professionalRepository.BulkAssignServicesToProfessional: insert: %w", err)
		}
		inserted = int(tag.RowsAffected())
		return nil
	})
	return inserted, err
}

// RemoveService elimina la asignación de un servicio a un profesional.
func (r *professionalRepository) RemoveService(ctx context.Context, tenantID, professionalID, serviceID uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var profExists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM professionals WHERE id=$1 AND tenant_id=$2)`,
			professionalID, tenantID,
		).Scan(&profExists); err != nil {
			return fmt.Errorf("professionalRepository.RemoveService: check prof: %w", err)
		}
		if !profExists {
			return domain.ErrNotFound
		}

		if _, err := tx.Exec(ctx, `
			DELETE FROM professional_services
			WHERE professional_id = $1 AND service_id = $2
		`, professionalID, serviceID); err != nil {
			return fmt.Errorf("professionalRepository.RemoveService: delete: %w", err)
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
			SELECT id, tenant_id, user_id, name, specialty, bio, avatar_url, phone, email, color, is_active, is_archived, created_at
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
