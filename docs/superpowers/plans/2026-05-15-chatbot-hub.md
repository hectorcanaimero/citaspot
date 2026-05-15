# Chatbot Hub Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Replace the "Conocimiento" section with a unified "Chatbot" hub featuring personality config, knowledge management, live testing, and AI validation.

**Architecture:** 3-column dashboard layout (progress checklist | tabbed content | test chat). New Go endpoints for chatbot config CRUD, test, and validate. AI service gains test mode (sync response, no WA publish). New `chatbot_configs` DB table with data migration from tenants.settings JSONB.

**Tech Stack:** Go (Fiber), Python (FastAPI), Next.js 14, PostgreSQL, Redis, pgvector

---

## Task 1: DB Migration — `033_chatbot_configs.sql`

**Files:**
- `apps/api/db/migrations/033_chatbot_configs.sql`

- [ ] Step 1.1: Create migration file

```sql
-- 033_chatbot_configs.sql
-- Tabla de configuración del chatbot por tenant.
-- Migra bot_name y bot_greeting desde tenants.settings JSONB.

CREATE TABLE IF NOT EXISTS chatbot_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_name            VARCHAR(30) NOT NULL DEFAULT '',
    bot_greeting        TEXT NOT NULL DEFAULT '',
    tone                VARCHAR(20) NOT NULL DEFAULT 'friendly',
    custom_instructions TEXT NOT NULL DEFAULT '',
    template_id         VARCHAR(30),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- RLS obligatorio — aislamiento por tenant
ALTER TABLE chatbot_configs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON chatbot_configs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Índice en tenant_id para lookups rápidos (la UK ya lo cubre, pero explícito para RLS)
CREATE INDEX idx_chatbot_configs_tenant ON chatbot_configs(tenant_id);

-- Migrar datos existentes de tenants.settings JSONB
INSERT INTO chatbot_configs (tenant_id, bot_name, bot_greeting)
SELECT
    id,
    COALESCE(settings->>'bot_name', ''),
    COALESCE(settings->>'bot_greeting', '')
FROM tenants
WHERE settings->>'bot_name' IS NOT NULL
   OR settings->>'bot_greeting' IS NOT NULL
ON CONFLICT (tenant_id) DO NOTHING;
```

**Commit:** `feat(db): add chatbot_configs table with data migration from tenant settings`

---

## Task 2: Go Domain — ChatbotConfig types + interfaces

**Files:**
- `apps/api/internal/domain/types.go`
- `apps/api/internal/domain/interfaces.go`
- `apps/api/internal/domain/errors.go`

- [ ] Step 2.1: Add ChatbotConfig types to `domain/types.go`

Append after the `KnowledgeVectorizePayload` section (after line ~463):

```go
// ── Chatbot Config ───────────────────────────────────────────────────────────

// ChatbotConfig configuración del chatbot IA de un tenant.
type ChatbotConfig struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	BotName            string     `json:"bot_name"`
	BotGreeting        string     `json:"bot_greeting"`
	Tone               string     `json:"tone"`
	CustomInstructions string     `json:"custom_instructions"`
	TemplateID         *string    `json:"template_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// UpdateChatbotConfigInput campos opcionales para PATCH del config.
type UpdateChatbotConfigInput struct {
	BotName            *string `json:"bot_name"             validate:"omitempty,max=30"`
	BotGreeting        *string `json:"bot_greeting"         validate:"omitempty,max=500"`
	Tone               *string `json:"tone"                 validate:"omitempty,oneof=friendly professional premium casual"`
	CustomInstructions *string `json:"custom_instructions"  validate:"omitempty,max=1000"`
	TemplateID         *string `json:"template_id"          validate:"omitempty,max=30"`
}

// ChatbotTestRequest mensaje de prueba para el chatbot.
type ChatbotTestRequest struct {
	Message string `json:"message" validate:"required,min=1,max=2000"`
}

// ChatbotTestResponse respuesta síncrona del chatbot en modo test.
type ChatbotTestResponse struct {
	Response         string   `json:"response"`
	IntentDetected   string   `json:"intent_detected"`
	RAGSourcesUsed   []string `json:"rag_sources_used"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
}

// ChatbotValidateResponse resultado de la validación IA del chatbot.
type ChatbotValidateResponse struct {
	TotalQuestions int                        `json:"total_questions"`
	Passed         int                        `json:"passed"`
	Results        []ChatbotValidationResult  `json:"results"`
}

// ChatbotValidationResult resultado de una pregunta de validación individual.
type ChatbotValidationResult struct {
	Question   string `json:"question"`
	Response   string `json:"response"`
	Passed     bool   `json:"passed"`
	Suggestion string `json:"suggestion,omitempty"`
}
```

- [ ] Step 2.2: Add ChatbotConfig interfaces to `domain/interfaces.go`

Append after the `KnowledgeSvc` interface (after line ~246):

```go
// ── Chatbot Config ───────────────────────────────────────────────────────────

// ChatbotConfigRepository operaciones DB para la configuración del chatbot.
type ChatbotConfigRepository interface {
	// GetByTenantID retorna la config del chatbot. Si no existe, crea una con defaults.
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*ChatbotConfig, error)
	// Update actualiza campos parciales de la config (PATCH semantics).
	Update(ctx context.Context, tenantID uuid.UUID, input *UpdateChatbotConfigInput) (*ChatbotConfig, error)
}

// ChatbotSvc lógica de negocio del chatbot (config, test, validación).
type ChatbotSvc interface {
	GetConfig(ctx context.Context, tenantID uuid.UUID) (*ChatbotConfig, error)
	UpdateConfig(ctx context.Context, tenantID uuid.UUID, input *UpdateChatbotConfigInput) (*ChatbotConfig, error)
	Test(ctx context.Context, tenantID uuid.UUID, tenantSlug string, req *ChatbotTestRequest) (*ChatbotTestResponse, error)
	Validate(ctx context.Context, tenantID uuid.UUID, tenantSlug string) (*ChatbotValidateResponse, error)
}
```

- [ ] Step 2.3: Add rate limit error to `domain/errors.go`

```go
ErrRateLimited = errors.New("demasiadas solicitudes, intentá de nuevo más tarde")
```

And add the case to `handler/errors.go`:

```go
case errors.Is(err, domain.ErrRateLimited):
    return c.Status(http.StatusTooManyRequests).JSON(newError("rate_limited", err.Error()))
```

**Commit:** `feat(domain): add ChatbotConfig types, interfaces, and rate limit error`

---

## Task 3: Go Repository — `chatbot_config.go`

**Files:**
- `apps/api/internal/repository/chatbot_config.go`

- [ ] Step 3.1: Create the repository file

```go
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

// NewChatbotConfigRepository crea el repositorio de chatbot config.
func NewChatbotConfigRepository(db *pgxpool.Pool) domain.ChatbotConfigRepository {
	return &chatbotConfigRepository{db: db}
}

// GetByTenantID retorna la config del chatbot. Si no existe, crea una con defaults (upsert).
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

// Update actualiza campos parciales de la config (PATCH semantics).
func (r *chatbotConfigRepository) Update(ctx context.Context, tenantID uuid.UUID, input *domain.UpdateChatbotConfigInput) (*domain.ChatbotConfig, error) {
	var cfg domain.ChatbotConfig
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// Primero obtener la config actual (o crear con defaults)
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

		// Aplicar PATCH: solo los campos que vienen en el input
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
```

**Commit:** `feat(repo): add chatbot config repository with upsert pattern`

---

## Task 4: Go Service — `chatbot.go`

**Files:**
- `apps/api/internal/service/chatbot.go`

- [ ] Step 4.1: Create the chatbot service

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/citaspot/api/internal/domain"
)

type chatbotSvc struct {
	repo         domain.ChatbotConfigRepository
	knowledgeRepo domain.KnowledgeRepository
	authRepo     domain.AuthRepository
	aiServiceURL string
	httpClient   *http.Client
	rdb          *redis.Client
}

// NewChatbotSvc crea el servicio de chatbot.
func NewChatbotSvc(
	repo domain.ChatbotConfigRepository,
	knowledgeRepo domain.KnowledgeRepository,
	authRepo domain.AuthRepository,
	aiServiceURL string,
	rdb *redis.Client,
) domain.ChatbotSvc {
	return &chatbotSvc{
		repo:          repo,
		knowledgeRepo: knowledgeRepo,
		authRepo:      authRepo,
		aiServiceURL:  aiServiceURL,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		rdb:           rdb,
	}
}

func (s *chatbotSvc) GetConfig(ctx context.Context, tenantID uuid.UUID) (*domain.ChatbotConfig, error) {
	cfg, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.GetConfig: %w", err)
	}
	return cfg, nil
}

func (s *chatbotSvc) UpdateConfig(ctx context.Context, tenantID uuid.UUID, input *domain.UpdateChatbotConfigInput) (*domain.ChatbotConfig, error) {
	cfg, err := s.repo.Update(ctx, tenantID, input)
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.UpdateConfig: %w", err)
	}
	return cfg, nil
}

// aiTestRequest es el payload que se envía al AI Service /process-test.
type aiTestRequest struct {
	TenantID       string `json:"tenant_id"`
	TenantSlug     string `json:"tenant_slug"`
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message_text"`
	Test           bool   `json:"test"`
}

// aiTestResponse es la respuesta del AI Service /process-test.
type aiTestResponse struct {
	Response         string   `json:"response"`
	IntentDetected   string   `json:"intent_detected"`
	RAGSourcesUsed   []string `json:"rag_sources_used"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
}

// Test envía un mensaje de prueba al AI Service y retorna la respuesta de forma síncrona.
func (s *chatbotSvc) Test(ctx context.Context, tenantID uuid.UUID, tenantSlug string, req *domain.ChatbotTestRequest) (*domain.ChatbotTestResponse, error) {
	// Usar un conversation_id fijo para tests de este tenant
	conversationID := fmt.Sprintf("test-%s", tenantID.String())

	payload := aiTestRequest{
		TenantID:       tenantID.String(),
		TenantSlug:     tenantSlug,
		ConversationID: conversationID,
		Message:        req.Message,
		Test:           true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.Test: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/process-test", s.aiServiceURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.Test: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.Test: http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.Test: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("chatbotSvc.Test: AI service error", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("chatbotSvc.Test: AI service returned %d", resp.StatusCode)
	}

	var aiResp aiTestResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return nil, fmt.Errorf("chatbotSvc.Test: unmarshal: %w", err)
	}

	elapsed := time.Since(start).Milliseconds()
	if aiResp.ProcessingTimeMs == 0 {
		aiResp.ProcessingTimeMs = elapsed
	}

	return &domain.ChatbotTestResponse{
		Response:         aiResp.Response,
		IntentDetected:   aiResp.IntentDetected,
		RAGSourcesUsed:   aiResp.RAGSourcesUsed,
		ProcessingTimeMs: aiResp.ProcessingTimeMs,
	}, nil
}

// validateRateLimitKey key de Redis para rate limit de validación por tenant.
func validateRateLimitKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("chatbot:validate:ratelimit:%s", tenantID.String())
}

// validationQuestionsByCategory genera preguntas según las categorías de documentos activos.
func validationQuestionsByCategory(docs []*domain.KnowledgeDocument) []string {
	categories := make(map[string]bool)
	for _, d := range docs {
		if d.IsActive {
			categories[d.Category] = true
		}
	}

	questions := []string{
		"¿Cómo puedo agendar una cita?",
	}

	if categories["services"] {
		questions = append(questions, "¿Qué servicios ofrecen?")
	}
	if categories["pricing"] || categories["services"] {
		questions = append(questions, "¿Cuánto cuesta el servicio más popular?")
	}
	if categories["faq"] {
		questions = append(questions, "¿Cuál es el horario de atención?")
	}
	if categories["policies"] {
		questions = append(questions, "¿Cuál es la política de cancelación?")
	}
	if categories["team"] {
		questions = append(questions, "¿Quiénes son los profesionales disponibles?")
	}
	if categories["location"] {
		questions = append(questions, "¿Dónde están ubicados?")
	}

	// Preguntas genéricas que siempre aplican
	questions = append(questions, "¿Aceptan tarjeta?")

	return questions
}

// vaguePhrases respuestas que indican que el bot no encontró información.
var vaguePhrases = []string{
	"no tengo información",
	"no cuento con",
	"no dispongo",
	"no puedo confirmar",
	"verificar",
	"voy a consultar",
	"no estoy segur",
	"no sé",
	"no lo sé",
}

// isVagueResponse evalúa si la respuesta del bot es vaga o sin información concreta.
func isVagueResponse(response string) bool {
	lower := strings.ToLower(response)
	for _, phrase := range vaguePhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// categoryForQuestion mapea preguntas de validación a categorías de conocimiento sugeridas.
func categoryForQuestion(question string) string {
	lower := strings.ToLower(question)
	switch {
	case strings.Contains(lower, "servicio"):
		return "services"
	case strings.Contains(lower, "cuánto cuesta") || strings.Contains(lower, "precio"):
		return "pricing"
	case strings.Contains(lower, "horario"):
		return "faq"
	case strings.Contains(lower, "cancelación") || strings.Contains(lower, "política"):
		return "policies"
	case strings.Contains(lower, "profesional") || strings.Contains(lower, "quiénes"):
		return "team"
	case strings.Contains(lower, "ubicad") || strings.Contains(lower, "dónde"):
		return "location"
	case strings.Contains(lower, "tarjeta") || strings.Contains(lower, "pago"):
		return "pricing"
	default:
		return "faq"
	}
}

// suggestionForCategory genera la sugerencia cuando una pregunta falla.
func suggestionForCategory(category string) string {
	switch category {
	case "services":
		return "Agregá información sobre tus servicios a la base de conocimiento"
	case "pricing":
		return "Documentá tus precios y métodos de pago"
	case "faq":
		return "Agregá preguntas frecuentes con horarios y datos generales"
	case "policies":
		return "Agregá tu política de cancelación y condiciones"
	case "team":
		return "Agregá información sobre tu equipo de profesionales"
	case "location":
		return "Agregá información de ubicación y cómo llegar"
	default:
		return "Agregá más información a tu base de conocimiento"
	}
}

// Validate ejecuta la validación IA del chatbot: genera preguntas, las envía al AI Service
// y evalúa si las respuestas contienen información específica.
func (s *chatbotSvc) Validate(ctx context.Context, tenantID uuid.UUID, tenantSlug string) (*domain.ChatbotValidateResponse, error) {
	// Rate limit: 1 por hora por tenant
	if s.rdb != nil {
		key := validateRateLimitKey(tenantID)
		exists, err := s.rdb.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return nil, domain.ErrRateLimited
		}
	}

	// Obtener documentos activos para generar preguntas
	docs, err := s.knowledgeRepo.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("chatbotSvc.Validate: list docs: %w", err)
	}

	questions := validationQuestionsByCategory(docs)

	results := make([]domain.ChatbotValidationResult, 0, len(questions))
	passed := 0

	for _, q := range questions {
		testResp, err := s.Test(ctx, tenantID, tenantSlug, &domain.ChatbotTestRequest{Message: q})
		if err != nil {
			slog.Warn("chatbotSvc.Validate: test failed for question", "question", q, "error", err)
			results = append(results, domain.ChatbotValidationResult{
				Question:   q,
				Response:   "Error al procesar la pregunta",
				Passed:     false,
				Suggestion: "Verificá que el servicio de IA esté funcionando correctamente",
			})
			continue
		}

		ok := !isVagueResponse(testResp.Response)
		if ok {
			passed++
		}

		result := domain.ChatbotValidationResult{
			Question: q,
			Response: testResp.Response,
			Passed:   ok,
		}
		if !ok {
			cat := categoryForQuestion(q)
			result.Suggestion = suggestionForCategory(cat)
		}
		results = append(results, result)
	}

	// Guardar rate limit: 1 hora
	if s.rdb != nil {
		_ = s.rdb.Set(ctx, validateRateLimitKey(tenantID), "1", time.Hour).Err()
	}

	return &domain.ChatbotValidateResponse{
		TotalQuestions: len(questions),
		Passed:         passed,
		Results:        results,
	}, nil
}
```

**Commit:** `feat(service): add chatbot service with test and validation logic`

---

## Task 5: Go Handler — `chatbot.go` + route registration

**Files:**
- `apps/api/internal/handler/chatbot.go`
- `apps/api/cmd/server/main.go`

- [ ] Step 5.1: Create the chatbot handler

```go
package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ChatbotHandler maneja los endpoints del chatbot hub.
type ChatbotHandler struct {
	svc      domain.ChatbotSvc
	validate *validator.Validate
}

// NewChatbotHandler crea el handler del chatbot.
func NewChatbotHandler(svc domain.ChatbotSvc) *ChatbotHandler {
	return &ChatbotHandler{svc: svc, validate: validator.New()}
}

// GetConfig GET /api/v1/chatbot/config
func (h *ChatbotHandler) GetConfig(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	cfg, err := h.svc.GetConfig(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(cfg)
}

// UpdateConfig PATCH /api/v1/chatbot/config
func (h *ChatbotHandler) UpdateConfig(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	var input domain.UpdateChatbotConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{Error: err.Error()})
	}

	cfg, err := h.svc.UpdateConfig(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(cfg)
}

// Test POST /api/v1/chatbot/test
func (h *ChatbotHandler) Test(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	tenant := middleware.TenantFromContext(c)
	if tenant == nil {
		return c.Status(http.StatusForbidden).JSON(errorResponse{Error: "Tenant no encontrado"})
	}

	var req domain.ChatbotTestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{Error: err.Error()})
	}

	resp, err := h.svc.Test(c.Context(), tenantID, tenant.Slug, &req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(resp)
}

// Validate POST /api/v1/chatbot/validate
func (h *ChatbotHandler) Validate(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	tenant := middleware.TenantFromContext(c)
	if tenant == nil {
		return c.Status(http.StatusForbidden).JSON(errorResponse{Error: "Tenant no encontrado"})
	}

	resp, err := h.svc.Validate(c.Context(), tenantID, tenant.Slug)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(resp)
}
```

- [ ] Step 5.2: Wire up DI in `cmd/server/main.go`

Add to the repository construction section (around line 129, near `knowledgeRepo`):

```go
chatbotConfigRepo := repository.NewChatbotConfigRepository(pool)
```

Add to the service construction section (around line 155, near `knowledgeSvc`):

```go
chatbotSvc := service.NewChatbotSvc(chatbotConfigRepo, knowledgeRepo, authRepo, cfg.AIServiceURL, rdb)
```

Add to the handler construction section (around line 211, near `knowledgeHandler`):

```go
chatbotHandler := handler.NewChatbotHandler(chatbotSvc)
```

Add route registration after the knowledge routes (after line ~533):

```go
chatbot := protected.Group("/chatbot")
chatbot.Get("/config", chatbotHandler.GetConfig)
chatbot.Patch("/config", chatbotHandler.UpdateConfig)
chatbot.Post("/test", chatbotHandler.Test)
chatbot.Post("/validate", chatbotHandler.Validate)
```

**Commit:** `feat(api): add chatbot handler with config, test, and validate endpoints`

---

## Task 6: AI Service — Test mode endpoint

**Files:**
- `apps/ai/app/routers/process_test.py` (new)
- `apps/ai/app/agent/orchestrator.py` (modify)
- `apps/ai/app/agent/messages.py` (modify)
- `apps/ai/app/agent/state.py` (modify)
- `apps/ai/main.py` (modify)

- [ ] Step 6.1: Add test state helpers to `state.py`

Append to the end of `apps/ai/app/agent/state.py`:

```python
# TTL para conversaciones de test: 30 minutos
_TEST_TTL = 1_800  # segundos


def _test_key(tenant_id: str) -> str:
    return f"test:{tenant_id}"


async def get_test_state(tenant_id: str) -> dict[str, Any]:
    """Retorna el estado de la conversación de test desde Redis."""
    r = get_redis()
    raw = await r.get(_test_key(tenant_id))
    if raw:
        return json.loads(raw)
    return {
        "state": ConvState.IDLE,
        "pending_slots": [],
        "pending_service_id": None,
        "pending_professional_id": None,
        "pending_date": None,
        "customer_name": None,
        "customer_phone": None,
    }


async def save_test_state(tenant_id: str, data: dict[str, Any]) -> None:
    """Persiste el estado de test en Redis con TTL de 30 min."""
    r = get_redis()
    await r.setex(
        _test_key(tenant_id),
        _TEST_TTL,
        json.dumps(data, default=str),
    )


async def reset_test_state(tenant_id: str) -> None:
    """Borra el estado de test de un tenant."""
    r = get_redis()
    await r.delete(_test_key(tenant_id))
```

- [ ] Step 6.2: Modify `_build_system_prompt` in `orchestrator.py` to accept tone and custom_instructions

Replace the existing `_build_system_prompt` function (lines 131-152) with:

```python
# Mapeo de tone a instrucciones de comunicación
_TONE_INSTRUCTIONS: dict[str, dict[str, str]] = {
    "friendly": {
        "es": "Usá un tono amigable y cercano. Podés usar emojis con moderación. Tuteá al cliente.",
        "en": "Use a friendly and warm tone. You can use emojis sparingly. Address the client informally.",
        "pt": "Use um tom amigável e próximo. Pode usar emojis com moderação. Trate o cliente por você.",
    },
    "professional": {
        "es": "Mantené un tono profesional y respetuoso. Evitá emojis. Usá usted.",
        "en": "Maintain a professional and respectful tone. Avoid emojis. Use formal address.",
        "pt": "Mantenha um tom profissional e respeitoso. Evite emojis. Trate o cliente por senhor(a).",
    },
    "premium": {
        "es": "Usá un tono cálido pero elegante. Transmití exclusividad y cuidado personalizado.",
        "en": "Use a warm but elegant tone. Convey exclusivity and personalized care.",
        "pt": "Use um tom caloroso mas elegante. Transmita exclusividade e cuidado personalizado.",
    },
    "casual": {
        "es": "Sé directo y relajado. Podés usar expresiones coloquiales. Tuteá al cliente.",
        "en": "Be direct and relaxed. You can use colloquial expressions. Address the client casually.",
        "pt": "Seja direto e descontraído. Pode usar expressões coloquiais. Trate o cliente por você.",
    },
}


def _build_system_prompt(
    profile: dict[str, Any] | None,
    rag_context: str,
    tone: str | None = None,
    custom_instructions: str | None = None,
) -> str:
    """Construye el system prompt del asistente con contexto del negocio."""
    msgs = get_messages(_LANG)
    business_name = profile.get("name", "el negocio") if profile else "el negocio"
    bot_name = (profile.get("bot_name") or "el asistente virtual") if profile else "el asistente virtual"

    services_text = ""
    if profile and profile.get("services"):
        lines = []
        for s in profile["services"]:
            price_str = f"${s['price']} USD" if s.get("price") else "consultar en cita"
            lines.append(f"- {s['name']} ({price_str}, {s['duration_min']} min)")
        services_text = f"\n\n{msgs['system_services_header']}\n" + "\n".join(lines)

    rag_section = (
        f"\n\n{msgs['system_rag_header']}\n{rag_context}" if rag_context else ""
    )

    intro = msgs["system_intro"].format(business_name=business_name, bot_name=bot_name)
    warning = msgs["system_warning"]

    # Instrucciones de tono (desde chatbot_configs o default del system_intro)
    tone_section = ""
    if tone and tone in _TONE_INSTRUCTIONS:
        lang = _LANG if _LANG in _TONE_INSTRUCTIONS[tone] else "es"
        tone_section = f"\n\nEstilo de comunicación: {_TONE_INSTRUCTIONS[tone][lang]}"

    # Instrucciones personalizadas del negocio
    custom_section = ""
    if custom_instructions and custom_instructions.strip():
        custom_section = f"\n\nInstrucciones adicionales del negocio: {custom_instructions.strip()}"

    return f"{intro}{tone_section}{custom_section}{services_text}{rag_section}\n\n{warning}"
```

- [ ] Step 6.3: Update `_handle_query` in `orchestrator.py` to pass tone and custom_instructions

Replace the `_handle_query` function (lines 500-517) with:

```python
async def _handle_query(
    tenant_id: str,
    tenant_slug: str,
    message_text: str,
    history: list[dict[str, Any]],
    tone: str | None = None,
    custom_instructions: str | None = None,
) -> str:
    """Responde preguntas usando RAG + LLM."""
    profile = await get_tenant_profile(tenant_slug)
    context = await rag_query(tenant_id, message_text)

    system = _build_system_prompt(profile, context, tone=tone, custom_instructions=custom_instructions)
    messages = [{"role": "system", "content": system}]

    for h in history[-8:]:
        messages.append({"role": h["role"], "content": h["content"]})
    messages.append({"role": "user", "content": message_text})

    return await chat(messages, temperature=0.3)
```

Also update the two call sites in `_handle` that call `_handle_query` (lines 273 and 279) to pass `tone` and `custom_instructions` if available. Since the normal WA flow doesn't yet fetch chatbot_configs, pass None for now — the test endpoint will pass them explicitly.

- [ ] Step 6.4: Create `apps/ai/app/routers/process_test.py`

```python
# POST /process-test — endpoint síncrono para test del chatbot desde el dashboard.
# NO publica en RabbitMQ. Usa Redis key prefix test:{tenant_id}. Retorna respuesta directa.
from __future__ import annotations

import time
from typing import Any

import structlog
from fastapi import APIRouter
from pydantic import BaseModel, Field

from app.agent.actions import get_tenant_profile, rag_query
from app.agent.intent import Intent
from app.agent.intent import detect as detect_intent
from app.agent.orchestrator import _build_system_prompt, _handle_query
from app.agent.state import get_test_state, reset_test_state, save_test_state, ConvState
from app.llm.router import chat

log = structlog.get_logger(__name__)

router = APIRouter(prefix="/process-test", tags=["process-test"])


class ProcessTestRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant")
    tenant_slug: str = Field(..., description="Slug del tenant")
    conversation_id: str = Field(..., description="ID de conversación de test")
    message_text: str = Field(..., max_length=2000, description="Texto del mensaje de test")
    test: bool = Field(True, description="Flag de modo test")


class ProcessTestResponse(BaseModel):
    response: str = Field("", description="Respuesta del bot")
    intent_detected: str = Field("", description="Intención detectada")
    rag_sources_used: list[str] = Field(default_factory=list, description="Fuentes RAG usadas")
    processing_time_ms: int = Field(0, description="Tiempo de procesamiento en ms")


@router.post("", response_model=ProcessTestResponse)
async def process_test(req: ProcessTestRequest) -> ProcessTestResponse:
    """
    Procesa un mensaje de test de forma SÍNCRONA.
    No publica en RabbitMQ. Usa estado de test separado en Redis.
    Retorna la respuesta del bot directamente.
    """
    start = time.time()

    state = await get_test_state(req.tenant_id)
    current_state = ConvState(state.get("state", ConvState.IDLE))

    # Detectar intención
    intent = await detect_intent(req.message_text, [])
    log.info(
        "process-test: intent detectado",
        tenant=req.tenant_id,
        intent=str(intent),
        state=str(current_state),
    )

    # Obtener perfil del tenant y config del chatbot
    profile = await get_tenant_profile(req.tenant_slug)

    # Obtener tone y custom_instructions del chatbot config via Core API
    # (por ahora el profile ya trae bot_name; tone y custom_instructions
    # se pasan cuando el Core API los incluya en el profile endpoint)
    tone = profile.get("tone") if profile else None
    custom_instructions = profile.get("custom_instructions") if profile else None

    # RAG search
    rag_context = ""
    rag_sources: list[str] = []
    if intent in (Intent.QUERY, Intent.UNKNOWN):
        rag_context = await rag_query(req.tenant_id, req.message_text)
        if rag_context:
            # Extraer nombres de fuentes del contexto RAG
            rag_sources = [line.split(":")[0].strip("- ") for line in rag_context.split("\n") if line.strip()]

    # Generar respuesta
    system = _build_system_prompt(profile, rag_context, tone=tone, custom_instructions=custom_instructions)
    messages: list[dict[str, Any]] = [{"role": "system", "content": system}]
    messages.append({"role": "user", "content": req.message_text})

    response_text = await chat(messages, temperature=0.3)

    elapsed_ms = int((time.time() - start) * 1000)

    return ProcessTestResponse(
        response=response_text,
        intent_detected=str(intent.value) if intent else "unknown",
        rag_sources_used=rag_sources,
        processing_time_ms=elapsed_ms,
    )


@router.delete("/{tenant_id}/state")
async def clear_test_state(tenant_id: str) -> dict[str, str]:
    """Borra el estado de test de un tenant."""
    await reset_test_state(tenant_id)
    return {"status": "ok"}
```

- [ ] Step 6.5: Register the new router in `apps/ai/main.py`

After `from app.routers import health, process, search, vectorize` (line 14), add:

```python
from app.routers import health, process, process_test, search, vectorize
```

After `app.include_router(search.router)` (line 28), add:

```python
app.include_router(process_test.router)
```

**Commit:** `feat(ai): add synchronous process-test endpoint with tone and custom instructions support`

---

## Task 7: Frontend — API client + types

**Files:**
- `apps/web/lib/api.ts`

- [ ] Step 7.1: Add TypeScript types near the top of `api.ts` (after existing type definitions)

```typescript
// ── Chatbot Config ──────────────────────────────────────────────────────────

export interface ChatbotConfig {
  id: string;
  tenant_id: string;
  bot_name: string;
  bot_greeting: string;
  tone: 'friendly' | 'professional' | 'premium' | 'casual';
  custom_instructions: string;
  template_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface UpdateChatbotConfigInput {
  bot_name?: string;
  bot_greeting?: string;
  tone?: 'friendly' | 'professional' | 'premium' | 'casual';
  custom_instructions?: string;
  template_id?: string;
}

export interface ChatbotTestResponse {
  response: string;
  intent_detected: string;
  rag_sources_used: string[];
  processing_time_ms: number;
}

export interface ChatbotValidationResult {
  question: string;
  response: string;
  passed: boolean;
  suggestion?: string;
}

export interface ChatbotValidateResponse {
  total_questions: number;
  passed: number;
  results: ChatbotValidationResult[];
}
```

- [ ] Step 7.2: Add chatbot API client (after `settingsApi` export, around line ~977)

```typescript
// ── Chatbot ─────────────────────────────────────────────────────────────────

export const chatbotApi = {
  async config(): Promise<ChatbotConfig> {
    return request('/api/v1/chatbot/config');
  },

  async updateConfig(data: UpdateChatbotConfigInput): Promise<ChatbotConfig> {
    return request('/api/v1/chatbot/config', {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async test(message: string): Promise<ChatbotTestResponse> {
    return request('/api/v1/chatbot/test', {
      method: 'POST',
      body: JSON.stringify({ message }),
    });
  },

  async validate(): Promise<ChatbotValidateResponse> {
    return request('/api/v1/chatbot/validate', {
      method: 'POST',
    });
  },
};
```

**Commit:** `feat(web): add chatbot API client with types`

---

## Task 8: Frontend — i18n keys

**Files:**
- `apps/web/lib/i18n/locales/es.ts`
- `apps/web/lib/i18n/locales/en.ts`
- `apps/web/lib/i18n/locales/pt.ts`

- [ ] Step 8.1: Update nav key in all three locales

In `es.ts`, change:
```typescript
knowledge: 'Conocimiento',
```
to:
```typescript
knowledge: 'Conocimiento',
chatbot: 'Chatbot',
```

In `en.ts`, add `chatbot: 'Chatbot',` in the nav section.
In `pt.ts`, add `chatbot: 'Chatbot',` in the nav section.

- [ ] Step 8.2: Add `chatbot` section to `es.ts`

```typescript
chatbot: {
  title: 'Chatbot',
  subtitle: 'Configurá y probá tu asistente de WhatsApp',
  tabs: {
    personality: 'Personalidad',
    knowledge: 'Conocimiento',
    validation: 'Validación',
  },
  // Progress checklist
  progress: {
    title: 'Progreso',
    botName: 'Nombre del bot',
    greeting: 'Saludo configurado',
    docs: 'Documentos activos',
    tested: 'Chat probado',
    validated: 'Validación pasada',
    completed: '{n} de {total}',
  },
  // Personality tab
  personality: {
    templateTitle: 'Elegí un template para empezar',
    templateChange: 'Cambiar template',
    salon: 'Peluquería',
    dental: 'Dental',
    spa: 'Spa / Estética',
    barbershop: 'Barbería',
    generic: 'Genérico',
    botName: 'Nombre del asistente',
    botNameHint: 'Nombre con el que se presentará por WhatsApp (máx 30 chars)',
    botGreeting: 'Saludo de bienvenida',
    botGreetingHint: 'Mensaje inicial al empezar una conversación. Soporta {business_name}',
    tone: 'Tono de comunicación',
    toneFriendly: 'Amigable',
    toneProfessional: 'Profesional',
    tonePremium: 'Premium',
    toneCasual: 'Casual',
    customInstructions: 'Instrucciones personalizadas',
    customInstructionsHint: 'Instrucciones específicas para tu negocio (máx 1000 chars). Ej: "Siempre ofrecer combo corte+barba"',
    saved: 'Guardado',
  },
  // Knowledge tab
  knowledgeTab: {
    templateDraft: 'Sugerido por template',
  },
  // Test chat
  testChat: {
    title: 'Probar chatbot',
    placeholder: 'Escribí un mensaje de prueba...',
    send: 'Enviar',
    clear: 'Limpiar chat',
    banner: 'Modo prueba — no se envía a clientes',
    debug: 'Detalles técnicos',
    intent: 'Intención',
    sources: 'Fuentes RAG',
    time: 'Tiempo',
    empty: 'Enviá un mensaje para probar tu chatbot',
    collapse: 'Ocultar chat',
    expand: 'Probar chat',
  },
  // Validation tab
  validation: {
    title: 'Validación',
    staticTitle: 'Checklist de configuración',
    aiTitle: 'Validación con IA',
    aiButton: 'Validar con IA',
    aiRunning: 'Validando...',
    aiDisabled: 'Activá al menos 1 documento para validar',
    aiRateLimit: 'Podés validar una vez por hora',
    result: '{passed} de {total} preguntas respondidas correctamente',
    addDoc: 'Agregar documento',
    rules: {
      botName: 'Dale un nombre a tu asistente para que se presente con tus clientes',
      greeting: 'Configurá un saludo de bienvenida',
      activeDocs: 'Activá al menos un documento de conocimiento',
      faqCategory: 'Agregá preguntas frecuentes — es lo que más consultan tus clientes',
      pricingDocs: 'Documentá tus precios para que el bot pueda informar a tus clientes',
      policyDocs: 'Agregá tu política de cancelación',
      chatTested: 'Probá tu chatbot en el panel de la derecha',
    },
  },
},
```

- [ ] Step 8.3: Add equivalent `chatbot` section to `en.ts`

```typescript
chatbot: {
  title: 'Chatbot',
  subtitle: 'Configure and test your WhatsApp assistant',
  tabs: {
    personality: 'Personality',
    knowledge: 'Knowledge',
    validation: 'Validation',
  },
  progress: {
    title: 'Progress',
    botName: 'Bot name',
    greeting: 'Greeting set',
    docs: 'Active documents',
    tested: 'Chat tested',
    validated: 'Validation passed',
    completed: '{n} of {total}',
  },
  personality: {
    templateTitle: 'Choose a template to get started',
    templateChange: 'Change template',
    salon: 'Hair Salon',
    dental: 'Dental',
    spa: 'Spa / Aesthetics',
    barbershop: 'Barbershop',
    generic: 'Generic',
    botName: 'Assistant name',
    botNameHint: 'Name the bot will use to introduce itself on WhatsApp (max 30 chars)',
    botGreeting: 'Welcome greeting',
    botGreetingHint: 'Initial message when starting a conversation. Supports {business_name}',
    tone: 'Communication tone',
    toneFriendly: 'Friendly',
    toneProfessional: 'Professional',
    tonePremium: 'Premium',
    toneCasual: 'Casual',
    customInstructions: 'Custom instructions',
    customInstructionsHint: 'Specific instructions for your business (max 1000 chars). E.g., "Always offer combo cut+beard"',
    saved: 'Saved',
  },
  knowledgeTab: {
    templateDraft: 'Suggested by template',
  },
  testChat: {
    title: 'Test chatbot',
    placeholder: 'Write a test message...',
    send: 'Send',
    clear: 'Clear chat',
    banner: 'Test mode — not sent to customers',
    debug: 'Technical details',
    intent: 'Intent',
    sources: 'RAG sources',
    time: 'Time',
    empty: 'Send a message to test your chatbot',
    collapse: 'Hide chat',
    expand: 'Test chat',
  },
  validation: {
    title: 'Validation',
    staticTitle: 'Configuration checklist',
    aiTitle: 'AI Validation',
    aiButton: 'Validate with AI',
    aiRunning: 'Validating...',
    aiDisabled: 'Activate at least 1 document to validate',
    aiRateLimit: 'You can validate once per hour',
    result: '{passed} of {total} questions answered correctly',
    addDoc: 'Add document',
    rules: {
      botName: 'Give your assistant a name so it can introduce itself to your clients',
      greeting: 'Set up a welcome greeting',
      activeDocs: 'Activate at least one knowledge document',
      faqCategory: 'Add FAQs — it\'s what clients ask most',
      pricingDocs: 'Document your prices so the bot can inform your clients',
      policyDocs: 'Add your cancellation policy',
      chatTested: 'Test your chatbot in the right panel',
    },
  },
},
```

- [ ] Step 8.4: Add equivalent `chatbot` section to `pt.ts`

```typescript
chatbot: {
  title: 'Chatbot',
  subtitle: 'Configure e teste seu assistente de WhatsApp',
  tabs: {
    personality: 'Personalidade',
    knowledge: 'Conhecimento',
    validation: 'Validação',
  },
  progress: {
    title: 'Progresso',
    botName: 'Nome do bot',
    greeting: 'Saudação configurada',
    docs: 'Documentos ativos',
    tested: 'Chat testado',
    validated: 'Validação aprovada',
    completed: '{n} de {total}',
  },
  personality: {
    templateTitle: 'Escolha um template para começar',
    templateChange: 'Alterar template',
    salon: 'Salão de Beleza',
    dental: 'Dental',
    spa: 'Spa / Estética',
    barbershop: 'Barbearia',
    generic: 'Genérico',
    botName: 'Nome do assistente',
    botNameHint: 'Nome que o bot usará para se apresentar no WhatsApp (máx 30 chars)',
    botGreeting: 'Saudação de boas-vindas',
    botGreetingHint: 'Mensagem inicial ao começar uma conversa. Suporta {business_name}',
    tone: 'Tom de comunicação',
    toneFriendly: 'Amigável',
    toneProfessional: 'Profissional',
    tonePremium: 'Premium',
    toneCasual: 'Casual',
    customInstructions: 'Instruções personalizadas',
    customInstructionsHint: 'Instruções específicas para o seu negócio (máx 1000 chars)',
    saved: 'Salvo',
  },
  knowledgeTab: {
    templateDraft: 'Sugerido pelo template',
  },
  testChat: {
    title: 'Testar chatbot',
    placeholder: 'Escreva uma mensagem de teste...',
    send: 'Enviar',
    clear: 'Limpar chat',
    banner: 'Modo teste — não enviado para clientes',
    debug: 'Detalhes técnicos',
    intent: 'Intenção',
    sources: 'Fontes RAG',
    time: 'Tempo',
    empty: 'Envie uma mensagem para testar seu chatbot',
    collapse: 'Ocultar chat',
    expand: 'Testar chat',
  },
  validation: {
    title: 'Validação',
    staticTitle: 'Checklist de configuração',
    aiTitle: 'Validação com IA',
    aiButton: 'Validar com IA',
    aiRunning: 'Validando...',
    aiDisabled: 'Ative pelo menos 1 documento para validar',
    aiRateLimit: 'Você pode validar uma vez por hora',
    result: '{passed} de {total} perguntas respondidas corretamente',
    addDoc: 'Adicionar documento',
    rules: {
      botName: 'Dê um nome ao seu assistente para que ele se apresente aos seus clientes',
      greeting: 'Configure uma saudação de boas-vindas',
      activeDocs: 'Ative pelo menos um documento de conhecimento',
      faqCategory: 'Adicione perguntas frequentes — é o que os clientes mais perguntam',
      pricingDocs: 'Documente seus preços para que o bot possa informar seus clientes',
      policyDocs: 'Adicione sua política de cancelamento',
      chatTested: 'Teste seu chatbot no painel à direita',
    },
  },
},
```

**Commit:** `feat(i18n): add chatbot hub translations for es, en, pt`

---

## Task 9: Frontend — Nav update

**Files:**
- `apps/web/components/dashboard/sidebar.tsx`

- [ ] Step 9.1: Replace knowledge nav item with chatbot

In `sidebar.tsx` at line 66, change:

```typescript
{ href: '/dashboard/knowledge',  icon: BookOpen,        label: t.nav.knowledge   },
```

to:

```typescript
{ href: '/dashboard/chatbot',    icon: Bot,             label: t.nav.chatbot     },
```

Also add `Bot` to the lucide-react imports at the top of the file:

```typescript
import { Bot, /* ...existing imports */ } from 'lucide-react';
```

**Commit:** `feat(nav): replace knowledge with chatbot in dashboard sidebar`

---

## Task 10: Frontend — Main ChatbotPage layout

**Files:**
- `apps/web/app/dashboard/chatbot/page.tsx` (new)

- [ ] Step 10.1: Create the main chatbot page

```typescript
'use client';

import { useState, useEffect, useCallback } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, knowledge, type ChatbotConfig, type KnowledgeDocument } from '@/lib/api';
import { Spinner } from '@/components/ui/spinner';
import { ProgressChecklist } from './components/ProgressChecklist';
import { PersonalityTab } from './components/PersonalityTab';
import { KnowledgeTab } from './components/KnowledgeTab';
import { ValidationTab } from './components/ValidationTab';
import { TestChatPanel } from './components/TestChatPanel';

type Tab = 'personality' | 'knowledge' | 'validation';

export default function ChatbotPage() {
  const t = useTranslations();
  const [activeTab, setActiveTab] = useState<Tab>('personality');
  const [config, setConfig] = useState<ChatbotConfig | null>(null);
  const [docs, setDocs] = useState<KnowledgeDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [chatTested, setChatTested] = useState(false);
  const [chatPanelOpen, setChatPanelOpen] = useState(true);

  // Cargar config y documentos al montar
  useEffect(() => {
    async function load() {
      try {
        const [cfgRes, docsRes] = await Promise.all([
          chatbotApi.config(),
          knowledge.list(),
        ]);
        setConfig(cfgRes);
        setDocs(docsRes.data);
      } catch (err) {
        console.error('Error loading chatbot data:', err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  // Callback para actualizar config tras un PATCH
  const handleConfigUpdate = useCallback((updated: ChatbotConfig) => {
    setConfig(updated);
  }, []);

  // Callback para refrescar documentos
  const refreshDocs = useCallback(async () => {
    try {
      const res = await knowledge.list();
      setDocs(res.data);
    } catch (err) {
      console.error('Error refreshing docs:', err);
    }
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <Spinner size="lg" />
      </div>
    );
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'personality', label: t.chatbot.tabs.personality },
    { key: 'knowledge', label: t.chatbot.tabs.knowledge },
    { key: 'validation', label: t.chatbot.tabs.validation },
  ];

  return (
    <div className="flex flex-col h-[calc(100vh-3.5rem)]">
      {/* Header */}
      <div className="px-4 py-3 border-b border-neutral-200 shrink-0">
        <h1 className="text-lg font-semibold text-neutral-900">{t.chatbot.title}</h1>
        <p className="text-sm text-neutral-500">{t.chatbot.subtitle}</p>
      </div>

      {/* 3-column layout */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Progress Checklist */}
        <aside className="hidden lg:block w-52 border-r border-neutral-200 p-4 overflow-y-auto shrink-0">
          <ProgressChecklist
            config={config}
            docs={docs}
            chatTested={chatTested}
            onNavigate={(tab) => setActiveTab(tab as Tab)}
          />
        </aside>

        {/* Center: Tabbed Content */}
        <main className="flex-1 overflow-y-auto">
          {/* Tabs */}
          <div className="flex border-b border-neutral-200 px-4">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.key
                    ? 'border-primary-600 text-primary-600'
                    : 'border-transparent text-neutral-500 hover:text-neutral-700'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="p-4">
            {activeTab === 'personality' && config && (
              <PersonalityTab
                config={config}
                docs={docs}
                onConfigUpdate={handleConfigUpdate}
                onDocsRefresh={refreshDocs}
              />
            )}
            {activeTab === 'knowledge' && (
              <KnowledgeTab
                docs={docs}
                onRefresh={refreshDocs}
              />
            )}
            {activeTab === 'validation' && config && (
              <ValidationTab
                config={config}
                docs={docs}
                chatTested={chatTested}
              />
            )}
          </div>
        </main>

        {/* Right: Test Chat Panel */}
        {chatPanelOpen ? (
          <aside className="hidden xl:flex w-[350px] border-l border-neutral-200 flex-col shrink-0">
            <TestChatPanel
              onMessageSent={() => setChatTested(true)}
              onCollapse={() => setChatPanelOpen(false)}
            />
          </aside>
        ) : (
          <button
            onClick={() => setChatPanelOpen(true)}
            className="hidden xl:flex items-center justify-center w-10 border-l border-neutral-200 bg-neutral-50 hover:bg-neutral-100 transition-colors"
            title={t.chatbot.testChat.expand}
          >
            <span className="text-xs font-medium text-neutral-500 [writing-mode:vertical-lr] rotate-180">
              {t.chatbot.testChat.expand}
            </span>
          </button>
        )}
      </div>
    </div>
  );
}
```

**Commit:** `feat(web): add chatbot hub page with 3-column layout`

---

## Task 11: Frontend — ProgressChecklist component

**Files:**
- `apps/web/app/dashboard/chatbot/components/ProgressChecklist.tsx` (new)

- [ ] Step 11.1: Create ProgressChecklist component

```typescript
'use client';

import { useTranslations } from '@/lib/i18n';
import type { ChatbotConfig, KnowledgeDocument } from '@/lib/api';
import { CheckCircle2, Circle } from 'lucide-react';

interface Props {
  config: ChatbotConfig | null;
  docs: KnowledgeDocument[];
  chatTested: boolean;
  onNavigate: (tab: string) => void;
}

export function ProgressChecklist({ config, docs, chatTested, onNavigate }: Props) {
  const t = useTranslations();

  const activeDocs = docs.filter(d => d.is_active);

  const items = [
    {
      label: t.chatbot.progress.botName,
      done: Boolean(config?.bot_name),
      tab: 'personality',
    },
    {
      label: t.chatbot.progress.greeting,
      done: Boolean(config?.bot_greeting),
      tab: 'personality',
    },
    {
      label: t.chatbot.progress.docs,
      done: activeDocs.length > 0,
      tab: 'knowledge',
    },
    {
      label: t.chatbot.progress.tested,
      done: chatTested,
      tab: 'personality', // clicking navigates but user should use test panel
    },
    {
      label: t.chatbot.progress.validated,
      done: false, // se marca tras la primera validación exitosa
      tab: 'validation',
    },
  ];

  const completed = items.filter(i => i.done).length;

  return (
    <div>
      <h2 className="text-sm font-semibold text-neutral-900 mb-3">
        {t.chatbot.progress.title}
      </h2>

      <ul className="space-y-2">
        {items.map((item, idx) => (
          <li key={idx}>
            <button
              onClick={() => onNavigate(item.tab)}
              className="flex items-center gap-2 text-sm w-full text-left hover:bg-neutral-50 rounded px-1.5 py-1 transition-colors"
            >
              {item.done ? (
                <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0" />
              ) : (
                <Circle className="w-4 h-4 text-neutral-300 shrink-0" />
              )}
              <span className={item.done ? 'text-neutral-700' : 'text-neutral-400'}>
                {item.label}
              </span>
            </button>
          </li>
        ))}
      </ul>

      <div className="mt-4 pt-3 border-t border-neutral-100">
        <p className="text-xs text-neutral-400">
          {t.chatbot.progress.completed
            .replace('{n}', String(completed))
            .replace('{total}', String(items.length))}
        </p>
      </div>
    </div>
  );
}
```

**Commit:** `feat(web): add ProgressChecklist component for chatbot hub`

---

## Task 12: Frontend — PersonalityTab component

**Files:**
- `apps/web/app/dashboard/chatbot/components/PersonalityTab.tsx` (new)
- `apps/web/app/dashboard/chatbot/data/templates.ts` (new)

- [ ] Step 12.1: Create templates static data

```typescript
// Datos estáticos de templates de chatbot — NO requieren API.
export interface ChatbotTemplate {
  id: string;
  name: string;
  icon: string;
  defaults: {
    bot_name: string;
    bot_greeting: string;
    tone: 'friendly' | 'professional' | 'premium' | 'casual';
  };
  suggested_docs: Array<{
    category: string;
    title: string;
    content: string;
  }>;
}

export const CHATBOT_TEMPLATES: ChatbotTemplate[] = [
  {
    id: 'salon',
    name: 'Peluquería',
    icon: '💇',
    defaults: {
      bot_name: 'Luna',
      bot_greeting: '¡Hola! Soy Luna, asistente virtual de {business_name}. ¿En qué puedo ayudarte hoy? 😊',
      tone: 'friendly',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Cómo puedo cancelar mi cita?**\nPodés cancelar tu cita hasta 4 horas antes del horario programado sin costo. Cancelaciones tardías pueden tener un cargo.\n\n**¿Qué métodos de pago aceptan?**\nAceptamos efectivo, tarjetas de débito y crédito.\n\n**¿Necesito reservar con anticipación?**\nRecomendamos reservar al menos 24 horas antes para garantizar tu horario preferido.',
      },
      {
        category: 'policies',
        title: 'Políticas del salón',
        content: '## Políticas\n\n**Cancelación:** Cancelar con al menos 4 horas de anticipación. Cancelaciones tardías: cargo del 50% del servicio.\n\n**Puntualidad:** Si llegás más de 15 minutos tarde, la cita podría ser reprogramada.\n\n**Menores de edad:** Los menores de 16 años deben venir acompañados de un adulto.',
      },
    ],
  },
  {
    id: 'dental',
    name: 'Dental',
    icon: '🦷',
    defaults: {
      bot_name: 'Dra. Asistente',
      bot_greeting: '¡Bienvenido/a a {business_name}! Soy la asistente virtual de la clínica. ¿En qué puedo ayudarle?',
      tone: 'professional',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes — primera cita',
        content: '## Preguntas frecuentes\n\n**¿Qué debo llevar a mi primera cita?**\nDocumento de identidad, historial médico y lista de medicamentos actuales.\n\n**¿Aceptan seguros dentales?**\nTrabajamos con las principales aseguradoras. Consulte si su plan está incluido.\n\n**¿Cuánto dura una consulta general?**\nLa primera consulta dura aproximadamente 45 minutos incluyendo evaluación y radiografías.',
      },
      {
        category: 'policies',
        title: 'Políticas de la clínica',
        content: '## Políticas\n\n**Cancelación:** Cancelar con 24 horas de anticipación. Cancelaciones sin aviso: cargo de consulta completo.\n\n**Preparación:** Cepillarse los dientes antes de la consulta. No consumir café o alimentos pigmentados 2 horas antes de blanqueamiento.',
      },
    ],
  },
  {
    id: 'spa',
    name: 'Spa / Estética',
    icon: '🧖',
    defaults: {
      bot_name: 'Serenity',
      bot_greeting: 'Bienvenido/a a {business_name}. Soy Serenity, tu asistente personal. ¿En qué puedo ayudarte para tu próxima experiencia de bienestar?',
      tone: 'premium',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Hay contraindicaciones para los tratamientos?**\nAlgunos tratamientos no son recomendados durante el embarazo o con ciertas condiciones médicas. Consultá con nuestro equipo.\n\n**¿Ofrecen paquetes?**\nSí, tenemos paquetes especiales de relajación y belleza con descuentos. Preguntá por nuestras promociones vigentes.',
      },
      {
        category: 'policies',
        title: 'Cuidados post-tratamiento',
        content: '## Cuidados post-tratamiento\n\n**Faciales:** Evitar maquillaje por 24 horas. Usar protector solar SPF 50+.\n\n**Masajes:** Hidratarse bien. Evitar ejercicio intenso por 12 horas.\n\n**Depilación:** No exponerse al sol por 48 horas. Hidratar la zona tratada.',
      },
    ],
  },
  {
    id: 'barbershop',
    name: 'Barbería',
    icon: '💈',
    defaults: {
      bot_name: 'Barber Bot',
      bot_greeting: '¡Qué onda! Soy el asistente de {business_name}. ¿Querés agendar un corte o tenés alguna pregunta?',
      tone: 'casual',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Necesito turno previo?**\nRecomendamos reservar, pero también atendemos sin turno según disponibilidad.\n\n**¿Cuánto demora un corte?**\nUn corte clásico toma unos 30 minutos. Corte + barba unos 45 minutos.',
      },
      {
        category: 'services',
        title: 'Servicios y precios',
        content: '## Servicios\n\n- Corte clásico\n- Corte + barba\n- Afeitado tradicional\n- Tratamiento capilar\n- Cejas\n\nConsultá precios actualizados con nuestro equipo.',
      },
    ],
  },
  {
    id: 'generic',
    name: 'Genérico',
    icon: '🤖',
    defaults: {
      bot_name: 'Asistente',
      bot_greeting: '¡Hola! Soy el asistente virtual de {business_name}. ¿En qué puedo ayudarte?',
      tone: 'professional',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Cómo puedo agendar una cita?**\nPodés agendar directamente por WhatsApp o desde nuestra página web.\n\n**¿Cuáles son los horarios de atención?**\nLunes a viernes de 9:00 a 18:00. Sábados de 9:00 a 13:00.',
      },
      {
        category: 'policies',
        title: 'Políticas generales',
        content: '## Políticas\n\n**Cancelación:** Cancelar con al menos 4 horas de anticipación.\n\n**Reagendamiento:** Podés reagendar tu cita sin costo hasta 2 horas antes.',
      },
    ],
  },
];
```

- [ ] Step 12.2: Create PersonalityTab component

```typescript
'use client';

import { useState, useRef } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, knowledge as knowledgeApi, type ChatbotConfig, type KnowledgeDocument } from '@/lib/api';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Check } from 'lucide-react';
import { CHATBOT_TEMPLATES, type ChatbotTemplate } from '../data/templates';

interface Props {
  config: ChatbotConfig;
  docs: KnowledgeDocument[];
  onConfigUpdate: (config: ChatbotConfig) => void;
  onDocsRefresh: () => void;
}

export function PersonalityTab({ config, docs, onConfigUpdate, onDocsRefresh }: Props) {
  const t = useTranslations();
  const [saving, setSaving] = useState<string | null>(null);
  const [showTemplates, setShowTemplates] = useState(!config.template_id);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);

  // Auto-save con debounce al hacer blur
  async function handleFieldSave(field: string, value: string) {
    setSaving(field);
    try {
      const updated = await chatbotApi.updateConfig({ [field]: value });
      onConfigUpdate(updated);
    } catch (err) {
      console.error(`Error saving ${field}:`, err);
    } finally {
      // Mostrar checkmark brevemente
      setTimeout(() => setSaving(null), 1000);
    }
  }

  // Seleccionar template
  async function handleTemplateSelect(template: ChatbotTemplate) {
    try {
      // 1. Actualizar config con defaults del template
      const updated = await chatbotApi.updateConfig({
        bot_name: template.defaults.bot_name,
        bot_greeting: template.defaults.bot_greeting,
        tone: template.defaults.tone,
        template_id: template.id,
      });
      onConfigUpdate(updated);

      // 2. Crear documentos sugeridos como drafts (is_active: false)
      for (const doc of template.suggested_docs) {
        try {
          await knowledgeApi.create({
            category: doc.category,
            title: doc.title,
            content: doc.content,
            is_active: false,
          });
        } catch (err) {
          console.error('Error creating suggested doc:', err);
        }
      }

      onDocsRefresh();
      setShowTemplates(false);
    } catch (err) {
      console.error('Error applying template:', err);
    }
  }

  const toneOptions = [
    { value: 'friendly', label: t.chatbot.personality.toneFriendly },
    { value: 'professional', label: t.chatbot.personality.toneProfessional },
    { value: 'premium', label: t.chatbot.personality.tonePremium },
    { value: 'casual', label: t.chatbot.personality.toneCasual },
  ] as const;

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Template Selector */}
      {showTemplates ? (
        <section>
          <h2 className="text-sm font-semibold text-neutral-900 mb-3">
            {t.chatbot.personality.templateTitle}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {CHATBOT_TEMPLATES.map((tmpl) => (
              <button
                key={tmpl.id}
                onClick={() => handleTemplateSelect(tmpl)}
                className="flex flex-col items-center gap-2 p-4 rounded-lg border border-neutral-200 hover:border-primary-400 hover:bg-primary-50 transition-colors text-center"
              >
                <span className="text-2xl">{tmpl.icon}</span>
                <span className="text-sm font-medium text-neutral-700">
                  {(t.chatbot.personality as Record<string, string>)[tmpl.id] ?? tmpl.name}
                </span>
              </button>
            ))}
          </div>
        </section>
      ) : (
        <button
          onClick={() => setShowTemplates(true)}
          className="text-sm text-primary-600 hover:underline"
        >
          {t.chatbot.personality.templateChange}
        </button>
      )}

      {/* Config Fields */}
      <section className="space-y-4">
        {/* Bot Name */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.botName}
            {saving === 'bot_name' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <input
            type="text"
            maxLength={30}
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            defaultValue={config.bot_name}
            onBlur={(e) => handleFieldSave('bot_name', e.target.value)}
          />
          <p className="text-xs text-neutral-500 mt-1">{t.chatbot.personality.botNameHint}</p>
        </div>

        {/* Bot Greeting */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.botGreeting}
            {saving === 'bot_greeting' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <textarea
            maxLength={500}
            rows={3}
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            defaultValue={config.bot_greeting}
            onBlur={(e) => handleFieldSave('bot_greeting', e.target.value)}
          />
          <p className="text-xs text-neutral-500 mt-1">{t.chatbot.personality.botGreetingHint}</p>
        </div>

        {/* Tone */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.tone}
            {saving === 'tone' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <div className="grid grid-cols-2 gap-2">
            {toneOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => handleFieldSave('tone', opt.value)}
                className={`px-3 py-2 rounded-lg border text-sm font-medium transition-colors ${
                  config.tone === opt.value
                    ? 'border-primary-500 bg-primary-50 text-primary-700'
                    : 'border-neutral-200 text-neutral-600 hover:border-neutral-300'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        {/* Custom Instructions */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.customInstructions}
            {saving === 'custom_instructions' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <textarea
            maxLength={1000}
            rows={4}
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            defaultValue={config.custom_instructions}
            onBlur={(e) => handleFieldSave('custom_instructions', e.target.value)}
          />
          <p className="text-xs text-neutral-500 mt-1">{t.chatbot.personality.customInstructionsHint}</p>
        </div>
      </section>
    </div>
  );
}
```

**Commit:** `feat(web): add PersonalityTab with template selector and config form`

---

## Task 13: Frontend — KnowledgeTab component

**Files:**
- `apps/web/app/dashboard/chatbot/components/KnowledgeTab.tsx` (new)

- [ ] Step 13.1: Create KnowledgeTab by extracting from knowledge/page.tsx

The KnowledgeTab wraps the existing knowledge page functionality. Since the knowledge page is 342 lines, the simplest approach is to extract the core CRUD component.

```typescript
'use client';

// Este componente migra la funcionalidad completa de /dashboard/knowledge/page.tsx.
// Se monta dentro del Chatbot Hub como tab "Conocimiento".
// La implementación es idéntica al componente original — solo cambia el wrapper.

import { useState, useEffect } from 'react';
import { useTranslations } from '@/lib/i18n';
import { knowledge as knowledgeApi, type KnowledgeDocument } from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { Plus, Upload, Pencil, Trash2, FileText, ToggleLeft, ToggleRight } from 'lucide-react';

interface Props {
  docs: KnowledgeDocument[];
  onRefresh: () => void;
}

// NOTA PARA EL AGENTE EJECUTOR:
// Migrar el contenido completo de apps/web/app/dashboard/knowledge/page.tsx a este componente.
// Cambios necesarios:
//   1. Convertir de page.tsx (con su propio fetch) a componente que recibe docs via props
//   2. Reemplazar el fetch interno por onRefresh() callback tras crear/editar/eliminar
//   3. Agregar badge "Sugerido por template" para docs con is_active=false que coincidan
//      con títulos de los templates (comparación por título del doc)
//   4. Mantener el modal de crear/editar, file upload, toggle activo/inactivo
//   5. Mantener las categorías: services, pricing, faq, policies, team, location, promotions
// El componente es prácticamente idéntico al original, solo cambia la fuente de datos.
export function KnowledgeTab({ docs, onRefresh }: Props) {
  const t = useTranslations();
  const [showForm, setShowForm] = useState(false);
  const [editingDoc, setEditingDoc] = useState<KnowledgeDocument | null>(null);
  const [formData, setFormData] = useState({
    category: 'faq',
    title: '',
    content: '',
    is_active: true,
  });
  const [saving, setSaving] = useState(false);
  const [uploadMode, setUploadMode] = useState(false);
  const [file, setFile] = useState<File | null>(null);

  // Categorías con labels traducidos
  const categories = [
    { value: 'services', label: t.knowledge.categories.services },
    { value: 'pricing', label: t.knowledge.categories.pricing },
    { value: 'faq', label: t.knowledge.categories.faq },
    { value: 'policies', label: t.knowledge.categories.policies },
    { value: 'team', label: t.knowledge.categories.team },
    { value: 'location', label: t.knowledge.categories.location },
    { value: 'promotions', label: t.knowledge.categories.promotions },
  ];

  function openCreate() {
    setEditingDoc(null);
    setFormData({ category: 'faq', title: '', content: '', is_active: true });
    setUploadMode(false);
    setFile(null);
    setShowForm(true);
  }

  function openEdit(doc: KnowledgeDocument) {
    setEditingDoc(doc);
    setFormData({
      category: doc.category,
      title: doc.title,
      content: doc.content,
      is_active: doc.is_active,
    });
    setUploadMode(false);
    setShowForm(true);
  }

  async function handleSave() {
    setSaving(true);
    try {
      if (uploadMode && file) {
        await knowledgeApi.upload(file, formData.title, formData.category, formData.is_active);
      } else if (editingDoc) {
        await knowledgeApi.update(editingDoc.id, formData);
      } else {
        await knowledgeApi.create(formData);
      }
      setShowForm(false);
      onRefresh();
    } catch (err) {
      console.error('Error saving document:', err);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm(t.knowledge.confirmDelete)) return;
    try {
      await knowledgeApi.remove(id);
      onRefresh();
    } catch (err) {
      console.error('Error deleting document:', err);
    }
  }

  async function handleToggle(doc: KnowledgeDocument) {
    try {
      await knowledgeApi.update(doc.id, {
        category: doc.category,
        title: doc.title,
        content: doc.content,
        is_active: !doc.is_active,
      });
      onRefresh();
    } catch (err) {
      console.error('Error toggling document:', err);
    }
  }

  return (
    <div className="space-y-4">
      {/* Header con botones */}
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-neutral-900">
          {t.knowledge.title} ({docs.length})
        </h2>
        <div className="flex gap-2">
          <Button size="sm" variant="secondary" onClick={() => { openCreate(); setUploadMode(true); }}>
            <Upload className="w-4 h-4 mr-1" /> {t.knowledge.upload}
          </Button>
          <Button size="sm" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" /> {t.knowledge.create}
          </Button>
        </div>
      </div>

      {/* Lista de documentos */}
      {docs.length === 0 ? (
        <Card className="p-8 text-center">
          <FileText className="w-8 h-8 mx-auto text-neutral-300 mb-2" />
          <p className="text-sm text-neutral-500">{t.knowledge.empty}</p>
        </Card>
      ) : (
        <div className="space-y-2">
          {docs.map(doc => (
            <Card key={doc.id} className="p-3 flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-neutral-900 truncate">{doc.title}</span>
                  <Badge variant={doc.is_active ? 'success' : 'default'} className="text-[10px]">
                    {doc.is_active ? t.common.active : t.common.inactive}
                  </Badge>
                  {!doc.is_active && doc.source_type === 'text' && (
                    <Badge variant="info" className="text-[10px]">
                      {t.chatbot.knowledgeTab.templateDraft}
                    </Badge>
                  )}
                  <Badge variant="default" className="text-[10px]">
                    {categories.find(c => c.value === doc.category)?.label ?? doc.category}
                  </Badge>
                </div>
                <p className="text-xs text-neutral-500 truncate">{doc.content.slice(0, 100)}</p>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <button onClick={() => handleToggle(doc)} className="p-1.5 rounded hover:bg-neutral-100" title={doc.is_active ? t.common.deactivate : t.common.activate}>
                  {doc.is_active ? <ToggleRight className="w-4 h-4 text-green-500" /> : <ToggleLeft className="w-4 h-4 text-neutral-400" />}
                </button>
                <button onClick={() => openEdit(doc)} className="p-1.5 rounded hover:bg-neutral-100">
                  <Pencil className="w-4 h-4 text-neutral-500" />
                </button>
                <button onClick={() => handleDelete(doc.id)} className="p-1.5 rounded hover:bg-red-50">
                  <Trash2 className="w-4 h-4 text-red-400" />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Modal/Form — simplificado para la implementación */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <h3 className="text-base font-semibold mb-4">
              {editingDoc ? t.knowledge.editTitle : uploadMode ? t.knowledge.uploadTitle : t.knowledge.createTitle}
            </h3>

            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-neutral-700 mb-1">{t.knowledge.category}</label>
                <select
                  className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                  value={formData.category}
                  onChange={e => setFormData(prev => ({ ...prev, category: e.target.value }))}
                >
                  {categories.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-neutral-700 mb-1">{t.knowledge.titleLabel}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                  value={formData.title}
                  onChange={e => setFormData(prev => ({ ...prev, title: e.target.value }))}
                />
              </div>

              {uploadMode ? (
                <div>
                  <label className="block text-sm font-medium text-neutral-700 mb-1">{t.knowledge.file}</label>
                  <input
                    type="file"
                    accept=".pdf,.docx,.doc,.xlsx,.xls,.csv"
                    className="w-full text-sm"
                    onChange={e => setFile(e.target.files?.[0] ?? null)}
                  />
                </div>
              ) : (
                <div>
                  <label className="block text-sm font-medium text-neutral-700 mb-1">{t.knowledge.content}</label>
                  <textarea
                    rows={8}
                    className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                    value={formData.content}
                    onChange={e => setFormData(prev => ({ ...prev, content: e.target.value }))}
                  />
                </div>
              )}
            </div>

            <div className="flex justify-end gap-2 mt-4">
              <Button variant="secondary" onClick={() => setShowForm(false)}>
                {t.common.cancel}
              </Button>
              <Button onClick={handleSave} loading={saving}>
                {t.common.save}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

**Commit:** `feat(web): add KnowledgeTab component migrated from knowledge page`

---

## Task 14: Frontend — TestChatPanel component

**Files:**
- `apps/web/app/dashboard/chatbot/components/TestChatPanel.tsx` (new)

- [ ] Step 14.1: Create TestChatPanel component

```typescript
'use client';

import { useState, useRef, useEffect } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, type ChatbotTestResponse } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Send, Trash2, ChevronDown, ChevronRight, PanelRightClose } from 'lucide-react';

interface ChatMessage {
  id: string;
  role: 'user' | 'bot';
  content: string;
  timestamp: Date;
  debug?: {
    intent: string;
    sources: string[];
    timeMs: number;
  };
}

interface Props {
  onMessageSent: () => void;
  onCollapse: () => void;
}

export function TestChatPanel({ onMessageSent, onCollapse }: Props) {
  const t = useTranslations();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [expandedDebug, setExpandedDebug] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll al final cuando hay nuevos mensajes
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  async function handleSend() {
    if (!input.trim() || sending) return;

    const userMsg: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setSending(true);
    onMessageSent();

    try {
      const resp = await chatbotApi.test(userMsg.content);
      const botMsg: ChatMessage = {
        id: `bot-${Date.now()}`,
        role: 'bot',
        content: resp.response,
        timestamp: new Date(),
        debug: {
          intent: resp.intent_detected,
          sources: resp.rag_sources_used,
          timeMs: resp.processing_time_ms,
        },
      };
      setMessages(prev => [...prev, botMsg]);
    } catch (err) {
      const errorMsg: ChatMessage = {
        id: `error-${Date.now()}`,
        role: 'bot',
        content: 'Error al procesar el mensaje. Verificá que el servicio de IA esté activo.',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMsg]);
    } finally {
      setSending(false);
      inputRef.current?.focus();
    }
  }

  function handleClear() {
    setMessages([]);
    setExpandedDebug(null);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-neutral-200 shrink-0">
        <h3 className="text-sm font-semibold text-neutral-900">{t.chatbot.testChat.title}</h3>
        <div className="flex gap-1">
          <button onClick={handleClear} className="p-1 rounded hover:bg-neutral-100" title={t.chatbot.testChat.clear}>
            <Trash2 className="w-4 h-4 text-neutral-400" />
          </button>
          <button onClick={onCollapse} className="p-1 rounded hover:bg-neutral-100" title={t.chatbot.testChat.collapse}>
            <PanelRightClose className="w-4 h-4 text-neutral-400" />
          </button>
        </div>
      </div>

      {/* Banner */}
      <div className="px-3 py-1.5 bg-amber-50 border-b border-amber-100 shrink-0">
        <p className="text-[11px] text-amber-700">{t.chatbot.testChat.banner}</p>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {messages.length === 0 ? (
          <p className="text-sm text-neutral-400 text-center mt-8">{t.chatbot.testChat.empty}</p>
        ) : (
          messages.map(msg => (
            <div key={msg.id}>
              <div className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                  msg.role === 'user'
                    ? 'bg-primary-600 text-white'
                    : 'bg-neutral-100 text-neutral-800'
                }`}>
                  <p className="whitespace-pre-wrap">{msg.content}</p>
                  <p className={`text-[10px] mt-1 ${
                    msg.role === 'user' ? 'text-primary-200' : 'text-neutral-400'
                  }`}>
                    {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  </p>
                </div>
              </div>

              {/* Debug metadata (collapsed) */}
              {msg.debug && (
                <div className="ml-1 mt-1">
                  <button
                    onClick={() => setExpandedDebug(expandedDebug === msg.id ? null : msg.id)}
                    className="flex items-center gap-1 text-[10px] text-neutral-400 hover:text-neutral-600"
                  >
                    {expandedDebug === msg.id ? (
                      <ChevronDown className="w-3 h-3" />
                    ) : (
                      <ChevronRight className="w-3 h-3" />
                    )}
                    {t.chatbot.testChat.debug}
                  </button>
                  {expandedDebug === msg.id && (
                    <div className="mt-1 ml-4 text-[10px] text-neutral-400 space-y-0.5">
                      <p>{t.chatbot.testChat.intent}: <span className="text-neutral-600">{msg.debug.intent}</span></p>
                      {msg.debug.sources.length > 0 && (
                        <p>{t.chatbot.testChat.sources}: <span className="text-neutral-600">{msg.debug.sources.join(', ')}</span></p>
                      )}
                      <p>{t.chatbot.testChat.time}: <span className="text-neutral-600">{msg.debug.timeMs}ms</span></p>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="px-3 py-2 border-t border-neutral-200 shrink-0">
        <div className="flex gap-2">
          <input
            ref={inputRef}
            type="text"
            className="flex-1 rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            placeholder={t.chatbot.testChat.placeholder}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={sending}
          />
          <Button size="sm" onClick={handleSend} loading={sending} disabled={!input.trim()}>
            <Send className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
```

**Commit:** `feat(web): add TestChatPanel with chat UI and debug metadata`

---

## Task 15: Frontend — ValidationTab component

**Files:**
- `apps/web/app/dashboard/chatbot/components/ValidationTab.tsx` (new)

- [ ] Step 15.1: Create ValidationTab component

```typescript
'use client';

import { useState } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, type ChatbotConfig, type KnowledgeDocument, type ChatbotValidateResponse } from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { CheckCircle2, Circle, AlertCircle, ArrowRight, Sparkles } from 'lucide-react';

interface Props {
  config: ChatbotConfig;
  docs: KnowledgeDocument[];
  chatTested: boolean;
}

interface StaticRule {
  key: string;
  label: string;
  passed: boolean;
  suggestion: string;
  tab?: string;
}

export function ValidationTab({ config, docs, chatTested }: Props) {
  const t = useTranslations();
  const [aiReport, setAiReport] = useState<ChatbotValidateResponse | null>(null);
  const [validating, setValidating] = useState(false);
  const [rateLimited, setRateLimited] = useState(false);

  const activeDocs = docs.filter(d => d.is_active);
  const hasActiveByCategory = (cat: string) => activeDocs.some(d => d.category === cat);

  const staticRules: StaticRule[] = [
    {
      key: 'botName',
      label: t.chatbot.progress.botName,
      passed: Boolean(config.bot_name),
      suggestion: t.chatbot.validation.rules.botName,
      tab: 'personality',
    },
    {
      key: 'greeting',
      label: t.chatbot.progress.greeting,
      passed: Boolean(config.bot_greeting),
      suggestion: t.chatbot.validation.rules.greeting,
      tab: 'personality',
    },
    {
      key: 'activeDocs',
      label: t.chatbot.progress.docs,
      passed: activeDocs.length > 0,
      suggestion: t.chatbot.validation.rules.activeDocs,
      tab: 'knowledge',
    },
    {
      key: 'faq',
      label: 'FAQ',
      passed: hasActiveByCategory('faq'),
      suggestion: t.chatbot.validation.rules.faqCategory,
      tab: 'knowledge',
    },
    {
      key: 'pricing',
      label: 'Precios',
      passed: hasActiveByCategory('pricing') || hasActiveByCategory('services'),
      suggestion: t.chatbot.validation.rules.pricingDocs,
      tab: 'knowledge',
    },
    {
      key: 'policies',
      label: 'Políticas',
      passed: hasActiveByCategory('policies'),
      suggestion: t.chatbot.validation.rules.policyDocs,
      tab: 'knowledge',
    },
    {
      key: 'chatTested',
      label: t.chatbot.progress.tested,
      passed: chatTested,
      suggestion: t.chatbot.validation.rules.chatTested,
    },
  ];

  async function handleValidate() {
    setValidating(true);
    setRateLimited(false);
    try {
      const report = await chatbotApi.validate();
      setAiReport(report);
    } catch (err: any) {
      if (err?.status === 429) {
        setRateLimited(true);
      } else {
        console.error('Error validating:', err);
      }
    } finally {
      setValidating(false);
    }
  }

  const canValidate = activeDocs.length > 0;

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Static Checklist */}
      <section>
        <h2 className="text-sm font-semibold text-neutral-900 mb-3">
          {t.chatbot.validation.staticTitle}
        </h2>
        <Card className="divide-y divide-neutral-100">
          {staticRules.map(rule => (
            <div key={rule.key} className="flex items-start gap-3 px-4 py-3">
              {rule.passed ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0 mt-0.5" />
              ) : (
                <Circle className="w-5 h-5 text-neutral-300 shrink-0 mt-0.5" />
              )}
              <div className="flex-1">
                <p className={`text-sm ${rule.passed ? 'text-neutral-700' : 'text-neutral-500'}`}>
                  {rule.label}
                </p>
                {!rule.passed && (
                  <p className="text-xs text-amber-600 mt-0.5">{rule.suggestion}</p>
                )}
              </div>
            </div>
          ))}
        </Card>
      </section>

      {/* AI Validator */}
      <section>
        <h2 className="text-sm font-semibold text-neutral-900 mb-3">
          {t.chatbot.validation.aiTitle}
        </h2>

        <Button
          onClick={handleValidate}
          disabled={!canValidate || validating}
          loading={validating}
        >
          <Sparkles className="w-4 h-4 mr-1" />
          {validating ? t.chatbot.validation.aiRunning : t.chatbot.validation.aiButton}
        </Button>

        {!canValidate && (
          <p className="text-xs text-neutral-400 mt-2">{t.chatbot.validation.aiDisabled}</p>
        )}

        {rateLimited && (
          <p className="text-xs text-amber-600 mt-2">{t.chatbot.validation.aiRateLimit}</p>
        )}

        {/* AI Report */}
        {aiReport && (
          <Card className="mt-4">
            <div className="px-4 py-3 border-b border-neutral-100">
              <p className="text-sm font-medium text-neutral-900">
                {t.chatbot.validation.result
                  .replace('{passed}', String(aiReport.passed))
                  .replace('{total}', String(aiReport.total_questions))}
              </p>
              <div className="w-full bg-neutral-100 rounded-full h-1.5 mt-2">
                <div
                  className="bg-green-500 h-1.5 rounded-full transition-all"
                  style={{ width: `${(aiReport.passed / aiReport.total_questions) * 100}%` }}
                />
              </div>
            </div>

            <div className="divide-y divide-neutral-100">
              {aiReport.results.map((result, idx) => (
                <div key={idx} className="px-4 py-3">
                  <div className="flex items-start gap-2">
                    {result.passed ? (
                      <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0 mt-0.5" />
                    ) : (
                      <AlertCircle className="w-4 h-4 text-red-400 shrink-0 mt-0.5" />
                    )}
                    <div className="flex-1">
                      <p className="text-sm text-neutral-700">"{result.question}"</p>
                      <p className="text-xs text-neutral-500 mt-1 line-clamp-2">
                        {result.response}
                      </p>
                      {!result.passed && result.suggestion && (
                        <p className="text-xs text-amber-600 mt-1 flex items-center gap-1">
                          <ArrowRight className="w-3 h-3" /> {result.suggestion}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        )}
      </section>
    </div>
  );
}
```

**Commit:** `feat(web): add ValidationTab with static checklist and AI validator`

---

## Task 16: Frontend — Settings cleanup

**Files:**
- `apps/web/app/dashboard/settings/customization/page.tsx`

- [ ] Step 16.1: Remove bot_name/bot_greeting fields and add redirect link

In `customization/page.tsx`, replace the "Asistente IA" section (lines ~97-123) with:

```typescript
{/* ── Asistente IA (migrado a Chatbot Hub) ─────────────────────────── */}
<section className="space-y-4 border-t border-neutral-100 pt-6">
  <h2 className="text-base font-semibold text-neutral-900">Asistente IA por WhatsApp</h2>
  <p className="text-sm text-neutral-500">
    La configuración del asistente se movió al nuevo hub de Chatbot.
  </p>
  <a
    href="/dashboard/chatbot"
    className="inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700"
  >
    Ir a Chatbot <span aria-hidden="true">&rarr;</span>
  </a>
</section>
```

**Commit:** `refactor(settings): replace bot config with link to chatbot hub`

---

## Task 17: Delete old knowledge page route

**Files:**
- `apps/web/app/dashboard/knowledge/page.tsx` (delete or redirect)

- [ ] Step 17.1: Replace knowledge page with redirect to chatbot

Instead of deleting the file (to avoid broken bookmarks), replace the content with a redirect:

```typescript
import { redirect } from 'next/navigation';

export default function KnowledgePage() {
  redirect('/dashboard/chatbot');
}
```

**Commit:** `refactor(web): redirect /dashboard/knowledge to /dashboard/chatbot`

---

## Summary of All Commits

1. `feat(db): add chatbot_configs table with data migration from tenant settings`
2. `feat(domain): add ChatbotConfig types, interfaces, and rate limit error`
3. `feat(repo): add chatbot config repository with upsert pattern`
4. `feat(service): add chatbot service with test and validation logic`
5. `feat(api): add chatbot handler with config, test, and validate endpoints`
6. `feat(ai): add synchronous process-test endpoint with tone and custom instructions support`
7. `feat(web): add chatbot API client with types`
8. `feat(i18n): add chatbot hub translations for es, en, pt`
9. `feat(nav): replace knowledge with chatbot in dashboard sidebar`
10. `feat(web): add chatbot hub page with 3-column layout`
11. `feat(web): add ProgressChecklist component for chatbot hub`
12. `feat(web): add PersonalityTab with template selector and config form`
13. `feat(web): add KnowledgeTab component migrated from knowledge page`
14. `feat(web): add TestChatPanel with chat UI and debug metadata`
15. `feat(web): add ValidationTab with static checklist and AI validator`
16. `refactor(settings): replace bot config with link to chatbot hub`
17. `refactor(web): redirect /dashboard/knowledge to /dashboard/chatbot`

## Files Touched

### New files (14)
- `apps/api/db/migrations/033_chatbot_configs.sql`
- `apps/api/internal/repository/chatbot_config.go`
- `apps/api/internal/service/chatbot.go`
- `apps/api/internal/handler/chatbot.go`
- `apps/ai/app/routers/process_test.py`
- `apps/web/app/dashboard/chatbot/page.tsx`
- `apps/web/app/dashboard/chatbot/data/templates.ts`
- `apps/web/app/dashboard/chatbot/components/ProgressChecklist.tsx`
- `apps/web/app/dashboard/chatbot/components/PersonalityTab.tsx`
- `apps/web/app/dashboard/chatbot/components/KnowledgeTab.tsx`
- `apps/web/app/dashboard/chatbot/components/TestChatPanel.tsx`
- `apps/web/app/dashboard/chatbot/components/ValidationTab.tsx`

### Modified files (12)
- `apps/api/internal/domain/types.go`
- `apps/api/internal/domain/interfaces.go`
- `apps/api/internal/domain/errors.go`
- `apps/api/internal/handler/errors.go`
- `apps/api/cmd/server/main.go`
- `apps/ai/app/agent/orchestrator.py`
- `apps/ai/app/agent/messages.py` (no changes needed beyond tone mapping)
- `apps/ai/app/agent/state.py`
- `apps/ai/main.py`
- `apps/web/lib/api.ts`
- `apps/web/lib/i18n/locales/es.ts`
- `apps/web/lib/i18n/locales/en.ts`
- `apps/web/lib/i18n/locales/pt.ts`
- `apps/web/components/dashboard/sidebar.tsx`
- `apps/web/app/dashboard/settings/customization/page.tsx`
- `apps/web/app/dashboard/knowledge/page.tsx`

## Dependencies Between Tasks

```
Task 1 (migration)
    |
Task 2 (domain types) ──────────────────┐
    |                                    |
Task 3 (repository) ──> Task 4 (service) ──> Task 5 (handler + routes)
                                         |
Task 6 (AI service) ────────────────────┘
    |
Task 7 (frontend API) ──> Task 8 (i18n) ──> Task 9 (nav)
    |                                        |
    └──> Task 10 (layout) ──> Task 11-15 (components) ──> Task 16-17 (cleanup)
```

Tasks 1-6 (backend) can be done in one batch. Tasks 7-17 (frontend) in another.
Within the backend: 1→2→3→4→5 is sequential; Task 6 is independent of 3-5.
Within the frontend: 7→8 first, then 9-15 can be partially parallelized, 16-17 last.
