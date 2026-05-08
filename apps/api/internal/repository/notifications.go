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

// MarkReminderSent marca un recordatorio específico como enviado.
func (r *notificationRepository) MarkReminderSent(ctx context.Context, appointmentID uuid.UUID, minutesBefore int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE appointments
		SET reminders_sent = COALESCE(reminders_sent, '{}') || jsonb_build_object($2::TEXT, true),
		    updated_at = NOW()
		WHERE id = $1
	`, appointmentID, fmt.Sprintf("%d", minutesBefore))
	if err != nil {
		return fmt.Errorf("notificationRepository.MarkReminderSent: %w", err)
	}
	return nil
}
