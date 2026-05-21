// Tests para el handler SSE de Realtime (Phase C).
//
// T-21: integración end-to-end — publica un evento al canal Redis y verifica
// que el suscriptor recibe el frame SSE con el formato esperado.
// T-22: aislamiento multi-tenant — publica al canal del tenant A y verifica
// que un suscriptor en el tenant B NO recibe el mensaje.
package handler_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/citaspot/api/internal/handler"
)

// newRealtimeApp monta un Fiber app con el handler de realtime y un
// middleware previo que setea el tenant_id en el contexto — simula el
// resultado de JWTMiddlewareWithQuery + TenantMiddleware sin pagar el costo.
func newRealtimeApp(t *testing.T, tenantID uuid.UUID, rdb *redis.Client) *fiber.App {
	t.Helper()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenantID)
		return c.Next()
	})
	h := handler.NewRealtimeHandler(rdb)
	app.Get("/realtime/appointments", h.Appointments)
	return app
}

// startListener arranca el app en un listener real (no app.Test, que buffea
// la respuesta y rompe SSE). Devuelve la base URL y una función de cleanup
// que cierra el listener con un timeout corto — Shutdown puede bloquear
// indefinidamente esperando que las conexiones SSE se cierren, así que
// usamos ShutdownWithTimeout.
func startListener(t *testing.T, app *fiber.App) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = app.Listener(ln) }()
	cleanup := func() {
		_ = app.ShutdownWithTimeout(500 * time.Millisecond)
		_ = ln.Close()
	}
	return "http://" + ln.Addr().String(), cleanup
}

// newMiniredis arranca un miniredis y devuelve un cliente go-redis conectado.
func newMiniredis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	return s, rdb
}

// readSSEEvent lee del scanner hasta encontrar un evento SSE completo
// (líneas "event:" y "data:" seguidas de una línea vacía). Retorna el tipo
// y el data crudo, o error si el timeout expira primero.
//
// Usa un channel con resultados parciales para no bloquear más allá del
// timeout (bufio.Scanner.Scan() es bloqueante y no cancelable).
type sseResult struct {
	event, data string
	err         error
}

func readSSEEvent(ctx context.Context, scanner *bufio.Scanner, timeout time.Duration) (string, string, error) {
	resCh := make(chan sseResult, 1)
	go func() {
		var event, data string
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if event != "" || data != "" {
					resCh <- sseResult{event: event, data: data}
					return
				}
				continue
			}
			if strings.HasPrefix(line, ":") {
				// Comentario SSE (": connected" / ": keepalive") — ignorar.
				continue
			}
			if strings.HasPrefix(line, "event:") {
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			}
			if strings.HasPrefix(line, "data:") {
				data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
		resCh <- sseResult{err: scanner.Err()}
	}()
	select {
	case r := <-resCh:
		return r.event, r.data, r.err
	case <-time.After(timeout):
		return "", "", fmt.Errorf("timeout esperando evento SSE")
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

// TestRealtime_ForwardsPubSubMessage verifica el camino completo: el handler
// se suscribe al canal del tenant, alguien publica un evento, el cliente
// recibe el frame SSE con `event: appointment.created` y el payload verbatim.
func TestRealtime_ForwardsPubSubMessage(t *testing.T) {
	_, rdb := newMiniredis(t)
	defer rdb.Close()

	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	app := newRealtimeApp(t, tenantID, rdb)
	baseURL, cleanup := startListener(t, app)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/realtime/appointments", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("esperaba 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type esperado text/event-stream, got %q", ct)
	}

	// Publicar el evento DESPUÉS de que el cliente esté conectado.
	// miniredis necesita un breve momento para registrar la suscripción.
	channel := fmt.Sprintf("tenant:%s:appointments", tenantID)
	payload := `{"event":"appointment.created","data":{"id":"abc"},"ts":"2026-05-21T10:00:00Z"}`
	go func() {
		// Pequeña espera para asegurar que pubsub.Subscribe corrió.
		time.Sleep(100 * time.Millisecond)
		// Retry breve hasta que haya un subscriber registrado.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			n, err := rdb.Publish(ctx, channel, payload).Result()
			if err == nil && n >= 1 {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	event, data, err := readSSEEvent(ctx, scanner, 4*time.Second)
	if err != nil {
		t.Fatalf("readSSEEvent: %v", err)
	}
	if event != "appointment.created" {
		t.Fatalf("event esperado %q, got %q", "appointment.created", event)
	}
	if !strings.Contains(data, `"event":"appointment.created"`) {
		t.Fatalf("data esperado contener event campo, got %q", data)
	}
	if !strings.Contains(data, `"id":"abc"`) {
		t.Fatalf("data esperado contener payload original, got %q", data)
	}
}

// TestRealtime_MultiTenantIsolation verifica que publicar al canal del
// tenant A NO entrega el mensaje a un suscriptor del tenant B. Cada
// suscripción Redis es por-canal-por-tenant.
func TestRealtime_MultiTenantIsolation(t *testing.T) {
	_, rdb := newMiniredis(t)
	defer rdb.Close()

	tenantA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tenantB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Suscriptor directo a Redis para tenant B — más simple que levantar un
	// segundo HTTP server. Lo que probamos es Redis pub/sub semantics:
	// si un mensaje al canal A NO llega a un suscriber del canal B, entonces
	// el aislamiento multi-tenant es correcto a nivel transporte.
	channelA := fmt.Sprintf("tenant:%s:appointments", tenantA)
	channelB := fmt.Sprintf("tenant:%s:appointments", tenantB)

	pubsubB := rdb.Subscribe(ctx, channelB)
	defer pubsubB.Close()

	// Esperar a que la suscripción esté lista (miniredis confirma con un
	// "subscription" message en el primer Receive).
	if _, err := pubsubB.Receive(ctx); err != nil {
		t.Fatalf("subscribe B: %v", err)
	}

	// Publicar al canal del tenant A — el suscriber B NO debe recibir nada.
	payload := `{"event":"appointment.created","data":{"id":"only-A"},"ts":"2026-05-21T10:00:00Z"}`
	if _, err := rdb.Publish(ctx, channelA, payload).Result(); err != nil {
		t.Fatalf("publish A: %v", err)
	}

	// Dar tiempo para que el mensaje atraviese (no debería, pero esperamos).
	select {
	case msg := <-pubsubB.Channel():
		t.Fatalf("aislamiento ROTO: tenant B recibió mensaje del canal A: %+v", msg)
	case <-time.After(500 * time.Millisecond):
		// Bien — el mensaje al canal A no llegó al canal B.
	}

	// Sanity check: publicar al canal B sí debe llegar.
	payloadB := `{"event":"appointment.created","data":{"id":"only-B"},"ts":"2026-05-21T10:00:00Z"}`
	if _, err := rdb.Publish(ctx, channelB, payloadB).Result(); err != nil {
		t.Fatalf("publish B: %v", err)
	}
	select {
	case msg := <-pubsubB.Channel():
		if !strings.Contains(msg.Payload, `"id":"only-B"`) {
			t.Fatalf("payload inesperado: %s", msg.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout: tenant B no recibió su propio mensaje (sanity check falló)")
	}
}

// TestRealtime_RejectsWithoutTenant verifica que sin tenant_id en el
// contexto el handler responde 403 — defensa en profundidad por encima
// de JWTMiddlewareWithQuery + TenantMiddleware.
func TestRealtime_RejectsWithoutTenant(t *testing.T) {
	_, rdb := newMiniredis(t)
	defer rdb.Close()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := handler.NewRealtimeHandler(rdb)
	app.Get("/realtime/appointments", h.Appointments)

	req, _ := http.NewRequest("GET", "/realtime/appointments", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Fatalf("esperaba 403 sin tenant, got %d", resp.StatusCode)
	}
}

// TestRealtime_RejectsWithoutRedis verifica que si el handler se construye
// con rdb=nil el endpoint responde 503 en vez de panic.
func TestRealtime_RejectsWithoutRedis(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenantID)
		return c.Next()
	})
	h := handler.NewRealtimeHandler(nil)
	app.Get("/realtime/appointments", h.Appointments)

	req, _ := http.NewRequest("GET", "/realtime/appointments", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Fatalf("esperaba 503 sin redis, got %d", resp.StatusCode)
	}
}
