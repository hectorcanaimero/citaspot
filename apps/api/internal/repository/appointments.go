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

type appointmentRepository struct {
	db *pgxpool.Pool
}

// NewAppointmentRepository crea el repositorio de citas.
func NewAppointmentRepository(db *pgxpool.Pool) domain.AppointmentRepository {
	return &appointmentRepository{db: db}
}

// Create inserta una nueva cita en la DB.
func (r *appointmentRepository) Create(ctx context.Context, a *domain.Appointment) error {
	return withTenant(ctx, r.db, a.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO appointments (
				id, tenant_id, customer_id, professional_id, service_id,
				starts_at, ends_at, status, source, price, notes,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10, $11,
				NOW(), NOW()
			)
		`, a.ID, a.TenantID, a.CustomerID, a.ProfessionalID, a.ServiceID,
			a.StartsAt, a.EndsAt, a.Status, a.Source, a.Price, a.Notes)
		if err != nil {
			return fmt.Errorf("appointmentRepository.Create: %w", err)
		}
		return nil
	})
}

// GetByID retorna una cita con detalles del cliente, profesional y servicio.
func (r *appointmentRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AppointmentWithDetails, error) {
	var a *domain.AppointmentWithDetails
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		a = &domain.AppointmentWithDetails{}
		// Campos nullable en DB se escanean en *string para soportar NULL
		var notes, internalNotes, cancellationReason *string
		err := tx.QueryRow(ctx, `
			SELECT
				ap.id, ap.tenant_id, ap.customer_id, ap.professional_id, ap.service_id,
				ap.starts_at, ap.ends_at, ap.status, ap.source, ap.price, ap.notes,
				ap.internal_notes, ap.confirmed_at, ap.cancelled_at, ap.cancellation_reason,
				ap.created_at, ap.updated_at,
				c.name, c.phone,
				p.name,
				s.name, s.duration_min
			FROM appointments ap
			JOIN customers     c ON c.id = ap.customer_id
			JOIN professionals p ON p.id = ap.professional_id
			JOIN services      s ON s.id = ap.service_id
			WHERE ap.tenant_id = $1 AND ap.id = $2
		`, tenantID, id).Scan(
			&a.ID, &a.TenantID, &a.CustomerID, &a.ProfessionalID, &a.ServiceID,
			&a.StartsAt, &a.EndsAt, &a.Status, &a.Source, &a.Price, &notes,
			&internalNotes, &a.ConfirmedAt, &a.CancelledAt, &cancellationReason,
			&a.CreatedAt, &a.UpdatedAt,
			&a.CustomerName, &a.CustomerPhone,
			&a.ProfessionalName,
			&a.ServiceName, &a.ServiceDuration,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("appointmentRepository.GetByID: %w", err)
		}
		if notes != nil {
			a.Notes = *notes
		}
		if internalNotes != nil {
			a.InternalNotes = *internalNotes
		}
		if cancellationReason != nil {
			a.CancellationReason = *cancellationReason
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ListByDate retorna las citas de un día específico (usando timezone del tenant).
func (r *appointmentRepository) ListByDate(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*domain.AppointmentWithDetails, error) {
	if timezone == "" {
		timezone = "UTC"
	}
	var result []*domain.AppointmentWithDetails
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT
				ap.id, ap.tenant_id, ap.customer_id, ap.professional_id, ap.service_id,
				ap.starts_at, ap.ends_at, ap.status, ap.source, ap.price, ap.notes,
				ap.internal_notes, ap.confirmed_at, ap.cancelled_at, ap.cancellation_reason,
				ap.created_at, ap.updated_at,
				c.name, c.phone,
				p.name,
				s.name, s.duration_min
			FROM appointments ap
			JOIN customers     c ON c.id = ap.customer_id
			JOIN professionals p ON p.id = ap.professional_id
			JOIN services      s ON s.id = ap.service_id
			WHERE ap.tenant_id = $1
			  AND DATE(ap.starts_at AT TIME ZONE $2) = $3::DATE
			ORDER BY ap.starts_at ASC
		`, tenantID, timezone, date)
		if err != nil {
			return fmt.Errorf("appointmentRepository.ListByDate: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			a := &domain.AppointmentWithDetails{}
			// Campos nullable en DB se escanean en *string para soportar NULL
			var notes, internalNotes, cancellationReason *string
			if err := rows.Scan(
				&a.ID, &a.TenantID, &a.CustomerID, &a.ProfessionalID, &a.ServiceID,
				&a.StartsAt, &a.EndsAt, &a.Status, &a.Source, &a.Price, &notes,
				&internalNotes, &a.ConfirmedAt, &a.CancelledAt, &cancellationReason,
				&a.CreatedAt, &a.UpdatedAt,
				&a.CustomerName, &a.CustomerPhone,
				&a.ProfessionalName,
				&a.ServiceName, &a.ServiceDuration,
			); err != nil {
				return fmt.Errorf("appointmentRepository.ListByDate: scan: %w", err)
			}
			if notes != nil {
				a.Notes = *notes
			}
			if internalNotes != nil {
				a.InternalNotes = *internalNotes
			}
			if cancellationReason != nil {
				a.CancellationReason = *cancellationReason
			}
			result = append(result, a)
		}
		return rows.Err()
	})
	return result, err
}

// ListFiltered retorna citas con filtros dinámicos y paginación.
func (r *appointmentRepository) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	// Valores por defecto de paginación
	page := q.Page
	if page < 1 {
		page = 1
	}
	perPage := q.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Timezone por defecto
	tz := q.Timezone
	if tz == "" {
		tz = "UTC"
	}

	var result *domain.PaginatedAppointments
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// Construir cláusula WHERE dinámica con parámetros posicionales
		args := []any{tenantID, tz}
		argIdx := 3 // $1=tenantID, $2=tz; el siguiente argumento empieza en $3

		where := "ap.tenant_id = $1"

		// Filtro por fecha desde
		if q.DateFrom != "" {
			where += fmt.Sprintf(" AND DATE(ap.starts_at AT TIME ZONE $2) >= $%d::DATE", argIdx)
			args = append(args, q.DateFrom)
			argIdx++
		}

		// Filtro por fecha hasta
		if q.DateTo != "" {
			where += fmt.Sprintf(" AND DATE(ap.starts_at AT TIME ZONE $2) <= $%d::DATE", argIdx)
			args = append(args, q.DateTo)
			argIdx++
		}

		// Filtro por profesional
		if q.ProfessionalID != nil {
			where += fmt.Sprintf(" AND ap.professional_id = $%d", argIdx)
			args = append(args, *q.ProfessionalID)
			argIdx++
		}

		// Filtro por servicio
		if q.ServiceID != nil {
			where += fmt.Sprintf(" AND ap.service_id = $%d", argIdx)
			args = append(args, *q.ServiceID)
			argIdx++
		}

		// Filtro por estado
		if q.Status != "" {
			where += fmt.Sprintf(" AND ap.status = $%d", argIdx)
			args = append(args, q.Status)
			argIdx++
		}

		// Filtro por nombre de cliente (búsqueda parcial)
		if q.Search != "" {
			where += fmt.Sprintf(" AND c.name ILIKE $%d", argIdx)
			args = append(args, "%"+q.Search+"%")
			argIdx++
		}

		// Determinar columna y dirección de ordenamiento
		orderCol := "ap.starts_at"
		if q.SortBy == "customer_name" {
			orderCol = "c.name"
		}
		orderDir := "DESC"
		if q.SortDir == "asc" {
			orderDir = "ASC"
		}

		// Query de conteo total (sin LIMIT/OFFSET)
		countSQL := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM appointments ap
			JOIN customers     c ON c.id = ap.customer_id
			JOIN professionals p ON p.id = ap.professional_id
			JOIN services      s ON s.id = ap.service_id
			WHERE %s
		`, where)

		var total int
		if err := tx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			return fmt.Errorf("appointmentRepository.ListFiltered: count: %w", err)
		}

		// Calcular paginación
		totalPages := (total + perPage - 1) / perPage
		if totalPages < 1 {
			totalPages = 1
		}
		offset := (page - 1) * perPage

		// Query de datos con LIMIT/OFFSET
		dataSQL := fmt.Sprintf(`
			SELECT
				ap.id, ap.tenant_id, ap.customer_id, ap.professional_id, ap.service_id,
				ap.starts_at, ap.ends_at, ap.status, ap.source, ap.price, ap.notes,
				ap.internal_notes, ap.confirmed_at, ap.cancelled_at, ap.cancellation_reason,
				ap.created_at, ap.updated_at,
				c.name, c.phone,
				p.name,
				s.name, s.duration_min
			FROM appointments ap
			JOIN customers     c ON c.id = ap.customer_id
			JOIN professionals p ON p.id = ap.professional_id
			JOIN services      s ON s.id = ap.service_id
			WHERE %s
			ORDER BY %s %s
			LIMIT $%d OFFSET $%d
		`, where, orderCol, orderDir, argIdx, argIdx+1)

		args = append(args, perPage, offset)

		rows, err := tx.Query(ctx, dataSQL, args...)
		if err != nil {
			return fmt.Errorf("appointmentRepository.ListFiltered: query: %w", err)
		}
		defer rows.Close()

		// Slice vacío (nunca nil) para JSON []
		data := make([]*domain.AppointmentWithDetails, 0)
		for rows.Next() {
			a := &domain.AppointmentWithDetails{}
			// Campos nullable en DB se escanean en *string para soportar NULL
			var notes, internalNotes, cancellationReason *string
			if err := rows.Scan(
				&a.ID, &a.TenantID, &a.CustomerID, &a.ProfessionalID, &a.ServiceID,
				&a.StartsAt, &a.EndsAt, &a.Status, &a.Source, &a.Price, &notes,
				&internalNotes, &a.ConfirmedAt, &a.CancelledAt, &cancellationReason,
				&a.CreatedAt, &a.UpdatedAt,
				&a.CustomerName, &a.CustomerPhone,
				&a.ProfessionalName,
				&a.ServiceName, &a.ServiceDuration,
			); err != nil {
				return fmt.Errorf("appointmentRepository.ListFiltered: scan: %w", err)
			}
			if notes != nil {
				a.Notes = *notes
			}
			if internalNotes != nil {
				a.InternalNotes = *internalNotes
			}
			if cancellationReason != nil {
				a.CancellationReason = *cancellationReason
			}
			data = append(data, a)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("appointmentRepository.ListFiltered: rows: %w", err)
		}

		result = &domain.PaginatedAppointments{
			Data: data,
			Pagination: domain.Pagination{
				Page:       page,
				PerPage:    perPage,
				Total:      total,
				TotalPages: totalPages,
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateStatus actualiza el estado y notas de una cita.
func (r *appointmentRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		var confirmedAt, cancelledAt *time.Time

		switch req.Status {
		case "confirmed":
			confirmedAt = &now
		case "cancelled":
			cancelledAt = &now
		}

		tag, err := tx.Exec(ctx, `
			UPDATE appointments
			SET status              = COALESCE(NULLIF($3, ''), status),
			    notes               = CASE WHEN $4 != '' THEN $4 ELSE notes END,
			    internal_notes      = CASE WHEN $5 != '' THEN $5 ELSE internal_notes END,
			    cancellation_reason = CASE WHEN $6 != '' THEN $6 ELSE cancellation_reason END,
			    confirmed_at        = COALESCE($7, confirmed_at),
			    cancelled_at        = COALESCE($8, cancelled_at),
			    updated_at          = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id,
			req.Status, req.Notes, req.InternalNotes, req.CancellationReason,
			confirmedAt, cancelledAt)
		if err != nil {
			return fmt.Errorf("appointmentRepository.UpdateStatus: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// CheckConflict retorna true si hay conflicto de horario con otra cita activa.
func (r *appointmentRepository) CheckConflict(ctx context.Context, tenantID, professionalID uuid.UUID, startsAt, endsAt time.Time, excludeID *uuid.UUID) (bool, error) {
	var conflict bool
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var row pgx.Row
		if excludeID != nil {
			row = tx.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM appointments
					WHERE tenant_id       = $1
					  AND professional_id = $2
					  AND status NOT IN ('cancelled')
					  AND id != $5
					  AND starts_at < $4
					  AND ends_at   > $3
				)
			`, tenantID, professionalID, startsAt, endsAt, *excludeID)
		} else {
			row = tx.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM appointments
					WHERE tenant_id       = $1
					  AND professional_id = $2
					  AND status NOT IN ('cancelled')
					  AND starts_at < $4
					  AND ends_at   > $3
				)
			`, tenantID, professionalID, startsAt, endsAt)
		}
		return row.Scan(&conflict)
	})
	return conflict, err
}

// Reschedule actualiza el profesional y horario de una cita existente.
func (r *appointmentRepository) Reschedule(ctx context.Context, tenantID, id, professionalID uuid.UUID, startsAt, endsAt time.Time) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE appointments
			SET professional_id = $3,
			    starts_at       = $4,
			    ends_at         = $5,
			    updated_at      = NOW()
			WHERE tenant_id = $1 AND id = $2
			  AND status NOT IN ('cancelled', 'completed')
		`, tenantID, id, professionalID, startsAt, endsAt)
		if err != nil {
			return fmt.Errorf("appointmentRepository.Reschedule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
