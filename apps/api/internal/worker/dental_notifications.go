package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// DentalNotificationsWorker corre 2 crons relacionados con tratamientos:
//   - Post-op WhatsApp 48h después de completar una sesión clínica
//   - Recall 6 meses después de completar un tratamiento
//
// Solo procesa tenants con el módulo "dental" activo (filtrado en el repo).
type DentalNotificationsWorker struct {
	treatmentRepo domain.TreatmentRepository
	sessionRepo   domain.TreatmentSessionRepository
	notifRepo     domain.NotificationRepository
	waClient      domain.WAClient

	postOpInterval time.Duration
	recallInterval time.Duration
	postOpWindow   time.Duration // hace cuánto tiempo se completó la sesión
	postOpLookback time.Duration // hasta cuánto atrás miramos (evita backfill masivo)
	recallWindow   time.Duration // hace cuánto se completó el tratamiento
}

// NewDentalNotificationsWorker crea el worker con defaults productivos.
func NewDentalNotificationsWorker(
	treatmentRepo domain.TreatmentRepository,
	sessionRepo domain.TreatmentSessionRepository,
	notifRepo domain.NotificationRepository,
	waClient domain.WAClient,
) *DentalNotificationsWorker {
	return &DentalNotificationsWorker{
		treatmentRepo:  treatmentRepo,
		sessionRepo:    sessionRepo,
		notifRepo:      notifRepo,
		waClient:       waClient,
		postOpInterval: 1 * time.Hour,
		recallInterval: 24 * time.Hour,
		postOpWindow:   48 * time.Hour,
		postOpLookback: 7 * 24 * time.Hour, // máximo 7 días atrás
		recallWindow:   180 * 24 * time.Hour, // 6 meses aprox
	}
}

// Start arranca ambos crons en goroutines separadas. Bloqueante.
func (w *DentalNotificationsWorker) Start(ctx context.Context) {
	slog.Info("DentalNotificationsWorker: iniciado",
		"post_op_interval", w.postOpInterval, "recall_interval", w.recallInterval)

	go w.loop(ctx, "post_op", w.postOpInterval, w.runPostOp)
	go w.loop(ctx, "recall", w.recallInterval, w.runRecall)

	<-ctx.Done()
	slog.Info("DentalNotificationsWorker: detenido")
}

func (w *DentalNotificationsWorker) loop(ctx context.Context, name string, interval time.Duration, fn func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	fn(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}

// runPostOp busca sessions completadas hace 48h–7d sin notificación y envía
// el mensaje post-operatorio dental.
func (w *DentalNotificationsWorker) runPostOp(ctx context.Context) {
	now := time.Now().UTC()
	completedBefore := now.Add(-w.postOpWindow)
	completedAfter := now.Add(-w.postOpLookback)

	jobs, err := w.sessionRepo.FindPendingPostOpJobs(ctx, completedBefore, completedAfter)
	if err != nil {
		slog.Error("DentalNotificationsWorker.runPostOp: query error", "error", err)
		return
	}

	sent := 0
	for _, j := range jobs {
		if err := w.sendPostOp(ctx, j); err != nil {
			slog.Warn("DentalNotificationsWorker.runPostOp: send failed",
				"session_id", j.SessionID, "tenant_slug", j.TenantSlug, "error", err)
			continue
		}
		if err := w.sessionRepo.MarkPostOpSent(ctx, j.SessionID); err != nil {
			slog.Error("DentalNotificationsWorker.runPostOp: mark sent error",
				"session_id", j.SessionID, "error", err)
		}
		sent++
	}
	if sent > 0 {
		slog.Info("DentalNotificationsWorker.runPostOp: enviados", "count", sent)
	}
}

// runRecall busca treatments completados hace 6 meses sin recall y envía
// el mensaje de control dental.
func (w *DentalNotificationsWorker) runRecall(ctx context.Context) {
	now := time.Now().UTC()
	olderThan := now.Add(-w.recallWindow)

	jobs, err := w.treatmentRepo.FindPendingRecallJobs(ctx, olderThan)
	if err != nil {
		slog.Error("DentalNotificationsWorker.runRecall: query error", "error", err)
		return
	}

	sent := 0
	for _, j := range jobs {
		if err := w.sendRecall(ctx, j); err != nil {
			slog.Warn("DentalNotificationsWorker.runRecall: send failed",
				"treatment_id", j.TreatmentID, "tenant_slug", j.TenantSlug, "error", err)
			continue
		}
		if err := w.treatmentRepo.MarkRecallSent(ctx, j.TreatmentID); err != nil {
			slog.Error("DentalNotificationsWorker.runRecall: mark sent error",
				"treatment_id", j.TreatmentID, "error", err)
		}
		sent++
	}
	if sent > 0 {
		slog.Info("DentalNotificationsWorker.runRecall: enviados", "count", sent)
	}
}

func (w *DentalNotificationsWorker) sendPostOp(ctx context.Context, j *domain.PostOpJob) error {
	connected, err := w.waClient.IsConnected(ctx, j.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("WA instance %s no conectada", j.TenantSlug)
	}
	time.Sleep(1 * time.Second) // rate limit 1msg/s por tenant

	text := fmt.Sprintf(
		"¡Hola %s! 👋 Hace 48 horas estuviste en tu sesión de %s con %s. ¿Cómo te sientes? Si tenés molestias o dudas, respondé este mensaje y te ayudamos. 🦷",
		firstName(j.CustomerName), j.TreatmentName, j.ProfessionalName,
	)

	waMessageID, sendErr := w.waClient.SendText(ctx, j.TenantSlug, j.CustomerPhone, text)

	custID := j.CustomerID
	nl := &domain.NotificationLog{
		ID:         uuid.New(),
		TenantID:   j.TenantID,
		CustomerID: &custID,
		Type:       "treatment_post_op",
		WAMessageID: waMessageID,
		Content:    text,
		SentAt:     time.Now(),
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
	} else {
		nl.Status = "sent"
	}
	if logErr := w.notifRepo.LogNotification(ctx, nl); logErr != nil {
		slog.Warn("DentalNotificationsWorker.sendPostOp: log error", "error", logErr)
	}
	return sendErr
}

func (w *DentalNotificationsWorker) sendRecall(ctx context.Context, j *domain.RecallJob) error {
	connected, err := w.waClient.IsConnected(ctx, j.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("WA instance %s no conectada", j.TenantSlug)
	}
	time.Sleep(1 * time.Second)

	text := fmt.Sprintf(
		"¡Hola %s! 👋 Hace 6 meses terminaste tu tratamiento de %s. ¿Te gustaría agendar un control? Respondé este mensaje y te coordinamos un turno. 🦷",
		firstName(j.CustomerName), j.TreatmentName,
	)

	waMessageID, sendErr := w.waClient.SendText(ctx, j.TenantSlug, j.CustomerPhone, text)

	custID := j.CustomerID
	nl := &domain.NotificationLog{
		ID:         uuid.New(),
		TenantID:   j.TenantID,
		CustomerID: &custID,
		Type:       "treatment_recall_6m",
		WAMessageID: waMessageID,
		Content:    text,
		SentAt:     time.Now(),
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
	} else {
		nl.Status = "sent"
	}
	if logErr := w.notifRepo.LogNotification(ctx, nl); logErr != nil {
		slog.Warn("DentalNotificationsWorker.sendRecall: log error", "error", logErr)
	}
	return sendErr
}

// firstName extrae el primer token del nombre completo para mensajes amigables.
func firstName(full string) string {
	for i, c := range full {
		if c == ' ' {
			return full[:i]
		}
	}
	return full
}
