// Package worker contiene los workers background del Core API.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// ReminderWorker envía recordatorios de citas vía WhatsApp cada 5 minutos.
type ReminderWorker struct {
	reminderRepo  domain.ReminderRepository
	notifRepo     domain.NotificationRepository
	waClient      domain.WAClient
	interval      time.Duration
}

// NewReminderWorker crea el worker de recordatorios.
func NewReminderWorker(
	reminderRepo domain.ReminderRepository,
	notifRepo domain.NotificationRepository,
	waClient domain.WAClient,
) *ReminderWorker {
	return &ReminderWorker{
		reminderRepo: reminderRepo,
		notifRepo:    notifRepo,
		waClient:     waClient,
		interval:     5 * time.Minute,
	}
}

// Start inicia el cron en una goroutine. Bloqueante — llamar con go.
func (w *ReminderWorker) Start(ctx context.Context) {
	slog.Info("ReminderWorker: iniciado (cada 5 min)")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Ejecutar inmediatamente al iniciar
	w.run(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("ReminderWorker: detenido")
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

// run procesa los recordatorios de 24h y 2h pendientes.
func (w *ReminderWorker) run(ctx context.Context) {
	// Recordatorios 24h
	jobs24h, err := w.reminderRepo.FindDue24hReminders(ctx)
	if err != nil {
		slog.Error("ReminderWorker: FindDue24h error", "error", err)
	}
	for _, job := range jobs24h {
		if err := w.send(ctx, job); err != nil {
			slog.Error("ReminderWorker: send 24h error", "appointment", job.AppointmentID, "error", err)
			continue
		}
		if err := w.notifRepo.MarkReminder24hSent(ctx, job.AppointmentID); err != nil {
			slog.Error("ReminderWorker: mark 24h error", "error", err)
		}
	}

	// Recordatorios 2h
	jobs2h, err := w.reminderRepo.FindDue2hReminders(ctx)
	if err != nil {
		slog.Error("ReminderWorker: FindDue2h error", "error", err)
	}
	for _, job := range jobs2h {
		if err := w.send(ctx, job); err != nil {
			slog.Error("ReminderWorker: send 2h error", "appointment", job.AppointmentID, "error", err)
			continue
		}
		if err := w.notifRepo.MarkReminder2hSent(ctx, job.AppointmentID); err != nil {
			slog.Error("ReminderWorker: mark 2h error", "error", err)
		}
	}

	if len(jobs24h)+len(jobs2h) > 0 {
		slog.Info("ReminderWorker: recordatorios procesados",
			"total", len(jobs24h)+len(jobs2h), "24h", len(jobs24h), "2h", len(jobs2h))
	}
}

// send envía el recordatorio vía WhatsApp y registra el log.
func (w *ReminderWorker) send(ctx context.Context, job *domain.ReminderJob) error {
	// Validar que la sesión WA está activa (backoff en IsConnected si falla)
	connected, err := w.waClient.IsConnected(ctx, job.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("send: instancia %s no conectada", job.TenantSlug)
	}

	// Rate limit: máximo 1 mensaje/segundo por tenant
	time.Sleep(1 * time.Second)

	// Formatear hora en el timezone del tenant
	loc, _ := time.LoadLocation(job.TenantTimezone)
	if loc == nil {
		loc = time.UTC
	}
	horaLocal := job.StartsAt.In(loc).Format("3:04 PM")

	// Construir mensaje según el tipo
	var text string
	switch job.Type {
	case "reminder_24h":
		text = fmt.Sprintf(
			"¡Hola %s! 👋 Te recordamos que mañana tienes una cita con %s a las %s para %s. ¿Tienes alguna pregunta? Puedes respondernos aquí.",
			job.CustomerName, job.ProfessionalName, horaLocal, job.ServiceName,
		)
	case "reminder_2h":
		text = fmt.Sprintf(
			"¡Hola %s! ⏰ Tu cita con %s es en 2 horas (%s). ¡Te esperamos! 😊",
			job.CustomerName, job.ProfessionalName, horaLocal,
		)
	default:
		return fmt.Errorf("send: tipo desconocido '%s'", job.Type)
	}

	// Enviar por WhatsApp
	waMessageID, sendErr := w.waClient.SendText(ctx, job.TenantSlug, job.CustomerPhone, text)

	// Log del resultado (exitoso o fallido)
	apptID := job.AppointmentID
	nl := &domain.NotificationLog{
		ID:            uuid.New(),
		TenantID:      job.TenantID,
		AppointmentID: &apptID,
		Type:          job.Type,
		WAMessageID:   waMessageID,
		Content:       text,
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
	} else {
		nl.Status = "sent"
	}

	if err := w.notifRepo.LogNotification(ctx, nl); err != nil {
		slog.Error("ReminderWorker: log notification error", "error", err)
	}

	return sendErr
}
