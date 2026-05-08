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

// FindDueReminders retorna citas que necesitan recordatorio para un tiempo dado (en minutos antes).
// Opera sin RLS — acceso global para el cron.
func (r *reminderRepository) FindDueReminders(ctx context.Context, minutesBefore int) ([]*domain.ReminderJob, error) {
	windowMin := minutesBefore - 15
	windowMax := minutesBefore + 15

	query := `
		SELECT ` + reminderColumns + `
		FROM appointments a
		JOIN tenants       t ON t.id = a.tenant_id
		JOIN customers     c ON c.id = a.customer_id
		JOIN professionals p ON p.id = a.professional_id
		JOIN services      s ON s.id = a.service_id
		WHERE a.status IN ('pending', 'confirmed')
		  AND c.wa_opt_in = TRUE
		  AND c.phone IS NOT NULL
		  AND NOT COALESCE((a.reminders_sent->>(($3)::TEXT))::BOOLEAN, FALSE)
		  AND a.starts_at BETWEEN NOW() + ($1 * interval '1 minute')
		                       AND NOW() + ($2 * interval '1 minute')
		  AND (t.settings->'reminder_minutes') @> to_jsonb($3)
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query, windowMin, windowMax, minutesBefore)
	if err != nil {
		return nil, fmt.Errorf("reminderRepository.FindDueReminders(%d): %w", minutesBefore, err)
	}
	defer rows.Close()

	reminderType := fmt.Sprintf("reminder_%d", minutesBefore)
	return scanReminderJobs(rows, reminderType)
}

// GetDistinctReminderMinutes retorna todos los tiempos de recordatorio únicos configurados por tenants.
func (r *reminderRepository) GetDistinctReminderMinutes(ctx context.Context) ([]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT value::INT
		FROM tenants, jsonb_array_elements(COALESCE(settings->'reminder_minutes', '[1440, 120]'::jsonb)) AS value
		ORDER BY value::INT DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("reminderRepository.GetDistinctReminderMinutes: %w", err)
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var m int
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("reminderRepository.GetDistinctReminderMinutes: scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
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
