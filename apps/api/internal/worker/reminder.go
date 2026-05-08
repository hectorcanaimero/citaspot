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

func (w *ReminderWorker) run(ctx context.Context) {
	minutes, err := w.reminderRepo.GetDistinctReminderMinutes(ctx)
	if err != nil {
		slog.Error("ReminderWorker: GetDistinctReminderMinutes error", "error", err)
		return
	}

	totalSent := 0
	for _, m := range minutes {
		jobs, err := w.reminderRepo.FindDueReminders(ctx, m)
		if err != nil {
			slog.Error("ReminderWorker: FindDueReminders error", "minutes", m, "error", err)
			continue
		}
		for _, job := range jobs {
			if err := w.send(ctx, job, m); err != nil {
				slog.Error("ReminderWorker: send error", "appointment", job.AppointmentID, "minutes", m, "error", err)
				continue
			}
			if err := w.notifRepo.MarkReminderSent(ctx, job.AppointmentID, m); err != nil {
				slog.Error("ReminderWorker: mark sent error", "error", err)
			}
			totalSent++
		}
	}
	if totalSent > 0 {
		slog.Info("ReminderWorker: recordatorios procesados", "total", totalSent)
	}
}

// send envía el recordatorio vía WhatsApp y registra el log.
func (w *ReminderWorker) send(ctx context.Context, job *domain.ReminderJob, minutesBefore int) error {
	connected, err := w.waClient.IsConnected(ctx, job.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("send: instancia %s no conectada", job.TenantSlug)
	}

	time.Sleep(1 * time.Second)

	loc, _ := time.LoadLocation(job.TenantTimezone)
	if loc == nil {
		loc = time.UTC
	}
	horaLocal := job.StartsAt.In(loc).Format("3:04 PM")

	var text string
	switch {
	case minutesBefore >= 1440:
		text = fmt.Sprintf(
			"¡Hola %s! 👋 Te recordamos que mañana tienes una cita con %s a las %s para %s. ¿Tienes alguna pregunta? Puedes respondernos aquí.",
			job.CustomerName, job.ProfessionalName, horaLocal, job.ServiceName,
		)
	case minutesBefore >= 60:
		hours := minutesBefore / 60
		text = fmt.Sprintf(
			"¡Hola %s! ⏰ Tu cita con %s es en %d hora(s) (%s). ¡Te esperamos! 😊",
			job.CustomerName, job.ProfessionalName, hours, horaLocal,
		)
	default:
		text = fmt.Sprintf(
			"¡Hola %s! ⏰ Tu cita con %s es en %d minutos (%s). ¡Te esperamos! 😊",
			job.CustomerName, job.ProfessionalName, minutesBefore, horaLocal,
		)
	}

	waMessageID, sendErr := w.waClient.SendText(ctx, job.TenantSlug, job.CustomerPhone, text)

	apptID := job.AppointmentID
	nl := &domain.NotificationLog{
		ID:            uuid.New(),
		TenantID:      job.TenantID,
		AppointmentID: &apptID,
		Type:          fmt.Sprintf("reminder_%d", minutesBefore),
		WAMessageID:   waMessageID,
		Content:       text,
		SentAt:        time.Now(),
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
