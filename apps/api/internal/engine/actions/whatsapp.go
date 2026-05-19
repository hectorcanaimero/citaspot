package actions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type SendWhatsAppAction struct {
	waClient domain.WAClient
}

func NewSendWhatsAppAction(waClient domain.WAClient) *SendWhatsAppAction {
	return &SendWhatsAppAction{waClient: waClient}
}

func (a *SendWhatsAppAction) Type() string { return "send_whatsapp" }

func (a *SendWhatsAppAction) Execute(ctx context.Context, params engine.ActionParams) error {
	if a.waClient == nil {
		slog.Warn("SendWhatsAppAction: waClient is nil, skipping")
		return nil
	}

	// Determinar destinatario: por defecto el cliente, opcionalmente el profesional.
	recipient, _ := params.Params["recipient"].(string)
	if recipient == "" {
		recipient = "customer"
	}

	var (
		phone   string
		ctxKey  string
	)
	switch recipient {
	case "professional":
		ctxKey = "professional_phone"
	case "customer":
		ctxKey = "customer_phone"
	default:
		return fmt.Errorf("SendWhatsAppAction.Execute: recipient inválido '%s'", recipient)
	}

	phone, _ = params.Context[ctxKey].(string)
	if phone == "" {
		// Best-effort: si el profesional no tiene telefono, no romper la regla.
		if recipient == "professional" {
			slog.Warn("SendWhatsAppAction: profesional sin telefono, skipping",
				"tenant", params.TenantSlug, "entity_id", params.EntityID)
			return nil
		}
		return fmt.Errorf("SendWhatsAppAction.Execute: %s no disponible en contexto", ctxKey)
	}

	connected, err := a.waClient.IsConnected(ctx, params.TenantSlug)
	if err != nil || !connected {
		return fmt.Errorf("SendWhatsAppAction.Execute: instancia '%s' no conectada", params.TenantSlug)
	}

	text := renderTemplate(params.Template, params.Context)
	if text == "" {
		return fmt.Errorf("SendWhatsAppAction.Execute: template vacío tras renderizar")
	}

	_, err = a.waClient.SendText(ctx, params.TenantSlug, phone, text)
	if err != nil {
		return fmt.Errorf("SendWhatsAppAction.Execute: %w", err)
	}

	slog.Info("SendWhatsAppAction: mensaje enviado",
		"tenant", params.TenantSlug, "recipient", recipient, "phone", phone)
	return nil
}

func renderTemplate(tmpl string, ctx map[string]any) string {
	result := tmpl
	for key, val := range ctx {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}
	return result
}
