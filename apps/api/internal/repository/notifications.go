package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type notificationRepository struct {
	db *pgxpool.Pool
}

// NewNotificationRepository crea el repositorio de notificaciones.
func NewNotificationRepository(db *pgxpool.Pool) domain.NotificationRepository {
	return &notificationRepository{db: db}
}

// LogNotification inserta un registro de notificación enviada.
func (r *notificationRepository) LogNotification(ctx context.Context, nl *domain.NotificationLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notification_logs
			(id, tenant_id, appointment_id, customer_id, type, wa_message_id, status, content, sent_at, error_message)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9)
	`, nl.ID, nl.TenantID, nl.AppointmentID, nl.CustomerID,
		nl.Type, nl.WAMessageID, nl.Status, nl.Content, nl.ErrorMessage)
	if err != nil {
		return fmt.Errorf("notificationRepository.LogNotification: %w", err)
	}
	return nil
}

// MarkReminder24hSent marca la cita como que ya recibió el recordatorio de 24h.
func (r *notificationRepository) MarkReminder24hSent(ctx context.Context, appointmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE appointments SET reminder_24h_sent = TRUE, updated_at = NOW() WHERE id = $1`,
		appointmentID,
	)
	if err != nil {
		return fmt.Errorf("notificationRepository.MarkReminder24hSent: %w", err)
	}
	return nil
}

// MarkReminder2hSent marca la cita como que ya recibió el recordatorio de 2h.
func (r *notificationRepository) MarkReminder2hSent(ctx context.Context, appointmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE appointments SET reminder_2h_sent = TRUE, updated_at = NOW() WHERE id = $1`,
		appointmentID,
	)
	if err != nil {
		return fmt.Errorf("notificationRepository.MarkReminder2hSent: %w", err)
	}
	return nil
}
