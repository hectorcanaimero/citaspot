package middleware

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

// TenantMiddleware resuelve el tenant del request a partir del auth_id (JWT sub).
// Debe ejecutarse DESPUÉS de JWTMiddleware.
//
// Por cada request protegido:
//  1. Lee el auth_id del contexto (puesto por JWTMiddleware)
//  2. Busca user + tenant en la DB por auth_id
//  3. Abre una conexión del pool y ejecuta SET app.tenant_id = $1
//  4. Almacena user y tenant en el contexto de Fiber
func TenantMiddleware(repo domain.AuthRepository, pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authID := AuthIDFromContext(c)
		if authID == "" {
			// JWTMiddleware debe ejecutarse primero
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "No autorizado",
			})
		}

		// Resolver usuario y tenant desde la DB
		user, tenant, err := repo.FindUserByAuthID(c.Context(), authID)
		if err != nil {
			if err == domain.ErrNotFound {
				return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
					"error": "Usuario no encontrado",
				})
			}
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error interno del servidor",
			})
		}

		// Setear app.tenant_id en la sesión de DB para activar RLS.
		// Obtenemos una conexión del pool, seteamos la variable de sesión
		// y la liberamos — las queries posteriores del handler deben hacer
		// lo mismo a través del helper SetTenantID.
		conn, err := pool.Acquire(c.Context())
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error interno del servidor",
			})
		}
		_, err = conn.Exec(c.Context(),
			"SELECT set_config('app.tenant_id', $1, true)", tenant.ID.String(),
		)
		conn.Release()
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error interno del servidor",
			})
		}

		// Almacenar en el contexto para uso downstream
		c.Locals(ctxKeyUser, user)
		c.Locals(ctxKeyTenant, tenant)
		c.Locals(ctxKeyTenantID, tenant.ID)

		return c.Next()
	}
}

// SetTenantID ejecuta SET app.tenant_id en la conexión del pool antes de queries.
// Los repositories deben llamar esto al inicio de cada operación protegida por RLS.
// Uso: defer conn.Release() después de llamar esta función.
//
//	conn, err := pool.Acquire(ctx)
//	if err != nil { ... }
//	defer conn.Release()
//	if err := middleware.SetTenantID(ctx, conn, tenantID.String()); err != nil { ... }
func SetTenantID(c *fiber.Ctx, pool *pgxpool.Pool) (*pgxpool.Conn, error) {
	tenantID := TenantIDFromContext(c)
	if tenantID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, domain.ErrUnauthorized
	}

	conn, err := pool.Acquire(c.Context())
	if err != nil {
		return nil, fmt.Errorf("SetTenantID: acquire: %w", err)
	}

	_, err = conn.Exec(c.Context(),
		"SELECT set_config('app.tenant_id', $1, true)", tenantID.String(),
	)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("SetTenantID: set: %w", err)
	}

	return conn, nil
}
