package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/citaspot/api/internal/domain"
)

// OutboundWorker consume wa.messages.outbound y envía los mensajes vía WhatsApp.
// El AI Service publica en esta cola cuando tiene una respuesta lista.
type OutboundWorker struct {
	amqpURL  string
	waClient domain.WAClient
	notifRepo domain.NotificationRepository
}

// NewOutboundWorker crea el worker de mensajes salientes.
func NewOutboundWorker(amqpURL string, waClient domain.WAClient, notifRepo domain.NotificationRepository) *OutboundWorker {
	return &OutboundWorker{
		amqpURL:   amqpURL,
		waClient:  waClient,
		notifRepo: notifRepo,
	}
}

// Start inicia el consumer de mensajes salientes. Bloqueante — llamar con go.
// Implementa reconexión automática con backoff exponencial.
func (w *OutboundWorker) Start(ctx context.Context) {
	slog.Info("OutboundWorker: iniciado")
	delays := []time.Duration{1, 2, 4, 8, 16}
	idx := 0

	for {
		select {
		case <-ctx.Done():
			slog.Info("OutboundWorker: detenido")
			return
		default:
		}

		if err := w.consume(ctx); err != nil {
			slog.Error("OutboundWorker: error en consumer", "error", err)
		}

		// Backoff exponencial antes de reconectar
		d := delays[idx]
		if idx < len(delays)-1 {
			idx++
		}
		slog.Info("OutboundWorker: reconectando", "delay_s", d)

		select {
		case <-ctx.Done():
			return
		case <-time.After(d * time.Second):
		}
	}
}

// consume conecta a RabbitMQ y procesa mensajes de wa.messages.outbound.
func (w *OutboundWorker) consume(ctx context.Context) error {
	conn, err := amqp.Dial(w.amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declarar la cola (idempotente)
	q, err := ch.QueueDeclare("wa.messages.outbound", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Prefetch: procesar de uno en uno para respetar rate limit de WA
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "core-api-outbound", false, false, false, false, nil)
	if err != nil {
		return err
	}

	slog.Info("OutboundWorker: consumiendo wa.messages.outbound")

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil // canal cerrado
			}
			w.processOutbound(ctx, msg)
		}
	}
}

// processOutbound envía un mensaje al cliente vía Evolution API.
func (w *OutboundWorker) processOutbound(ctx context.Context, msg amqp.Delivery) {
	var payload domain.WAOutboundPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		slog.Error("OutboundWorker: unmarshal error", "error", err)
		msg.Nack(false, false) // descartar mensaje mal formateado
		return
	}

	tenantID, _ := uuid.Parse(payload.TenantID)

	// Verificar que la sesión WA está activa
	instanceName := payload.TenantSlug // slug = nombre de la instancia en Evolution API
	connected, err := w.waClient.IsConnected(ctx, instanceName)
	if err != nil || !connected {
		slog.Warn("OutboundWorker: instancia no conectada, reencolar", "instance", instanceName)
		msg.Nack(false, true) // reencolar
		return
	}

	// Rate limit: 1 mensaje/segundo por sesión
	time.Sleep(1 * time.Second)

	waMessageID, sendErr := w.waClient.SendText(ctx, instanceName, payload.WAPhone, payload.Content)

	nl := &domain.NotificationLog{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Content:     payload.Content,
		WAMessageID: waMessageID,
		Type:        "ai_response",
	}
	if sendErr != nil {
		nl.Status = "failed"
		nl.ErrorMessage = sendErr.Error()
		slog.Error("OutboundWorker: send error", "phone", payload.WAPhone, "error", sendErr)
		// Fallo permanente (número no existe, etc.) → descartar; fallo transitorio → reencolar.
		requeue := !errors.Is(sendErr, domain.ErrWAPermanentFailure)
		msg.Nack(false, requeue)
	} else {
		nl.Status = "sent"
		slog.Info("OutboundWorker: mensaje enviado", "phone", payload.WAPhone, "len", len(payload.Content))
		msg.Ack(false)
	}

	if err := w.notifRepo.LogNotification(ctx, nl); err != nil {
		slog.Error("OutboundWorker: log notification error", "error", err)
	}
}
