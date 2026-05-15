package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// retentionByPlan define los días de retención por plan de suscripción.
var retentionByPlan = map[string]int{
	"basic":   90,
	"starter": 90,
	"trial":   90,
	"pro":     365,
	// enterprise: sin límite — no aparece en el mapa
}

// EventsMaintenanceWorker ejecuta mantenimiento de la tabla events:
// - Diario: limpieza de eventos expirados según el plan del tenant.
// - Mensual: creación de particiones futuras.
type EventsMaintenanceWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
}

func NewEventsMaintenanceWorker(pool *pgxpool.Pool) *EventsMaintenanceWorker {
	return &EventsMaintenanceWorker{
		pool:     pool,
		interval: 24 * time.Hour,
	}
}

// Start inicia el cron en una goroutine. Bloqueante — llamar con go.
func (w *EventsMaintenanceWorker) Start(ctx context.Context) {
	slog.Info("EventsMaintenanceWorker: iniciado (cada 24h)")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Ejecutar inmediatamente al iniciar: asegurar particiones futuras
	w.ensurePartitions(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("EventsMaintenanceWorker: detenido")
			return
		case <-ticker.C:
			w.cleanupExpired(ctx)
			w.ensurePartitions(ctx)
		}
	}
}

// cleanupExpired elimina eventos que superan el período de retención del plan del tenant.
func (w *EventsMaintenanceWorker) cleanupExpired(ctx context.Context) {
	for plan, days := range retentionByPlan {
		result, err := w.pool.Exec(ctx, `
			DELETE FROM events
			WHERE tenant_id IN (
				SELECT id FROM tenants WHERE plan = $1
			)
			AND occurred_at < NOW() - make_interval(days => $2)
		`, plan, days)
		if err != nil {
			slog.Warn("EventsMaintenanceWorker.cleanupExpired: error", "plan", plan, "error", err)
			continue
		}
		if result.RowsAffected() > 0 {
			slog.Info("EventsMaintenanceWorker.cleanupExpired: deleted", "plan", plan, "rows", result.RowsAffected())
		}
	}
}

// ensurePartitions crea particiones para los próximos 2 meses si no existen.
func (w *EventsMaintenanceWorker) ensurePartitions(ctx context.Context) {
	now := time.Now()

	for i := 0; i <= 2; i++ {
		month := now.AddDate(0, i, 0)
		partName := fmt.Sprintf("events_y%dm%02d", month.Year(), month.Month())
		rangeStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		rangeEnd := rangeStart.AddDate(0, 1, 0)

		var exists bool
		err := w.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_class WHERE relname = $1
			)
		`, partName).Scan(&exists)
		if err != nil {
			slog.Warn("EventsMaintenanceWorker.ensurePartitions: check error", "partition", partName, "error", err)
			continue
		}
		if exists {
			continue
		}

		_, err = w.pool.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE %s PARTITION OF events FOR VALUES FROM ('%s') TO ('%s')`,
			partName,
			rangeStart.Format("2006-01-02"),
			rangeEnd.Format("2006-01-02"),
		))
		if err != nil {
			slog.Warn("EventsMaintenanceWorker.ensurePartitions: create error", "partition", partName, "error", err)
		} else {
			slog.Info("EventsMaintenanceWorker.ensurePartitions: created", "partition", partName)
		}
	}
}
