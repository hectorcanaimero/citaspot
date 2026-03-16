package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/handler"
)

const testWebhookSecret = "test-evolution-api-key"

func newWAHandler(waSvc *mockWhatsAppSvc, waClient *mockWAClient, secret string) *handler.WhatsAppHandler {
	if waSvc == nil {
		waSvc = &mockWhatsAppSvc{}
	}
	if waClient == nil {
		waClient = &mockWAClient{}
	}
	return handler.NewWhatsAppHandler(waSvc, waClient, secret)
}

func TestWhatsAppHandler_Status(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			isConnectedFn: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Get("/whatsapp/status", h.Status)

		resp := doRequest(app, "GET", "/whatsapp/status")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "CONNECTED")
	})

	t.Run("disconnected", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			isConnectedFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Get("/whatsapp/status", h.Status)

		resp := doRequest(app, "GET", "/whatsapp/status")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "DISCONNECTED")
	})

	t.Run("evolution api error returns disconnected", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			isConnectedFn: func(_ context.Context, _ string) (bool, error) {
				return false, errors.New("connection refused")
			},
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Get("/whatsapp/status", h.Status)

		resp := doRequest(app, "GET", "/whatsapp/status")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "DISCONNECTED")
	})
}

func TestWhatsAppHandler_Connect(t *testing.T) {
	t.Run("success returns CONNECTING", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			connectFn: func(_ context.Context, _ string) error { return nil },
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Post("/whatsapp/connect", h.Connect)

		resp := doJSON(app, "POST", "/whatsapp/connect", "")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "CONNECTING")
	})

	t.Run("evolution api error returns 500", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			connectFn: func(_ context.Context, _ string) error {
				return errors.New("evolution API unavailable")
			},
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Post("/whatsapp/connect", h.Connect)

		resp := doJSON(app, "POST", "/whatsapp/connect", "")
		assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

// TestWhatsAppHandler_GetQR verifica el endpoint de consulta de QR.
// Nota: usa un slug único para evitar colisiones con qrStore (var global del paquete handler).
func TestWhatsAppHandler_GetQR(t *testing.T) {
	t.Run("no QR available returns 204", func(t *testing.T) {
		// testTenant.Slug = "test-biz" — no tiene QR almacenado (instancia nueva/única)
		h := newWAHandler(nil, nil, testWebhookSecret)
		app := newProtectedApp()
		app.Get("/whatsapp/qr", h.GetQR)

		resp := doRequest(app, "GET", "/whatsapp/qr")
		assertStatus(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestWhatsAppHandler_Disconnect(t *testing.T) {
	t.Run("success returns 204", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			disconnectFn: func(_ context.Context, _ string) error { return nil },
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Delete("/whatsapp/disconnect", h.Disconnect)

		resp := doRequest(app, "DELETE", "/whatsapp/disconnect")
		assertStatus(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("evolution error returns 500", func(t *testing.T) {
		h := newWAHandler(nil, &mockWAClient{
			disconnectFn: func(_ context.Context, _ string) error {
				return errors.New("evolution API error")
			},
		}, testWebhookSecret)
		app := newProtectedApp()
		app.Delete("/whatsapp/disconnect", h.Disconnect)

		resp := doRequest(app, "DELETE", "/whatsapp/disconnect")
		assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestWhatsAppHandler_Webhook(t *testing.T) {
	// Nota: usamos "other-instance" en lugar de "test-biz" para evitar
	// interferencias con el qrStore global entre tests.
	tests := []struct {
		name       string
		apiKey     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid qrcode event",
			apiKey:     testWebhookSecret,
			body:       `{"event":"qrcode.updated","instance":"other-instance","data":{"qrcode":{"base64":"data:image/png;base64,abc"}}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid connection open event",
			apiKey:     testWebhookSecret,
			body:       `{"event":"connection.update","instance":"other-instance","data":{"state":"open"}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "messages.upsert event dispatched async",
			apiKey:     testWebhookSecret,
			body:       `{"event":"messages.upsert","instance":"other-instance","data":{"message":{"conversation":"Hola"}}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "QRCODE_UPDATED normalized to qrcode.updated",
			apiKey:     testWebhookSecret,
			body:       `{"event":"QRCODE_UPDATED","instance":"other-instance","data":{"qrcode":{"base64":"abc"}}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown event is ignored gracefully",
			apiKey:     testWebhookSecret,
			body:       `{"event":"unknown.event","instance":"other-instance"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong api key returns 401",
			apiKey:     "wrong-key",
			body:       `{"event":"messages.upsert","instance":"other-instance"}`,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "autorizado",
		},
		{
			name:       "empty api key with secret set returns 401",
			apiKey:     "",
			body:       `{"event":"messages.upsert","instance":"other-instance"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json returns 400",
			apiKey:     testWebhookSecret,
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "empty secret bypasses validation",
			apiKey:     "any-key-is-ok",
			body:       `{"event":"messages.upsert","instance":"other-instance"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret := testWebhookSecret
			if tc.name == "empty secret bypasses validation" {
				secret = ""
			}
			h := newWAHandler(&mockWhatsAppSvc{}, nil, secret)
			app := newPublicApp()
			app.Post("/whatsapp/webhook", h.Webhook)

			resp := doRequestWithHeader(app, "POST", "/whatsapp/webhook", "apikey", tc.apiKey, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
