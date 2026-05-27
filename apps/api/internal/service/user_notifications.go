// Package service — feed in-app de notificaciones del dashboard.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/citaspot/api/internal/domain"
)

type userNotificationSvc struct {
	repo domain.UserNotificationRepository
	// rdb opcional. Si es nil, Create persiste en DB pero no publica al canal
	// Redis — el cliente verá la notificación al recargar (no en realtime).
	rdb *redis.Client
}

// NewUserNotificationSvc crea el servicio de notificaciones in-app.
func NewUserNotificationSvc(repo domain.UserNotificationRepository, rdb *redis.Client) domain.UserNotificationSvc {
	return &userNotificationSvc{repo: repo, rdb: rdb}
}

// Create persiste la notificación en DB y publica al canal Redis Pub/Sub
// `tenant:{id}:notifications`. El publish es best-effort: si falla, la fila
// queda en DB y el cliente la verá al recargar.
func (s *userNotificationSvc) Create(
	ctx context.Context,
	tenantID uuid.UUID,
	notifType, title, body string,
	metadata map[string]any,
) (*domain.UserNotification, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("userNotificationSvc.Create: marshal metadata: %w", err)
	}

	n := &domain.UserNotification{
		TenantID: tenantID,
		Type:     notifType,
		Title:    title,
		Body:     body,
		Metadata: metaBytes,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("userNotificationSvc.Create: %w", err)
	}

	// Publish realtime — el mismo wire format que appointments para que la
	// SSE handler única pueda enrutar por el campo `event`.
	s.publishRealtime(ctx, tenantID, n)

	return n, nil
}

// List delega al repo. El handler ya clampó limit/cursor — aquí no aplicamos
// reglas de negocio adicionales.
func (s *userNotificationSvc) List(
	ctx context.Context, tenantID uuid.UUID, limit int, cursor *time.Time,
) ([]*domain.UserNotification, *time.Time, error) {
	return s.repo.ListByTenant(ctx, tenantID, limit, cursor)
}

func (s *userNotificationSvc) UnreadCount(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, tenantID)
}

func (s *userNotificationSvc) MarkRead(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.MarkRead(ctx, tenantID, id)
}

func (s *userNotificationSvc) MarkAllRead(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return s.repo.MarkAllRead(ctx, tenantID)
}

// publishRealtime publica el evento al canal Redis del tenant.
// Best-effort: nunca falla la operación principal.
func (s *userNotificationSvc) publishRealtime(ctx context.Context, tenantID uuid.UUID, n *domain.UserNotification) {
	if s.rdb == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"event": "notification.created",
		"data":  n,
		"ts":    time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.Warn("userNotificationSvc.publishRealtime: marshal", "error", err)
		return
	}
	channel := fmt.Sprintf("tenant:%s:notifications", tenantID)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		slog.Warn("userNotificationSvc.publishRealtime: publish", "channel", channel, "error", err)
	}
}
