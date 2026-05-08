// Package service — lógica del webhook de WhatsApp.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type whatsAppSvc struct {
	authRepo     domain.AuthRepository
	convRepo     domain.ConversationRepository
	customerRepo domain.CustomerRepository
	publisher    domain.MessagePublisher
}

// NewWhatsAppSvc crea el servicio de procesamiento de mensajes WA.
func NewWhatsAppSvc(
	authRepo domain.AuthRepository,
	convRepo domain.ConversationRepository,
	customerRepo domain.CustomerRepository,
	publisher domain.MessagePublisher,
) domain.WhatsAppSvc {
	return &whatsAppSvc{
		authRepo:     authRepo,
		convRepo:     convRepo,
		customerRepo: customerRepo,
		publisher:    publisher,
	}
}

// HandleConnectionUpdate persiste el estado de conexión de WhatsApp en la DB.
// Se llama de forma asíncrona desde el webhook cuando llega un connection.update.
func (s *whatsAppSvc) HandleConnectionUpdate(ctx context.Context, instanceName, state string) error {
	status := "disconnected"
	if state == "open" {
		status = "connected"
	}
	if err := s.authRepo.UpdateTenantWAStatus(ctx, instanceName, status); err != nil {
		return fmt.Errorf("whatsAppSvc.HandleConnectionUpdate: %w", err)
	}
	slog.Info("wa.status: actualizado", "instance", instanceName, "status", status)
	return nil
}

// ProcessInbound procesa un evento de webhook de Evolution API.
// instanceName corresponde al slug del tenant en nuestra DB.
//
// Flujo:
//  1. Resolver tenant por instanceName (slug).
//  2. Validar que es un mensaje entrante de texto.
//  3. Extraer teléfono, contenido y pushName.
//  4. Sanitizar contenido (max 2000 chars, sin scripts).
//  5. FindOrCreate conversación y cliente.
//  6. Guardar mensaje en DB.
//  7. Publicar en wa.messages.inbound.
func (s *whatsAppSvc) ProcessInbound(ctx context.Context, instanceName string, payload map[string]any) error {
	// 1. Resolver tenant por slug = instanceName
	tenant, err := s.authRepo.FindTenantBySlug(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("whatsAppSvc.ProcessInbound: tenant '%s': %w", instanceName, err)
	}

	// 2. Verificar que es un mensaje entrante (no propio)
	data, ok := payload["data"].(map[string]any)
	if !ok {
		return nil // evento que no nos interesa
	}
	key, _ := data["key"].(map[string]any)
	fromMe, _ := key["fromMe"].(bool)
	if fromMe {
		return nil // ignorar mensajes propios
	}

	// 3. Extraer datos del mensaje
	waMessageID, _ := key["id"].(string)
	remoteJid, _ := key["remoteJid"].(string)
	// Normalizar teléfono: "18091234567@s.whatsapp.net" → "+18091234567"
	phone := normalizeWAPhone(remoteJid)
	if phone == "" {
		return nil
	}

	pushName, _ := data["pushName"].(string)

	// Extraer contenido (texto plano)
	msgBody, _ := data["message"].(map[string]any)
	content := extractMessageContent(msgBody)
	if content == "" {
		return nil // solo procesamos mensajes con texto
	}

	// 4. Sanitizar: limitar longitud, evitar XSS básico
	content = sanitizeWAInput(content)

	// 5. FindOrCreate conversación y cliente
	conv, err := s.convRepo.FindOrCreateConversation(ctx, tenant.ID, phone)
	if err != nil {
		return fmt.Errorf("whatsAppSvc.ProcessInbound: conversation: %w", err)
	}

	// Crear/actualizar cliente por teléfono
	customerName := pushName
	if customerName == "" {
		customerName = phone
	}
	customer, err := s.customerRepo.FindOrCreateByPhone(ctx, tenant.ID, customerName, phone)
	if err != nil {
		slog.Warn("whatsAppSvc.ProcessInbound: customer create warning", "error", err)
		// No fatal — continuamos sin customer vinculado
	}

	// Emitir evento customer.created si el cliente acaba de ser creado
	if customer != nil && customer.TotalVisits == 0 && customer.CreatedAt.After(time.Now().Add(-5*time.Second)) {
		s.publishRuleEvent(ctx, domain.RuleEvent{
			TenantID:   tenant.ID,
			EventType:  "customer.created",
			CustomerID: customer.ID,
			EntityID:   customer.ID,
			EntityType: "customer",
			Payload: map[string]any{
				"customer_id":    customer.ID.String(),
				"customer_name":  customer.Name,
				"customer_phone": customer.Phone,
			},
			Timestamp: time.Now(),
		})
	}

	// 6. Guardar mensaje en DB
	msg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: conv.ID,
		TenantID:       tenant.ID,
		Role:           "user",
		Content:        content,
		WAMessageID:    waMessageID,
	}
	if err := s.convRepo.SaveMessage(ctx, msg); err != nil {
		slog.Warn("whatsAppSvc.ProcessInbound: save message warning", "error", err)
	}
	_ = s.convRepo.UpdateConversationTimestamp(ctx, conv.ID)

	// 7. Obtener historial reciente para contexto del AI Service (últimos 10)
	recentMsgs, err := s.convRepo.GetRecentMessages(ctx, tenant.ID, conv.ID, 10)
	if err != nil {
		slog.Warn("whatsAppSvc.ProcessInbound: get history warning", "error", err)
	}
	history := make([]map[string]interface{}, 0, len(recentMsgs))
	for _, m := range recentMsgs {
		history = append(history, map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	// 8. Publicar en RabbitMQ para que el AI Service procese
	inbound := domain.WAInboundPayload{
		TenantID:       tenant.ID.String(),
		TenantSlug:     tenant.Slug,
		ConversationID: conv.ID.String(),
		WAPhone:        phone,
		WAMessageID:    waMessageID,
		Content:        content,
		MessageType:    "text",
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		History:        history,
	}
	_ = customer // referenciado para evitar warning del compilador

	body, err := json.Marshal(inbound)
	if err != nil {
		return fmt.Errorf("whatsAppSvc.ProcessInbound: marshal: %w", err)
	}
	if err := s.publisher.Publish(ctx, "wa.messages.inbound", body); err != nil {
		return fmt.Errorf("whatsAppSvc.ProcessInbound: publish: %w", err)
	}

	slog.Info("wa.inbound: mensaje procesado", "tenant", tenant.Slug, "phone", phone, "len", len(content))
	return nil
}

// normalizeWAPhone convierte "18091234567@s.whatsapp.net" en "+18091234567".
func normalizeWAPhone(remoteJid string) string {
	phone := strings.Split(remoteJid, "@")[0]
	if phone == "" {
		return ""
	}
	// Grupos de WhatsApp tienen "-" en el número — ignorarlos
	if strings.Contains(phone, "-") {
		return ""
	}
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	return phone
}

// extractMessageContent extrae el texto de distintos tipos de mensaje WA.
func extractMessageContent(msgBody map[string]any) string {
	if msgBody == nil {
		return ""
	}
	// Texto plano
	if text, ok := msgBody["conversation"].(string); ok && text != "" {
		return text
	}
	// Mensaje extendido (respuestas, emojis, etc.)
	if ext, ok := msgBody["extendedTextMessage"].(map[string]any); ok {
		if text, ok := ext["text"].(string); ok {
			return text
		}
	}
	return ""
}

// sanitizeWAInput limita el input y elimina caracteres problemáticos.
// El LLM nunca debe recibir instrucciones embebidas en mensajes de usuario.
func sanitizeWAInput(content string) string {
	// Limitar a 2000 caracteres
	runes := []rune(content)
	if len(runes) > 2000 {
		content = string(runes[:2000])
	}
	// Eliminar caracteres de control excepto newlines
	var sb strings.Builder
	for _, r := range content {
		if r == '\n' || r == '\r' || (r >= 32 && r != 127) {
			sb.WriteRune(r)
		}
	}
	return strings.TrimSpace(sb.String())
}

// publishRuleEvent publica un evento de dominio en la cola rules.events.
func (s *whatsAppSvc) publishRuleEvent(ctx context.Context, event domain.RuleEvent) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("whatsAppSvc.publishRuleEvent: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "rules.events", body); err != nil {
		slog.Warn("whatsAppSvc.publishRuleEvent: publish error", "event", event.EventType, "error", err)
	}
}
