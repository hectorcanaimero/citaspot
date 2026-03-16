// Package middleware contiene los middlewares de Fiber para auth y tenant resolution.
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// Claves para los valores almacenados en el contexto de Fiber.
const (
	ctxKeyAuthID   = "auth_id"    // Supabase Auth UID (string)
	ctxKeyUser     = "user"       // *domain.User
	ctxKeyTenant   = "tenant"     // *domain.Tenant
	ctxKeyTenantID = "tenant_id"  // uuid.UUID
)

// AuthIDFromContext retorna el Supabase Auth UID del contexto.
// Populated por JWTMiddleware.
func AuthIDFromContext(c *fiber.Ctx) string {
	v, _ := c.Locals(ctxKeyAuthID).(string)
	return v
}

// UserFromContext retorna el usuario autenticado del contexto.
// Populated por TenantMiddleware.
func UserFromContext(c *fiber.Ctx) *domain.User {
	u, _ := c.Locals(ctxKeyUser).(*domain.User)
	return u
}

// TenantFromContext retorna el tenant del request actual.
// Populated por TenantMiddleware.
func TenantFromContext(c *fiber.Ctx) *domain.Tenant {
	t, _ := c.Locals(ctxKeyTenant).(*domain.Tenant)
	return t
}

// TenantIDFromContext retorna el UUID del tenant del request actual.
// Populated por TenantMiddleware.
func TenantIDFromContext(c *fiber.Ctx) uuid.UUID {
	id, _ := c.Locals(ctxKeyTenantID).(uuid.UUID)
	return id
}
