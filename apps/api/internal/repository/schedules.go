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
func (r *scheduleRepository) GetBlocks(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*domain.ScheduleBlock, error) {
	var result []*domain.ScheduleBlock
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, professional_id, starts_at, ends_at, reason, created_at
			FROM schedule_blocks
			WHERE tenant_id = $1
			  AND (professional_id = $2 OR professional_id IS NULL)
			  AND starts_at < $4
			  AND ends_at   > $3
			ORDER BY starts_at ASC
		`, tenantID, professionalID, from, to)
		if err != nil {
			return fmt.Errorf("scheduleRepository.GetBlocks: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			b := &domain.ScheduleBlock{}
			if err := rows.Scan(
				&b.ID, &b.TenantID, &b.ProfessionalID,
				&b.StartsAt, &b.EndsAt, &b.Reason, &b.CreatedAt,
			); err != nil {
				return fmt.Errorf("scheduleRepository.GetBlocks: scan: %w", err)
			}
			result = append(result, b)
		}
		return rows.Err()
	})
	return result, err
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
