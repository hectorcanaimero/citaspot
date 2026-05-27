package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// UserNotificationHandler expone el feed de notificaciones in-app.
//
// Acceso restringido a owner/admin: staff no debe ver el feed agregado del
// tenant (políticas RBAC del negocio — staff opera sobre sus propias tareas).
type UserNotificationHandler struct {
	svc domain.UserNotificationSvc
}

// NewUserNotificationHandler crea el handler.
func NewUserNotificationHandler(svc domain.UserNotificationSvc) *UserNotificationHandler {
	return &UserNotificationHandler{svc: svc}
}

// canViewNotifications retorna true si el usuario tiene rol owner o admin.
// Defensa en profundidad: el middleware ya validó JWT + tenant, aquí
// chequeamos rol para el endpoint específico.
func canViewNotifications(c *fiber.Ctx) bool {
	u := middleware.UserFromContext(c)
	if u == nil {
		return false
	}
	return u.Role == "owner" || u.Role == "admin"
}

// List GET /api/v1/notifications?limit=20&cursor=2026-05-20T10:00:00Z
//
// Respuesta:
//
//	{ "items": [UserNotification...], "nextCursor": "2026-...Z" | null }
func (h *UserNotificationHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Tenant no identificado"))
	}
	if !canViewNotifications(c) {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Acceso denegado"))
	}

	// limit: default 20, max 50
	limit := 20
	if q := c.Query("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			limit = v
		}
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// cursor: opcional, RFC3339
	var cursor *time.Time
	if q := c.Query("cursor"); q != "" {
		t, err := time.Parse(time.RFC3339, q)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				newError("validation_error", "cursor inválido (usar RFC3339)"),
			)
		}
		cursor = &t
	}

	items, nextCursor, err := h.svc.List(c.Context(), tenantID, limit, cursor)
	if err != nil {
		return handleServiceError(c, err)
	}

	// Garantizar []  (no null) cuando no hay items
	if items == nil {
		items = []*domain.UserNotification{}
	}

	resp := fiber.Map{"items": items}
	if nextCursor != nil {
		resp["nextCursor"] = nextCursor.Format(time.RFC3339Nano)
	} else {
		resp["nextCursor"] = nil
	}
	return c.JSON(resp)
}

// UnreadCount GET /api/v1/notifications/unread-count
//
// Respuesta: { "count": N }
func (h *UserNotificationHandler) UnreadCount(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Tenant no identificado"))
	}
	if !canViewNotifications(c) {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Acceso denegado"))
	}

	count, err := h.svc.UnreadCount(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"count": count})
}

// MarkRead POST /api/v1/notifications/:id/read
//
// 204 No Content. Retorna 404 si la notificación no existe o ya estaba leída.
func (h *UserNotificationHandler) MarkRead(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Tenant no identificado"))
	}
	if !canViewNotifications(c) {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Acceso denegado"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(newError("validation_error", "ID inválido"))
	}

	if err := h.svc.MarkRead(c.Context(), tenantID, id); err != nil {
		if errors.Is(err, domain.ErrUserNotificationNotFound) {
			return c.Status(http.StatusNotFound).JSON(newError("not_found", "Notificación no encontrada"))
		}
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// MarkAllRead POST /api/v1/notifications/read-all
//
// Respuesta: { "affected": N }
func (h *UserNotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Tenant no identificado"))
	}
	if !canViewNotifications(c) {
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Acceso denegado"))
	}

	affected, err := h.svc.MarkAllRead(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"affected": affected})
}
