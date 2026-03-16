package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type reminderRepository struct {
	db *pgxpool.Pool
}

// NewReminderRepository crea el repositorio de recordatorios.
// Opera sin RLS — accede a citas de todos los tenants para el cron global.
func NewReminderRepository(db *pgxpool.Pool) domain.ReminderRepository {
	return &reminderRepository{db: db}
}

const reminderColumns = `
	a.id, a.tenant_id,
	t.slug, t.timezone,
	c.name, c.phone,
	p.name, s.name,
	a.starts_at
`

// FindDue24hReminders retorna citas que comienzan en ~24h y aún no recibieron recordatorio.
func (r *reminderRepository) FindDue24hReminders(ctx context.Context) ([]*domain.ReminderJob, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+reminderColumns+`
		FROM appointments a
		JOIN tenants       t ON t.id = a.tenant_id
		JOIN customers     c ON c.id = a.customer_id
		JOIN professionals p ON p.id = a.professional_id
		JOIN services      s ON s.id = a.service_id
		WHERE a.status IN ('pending', 'confirmed')
		  AND a.reminder_24h_sent = FALSE
		  AND c.wa_opt_in = TRUE
		  AND c.phone IS NOT NULL
		  AND a.starts_at BETWEEN NOW() + INTERVAL '23 hours' AND NOW() + INTERVAL '25 hours'
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("reminderRepository.FindDue24h: %w", err)
	}
	defer rows.Close()
	return scanReminderJobs(rows, "reminder_24h")
}

// FindDue2hReminders retorna citas que comienzan en ~2h y aún no recibieron recordatorio.
func (r *reminderRepository) FindDue2hReminders(ctx context.Context) ([]*domain.ReminderJob, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+reminderColumns+`
		FROM appointments a
		JOIN tenants       t ON t.id = a.tenant_id
		JOIN customers     c ON c.id = a.customer_id
		JOIN professionals p ON p.id = a.professional_id
		JOIN services      s ON s.id = a.service_id
		WHERE a.status IN ('pending', 'confirmed')
		  AND a.reminder_2h_sent = FALSE
		  AND c.wa_opt_in = TRUE
		  AND c.phone IS NOT NULL
		  AND a.starts_at BETWEEN NOW() + INTERVAL '1 hour 45 minutes' AND NOW() + INTERVAL '2 hours 15 minutes'
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("reminderRepository.FindDue2h: %w", err)
	}
	defer rows.Close()
	return scanReminderJobs(rows, "reminder_2h")
}

func scanReminderJobs(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}, reminderType string) ([]*domain.ReminderJob, error) {
	var result []*domain.ReminderJob
	for rows.Next() {
		j := &domain.ReminderJob{Type: reminderType}
		if err := rows.Scan(
			&j.AppointmentID, &j.TenantID,
			&j.TenantSlug, &j.TenantTimezone,
			&j.CustomerName, &j.CustomerPhone,
			&j.ProfessionalName, &j.ServiceName,
			&j.StartsAt,
		); err != nil {
			return nil, fmt.Errorf("scanReminderJobs: %w", err)
		}
		result = append(result, j)
	}
	return result, rows.Err()
}
