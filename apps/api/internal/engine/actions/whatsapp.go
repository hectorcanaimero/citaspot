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

	phone, ok := params.Context["customer_phone"].(string)
	if !ok || phone == "" {
		return fmt.Errorf("SendWhatsAppAction.Execute: customer_phone no disponible en contexto")
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

	slog.Info("SendWhatsAppAction: mensaje enviado", "tenant", params.TenantSlug, "phone", phone)
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
