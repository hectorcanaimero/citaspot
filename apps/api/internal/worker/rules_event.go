package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
)

type RulesEventWorker struct {
	amqpURL  string
	executor *engine.RuleExecutor
}

func NewRulesEventWorker(amqpURL string, executor *engine.RuleExecutor) *RulesEventWorker {
	return &RulesEventWorker{amqpURL: amqpURL, executor: executor}
}

func (w *RulesEventWorker) Start(ctx context.Context) {
	slog.Info("RulesEventWorker: iniciado")
	delays := []time.Duration{1, 2, 4, 8, 16}
	idx := 0

	for {
		select {
		case <-ctx.Done():
			slog.Info("RulesEventWorker: detenido")
			return
		default:
		}

		if err := w.consume(ctx); err != nil {
			slog.Error("RulesEventWorker: error en consumer", "error", err)
		}

		d := delays[idx]
		if idx < len(delays)-1 {
			idx++
		}
		slog.Info("RulesEventWorker: reconectando", "delay_s", d)

		select {
		case <-ctx.Done():
			return
		case <-time.After(d * time.Second):
		}
	}
}

func (w *RulesEventWorker) consume(ctx context.Context) error {
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

	q, err := ch.QueueDeclare("rules.events", true, false, false, false, nil)
	if err != nil {
		return err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "core-api-rules-event", false, false, false, false, nil)
	if err != nil {
		return err
	}

	slog.Info("RulesEventWorker: consumiendo rules.events")

	closeCh := make(chan *amqp.Error)
	conn.NotifyClose(closeCh)

	for {
		select {
		case <-ctx.Done():
			return nil
		case amqpErr := <-closeCh:
			if amqpErr != nil {
				return amqpErr
			}
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			w.processEvent(ctx, msg)
		}
	}
}

func (w *RulesEventWorker) processEvent(ctx context.Context, msg amqp.Delivery) {
	var event domain.RuleEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("RulesEventWorker: unmarshal error", "error", err)
		msg.Nack(false, false)
		return
	}

	slog.Info("RulesEventWorker: procesando evento",
		"event_type", event.EventType, "tenant_id", event.TenantID, "customer_id", event.CustomerID)

	w.executor.ExecuteEventRules(ctx, event)
	msg.Ack(false)
}
