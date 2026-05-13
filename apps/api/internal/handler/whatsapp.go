package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// qrStore guarda el último QR base64 por slug de tenant.
// TTL implícito: se borra cuando el tenant se conecta o desconecta.
var qrStore sync.Map // map[string]string: slug → base64 QR

// WhatsAppHandler maneja el webhook de Evolution API.
type WhatsAppHandler struct {
	svc           domain.WhatsAppSvc
	waClient      domain.WAClient
	webhookSecret string // Evolution API key usada para validar el webhook (global)
}

// NewWhatsAppHandler crea el handler del webhook de WhatsApp.
func NewWhatsAppHandler(svc domain.WhatsAppSvc, waClient domain.WAClient, webhookSecret string) *WhatsAppHandler {
	return &WhatsAppHandler{svc: svc, waClient: waClient, webhookSecret: webhookSecret}
}

// Status GET /api/v1/whatsapp/status
// Consulta el estado de la sesión de WhatsApp del tenant en Evolution API.
func (h *WhatsAppHandler) Status(c *fiber.Ctx) error {
	tenant := middleware.TenantFromContext(c)
	if tenant == nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	connected, err := h.waClient.IsConnected(c.Context(), tenant.Slug)
	if err != nil {
		slog.Warn("whatsapp.Status: evolution error", "tenant", tenant.Slug, "error", err)
		return c.JSON(fiber.Map{"status": "DISCONNECTED", "instance": tenant.Slug})
	}

	status := "DISCONNECTED"
	if connected {
		status = "CONNECTED"
		qrStore.Delete(tenant.Slug) // limpiar QR si ya está conectado
	}
	return c.JSON(fiber.Map{"status": status, "instance": tenant.Slug})
}

// Connect POST /api/v1/whatsapp/connect
// Crea la instancia en Evolution API. El QR llegará vía webhook (evento qrcode.updated).
// El frontend debe hacer polling a GET /whatsapp/qr para obtenerlo.
func (h *WhatsAppHandler) Connect(c *fiber.Ctx) error {
	tenant := middleware.TenantFromContext(c)
	if tenant == nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	// Limpiar QR previo antes de reconectar
	qrStore.Delete(tenant.Slug)

	if err := h.waClient.Connect(c.Context(), tenant.Slug); err != nil {
		slog.Error("whatsapp.Connect: error", "tenant", tenant.Slug, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al iniciar la conexión con WhatsApp"})
	}

	// Intentar capturar el QR inmediatamente (best-effort).
	// Si Evolution ya generó el QR, lo tenemos disponible sin esperar el webhook.
	if qr, err := h.waClient.FetchQR(c.Context(), tenant.Slug); err == nil && qr != "" {
		qrStore.Store(tenant.Slug, qr)
		slog.Info("whatsapp.Connect: QR capturado directamente", "tenant", tenant.Slug)
	}

	return c.JSON(fiber.Map{"status": "CONNECTING", "instance": tenant.Slug})
}

// GetQR GET /api/v1/whatsapp/qr
// Retorna el QR más reciente para el tenant (almacenado al recibir el webhook qrcode.updated).
// El frontend hace polling a este endpoint hasta obtener el QR o CONNECTED status.
func (h *WhatsAppHandler) GetQR(c *fiber.Ctx) error {
	tenant := middleware.TenantFromContext(c)
	if tenant == nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	qr, ok := qrStore.Load(tenant.Slug)
	if !ok || qr == "" {
		// Fallback: consultar Evolution API directamente.
		// Útil en desarrollo donde el webhook QRCODE_UPDATED puede no llegar.
		fetchedQR, err := h.waClient.FetchQR(c.Context(), tenant.Slug)
		if err != nil {
			slog.Warn("whatsapp.GetQR: FetchQR fallback error", "tenant", tenant.Slug, "error", err)
		}
		if fetchedQR != "" {
			qrStore.Store(tenant.Slug, fetchedQR)
			return c.JSON(fiber.Map{"qr": fetchedQR})
		}
		// QR aún no disponible — el frontend vuelve a intentar
		return c.Status(http.StatusNoContent).Send(nil)
	}

	return c.JSON(fiber.Map{"qr": qr})
}

// Disconnect DELETE /api/v1/whatsapp/disconnect
// Cierra la sesión de WhatsApp del tenant.
func (h *WhatsAppHandler) Disconnect(c *fiber.Ctx) error {
	tenant := middleware.TenantFromContext(c)
	if tenant == nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	qrStore.Delete(tenant.Slug)

	if err := h.waClient.Disconnect(c.Context(), tenant.Slug); err != nil {
		slog.Error("whatsapp.Disconnect: error", "tenant", tenant.Slug, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al desconectar WhatsApp"})
	}

	return c.SendStatus(http.StatusNoContent)
}

// Webhook POST /api/v1/whatsapp/webhook
// Evolution API envía todos los eventos aquí con el global API key.
// Maneja: messages.upsert, qrcode.updated, connection.update.
func (h *WhatsAppHandler) Webhook(c *fiber.Ctx) error {
	// Validar webhook secret si WEBHOOK_SECRET está configurado.
	// En desarrollo dentro de Docker se deja vacío para que Evolution pueda llamar sin header adicional.
	if h.webhookSecret != "" {
		if apiKey := c.Get("apikey"); apiKey != h.webhookSecret {
			return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "Webhook no autorizado"})
		}
	}

	var payload map[string]any
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	event, _ := payload["event"].(string)
	instanceName, _ := payload["instance"].(string)

	// Normalizar: Evolution v2 puede enviar "QRCODE_UPDATED" o "qrcode.updated"
	eventNorm := strings.ToLower(strings.ReplaceAll(event, "_", "."))
	slog.Info("whatsapp.Webhook: evento recibido", "event", event, "instance", instanceName)

	switch eventNorm {
	case "qrcode.updated":
		// Almacenar el QR para que el frontend lo obtenga vía GET /whatsapp/qr
		if instanceName != "" {
			if data, ok := payload["data"].(map[string]any); ok {
				if qr, ok := data["qrcode"].(map[string]any); ok {
					if base64, ok := qr["base64"].(string); ok && base64 != "" {
						qrStore.Store(instanceName, base64)
						slog.Info("whatsapp.Webhook: QR recibido", "instance", instanceName)
					}
				}
			}
		}

	case "connection.update":
		// Limpiar QR y persistir estado cuando la sesión cambia
		if instanceName != "" {
			if data, ok := payload["data"].(map[string]any); ok {
				state, _ := data["state"].(string)
				if state == "open" {
					qrStore.Delete(instanceName)
					slog.Info("whatsapp.Webhook: conexión establecida", "instance", instanceName)
				}
				// Persistir estado en DB para "open" (connected) y "close" (disconnected)
				if state == "open" || state == "close" {
					go func(inst, st string) {
						if err := h.svc.HandleConnectionUpdate(context.Background(), inst, st); err != nil {
							slog.Error("webhook.HandleConnectionUpdate error", "instance", inst, "error", err)
						}
					}(instanceName, state)
				}
			}
		}

	case "messages.upsert":
		// Procesar mensaje entrante de forma asíncrona
		if instanceName == "" {
			return c.SendStatus(http.StatusOK)
		}
		go func() {
			// c.Context() es fasthttp.RequestCtx — Fiber lo recicla al retornar el handler.
			// Usar context.Background() para evitar data race en la goroutine.
			if err := h.svc.ProcessInbound(context.Background(), instanceName, payload); err != nil {
				slog.Error("webhook.ProcessInbound error", "instance", instanceName, "error", err)
			}
		}()
	}

	return c.SendStatus(http.StatusOK)
}
