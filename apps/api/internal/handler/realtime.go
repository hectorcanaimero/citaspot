// Package handler — HTTP handler para streams en tiempo real (SSE).
//
// Phase C: suscripción por conexión a Redis Pub/Sub. Cada cliente abre su
// propia suscripción a los canales `tenant:{tenantID}:appointments` y
// `tenant:{tenantID}:notifications` — el tenant se deriva SIEMPRE del
// contexto autenticado, NUNCA de la URL. Los dos canales viajan por el mismo
// stream SSE; el campo `event` del payload (appointment.* | notification.*)
// le indica al cliente qué tipo de evento es.
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
// Stream SSE multiplexado para el tenant autenticado. Se suscribe a DOS canales
// Redis con la misma suscripción pub/sub:
//
//   - `tenant:{tenantID}:appointments` — eventos de citas (appointment.*)
//   - `tenant:{tenantID}:notifications` — feed in-app (notification.created)
//
// Por compatibilidad histórica la URL sigue siendo `/realtime/appointments`:
// el cliente diferencia eventos por el campo `event` del payload, no por la URL.
//
// La conexión se cierra limpiamente cuando el cliente desconecta
// (UserContext().Done()).
//
// Formato del payload publicado por los services:
//
//	{"event":"appointment.created|updated|cancelled|rescheduled|notification.created",
//	 "data": <...>, "ts":"<RFC3339>"}
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

	apptChannel := fmt.Sprintf("tenant:%s:appointments", tenantID)
	notifChannel := fmt.Sprintf("tenant:%s:notifications", tenantID)
	// Capturamos UserContext ANTES de SetBodyStreamWriter — fasthttp recicla
	// c.Context() después de que el handler retorna, por lo que dentro del
	// stream writer no podemos depender del request context original.
	ctx := c.UserContext()
	rdb := h.rdb

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Suscripción PER-CONEXIÓN — NUNCA compartir entre clientes.
		// go-redis acepta múltiples canales en una sola Subscribe — los
		// mensajes de ambos canales fluyen por el mismo Channel().
		pubsub := rdb.Subscribe(ctx, apptChannel, notifChannel)
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
						slog.String("channel", m.Channel),
						slog.String("err", err.Error()),
					)
				}
				if env.Event == "" {
					// Fallback por canal de origen si el publisher omitió `event`.
					if m.Channel == notifChannel {
						env.Event = "notification.event"
					} else {
						env.Event = "appointment.event"
					}
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
