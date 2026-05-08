package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type scheduleRepository struct {
	db *pgxpool.Pool
}

// NewScheduleRepository crea el repositorio de horarios.
func NewScheduleRepository(db *pgxpool.Pool) domain.ScheduleRepository {
	return &scheduleRepository{db: db}
}

// GetSchedules retorna los horarios semanales de un profesional.
func (r *scheduleRepository) GetSchedules(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Schedule, error) {
	var result []*domain.Schedule
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, professional_id, day_of_week,
			       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), is_active
			FROM schedules
			WHERE tenant_id = $1 AND professional_id = $2
			ORDER BY day_of_week ASC
		`, tenantID, professionalID)
		if err != nil {
			return fmt.Errorf("scheduleRepository.GetSchedules: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.Schedule{}
			if err := rows.Scan(
				&s.ID, &s.TenantID, &s.ProfessionalID,
				&s.DayOfWeek, &s.StartTime, &s.EndTime, &s.IsActive,
			); err != nil {
				return fmt.Errorf("scheduleRepository.GetSchedules: scan: %w", err)
			}
			result = append(result, s)
		}
		return rows.Err()
	})
	return result, err
}

// UpsertSchedules reemplaza los horarios de un profesional con los nuevos.
// Usa INSERT ... ON CONFLICT para actualizar si ya existe el día.
func (r *scheduleRepository) UpsertSchedules(ctx context.Context, tenantID, professionalID uuid.UUID, schedules []*domain.Schedule) ([]*domain.Schedule, error) {
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		for _, s := range schedules {
			_, err := tx.Exec(ctx, `
				INSERT INTO schedules
					(id, tenant_id, professional_id, day_of_week, start_time, end_time, is_active)
				VALUES
					(uuid_generate_v4(), $1, $2, $3, $4::TIME, $5::TIME, $6)
				ON CONFLICT (professional_id, day_of_week)
				DO UPDATE SET
					start_time = EXCLUDED.start_time,
					end_time   = EXCLUDED.end_time,
					is_active  = EXCLUDED.is_active,
					tenant_id  = EXCLUDED.tenant_id
			`, tenantID, professionalID, s.DayOfWeek, s.StartTime, s.EndTime, s.IsActive)
			if err != nil {
				return fmt.Errorf("scheduleRepository.UpsertSchedules day=%d: %w", s.DayOfWeek, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Retornar los horarios actualizados
	return r.GetSchedules(ctx, tenantID, professionalID)
}

// GetBlocks retorna los bloqueos de tiempo de un profesional en un rango.
// Incluye bloqueos puntuales y recurrentes (expandidos al rango solicitado).
func (r *scheduleRepository) GetBlocks(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*domain.ScheduleBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days, created_at
		FROM schedule_blocks
		WHERE tenant_id = $1
		  AND (professional_id = $2 OR professional_id IS NULL)
		  AND is_recurring = FALSE
		  AND starts_at < $4
		  AND ends_at   > $3
		UNION ALL
		SELECT id, tenant_id, professional_id,
		       ($3::DATE + starts_at::TIME) AT TIME ZONE 'UTC',
		       ($3::DATE + ends_at::TIME) AT TIME ZONE 'UTC',
		       reason, is_recurring, recurrence_days, created_at
		FROM schedule_blocks
		WHERE tenant_id = $1
		  AND (professional_id = $2 OR professional_id IS NULL)
		  AND is_recurring = TRUE
		  AND EXTRACT(DOW FROM $3 AT TIME ZONE 'UTC')::INT = ANY(recurrence_days)
		ORDER BY starts_at ASC
	`, tenantID, professionalID, from, to)
	if err != nil {
		return nil, fmt.Errorf("scheduleRepository.GetBlocks: %w", err)
	}
	defer rows.Close()

	var blocks []*domain.ScheduleBlock
	for rows.Next() {
		b := &domain.ScheduleBlock{}
		var profID *uuid.UUID
		if err := rows.Scan(
			&b.ID, &b.TenantID, &profID,
			&b.StartsAt, &b.EndsAt, &b.Reason,
			&b.IsRecurring, &b.RecurrenceDays, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scheduleRepository.GetBlocks: scan: %w", err)
		}
		b.ProfessionalID = profID
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// GetAppointmentsInRange retorna citas activas de un profesional en un rango.
// Usado por el engine de disponibilidad para detectar conflictos.
func (r *scheduleRepository) GetAppointmentsInRange(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*domain.Appointment, error) {
	var result []*domain.Appointment
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, customer_id, professional_id, service_id,
			       starts_at, ends_at, status, source
			FROM appointments
			WHERE tenant_id       = $1
			  AND professional_id = $2
			  AND status NOT IN ('cancelled')
			  AND starts_at < $4
			  AND ends_at   > $3
			ORDER BY starts_at ASC
		`, tenantID, professionalID, from, to)
		if err != nil {
			return fmt.Errorf("scheduleRepository.GetAppointmentsInRange: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			a := &domain.Appointment{}
			if err := rows.Scan(
				&a.ID, &a.TenantID, &a.CustomerID, &a.ProfessionalID, &a.ServiceID,
				&a.StartsAt, &a.EndsAt, &a.Status, &a.Source,
			); err != nil {
				return fmt.Errorf("scheduleRepository.GetAppointmentsInRange: scan: %w", err)
			}
			result = append(result, a)
		}
		return rows.Err()
	})
	return result, err
}

// CreateBlock crea un bloqueo de horario (individual o recurrente).
func (r *scheduleRepository) CreateBlock(ctx context.Context, b *domain.ScheduleBlock) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO schedule_blocks (id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, b.ID, b.TenantID, b.ProfessionalID, b.StartsAt, b.EndsAt, b.Reason, b.IsRecurring, b.RecurrenceDays)
	if err != nil {
		return fmt.Errorf("scheduleRepository.CreateBlock: %w", err)
	}
	return nil
}

// ListBlocks retorna todos los bloqueos de un tenant, opcionalmente filtrados por profesional.
func (r *scheduleRepository) ListBlocks(ctx context.Context, tenantID uuid.UUID, professionalID *uuid.UUID) ([]*domain.ScheduleBlock, error) {
	var query string
	var args []any

	if professionalID != nil {
		query = `
			SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days, created_at
			FROM schedule_blocks
			WHERE tenant_id = $1 AND (professional_id = $2 OR professional_id IS NULL)
			ORDER BY is_recurring DESC, starts_at ASC
		`
		args = []any{tenantID, *professionalID}
	} else {
		query = `
			SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, is_recurring, recurrence_days, created_at
			FROM schedule_blocks
			WHERE tenant_id = $1
			ORDER BY is_recurring DESC, starts_at ASC
		`
		args = []any{tenantID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("scheduleRepository.ListBlocks: %w", err)
	}
	defer rows.Close()

	var blocks []*domain.ScheduleBlock
	for rows.Next() {
		b := &domain.ScheduleBlock{}
		var profID *uuid.UUID
		if err := rows.Scan(
			&b.ID, &b.TenantID, &profID, &b.StartsAt, &b.EndsAt, &b.Reason,
			&b.IsRecurring, &b.RecurrenceDays, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scheduleRepository.ListBlocks: scan: %w", err)
		}
		b.ProfessionalID = profID
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// DeleteBlock elimina un bloqueo de horario.
func (r *scheduleRepository) DeleteBlock(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM schedule_blocks WHERE tenant_id = $1 AND id = $2", tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("scheduleRepository.DeleteBlock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
