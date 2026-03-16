package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newBillingHandler(repo *mockAuthRepo) *handler.BillingHandler {
	// Stripe key vacía: las llamadas reales a Stripe fallarán, pero los tests de
	// validación de input no llegan al punto de llamar a Stripe.
	return handler.NewBillingHandler(repo, "", "test-webhook-secret", "price_starter_test", "price_pro_test")
}

func TestBillingHandler_CreateCheckout(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID) (*domain.Tenant, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "invalid plan name",
			body:       `{"plan":"enterprise","success_url":"https://example.com/ok","cancel_url":"https://example.com/cancel"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Plan no válido",
		},
		{
			name:       "empty plan",
			body:       `{"success_url":"https://example.com/ok","cancel_url":"https://example.com/cancel"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Plan",
		},
		{
			// Con plan válido: llama a authRepo.FindTenantByID y luego a Stripe.
			// Stripe fallará (sin key real) pero validamos que el plan es aceptado
			// y el error resultante es 500 (error interno de Stripe).
			name: "valid starter plan triggers stripe call",
			body: `{"plan":"starter","success_url":"https://example.com/ok","cancel_url":"https://example.com/cancel"}`,
			mockFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
				return &domain.Tenant{ID: testTenantID, Email: "owner@test.com"}, nil
			},
			wantStatus: http.StatusInternalServerError, // Stripe error (sin key real)
		},
		{
			name: "valid professional plan triggers stripe call",
			body: `{"plan":"professional","success_url":"https://example.com/ok","cancel_url":"https://example.com/cancel"}`,
			mockFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
				return &domain.Tenant{ID: testTenantID, Email: "owner@test.com"}, nil
			},
			wantStatus: http.StatusInternalServerError, // Stripe error (sin key real)
		},
		{
			name: "tenant not found",
			body: `{"plan":"starter","success_url":"https://example.com/ok","cancel_url":"https://example.com/cancel"}`,
			mockFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAuthRepo{findTenantByIDFn: tc.mockFn}
			h := newBillingHandler(repo)
			app := newProtectedApp()
			app.Post("/billing/checkout", h.CreateCheckout)

			resp := doJSON(app, "POST", "/billing/checkout", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestBillingHandler_Webhook(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		stripeSignature  string
		wantStatus       int
		wantBody         string
	}{
		{
			name:            "invalid stripe signature returns 400",
			body:            `{"type":"checkout.session.completed"}`,
			stripeSignature: "t=invalid,v1=badhash",
			wantStatus:      http.StatusBadRequest,
			wantBody:        "Firma",
		},
		{
			name:            "missing stripe signature returns 400",
			body:            `{"type":"checkout.session.completed"}`,
			stripeSignature: "",
			wantStatus:      http.StatusBadRequest,
			wantBody:        "Firma",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newBillingHandler(&mockAuthRepo{})
			app := newPublicApp()
			app.Post("/billing/webhook", h.Webhook)

			resp := doRequestWithHeader(app, "POST", "/billing/webhook", "Stripe-Signature", tc.stripeSignature, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
