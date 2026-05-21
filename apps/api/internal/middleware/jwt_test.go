// Tests para JWTMiddlewareWithQuery — verifica el fallback ?token= cuando
// el header Authorization no está presente (necesario para EventSource).
package middleware

import (
	"crypto/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// makeHS256Token genera un JWT HS256 válido firmado con secret.
func makeHS256Token(t *testing.T, secret []byte, sub string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	signed, err := tok.SignedString(secret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

// randSecret genera un secret random para los tests.
func randSecret(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return b
}

// TestJWTMiddlewareWithQuery_FallsBackToQuery verifica que cuando el header
// Authorization no está, el middleware acepta el JWT en ?token=.
func TestJWTMiddlewareWithQuery_FallsBackToQuery(t *testing.T) {
	secret := randSecret(t)
	token := makeHS256Token(t, secret, "user-123")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(JWTMiddlewareWithQuery(string(secret), ""))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString(AuthIDFromContext(c))
	})

	req := httptest.NewRequest("GET", "/test?token="+token, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("esperaba 200 con ?token=, got %d", resp.StatusCode)
	}
}

// TestJWTMiddlewareWithQuery_PrefersHeader verifica que si AMBOS están
// presentes (header y query), el header tiene precedencia — query es solo
// fallback. Esto evita que un atacante con un token expirado en el header
// "actualice" su sesión vía query string.
func TestJWTMiddlewareWithQuery_PrefersHeader(t *testing.T) {
	secret := randSecret(t)
	headerToken := makeHS256Token(t, secret, "user-header")
	queryToken := makeHS256Token(t, secret, "user-query")

	var got string
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(JWTMiddlewareWithQuery(string(secret), ""))
	app.Get("/test", func(c *fiber.Ctx) error {
		got = AuthIDFromContext(c)
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test?token="+queryToken, nil)
	req.Header.Set("Authorization", "Bearer "+headerToken)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("esperaba 200, got %d", resp.StatusCode)
	}
	if got != "user-header" {
		t.Fatalf("esperaba sub=user-header (header gana), got %q", got)
	}
}

// TestJWTMiddlewareWithQuery_RejectsMissing verifica que sin token (ni
// header ni query) responde 401.
func TestJWTMiddlewareWithQuery_RejectsMissing(t *testing.T) {
	secret := randSecret(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(JWTMiddlewareWithQuery(string(secret), ""))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("esperaba 401, got %d", resp.StatusCode)
	}
}

// TestJWTMiddlewareWithQuery_RejectsInvalidToken verifica que un token
// inválido en ?token= sigue siendo rechazado (el fallback no debilita la
// validación, solo cambia el transporte).
func TestJWTMiddlewareWithQuery_RejectsInvalidToken(t *testing.T) {
	secret := randSecret(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(JWTMiddlewareWithQuery(string(secret), ""))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("GET", "/test?token=not-a-jwt", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("esperaba 401 con token inválido, got %d", resp.StatusCode)
	}
}
