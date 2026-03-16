package middleware

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// PlanCheckMiddleware verifica que el tenant tenga un plan activo.
// Se aplica solo a rutas del dashboard — no a /public/*, /billing/webhook, /health.
//
// Estados:
//   - active / trialing → permitir
//   - past_due          → permitir, añade header X-Plan-Warning para que el frontend muestre aviso
//   - cancelled         → 402 Payment Required
func PlanCheckMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenant := TenantFromContext(c)
		if tenant == nil {
			return c.Next()
		}

		switch tenant.PlanStatus {
		case "cancelled":
			return c.Status(http.StatusPaymentRequired).JSON(fiber.Map{
				"code":    "SUBSCRIPTION_CANCELLED",
				"message": "Tu suscripción ha sido cancelada. Renuévala en Configuración → Suscripción.",
			})
		case "past_due":
			// Avisar al frontend sin bloquear — el usuario puede seguir operando
			c.Set("X-Plan-Warning", "past_due")
		}

		return c.Next()
	}
}
