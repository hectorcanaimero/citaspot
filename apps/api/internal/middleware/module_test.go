package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type stubModuleRepo struct {
	isActiveFn func(ctx context.Context, tenantID uuid.UUID, key string) (bool, error)
}

func (s *stubModuleRepo) IsActive(ctx context.Context, tenantID uuid.UUID, key string) (bool, error) {
	return s.isActiveFn(ctx, tenantID, key)
}
func (s *stubModuleRepo) ListActiveKeys(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return nil, nil
}
func (s *stubModuleRepo) Enable(ctx context.Context, tenantID uuid.UUID, key string) error {
	return nil
}
func (s *stubModuleRepo) Disable(ctx context.Context, tenantID uuid.UUID, key string) error {
	return nil
}

// buildAppWithTenant monta una app con un middleware previo que setea el
// tenant_id en el contexto (simula TenantMiddleware) y luego aplica RequireModule.
func buildAppWithTenant(tenantID uuid.UUID, repo domain.TenantModuleRepository, moduleKey string) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(ctxKeyTenantID, tenantID)
		return c.Next()
	})
	app.Get("/protected", RequireModule(repo, moduleKey), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func TestRequireModule_Allows200WhenActive(t *testing.T) {
	tenantID := uuid.New()
	repo := &stubModuleRepo{
		isActiveFn: func(_ context.Context, tID uuid.UUID, key string) (bool, error) {
			if tID != tenantID {
				t.Fatalf("tenant_id esperado %s, recibido %s", tenantID, tID)
			}
			if key != domain.ModuleDental {
				t.Fatalf("module_key esperado %s, recibido %s", domain.ModuleDental, key)
			}
			return true, nil
		},
	}
	app := buildAppWithTenant(tenantID, repo, domain.ModuleDental)
	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperaba 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestRequireModule_Blocks403WhenInactive(t *testing.T) {
	tenantID := uuid.New()
	repo := &stubModuleRepo{
		isActiveFn: func(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
			return false, nil
		},
	}
	app := buildAppWithTenant(tenantID, repo, domain.ModuleDental)
	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Fatalf("esperaba 403, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["module"] != domain.ModuleDental {
		t.Fatalf("body.module esperado %q, recibido %v", domain.ModuleDental, body["module"])
	}
}

func TestRequireModule_Returns401WhenTenantMissing(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", RequireModule(&stubModuleRepo{}, domain.ModuleDental), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("esperaba 401, got %d", resp.StatusCode)
	}
}

func TestRequireModule_Returns500OnRepoError(t *testing.T) {
	tenantID := uuid.New()
	repo := &stubModuleRepo{
		isActiveFn: func(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
			return false, errors.New("db down")
		},
	}
	app := buildAppWithTenant(tenantID, repo, domain.ModuleDental)
	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Fatalf("esperaba 500, got %d", resp.StatusCode)
	}
}
