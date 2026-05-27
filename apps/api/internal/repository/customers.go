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

type customerRepository struct {
	db *pgxpool.Pool
}

// NewCustomerRepository crea el repositorio de clientes.
func NewCustomerRepository(db *pgxpool.Pool) domain.CustomerRepository {
	return &customerRepository{db: db}
}

// scanCustomer escanea una fila en un Customer, manejando campos nullable.
func scanCustomer(row interface{ Scan(dest ...any) error }, c *domain.Customer) error {
	var email, notes, acquisitionSource *string
	if err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Phone, &email,
		&notes, &c.Tags, &c.WaOptIn, &c.TotalVisits,
		&c.StageID, &c.LastVisitAt, &c.NextRecallAt, &c.LifetimeValue, &acquisitionSource,
		&c.CreatedAt,
	); err != nil {
		return err
	}
	if email != nil {
		c.Email = *email
	}
	if notes != nil {
		c.Notes = *notes
	}
	if acquisitionSource != nil {
		c.AcquisitionSource = *acquisitionSource
	}
	return nil
}

// scanCustomerWithLast escanea un Customer junto con datos del último profesional que lo atendió.
func scanCustomerWithLast(row interface{ Scan(dest ...any) error }, c *domain.Customer) error {
	var email, notes, acquisitionSource *string
	if err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Phone, &email,
		&notes, &c.Tags, &c.WaOptIn, &c.TotalVisits,
		&c.StageID, &c.LastVisitAt, &c.NextRecallAt, &c.LifetimeValue, &acquisitionSource,
		&c.CreatedAt,
		&c.LastProfessionalID, &c.LastProfessionalName, &c.LastAttendedAt,
	); err != nil {
		return err
	}
	if email != nil {
		c.Email = *email
	}
	if notes != nil {
		c.Notes = *notes
	}
	if acquisitionSource != nil {
		c.AcquisitionSource = *acquisitionSource
	}
	return nil
}

// FindOrCreateByPhone busca un cliente por teléfono o lo crea si no existe.
// Usado por el booking público y el asistente de WhatsApp.
func (r *customerRepository) FindOrCreateByPhone(ctx context.Context, tenantID uuid.UUID, name, phone string) (*domain.Customer, error) {
	var c *domain.Customer
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		c = &domain.Customer{}
		// Intentar encontrar primero
		err := scanCustomer(tx.QueryRow(ctx, `
			SELECT id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
				       stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
				       created_at
			FROM customers
			WHERE tenant_id = $1 AND phone = $2
		`, tenantID, phone), c)
		if err == nil {
			return nil // encontrado
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("customerRepository.FindOrCreateByPhone: find: %w", err)
		}

		// Crear nuevo cliente
		c = &domain.Customer{ID: uuid.New(), TenantID: tenantID, Name: name, Phone: phone, Tags: []string{}, WaOptIn: true}

		err = scanCustomer(tx.QueryRow(ctx, `
			INSERT INTO customers (id, tenant_id, name, phone, wa_opt_in, created_at)
			VALUES ($1, $2, $3, $4, TRUE, NOW())
			RETURNING id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
			          stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
			          created_at
		`, c.ID, c.TenantID, c.Name, c.Phone), c)
		if err != nil {
			return fmt.Errorf("customerRepository.FindOrCreateByPhone: create: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// FindByPhone busca un cliente por teléfono sin crearlo. Retorna nil, nil si no existe.
func (r *customerRepository) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*domain.Customer, error) {
	var c *domain.Customer
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		c = &domain.Customer{}
		err := scanCustomer(tx.QueryRow(ctx, `
			SELECT id, tenant_id, name, phone, email, notes, tags, wa_opt_in, total_visits,
				       stage_id, last_visit_at, next_recall_at, lifetime_value, acquisition_source,
				       created_at
			FROM customers
			WHERE tenant_id = $1 AND phone = $2
		`, tenantID, phone), c)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c = nil
				return nil
			}
			return fmt.Errorf("customerRepository.FindByPhone: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetByID retorna un cliente por ID.
func (r *customerRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	var c *domain.Customer
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		c = &domain.Customer{}
		err := scanCustomerWithLast(tx.QueryRow(ctx, `
			SELECT c.id, c.tenant_id, c.name, c.phone, c.email, c.notes, c.tags, c.wa_opt_in,
			       c.total_visits, c.stage_id, c.last_visit_at, c.next_recall_at,
			       c.lifetime_value, c.acquisition_source, c.created_at,
			       lp.professional_id AS last_professional_id,
			       lp.professional_name AS last_professional_name,
			       lp.starts_at AS last_attended_at
			FROM customers c
			LEFT JOIN LATERAL (
			  SELECT a.professional_id, p.name AS professional_name, a.starts_at
			  FROM appointments a
			  JOIN professionals p ON p.id = a.professional_id
			  WHERE a.customer_id = c.id AND a.status = 'completed'
			  ORDER BY a.starts_at DESC
			  LIMIT 1
			) lp ON TRUE
			WHERE c.tenant_id = $1 AND c.id = $2
		`, tenantID, id), c)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("customerRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// List retorna clientes con paginación y búsqueda opcional por nombre.
func (r *customerRepository) List(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*domain.Customer, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var result []*domain.Customer
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var rows pgx.Rows
		var err error

		if search != "" {
			rows, err = tx.Query(ctx, `
				SELECT c.id, c.tenant_id, c.name, c.phone, c.email, c.notes, c.tags, c.wa_opt_in,
				       c.total_visits, c.stage_id, c.last_visit_at, c.next_recall_at,
				       c.lifetime_value, c.acquisition_source, c.created_at,
				       lp.professional_id AS last_professional_id,
				       lp.professional_name AS last_professional_name,
				       lp.starts_at AS last_attended_at
				FROM customers c
				LEFT JOIN LATERAL (
				  SELECT a.professional_id, p.name AS professional_name, a.starts_at
				  FROM appointments a
				  JOIN professionals p ON p.id = a.professional_id
				  WHERE a.customer_id = c.id AND a.status = 'completed'
				  ORDER BY a.starts_at DESC
				  LIMIT 1
				) lp ON TRUE
				WHERE c.tenant_id = $1
				  AND (c.name ILIKE $2 OR c.phone ILIKE $2)
				ORDER BY c.name ASC
				LIMIT $3 OFFSET $4
			`, tenantID, "%"+search+"%", limit, offset)
		} else {
			rows, err = tx.Query(ctx, `
				SELECT c.id, c.tenant_id, c.name, c.phone, c.email, c.notes, c.tags, c.wa_opt_in,
				       c.total_visits, c.stage_id, c.last_visit_at, c.next_recall_at,
				       c.lifetime_value, c.acquisition_source, c.created_at,
				       lp.professional_id AS last_professional_id,
				       lp.professional_name AS last_professional_name,
				       lp.starts_at AS last_attended_at
				FROM customers c
				LEFT JOIN LATERAL (
				  SELECT a.professional_id, p.name AS professional_name, a.starts_at
				  FROM appointments a
				  JOIN professionals p ON p.id = a.professional_id
				  WHERE a.customer_id = c.id AND a.status = 'completed'
				  ORDER BY a.starts_at DESC
				  LIMIT 1
				) lp ON TRUE
				WHERE c.tenant_id = $1
				ORDER BY c.name ASC
				LIMIT $2 OFFSET $3
			`, tenantID, limit, offset)
		}
		if err != nil {
			return fmt.Errorf("customerRepository.List: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			c := &domain.Customer{}
			if err := scanCustomerWithLast(rows, c); err != nil {
				return fmt.Errorf("customerRepository.List: scan: %w", err)
			}
			result = append(result, c)
		}
		return rows.Err()
	})
	return result, err
}

// UpdateStage actualiza la etapa del pipeline de un cliente. stageID nil limpia el campo.
func (r *customerRepository) UpdateStage(ctx context.Context, tenantID, customerID uuid.UUID, stageID *uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE customers SET stage_id = $3 WHERE tenant_id = $1 AND id = $2`, tenantID, customerID, stageID)
		if err != nil {
			return fmt.Errorf("customerRepository.UpdateStage: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// UpdateField actualiza un campo específico permitido de un cliente.
func (r *customerRepository) UpdateField(ctx context.Context, tenantID, customerID uuid.UUID, field string, value any) error {
	allowed := map[string]bool{"next_recall_at": true, "last_visit_at": true, "total_visits": true, "lifetime_value": true}
	if !allowed[field] {
		return fmt.Errorf("customerRepository.UpdateField: campo '%s' no permitido: %w", field, domain.ErrValidation)
	}
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := fmt.Sprintf(`UPDATE customers SET %s = $3 WHERE tenant_id = $1 AND id = $2`, field)
		tag, err := tx.Exec(ctx, query, tenantID, customerID, value)
		if err != nil {
			return fmt.Errorf("customerRepository.UpdateField: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// IncrementVisits suma delta a total_visits de forma atómica y opcionalmente actualiza last_visit_at.
func (r *customerRepository) IncrementVisits(ctx context.Context, tenantID, customerID uuid.UUID, delta int, lastVisit *time.Time) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// Asegurar que total_visits nunca quede negativo
		tag, err := tx.Exec(ctx, `
			UPDATE customers
			SET total_visits  = GREATEST(0, total_visits + $3),
			    last_visit_at = COALESCE($4, last_visit_at)
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, customerID, delta, lastVisit)
		if err != nil {
			return fmt.Errorf("customerRepository.IncrementVisits: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
