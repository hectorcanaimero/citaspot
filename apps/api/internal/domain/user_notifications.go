// Package domain — tipos para notificaciones in-app del dashboard.
//
// Estas notificaciones son distintas de NotificationLog (que trackea mensajes
// WhatsApp salientes hacia clientes). UserNotification es el feed de novedades
// que ve el owner/admin en el bell-icon del dashboard.
package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Tipos de notificación in-app soportados.
// Mantenerlos sincronizados con el frontend (lib/i18n para iconos/colores).
const (
	UserNotificationAppointmentCreated     = "appointment.created"
	UserNotificationAppointmentCancelled   = "appointment.cancelled"
	UserNotificationAppointmentRescheduled = "appointment.rescheduled"
	UserNotificationWhatsAppInbound        = "whatsapp.inbound"
)

// ErrUserNotificationNotFound se retorna cuando la notificación no existe
// o ya estaba marcada como leída (en operaciones que mutan).
var ErrUserNotificationNotFound = errors.New("notificación no encontrada")

// UserNotification es una entrada en el feed de notificaciones del dashboard.
// El campo Metadata es JSONB y puede incluir IDs y nombres del recurso
// asociado (appointment, customer, professional) para que el frontend pueda
// linkear a la pantalla correspondiente sin un fetch adicional.
type UserNotification struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// UserNotificationRepository persistencia del feed in-app.
// Todas las operaciones aplican RLS por tenant.
type UserNotificationRepository interface {
	// Create persiste una nueva notificación. Asigna ID y CreatedAt al struct
	// pasado (si vienen vacíos).
	Create(ctx context.Context, n *UserNotification) error

	// ListByTenant retorna notificaciones del tenant ordenadas por created_at DESC,
	// con paginación tipo cursor (cursor = created_at del último item de la página
	// previa; nil para la primera página). nextCursor es nil cuando ya no hay más.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int, cursor *time.Time) ([]*UserNotification, *time.Time, error)

	// UnreadCount retorna el número de notificaciones sin leer del tenant.
	UnreadCount(ctx context.Context, tenantID uuid.UUID) (int, error)

	// MarkRead marca una notificación específica como leída. Retorna
	// ErrUserNotificationNotFound si la fila no existe (o ya estaba leída).
	MarkRead(ctx context.Context, tenantID, id uuid.UUID) error

	// MarkAllRead marca todas las no-leídas del tenant como leídas y retorna
	// la cantidad de filas afectadas.
	MarkAllRead(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// UserNotificationSvc lógica de negocio del feed in-app.
// El service persiste en DB y publica el evento a Redis Pub/Sub para que
// el dashboard reciba el push vía SSE en tiempo real.
type UserNotificationSvc interface {
	// Create crea, persiste y publica una notificación. Best-effort en el
	// publish — si Redis falla, el item queda en DB y se recupera al recargar.
	Create(ctx context.Context, tenantID uuid.UUID, notifType, title, body string, metadata map[string]any) (*UserNotification, error)

	List(ctx context.Context, tenantID uuid.UUID, limit int, cursor *time.Time) ([]*UserNotification, *time.Time, error)
	UnreadCount(ctx context.Context, tenantID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, tenantID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, tenantID uuid.UUID) (int, error)
}
