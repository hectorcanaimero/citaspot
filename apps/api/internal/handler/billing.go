package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	stripeinv "github.com/stripe/stripe-go/v76/invoice"
	stripesub "github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// BillingHandler maneja pagos y subscripciones con Stripe.
type BillingHandler struct {
	authRepo      domain.AuthRepository
	rdb           *redis.Client // puede ser nil si Redis no está disponible
	secretKey     string
	webhookSecret string
	plans         map[string]string
}

// NewBillingHandler crea el handler de facturación.
// rdb puede ser nil — en ese caso la idempotencia del webhook queda desactivada.
// priceStarter y pricePro son los Price IDs de Stripe (env: STRIPE_PRICE_STARTER, STRIPE_PRICE_PRO).
func NewBillingHandler(authRepo domain.AuthRepository, rdb *redis.Client, stripeSecretKey, webhookSecret, priceStarter, pricePro string) *BillingHandler {
	stripe.Key = stripeSecretKey
	return &BillingHandler{
		authRepo:      authRepo,
		rdb:           rdb,
		secretKey:     stripeSecretKey,
		webhookSecret: webhookSecret,
		plans: map[string]string{
			"starter":      priceStarter,
			"professional": pricePro,
		},
	}
}

// CreateCheckout POST /api/v1/billing/checkout
// Crea una sesión de Stripe Checkout y retorna la URL.
func (h *BillingHandler) CreateCheckout(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	var req struct {
		Plan       string `json:"plan"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	priceID, ok := h.plans[req.Plan]
	if !ok {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Plan no válido"})
	}

	tenant, err := h.authRepo.FindTenantByID(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	tenantIDStr := tenantID.String()
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
		Metadata: map[string]string{
			"tenant_id": tenantIDStr,
			"plan":      req.Plan,
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tenant_id": tenantIDStr,
				"plan":      req.Plan,
			},
		},
		CustomerEmail: stripe.String(tenant.Email),
	}

	sess, err := session.New(params)
	if err != nil {
		slog.Error("billing.CreateCheckout: stripe error", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al crear la sesión de pago"})
	}

	return c.JSON(fiber.Map{"url": sess.URL})
}

// Subscription GET /api/v1/billing/subscription
// Retorna el estado actual de la suscripción del tenant en Stripe.
func (h *BillingHandler) Subscription(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	_, subID, err := h.authRepo.FindTenantStripeIDs(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	if subID == "" {
		return c.JSON(fiber.Map{"subscription": nil})
	}

	sub, err := stripesub.Get(subID, nil)
	if err != nil {
		slog.Error("billing.Subscription: stripe error", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al obtener suscripción"})
	}

	return c.JSON(fiber.Map{
		"subscription": fiber.Map{
			"id":                   sub.ID,
			"status":               string(sub.Status),
			"current_period_end":   time.Unix(sub.CurrentPeriodEnd, 0).UTC(),
			"cancel_at_period_end": sub.CancelAtPeriodEnd,
		},
	})
}

// Invoices GET /api/v1/billing/invoices
// Retorna las últimas facturas del tenant en Stripe (máximo 24).
func (h *BillingHandler) Invoices(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	customerID, _, err := h.authRepo.FindTenantStripeIDs(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	if customerID == "" {
		return c.JSON(fiber.Map{"data": []any{}})
	}

	params := &stripe.InvoiceListParams{}
	params.Customer = stripe.String(customerID)
	params.Limit = stripe.Int64(24)

	iter := stripeinv.List(params)
	var invoices []fiber.Map
	for iter.Next() {
		inv := iter.Invoice()
		invoices = append(invoices, fiber.Map{
			"id":         inv.ID,
			"number":     inv.Number,
			"amount":     inv.AmountPaid, // centavos
			"currency":   string(inv.Currency),
			"status":     string(inv.Status),
			"created_at": time.Unix(inv.Created, 0).UTC(),
			"pdf_url":    inv.InvoicePDF,
			"hosted_url": inv.HostedInvoiceURL,
		})
	}
	if err := iter.Err(); err != nil {
		slog.Error("billing.Invoices: stripe error", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al obtener facturas"})
	}

	if invoices == nil {
		invoices = []fiber.Map{}
	}
	return c.JSON(fiber.Map{"data": invoices})
}

// CancelSubscription POST /api/v1/billing/cancel
// Marca la suscripción para cancelarse al final del período (no inmediata).
func (h *BillingHandler) CancelSubscription(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	_, subID, err := h.authRepo.FindTenantStripeIDs(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	if subID == "" {
		return c.Status(http.StatusNotFound).JSON(errorResponse{Error: "No hay suscripción activa"})
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	sub, err := stripesub.Update(subID, params)
	if err != nil {
		slog.Error("billing.CancelSubscription: stripe error", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al cancelar suscripción"})
	}

	return c.JSON(fiber.Map{
		"cancel_at_period_end": sub.CancelAtPeriodEnd,
		"current_period_end":   time.Unix(sub.CurrentPeriodEnd, 0).UTC(),
	})
}

// Webhook POST /api/v1/billing/webhook
// Recibe eventos de Stripe y actualiza el plan del tenant en la DB.
func (h *BillingHandler) Webhook(c *fiber.Ctx) error {
	payload := c.Body()
	sig := c.Get("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(payload, sig, h.webhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		slog.Error("billing.Webhook: signature error", "error", err)
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Firma inválida"})
	}

	ctx := c.Context()

	// Idempotencia: evitar procesar el mismo evento dos veces si Stripe reintenta.
	// SetNX retorna false si la clave ya existe → el evento fue procesado antes.
	if h.rdb != nil {
		eventKey := "stripe:event:" + event.ID
		set, redisErr := h.rdb.SetNX(context.Background(), eventKey, "1", 24*time.Hour).Result()
		if redisErr != nil {
			slog.Warn("billing.Webhook: redis unavailable para idempotencia", "event", event.ID, "error", redisErr)
			// Continuar igualmente — mejor procesar dos veces que no procesar
		} else if !set {
			slog.Info("billing.Webhook: evento ya procesado, ignorando", "event", event.ID)
			return c.SendStatus(http.StatusOK)
		}
	}

	switch event.Type {

	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("billing.Webhook: unmarshal checkout session", "error", err)
			break
		}
		tenantID, plan, err := extractTenantAndPlan(sess.Metadata)
		if err != nil {
			slog.Error("billing.Webhook: checkout.session.completed", "error", err)
			break
		}
		customerID := ""
		if sess.Customer != nil {
			customerID = sess.Customer.ID
		}
		subID := ""
		if sess.Subscription != nil {
			subID = sess.Subscription.ID
		}
		if err := h.authRepo.UpdateTenantBilling(ctx, tenantID, plan, "active", customerID, subID); err != nil {
			slog.Error("billing.Webhook: update billing", "event", "checkout.session.completed", "tenant", tenantID, "error", err)
		} else {
			slog.Info("billing.Webhook: tenant activado", "tenant", tenantID, "plan", plan, "sub", subID)
		}

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			slog.Error("billing.Webhook: unmarshal subscription updated", "error", err)
			break
		}
		tenantID, plan, err := extractTenantAndPlan(sub.Metadata)
		if err != nil {
			slog.Error("billing.Webhook: customer.subscription.updated", "error", err)
			break
		}
		planStatus := mapSubscriptionStatus(string(sub.Status))
		customerID := ""
		if sub.Customer != nil {
			customerID = sub.Customer.ID
		}
		if err := h.authRepo.UpdateTenantBilling(ctx, tenantID, plan, planStatus, customerID, sub.ID); err != nil {
			slog.Error("billing.Webhook: update billing subscription.updated", "tenant", tenantID, "error", err)
		} else {
			slog.Info("billing.Webhook: tenant actualizado", "tenant", tenantID, "plan_status", planStatus)
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			slog.Error("billing.Webhook: unmarshal subscription deleted", "error", err)
			break
		}
		tenantID, plan, err := extractTenantAndPlan(sub.Metadata)
		if err != nil {
			slog.Error("billing.Webhook: customer.subscription.deleted", "error", err)
			break
		}
		customerID := ""
		if sub.Customer != nil {
			customerID = sub.Customer.ID
		}
		if err := h.authRepo.UpdateTenantBilling(ctx, tenantID, plan, "cancelled", customerID, sub.ID); err != nil {
			slog.Error("billing.Webhook: update billing subscription.deleted", "tenant", tenantID, "error", err)
		} else {
			slog.Info("billing.Webhook: tenant cancelado", "tenant", tenantID)
		}
	}

	return c.SendStatus(http.StatusOK)
}

// extractTenantAndPlan extrae tenant_id y plan del metadata de un objeto de Stripe.
func extractTenantAndPlan(metadata map[string]string) (uuid.UUID, string, error) {
	tenantIDStr, ok := metadata["tenant_id"]
	if !ok || tenantIDStr == "" {
		return uuid.Nil, "", fmt.Errorf("tenant_id ausente en metadata de Stripe")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("tenant_id inválido en metadata: %s", tenantIDStr)
	}
	plan := metadata["plan"]
	if plan == "" {
		plan = "starter"
	}
	return tenantID, plan, nil
}

// mapSubscriptionStatus convierte el estado de Stripe al plan_status de CitaSpot.
func mapSubscriptionStatus(status string) string {
	switch status {
	case "active", "trialing":
		return "active"
	case "past_due", "unpaid":
		return "past_due"
	case "canceled":
		return "cancelled"
	default:
		return "past_due"
	}
}
