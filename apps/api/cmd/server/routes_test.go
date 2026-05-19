// Tests del wiring de rutas. En particular, verifica que el gate del módulo
// dental esté aplicado a TODAS las rutas dental-specific (treatments,
// treatment sessions, clinical notes y clinical files).
//
// Bug histórico (2026-05-19): el gate estaba aplicado al grupo /treatments
// pero faltaba en las rutas de clinical notes/files que viven en grupos
// no-dental (/appointments, /customers). Este test previene que se repita.
package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// sentinelStatus es el código que devuelve el ModuleGate de prueba.
// Cualquier código se sirve — usamos 418 (I'm a teapot) porque no aparece
// en ningún handler real, así sabemos sin ambigüedad si el gate corrió.
const sentinelStatus = fiber.StatusTeapot

// dentalRoutes es el manifest de rutas que DEBEN estar gateadas por el módulo
// dental. Agregar acá cualquier ruta dental nueva — si el test no la cubre,
// es porque falta en el wiring.
var dentalRoutes = []struct {
	method string
	path   string
}{
	// Treatments — grupo /treatments con dentalGate
	{"GET", "/api/v1/treatments/"},
	{"POST", "/api/v1/treatments/"},
	{"GET", "/api/v1/treatments/00000000-0000-0000-0000-000000000001"},
	{"PATCH", "/api/v1/treatments/00000000-0000-0000-0000-000000000001"},
	{"PATCH", "/api/v1/treatments/00000000-0000-0000-0000-000000000001/status"},

	// Treatment sessions — heredan del grupo /treatments
	{"GET", "/api/v1/treatments/00000000-0000-0000-0000-000000000001/sessions/"},
	{"POST", "/api/v1/treatments/00000000-0000-0000-0000-000000000001/sessions/"},
	{"GET", "/api/v1/treatments/00000000-0000-0000-0000-000000000001/sessions/00000000-0000-0000-0000-000000000002"},
	{"PATCH", "/api/v1/treatments/00000000-0000-0000-0000-000000000001/sessions/00000000-0000-0000-0000-000000000002"},
	{"DELETE", "/api/v1/treatments/00000000-0000-0000-0000-000000000001/sessions/00000000-0000-0000-0000-000000000002"},

	// Clinical notes — salpicadas entre /appointments, /customers y /clinical-notes
	{"POST", "/api/v1/appointments/00000000-0000-0000-0000-000000000001/clinical-note"},
	{"GET", "/api/v1/appointments/00000000-0000-0000-0000-000000000001/clinical-note"},
	{"GET", "/api/v1/customers/00000000-0000-0000-0000-000000000001/clinical-notes"},
	{"GET", "/api/v1/customers/00000000-0000-0000-0000-000000000001/clinical-notes/00000000-0000-0000-0000-000000000002"},
	{"PATCH", "/api/v1/clinical-notes/00000000-0000-0000-0000-000000000001"},
	{"DELETE", "/api/v1/clinical-notes/00000000-0000-0000-0000-000000000001"},
	{"POST", "/api/v1/clinical-notes/00000000-0000-0000-0000-000000000001/files/upload"},
	{"GET", "/api/v1/clinical-notes/00000000-0000-0000-0000-000000000001/files"},

	// Clinical files
	{"GET", "/api/v1/customers/00000000-0000-0000-0000-000000000001/clinical-files"},
	{"DELETE", "/api/v1/clinical-files/00000000-0000-0000-0000-000000000001"},
}

// nonDentalRoutes son rutas que NO deben ejecutar el gate dental. Si el test
// gate corre en alguna de estas, significa que aplicamos el gate de más
// (probablemente al grupo padre por error, rompiendo rutas no-dental).
var nonDentalRoutes = []struct {
	method string
	path   string
}{
	{"GET", "/api/v1/appointments/"},                                                    // /appointments root
	{"GET", "/api/v1/customers/"},                                                       // /customers root
	{"GET", "/api/v1/customers/00000000-0000-0000-0000-000000000001"},                   // customer detail
	{"GET", "/api/v1/tasks/"},                                                           // tasks
	{"GET", "/api/v1/rules/"},                                                           // rules
	{"GET", "/api/v1/professionals/"},                                                   // professionals
	{"GET", "/api/v1/services/"},                                                        // services
}

// buildTestApp arma un app fiber con SetupRoutes inyectando:
//   - ModuleGate que devuelve sentinelStatus (para detectar que se aplicó)
//   - AuthMiddlewares passthrough (saltea JWT/tenant)
//   - Handlers nil — no se invocan porque el gate corta antes; si el gate falta,
//     el dispatch al handler nil produce panic → 500, distinto de sentinelStatus.
func buildTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	// recover captura el panic cuando una ruta no-dental dispatch a un handler nil.
	// El panic se convierte en 500 — distinto de sentinelStatus, así el test puede
	// distinguir "gate aplicado" de "gate NO aplicado pero handler crashea".
	app.Use(recover.New())

	moduleGate := func(c *fiber.Ctx) error {
		return c.SendStatus(sentinelStatus)
	}
	passthrough := func(c *fiber.Ctx) error { return c.Next() }

	SetupRoutes(app, &RouteDeps{
		AuthMiddlewares: []fiber.Handler{passthrough},
		ModuleGate:      moduleGate,
		// Handlers nil — Fiber acepta method values de receiver nil al registrar.
		// Solo crashean si se invocan, lo cual indica que el gate falló.
	})
	return app
}

func TestSetupRoutes_DentalGateAppliedToDentalRoutes(t *testing.T) {
	app := buildTestApp()

	for _, r := range dentalRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(""))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != sentinelStatus {
				t.Fatalf("ruta dental NO está gateada — esperaba %d (sentinel), got %d. "+
					"El middleware dental no corrió antes del handler para %s %s.",
					sentinelStatus, resp.StatusCode, r.method, r.path)
			}
		})
	}
}

func TestSetupRoutes_DentalGateNotAppliedToOtherRoutes(t *testing.T) {
	app := buildTestApp()

	for _, r := range nonDentalRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(""))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			// No debe ser sentinelStatus — si lo es, el gate dental se aplicó
			// a una ruta que no debería tenerlo (probablemente porque alguien
			// le puso el gate al grupo padre como `/appointments` entero).
			if resp.StatusCode == sentinelStatus {
				t.Fatalf("ruta no-dental %s %s recibió el gate dental — verificá que el "+
					"middleware no esté aplicado al grupo padre por error.",
					r.method, r.path)
			}
		})
	}
}
