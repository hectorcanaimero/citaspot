// Package handler — HTTP handler para streams en tiempo real (SSE).
//
// Phase C: suscripción por conexión a Redis Pub/Sub. Cada cliente abre su
// propia suscripción al canal `tenant:{tenantID}:appointments` — el tenant
// se deriva SIEMPRE del contexto autenticado, NUNCA de la URL.
package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"

	"github.com/citaspot/api/internal/middleware"
)

// RealtimeHandler agrupa los endpoints SSE del dashboard.
//
// rdb es opcional al construir el handler — si es nil, /realtime/appointments
// responderá 503 (Redis es prerequisito para pub/sub). En producción siempre
// debe estar conectado; en tests con miniredis se inyecta un cliente válido.
type RealtimeHandler struct {
	rdb *redis.Client
}

// NewRealtimeHandler crea el handler de SSE con un cliente Redis.
func NewRealtimeHandler(rdb *redis.Client) *RealtimeHandler {
	return &RealtimeHandler{rdb: rdb}
}

// Appointments GET /api/v1/realtime/appointments
//
// Stream SSE de eventos de citas para el tenant autenticado. Se suscribe al
// canal Redis `tenant:{tenantID}:appointments` y reenvía cada mensaje
// recibido como un evento SSE. La conexión se cierra limpiamente cuando el
// cliente desconecta (UserContext().Done()).
//
// Formato del payload publicado por el service (ver service/appointments.go):
//
//	{"event":"appointment.created|updated|cancelled|rescheduled",
//	 "data": <AppointmentWithDetails>, "ts":"<RFC3339>"}
//
// Salida SSE:
//
//	event: appointment.created
//	data: {"event":"appointment.created","data":{...},"ts":"..."}
//
// El payload de Redis se reenvía VERBATIM como `data:` — no se reserializa,
// solo se extrae el campo `event` para el header SSE.
func (h *RealtimeHandler) Appointments(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(fiber.StatusForbidden, "tenant no identificado")
	}
	if h.rdb == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "realtime no disponible")
	}

	// Cabeceras SSE — Cache-Control y X-Accel-Buffering evitan que proxies
	// (nginx, Cloudflare) bufferen la respuesta y rompan el stream.
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	channel := fmt.Sprintf("tenant:%s:appointments", tenantID)
	// Capturamos UserContext ANTES de SetBodyStreamWriter — fasthttp recicla
	// c.Context() después de que el handler retorna, por lo que dentro del
	// stream writer no podemos depender del request context original.
	ctx := c.UserContext()
	rdb := h.rdb

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Suscripción PER-CONEXIÓN — NUNCA compartir entre clientes.
		// Cada tenant tiene su propia suscripción a su propio canal.
		pubsub := rdb.Subscribe(ctx, channel)
		defer pubsub.Close()

		// Mensaje inicial — confirma al cliente que la conexión está viva
		// y fuerza al proxy a comprometer las cabeceras SSE.
		if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		msgCh := pubsub.Channel()
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Cliente desconectado — salimos limpiamente.
				return

			case <-ticker.C:
				// Comentario SSE (línea iniciada con `:`) — el cliente la
				// ignora, pero mantiene la conexión TCP viva a través de
				// proxies con timeout corto (típico nginx 60s).
				if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

			case m, ok := <-msgCh:
				if !ok {
					// Canal cerrado (pubsub.Close() llamado) — salimos.
					return
				}
				// Extraer el campo `event` para el header SSE. El payload
				// completo se reenvía verbatim en `data:`.
				var env struct {
					Event string `json:"event"`
				}
				if err := json.Unmarshal([]byte(m.Payload), &env); err != nil {
					slog.Warn("realtime: payload sin event válido",
						slog.String("channel", channel),
						slog.String("err", err.Error()),
					)
				}
				if env.Event == "" {
					env.Event = "appointment.event"
				}
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", env.Event, m.Payload); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}
