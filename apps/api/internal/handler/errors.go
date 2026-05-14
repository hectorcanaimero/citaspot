// Package handler contiene los handlers HTTP — solo reciben requests, delegan al service.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/citaspot/api/internal/domain"
)

// errorResponse es el formato estándar de error de la API.
// Code es un código legible por máquina (e.g. "not_found") — el frontend lo usa para i18n.
// Message es el mensaje en español por defecto (fallback si el frontend no tiene el código).
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Mantener "error" por compatibilidad con clientes que ya usan este campo
	Error string `json:"error"`
}

func newError(code, message string) errorResponse {
	return errorResponse{Code: code, Message: message, Error: message}
}

// handleServiceError mapea errores de dominio a respuestas HTTP apropiadas.
// Nunca expone stack traces ni detalles internos al cliente.
func handleServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(http.StatusNotFound).JSON(newError("not_found", "Recurso no encontrado"))
	case errors.Is(err, domain.ErrUnauthorized):
		return c.Status(http.StatusUnauthorized).JSON(newError("unauthorized", "No autorizado"))
	case errors.Is(err, domain.ErrInvalidToken):
		return c.Status(http.StatusUnauthorized).JSON(newError("invalid_token", "Token inválido o expirado"))
	case errors.Is(err, domain.ErrInvalidCredentials):
		return c.Status(http.StatusUnauthorized).JSON(newError("invalid_credentials", "Email o contraseña incorrectos"))
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return c.Status(http.StatusConflict).JSON(newError("email_exists", "Ya existe una cuenta con ese email"))
	case errors.Is(err, domain.ErrSlugAlreadyExists):
		return c.Status(http.StatusConflict).JSON(newError("slug_exists", "Ese nombre de negocio ya está en uso"))
	case errors.Is(err, domain.ErrSlotUnavailable):
		return c.Status(http.StatusConflict).JSON(newError("slot_unavailable", err.Error()))
	case errors.Is(err, domain.ErrConflict):
		return c.Status(http.StatusConflict).JSON(newError("conflict", err.Error()))
	case errors.Is(err, domain.ErrForbidden):
		return c.Status(http.StatusForbidden).JSON(newError("forbidden", "Acceso denegado"))
	case errors.Is(err, domain.ErrValidation):
		return c.Status(http.StatusBadRequest).JSON(newError("validation_error", err.Error()))
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		return c.Status(http.StatusBadRequest).JSON(newError("invalid_status_transition", err.Error()))
	case errors.Is(err, domain.ErrClinicalNoteExists):
		return c.Status(http.StatusConflict).JSON(newError("clinical_note_exists", err.Error()))
	case errors.Is(err, domain.ErrAppointmentNotCompleted):
		return c.Status(http.StatusBadRequest).JSON(newError("appointment_not_completed", err.Error()))
	case errors.Is(err, domain.ErrFileTooLarge):
		return c.Status(http.StatusRequestEntityTooLarge).JSON(newError("file_too_large", err.Error()))
	case errors.Is(err, domain.ErrFileTypeNotAllowed):
		return c.Status(http.StatusUnprocessableEntity).JSON(newError("file_type_not_allowed", err.Error()))
	case errors.Is(err, domain.ErrMaxFilesReached):
		return c.Status(http.StatusConflict).JSON(newError("max_files_reached", err.Error()))
	default:
		// Error inesperado — loggear internamente, no exponer detalles al cliente
		slog.Error("handler internal error",
			"request_id", c.GetRespHeader("X-Request-ID"),
			"method", c.Method(),
			"path", c.Path(),
			"error", err,
		)
		return c.Status(http.StatusInternalServerError).JSON(newError("internal", "Error interno del servidor"))
	}
}
