// Helpers y fixtures comunes para todos los tests de handlers.
package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/citaspot/api/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ── Fixtures ──────────────────────────────────────────────────────────────────

var (
	testTenantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testUserID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testID       = uuid.MustParse("33333333-3333-3333-3333-333333333333")

	testTenant = &domain.Tenant{
		ID:           testTenantID,
		Slug:         "test-biz",
		Name:         "Test Business",
		BusinessType: "beauty",
		Email:        "owner@test.com",
		Timezone:     "America/Santo_Domingo",
		Country:      "DO",
		Plan:         "starter",
		PlanStatus:   "trial",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	testUser = &domain.User{
		ID:        testUserID,
		TenantID:  testTenantID,
		Email:     "owner@test.com",
		Name:      "Test Owner",
		Role:      "owner",
		CreatedAt: time.Now(),
	}
)

// ── App factories ─────────────────────────────────────────────────────────────

// newProtectedApp crea un Fiber app con middleware de tenant mockeado.
// Usar para endpoints protegidos — evita necesidad de JWT real.
func newProtectedApp() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("tenant_id", testTenantID)
		c.Locals("tenant", testTenant)
		c.Locals("user", testUser)
		return c.Next()
	})
	return app
}

// newPublicApp crea un Fiber app sin middleware de auth.
// Usar para endpoints públicos.
func newPublicApp() *fiber.App {
	return fiber.New(fiber.Config{DisableStartupMessage: true})
}

// ── Request helpers ───────────────────────────────────────────────────────────

// doJSON hace una petición HTTP con body JSON al test app.
func doJSON(app *fiber.App, method, path, body string) *http.Response {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		panic("app.Test error: " + err.Error())
	}
	return resp
}

// doRequest hace una petición HTTP simple sin body.
func doRequest(app *fiber.App, method, path string) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		panic("app.Test error: " + err.Error())
	}
	return resp
}

// doRequestWithHeader hace una petición HTTP con un header adicional.
func doRequestWithHeader(app *fiber.App, method, path, headerKey, headerVal, body string) *http.Response {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerKey, headerVal)
	resp, err := app.Test(req, -1)
	if err != nil {
		panic("app.Test error: " + err.Error())
	}
	return resp
}

// ── Assertion helpers ─────────────────────────────────────────────────────────

func assertStatus(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("status: expected %d, got %d", expected, actual)
	}
}

func assertBodyContains(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(b), want) {
		t.Errorf("body does not contain %q\nbody: %s", want, string(b))
	}
}
