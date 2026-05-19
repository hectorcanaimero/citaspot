package handler

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/citaspot/api/internal/domain"
)

// WaitlistHandler maneja los endpoints de la lista de espera pre-launch (sin auth).
type WaitlistHandler struct {
	svc domain.WaitlistSvc
}

// NewWaitlistHandler crea el handler de la lista de espera.
func NewWaitlistHandler(svc domain.WaitlistSvc) *WaitlistHandler {
	return &WaitlistHandler{svc: svc}
}

// Join POST /api/v1/public/waitlist
// Body: { "email": "...", "business_name": "..." }
// Respuestas:
//   - 201 Created    → { id, email, business_name, created_at }
//   - 400 Bad Request → email inválido o business_name vacío
//   - 409 Conflict   → email ya registrado en la lista
func (h *WaitlistHandler) Join(c *fiber.Ctx) error {
	var req domain.JoinWaitlistInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(newError("invalid_json", "JSON inválido"))
	}

	// Capturar IP del request. Prioridad: X-Forwarded-For (primer hop) → c.IP().
	// Importante: detrás de un proxy/load-balancer X-Forwarded-For tiene el
	// formato "ip1, ip2, ip3" donde ip1 es el cliente real.
	req.IPAddress = clientIP(c)
	req.UserAgent = string(c.Request().Header.UserAgent())

	signup, err := h.svc.Join(c.Context(), &req)
	if err != nil {
		return handleServiceError(c, err)
	}

	// No devolvemos ip_address ni user_agent en la respuesta — son metadatos internos.
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":            signup.ID,
		"email":         signup.Email,
		"business_name": signup.BusinessName,
		"created_at":    signup.CreatedAt,
	})
}

// clientIP extrae la IP del cliente respetando X-Forwarded-For si está presente.
// En desarrollo (sin proxy) devuelve c.IP() directamente.
func clientIP(c *fiber.Ctx) string {
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For: "client, proxy1, proxy2" → tomar el primero
		if idx := strings.Index(xff, ","); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	return c.IP()
}
