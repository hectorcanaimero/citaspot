// Package service — lógica de negocio para el chatbot IA.
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
	repo          domain.ChatbotConfigRepository
	knowledgeRepo domain.KnowledgeRepository
	authRepo      domain.AuthRepository
	aiServiceURL  string
	httpClient    *http.Client
	rdb           *redis.Client
}

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

type aiTestRequest struct {
	TenantID       string `json:"tenant_id"`
	TenantSlug     string `json:"tenant_slug"`
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message_text"`
	Test           bool   `json:"test"`
}

type aiTestResponse struct {
	Response         string   `json:"response"`
	IntentDetected   string   `json:"intent_detected"`
	RAGSourcesUsed   []string `json:"rag_sources_used"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
}

func (s *chatbotSvc) Test(ctx context.Context, tenantID uuid.UUID, tenantSlug string, req *domain.ChatbotTestRequest) (*domain.ChatbotTestResponse, error) {
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

func validateRateLimitKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("chatbot:validate:ratelimit:%s", tenantID.String())
}

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

	questions = append(questions, "¿Aceptan tarjeta?")

	return questions
}

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

func isVagueResponse(response string) bool {
	lower := strings.ToLower(response)
	for _, phrase := range vaguePhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

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

func (s *chatbotSvc) Validate(ctx context.Context, tenantID uuid.UUID, tenantSlug string) (*domain.ChatbotValidateResponse, error) {
	if s.rdb != nil {
		key := validateRateLimitKey(tenantID)
		exists, err := s.rdb.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return nil, domain.ErrRateLimited
		}
	}

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

	if s.rdb != nil {
		_ = s.rdb.Set(ctx, validateRateLimitKey(tenantID), "1", time.Hour).Err()
	}

	return &domain.ChatbotValidateResponse{
		TotalQuestions: len(questions),
		Passed:         passed,
		Results:        results,
	}, nil
}
