package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type chatbotConfigRepository struct {
	db *pgxpool.Pool
}

// NewChatbotConfigRepository crea el repositorio de configuración del chatbot.
func NewChatbotConfigRepository(db *pgxpool.Pool) domain.ChatbotConfigRepository {
	return &chatbotConfigRepository{db: db}
}

// GetByTenantID retorna la configuración del chatbot para un tenant.
// Si no existe, la crea con valores por defecto (upsert con DO NOTHING).
func (r *chatbotConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.ChatbotConfig, error) {
	var cfg domain.ChatbotConfig
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO chatbot_configs (tenant_id)
			VALUES ($1)
			ON CONFLICT (tenant_id) DO UPDATE SET tenant_id = EXCLUDED.tenant_id
			RETURNING id, tenant_id, bot_name, bot_greeting, tone,
			          custom_instructions, template_id, created_at, updated_at
		`, tenantID).Scan(
			&cfg.ID, &cfg.TenantID, &cfg.BotName, &cfg.BotGreeting, &cfg.Tone,
			&cfg.CustomInstructions, &cfg.TemplateID, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("chatbotConfig.GetByTenantID: %w", err)
	}
	return &cfg, nil
}

// Update aplica un PATCH sobre la configuración del chatbot.
// Solo actualiza los campos presentes en el input (punteros no nil).
func (r *chatbotConfigRepository) Update(ctx context.Context, tenantID uuid.UUID, input *domain.UpdateChatbotConfigInput) (*domain.ChatbotConfig, error) {
	var cfg domain.ChatbotConfig
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// Obtener o crear config actual
		var current domain.ChatbotConfig
		err := tx.QueryRow(ctx, `
			INSERT INTO chatbot_configs (tenant_id)
			VALUES ($1)
			ON CONFLICT (tenant_id) DO UPDATE SET tenant_id = EXCLUDED.tenant_id
			RETURNING id, tenant_id, bot_name, bot_greeting, tone,
			          custom_instructions, template_id, created_at, updated_at
		`, tenantID).Scan(
			&current.ID, &current.TenantID, &current.BotName, &current.BotGreeting,
			&current.Tone, &current.CustomInstructions, &current.TemplateID,
			&current.CreatedAt, &current.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// Aplicar PATCH: solo los campos presentes en el input
		if input.BotName != nil {
			current.BotName = *input.BotName
		}
		if input.BotGreeting != nil {
			current.BotGreeting = *input.BotGreeting
		}
		if input.Tone != nil {
			current.Tone = *input.Tone
		}
		if input.CustomInstructions != nil {
			current.CustomInstructions = *input.CustomInstructions
		}
		if input.TemplateID != nil {
			current.TemplateID = input.TemplateID
		}

		err = tx.QueryRow(ctx, `
			UPDATE chatbot_configs
			SET bot_name = $1, bot_greeting = $2, tone = $3,
			    custom_instructions = $4, template_id = $5, updated_at = $6
			WHERE tenant_id = $7
			RETURNING id, tenant_id, bot_name, bot_greeting, tone,
			          custom_instructions, template_id, created_at, updated_at
		`, current.BotName, current.BotGreeting, current.Tone,
			current.CustomInstructions, current.TemplateID, time.Now().UTC(),
			tenantID,
		).Scan(
			&cfg.ID, &cfg.TenantID, &cfg.BotName, &cfg.BotGreeting, &cfg.Tone,
			&cfg.CustomInstructions, &cfg.TemplateID, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("chatbotConfig.Update: %w", err)
	}
	return &cfg, nil
}
