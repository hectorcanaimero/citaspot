package middleware

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// RequireModule devuelve un middleware que bloquea con 403 si el tenant del
// request no tiene el módulo activo. Debe ejecutarse DESPUÉS de TenantMiddleware
// (necesita TenantIDFromContext resuelto).
//
// Ejemplo de uso:
//
//	treatments := protected.Group("/treatments",
//	    middleware.RequireModule(moduleRepo, domain.ModuleDental),
//	)
func RequireModule(repo domain.TenantModuleRepository, moduleKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := TenantIDFromContext(c)
		if tenantID == uuid.Nil {
			// TenantMiddleware no se ejecutó o no resolvió tenant
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "tenant no identificado",
			})
		}

		active, err := repo.IsActive(c.Context(), tenantID, moduleKey)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "error verificando módulo",
			})
		}
		if !active {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error":  "módulo no activo para este tenant",
				"module": moduleKey,
			})
		}
		return c.Next()
	}
}
