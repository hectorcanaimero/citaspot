package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/citaspot/api/internal/domain"
)

// AuthHandler maneja los endpoints de autenticación.
type AuthHandler struct {
	svc      domain.AuthService
	validate *validator.Validate
}

// NewAuthHandler crea el handler con el servicio de auth inyectado.
func NewAuthHandler(svc domain.AuthService) *AuthHandler {
	return &AuthHandler{
		svc:      svc,
		validate: validator.New(),
	}
}

// Register godoc
// POST /api/v1/auth/register
// Crea un nuevo tenant y su usuario propietario.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req domain.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	resp, err := h.svc.Register(c.Context(), &req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(resp)
}

// Login godoc
// POST /api/v1/auth/login
// Autentica un usuario existente y retorna tokens JWT.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req domain.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	resp, err := h.svc.Login(c.Context(), &req)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// RefreshToken godoc
// POST /api/v1/auth/refresh
// Renueva el access_token usando el refresh_token.
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "refresh_token requerido"})
	}

	resp, err := h.svc.RefreshToken(c.Context(), body.RefreshToken)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// validationMessage convierte errores de validator en mensajes amigables.
func validationMessage(err error) string {
	if ve, ok := err.(validator.ValidationErrors); ok && len(ve) > 0 {
		f := ve[0]
		switch f.Tag() {
		case "required":
			return "el campo '" + f.Field() + "' es requerido"
		case "email":
			return "el campo 'email' debe ser una dirección válida"
		case "min":
			return "el campo '" + f.Field() + "' es demasiado corto"
		case "oneof":
			return "el campo '" + f.Field() + "' tiene un valor no permitido"
		case "len":
			return "el campo '" + f.Field() + "' tiene longitud incorrecta"
		}
	}
	return err.Error()
}
