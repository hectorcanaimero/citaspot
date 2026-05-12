package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

const sessionCols = `
	ts.id, ts.tenant_id, ts.treatment_id, ts.professional_id,
	ts.status, ts.scheduled_at, ts.duration_minutes, ts.completed_at,
	ts.procedures_done, ts.notes, ts.paid_in_session, ts.currency,
	ts.next_session_at, ts.created_at, ts.updated_at,
	p.name AS professional_name`

type treatmentSessionRepo struct {
	db *pgxpool.Pool
}

// NewTreatmentSessionRepository crea el repositorio de sesiones de tratamiento.
func NewTreatmentSessionRepository(db *pgxpool.Pool) domain.TreatmentSessionRepository {
	return &treatmentSessionRepo{db: db}
}

func scanSession(row pgx.Row, s *domain.TreatmentSession) error {
	var (
		proceduresDone   *string
		notes            *string
		currency         *string
		professionalName *string
	)
	err := row.Scan(
		&s.ID, &s.TenantID, &s.TreatmentID, &s.ProfessionalID,
		&s.Status, &s.ScheduledAt, &s.DurationMinutes, &s.CompletedAt,
		&proceduresDone, &notes, &s.PaidInSession, &currency,
		&s.NextSessionAt, &s.CreatedAt, &s.UpdatedAt,
		&professionalName,
	)
	if err != nil {
		return err
	}
	if proceduresDone != nil {
		s.ProceduresDone = *proceduresDone
	}
	if notes != nil {
		s.Notes = *notes
	}
	if currency != nil {
		s.Currency = *currency
	}
	if professionalName != nil {
		s.ProfessionalName = *professionalName
	}
	return nil
}

// recalcCompleted actualiza completed_sessions en treatments dentro de la tx.
func recalcCompleted(ctx context.Context, tx pgx.Tx, treatmentID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE treatments
		SET completed_sessions = (
			SELECT COUNT(*) FROM treatment_sessions
			WHERE treatment_id = $1 AND status = 'completed'
		),
		updated_at = NOW()
		WHERE id = $1
	`, treatmentID)
	return err
}

func (r *treatmentSessionRepo) withTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
	return err
}

func (r *treatmentSessionRepo) Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *domain.TreatmentSessionInput) (*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	id := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO treatment_sessions
			(id, tenant_id, treatment_id, professional_id, status,
			 scheduled_at, duration_minutes, procedures_done, notes,
			 paid_in_session, currency, next_session_at,
			 completed_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,
		        CASE WHEN $5::text = 'completed' THEN NOW() ELSE NULL END,
		        NOW(), NOW())
	`,
		id, tenantID, treatmentID, input.ProfessionalID, input.Status,
		input.ScheduledAt, input.DurationMinutes,
		nullString(input.ProceduresDone), nullString(input.Notes),
		input.PaidInSession, nullString(input.Currency), input.NextSessionAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	if input.Status == "completed" {
		if err := recalcCompleted(ctx, tx, treatmentID); err != nil {
			return nil, fmt.Errorf("recalc completed: %w", err)
		}
	}

	s := &domain.TreatmentSession{}
	row2 := tx.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.id = $1
	`, id)
	if err := scanSession(row2, s); err != nil {
		return nil, fmt.Errorf("fetch created session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s, nil
}

func (r *treatmentSessionRepo) List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.treatment_id = $1
		ORDER BY ts.scheduled_at DESC
	`, treatmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.TreatmentSession
	for rows.Next() {
		s := &domain.TreatmentSession{}
		if err := scanSession(rows, s); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *treatmentSessionRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	s := &domain.TreatmentSession{}
	row := tx.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.id = $1
	`, id)
	if err := scanSession(row, s); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	tx.Commit(ctx)
	return s, nil
}

func (r *treatmentSessionRepo) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	// Leer treatment_id antes de actualizar
	var treatmentID uuid.UUID
	var oldStatus string
	err = tx.QueryRow(ctx, `SELECT treatment_id, status FROM treatment_sessions WHERE id = $1`, id).
		Scan(&treatmentID, &oldStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE treatment_sessions SET
			status           = $2,
			duration_minutes = $3,
			procedures_done  = $4,
			notes            = $5,
			paid_in_session  = $6,
			currency         = $7,
			next_session_at  = $8,
			completed_at     = CASE WHEN $2::text = 'completed' THEN NOW() ELSE NULL END,
			updated_at       = NOW()
		WHERE id = $1
	`,
		id, input.Status, input.DurationMinutes,
		nullString(input.ProceduresDone), nullString(input.Notes),
		input.PaidInSession, nullString(input.Currency), input.NextSessionAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Recalcular si el status cambia y afecta el conteo
	if oldStatus != input.Status && (input.Status == "completed" || input.Status == "cancelled" || oldStatus == "completed") {
		if err := recalcCompleted(ctx, tx, treatmentID); err != nil {
			return nil, fmt.Errorf("recalc: %w", err)
		}
	}

	s := &domain.TreatmentSession{}
	row := tx.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM treatment_sessions ts
		JOIN professionals p ON p.id = ts.professional_id
		WHERE ts.id = $1
	`, id)
	if err := scanSession(row, s); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s, nil
}

func (r *treatmentSessionRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.withTenant(ctx, tx, tenantID); err != nil {
		return err
	}

	var treatmentID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `DELETE FROM treatment_sessions WHERE id = $1 RETURNING treatment_id, status`, id).
		Scan(&treatmentID, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}

	// Solo recalcular si era una sesión completada (afecta el conteo)
	if status == "completed" {
		if err := recalcCompleted(ctx, tx, treatmentID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// nullString convierte string vacío en nil para columnas TEXT nullables.
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
