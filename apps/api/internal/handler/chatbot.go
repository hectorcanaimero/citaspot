// Package handler — HTTP handlers para el chatbot IA.
package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ChatbotHandler maneja los endpoints de configuración y prueba del chatbot IA.
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
