// Package rabbitmq contiene el publisher de mensajes para RabbitMQ.
package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/citaspot/api/internal/domain"
)

// Publisher implementa domain.MessagePublisher usando amqp091-go.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

// New conecta a RabbitMQ y declara las colas necesarias.
func New(url string) (domain.MessagePublisher, error) {
	p := &Publisher{url: url}
	if err := p.connect(); err != nil {
		return nil, err
	}
	return p, nil
}

// connect establece la conexión y el canal con RabbitMQ.
func (p *Publisher) connect() error {
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("rabbitmq.Publisher: dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("rabbitmq.Publisher: channel: %w", err)
	}

	// Declarar todas las colas del sistema (idempotente)
	queues := []string{
		"wa.messages.inbound",
		"wa.messages.outbound",
		"notifications.reminders",
		"knowledge.vectorize",
		"notifications.review",
	}
	for _, q := range queues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("rabbitmq.Publisher: declare queue %s: %w", q, err)
		}
	}

	p.conn = conn
	p.channel = ch
	return nil
}

// Publish publica un mensaje en la cola especificada.
// Si la conexión está cerrada, intenta reconectar con backoff exponencial.
func (p *Publisher) Publish(ctx context.Context, queue string, body []byte) error {
	if p.channel == nil || p.conn.IsClosed() {
		if err := p.reconnect(); err != nil {
			return fmt.Errorf("rabbitmq.Publish: reconnect: %w", err)
		}
	}

	return p.channel.PublishWithContext(ctx,
		"",    // exchange por defecto
		queue, // routing key = nombre de la cola
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // mensajes duraderos
			Body:         body,
		},
	)
}

// reconnect intenta reconectar con backoff exponencial (1s, 2s, 4s, 8s, 16s).
func (p *Publisher) reconnect() error {
	delays := []time.Duration{1, 2, 4, 8, 16}
	for _, d := range delays {
		slog.Info("RabbitMQ: reconectando", "delay_s", d)
		time.Sleep(d * time.Second)
		if err := p.connect(); err == nil {
			slog.Info("RabbitMQ: reconexión exitosa")
			return nil
		}
	}
	return fmt.Errorf("rabbitmq.reconnect: todos los intentos fallaron")
}

// Close cierra el canal y la conexión.
func (p *Publisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil && !p.conn.IsClosed() {
		return p.conn.Close()
	}
	return nil
}
