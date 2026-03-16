package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type conversationRepository struct {
	db *pgxpool.Pool
}

// NewConversationRepository crea el repositorio de conversaciones.
func NewConversationRepository(db *pgxpool.Pool) domain.ConversationRepository {
	return &conversationRepository{db: db}
}

// FindOrCreateConversation busca una conversación activa por teléfono o crea una nueva.
func (r *conversationRepository) FindOrCreateConversation(ctx context.Context, tenantID uuid.UUID, waPhone string) (*domain.Conversation, error) {
	var conv *domain.Conversation
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		conv = &domain.Conversation{}
		// Buscar conversación activa existente
		err := tx.QueryRow(ctx, `
			SELECT id, tenant_id, customer_id, wa_phone, status, last_message_at, created_at
			FROM conversations
			WHERE tenant_id = $1 AND wa_phone = $2 AND status = 'active'
			ORDER BY created_at DESC
			LIMIT 1
		`, tenantID, waPhone).Scan(
			&conv.ID, &conv.TenantID, &conv.CustomerID,
			&conv.WAPhone, &conv.Status, &conv.LastMessageAt, &conv.CreatedAt,
		)
		if err == nil {
			return nil // encontrada
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("conversationRepository.FindOrCreate: find: %w", err)
		}

		// Crear nueva conversación
		conv.ID = uuid.New()
		conv.TenantID = tenantID
		conv.WAPhone = waPhone
		conv.Status = "active"
		now := time.Now().UTC()
		conv.LastMessageAt = &now

		err = tx.QueryRow(ctx, `
			INSERT INTO conversations (id, tenant_id, wa_phone, status, last_message_at, created_at)
			VALUES ($1, $2, $3, 'active', NOW(), NOW())
			RETURNING id, tenant_id, customer_id, wa_phone, status, last_message_at, created_at
		`, conv.ID, conv.TenantID, conv.WAPhone).Scan(
			&conv.ID, &conv.TenantID, &conv.CustomerID,
			&conv.WAPhone, &conv.Status, &conv.LastMessageAt, &conv.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("conversationRepository.FindOrCreate: create: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return conv, nil
}

// SaveMessage guarda un mensaje en la conversación.
func (r *conversationRepository) SaveMessage(ctx context.Context, m *domain.Message) error {
	return withTenant(ctx, r.db, m.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO messages (id, conversation_id, tenant_id, role, content, wa_message_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`, m.ID, m.ConversationID, m.TenantID, m.Role, m.Content, m.WAMessageID)
		if err != nil {
			return fmt.Errorf("conversationRepository.SaveMessage: %w", err)
		}
		return nil
	})
}

// GetRecentMessages retorna los últimos N mensajes de una conversación.
// Usado por el AI Service (los últimos 10 para el contexto del LLM).
// El tenantID es obligatorio para aplicar RLS via withTenant y filtrar por tenant.
func (r *conversationRepository) GetRecentMessages(ctx context.Context, tenantID, conversationID uuid.UUID, limit int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = 10
	}
	var result []*domain.Message
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, conversation_id, tenant_id, role, content, wa_message_id, created_at
			FROM messages
			WHERE tenant_id = $1 AND conversation_id = $2
			ORDER BY created_at DESC
			LIMIT $3
		`, tenantID, conversationID, limit)
		if err != nil {
			return fmt.Errorf("conversationRepository.GetRecentMessages: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			msg := &domain.Message{}
			if err := rows.Scan(
				&msg.ID, &msg.ConversationID, &msg.TenantID,
				&msg.Role, &msg.Content, &msg.WAMessageID, &msg.CreatedAt,
			); err != nil {
				return fmt.Errorf("conversationRepository.GetRecentMessages: scan: %w", err)
			}
			result = append(result, msg)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateConversationTimestamp actualiza last_message_at de la conversación.
func (r *conversationRepository) UpdateConversationTimestamp(ctx context.Context, conversationID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversations SET last_message_at = NOW() WHERE id = $1
	`, conversationID)
	return err
}
