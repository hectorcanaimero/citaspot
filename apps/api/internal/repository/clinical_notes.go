package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type clinicalNoteRepository struct {
	db *pgxpool.Pool
}

func NewClinicalNoteRepository(db *pgxpool.Pool) domain.ClinicalNoteRepository {
	return &clinicalNoteRepository{db: db}
}

func (r *clinicalNoteRepository) Create(ctx context.Context, note *domain.ClinicalNote) error {
	return withTenant(ctx, r.db, note.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO clinical_notes
				(id, tenant_id, appointment_id, customer_id, professional_id,
				 subjective, objective, assessment, plan, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`, note.ID, note.TenantID, note.AppointmentID, note.CustomerID,
			note.ProfessionalID, note.Subjective, note.Objective,
			note.Assessment, note.Plan)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return domain.ErrClinicalNoteExists
			}
			return fmt.Errorf("clinicalNoteRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *clinicalNoteRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
	var result *domain.ClinicalNoteWithDetails
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		result = &domain.ClinicalNoteWithDetails{}
		err := tx.QueryRow(ctx, `
			SELECT cn.id, cn.tenant_id, cn.appointment_id, cn.customer_id, cn.professional_id,
			       cn.subjective, cn.objective, cn.assessment, cn.plan,
			       cn.created_at, cn.updated_at,
			       p.name, s.name, a.starts_at
			FROM clinical_notes cn
			JOIN professionals p ON p.id = cn.professional_id
			JOIN appointments a ON a.id = cn.appointment_id
			JOIN services s ON s.id = a.service_id
			WHERE cn.tenant_id = $1 AND cn.id = $2
		`, tenantID, id).Scan(
			&result.ID, &result.TenantID, &result.AppointmentID,
			&result.CustomerID, &result.ProfessionalID,
			&result.Subjective, &result.Objective, &result.Assessment, &result.Plan,
			&result.CreatedAt, &result.UpdatedAt,
			&result.ProfessionalName, &result.ServiceName, &result.AppointmentDate,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("clinicalNoteRepository.GetByID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *clinicalNoteRepository) GetByAppointmentID(ctx context.Context, tenantID, appointmentID uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
	var result *domain.ClinicalNoteWithDetails
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		result = &domain.ClinicalNoteWithDetails{}
		err := tx.QueryRow(ctx, `
			SELECT cn.id, cn.tenant_id, cn.appointment_id, cn.customer_id, cn.professional_id,
			       cn.subjective, cn.objective, cn.assessment, cn.plan,
			       cn.created_at, cn.updated_at,
			       p.name, s.name, a.starts_at
			FROM clinical_notes cn
			JOIN professionals p ON p.id = cn.professional_id
			JOIN appointments a ON a.id = cn.appointment_id
			JOIN services s ON s.id = a.service_id
			WHERE cn.tenant_id = $1 AND cn.appointment_id = $2
		`, tenantID, appointmentID).Scan(
			&result.ID, &result.TenantID, &result.AppointmentID,
			&result.CustomerID, &result.ProfessionalID,
			&result.Subjective, &result.Objective, &result.Assessment, &result.Plan,
			&result.CreatedAt, &result.UpdatedAt,
			&result.ProfessionalName, &result.ServiceName, &result.AppointmentDate,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("clinicalNoteRepository.GetByAppointmentID: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *clinicalNoteRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit, offset int) ([]*domain.ClinicalNoteWithDetails, int, error) {
	var result []*domain.ClinicalNoteWithDetails
	var total int
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT cn.id, cn.tenant_id, cn.appointment_id, cn.customer_id, cn.professional_id,
			       cn.subjective, cn.objective, cn.assessment, cn.plan,
			       cn.created_at, cn.updated_at,
			       p.name, s.name, a.starts_at,
			       COUNT(*) OVER() AS total_count
			FROM clinical_notes cn
			JOIN professionals p ON p.id = cn.professional_id
			JOIN appointments a ON a.id = cn.appointment_id
			JOIN services s ON s.id = a.service_id
			WHERE cn.tenant_id = $1 AND cn.customer_id = $2
			ORDER BY cn.created_at DESC
			LIMIT $3 OFFSET $4
		`, tenantID, customerID, limit, offset)
		if err != nil {
			return fmt.Errorf("clinicalNoteRepository.ListByCustomer: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			n := &domain.ClinicalNoteWithDetails{}
			if err := rows.Scan(
				&n.ID, &n.TenantID, &n.AppointmentID,
				&n.CustomerID, &n.ProfessionalID,
				&n.Subjective, &n.Objective, &n.Assessment, &n.Plan,
				&n.CreatedAt, &n.UpdatedAt,
				&n.ProfessionalName, &n.ServiceName, &n.AppointmentDate,
				&total,
			); err != nil {
				return fmt.Errorf("clinicalNoteRepository.ListByCustomer: scan: %w", err)
			}
			result = append(result, n)
		}
		return rows.Err()
	})
	return result, total, err
}

func (r *clinicalNoteRepository) Update(ctx context.Context, note *domain.ClinicalNote) error {
	return withTenant(ctx, r.db, note.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE clinical_notes
			SET subjective = $3, objective = $4, assessment = $5, plan = $6,
			    updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, note.TenantID, note.ID, note.Subjective, note.Objective,
			note.Assessment, note.Plan)
		if err != nil {
			return fmt.Errorf("clinicalNoteRepository.Update: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *clinicalNoteRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM clinical_notes
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("clinicalNoteRepository.Delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
