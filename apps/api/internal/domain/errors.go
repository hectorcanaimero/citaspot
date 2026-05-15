// Package domain contiene los tipos, interfaces y errores de dominio del Core API.
// Sin dependencias externas — solo stdlib.
package domain

import "errors"

// Errores de dominio tipados.
// Usar errors.Is() en los handlers para mapear a códigos HTTP.
var (
	// ErrNotFound se retorna cuando un recurso no existe o no pertenece al tenant.
	ErrNotFound = errors.New("recurso no encontrado")

	// ErrUnauthorized se retorna cuando el token JWT es inválido o falta.
	ErrUnauthorized = errors.New("no autorizado")

	// ErrForbidden se retorna cuando el tenant no tiene acceso al recurso.
	ErrForbidden = errors.New("acceso denegado")

	// ErrSlotUnavailable se retorna cuando el horario solicitado ya está ocupado.
	ErrSlotUnavailable = errors.New("el horario solicitado no está disponible")

	// ErrConflict se retorna cuando hay un conflicto con un recurso existente.
	ErrConflict = errors.New("conflicto con un recurso existente")

	// ErrValidation se retorna cuando los datos de entrada no son válidos.
	ErrValidation = errors.New("datos inválidos")

	// ErrInternal se retorna para errores internos no esperados.
	ErrInternal = errors.New("error interno del servidor")

	// ErrEmailAlreadyExists se retorna cuando el email ya existe en Supabase Auth.
	ErrEmailAlreadyExists = errors.New("email ya registrado")

	// ErrSlugAlreadyExists se retorna cuando el slug de negocio ya está en uso.
	ErrSlugAlreadyExists = errors.New("slug ya existe")

	// ErrInvalidCredentials se retorna cuando el email o contraseña son incorrectos.
	ErrInvalidCredentials = errors.New("credenciales inválidas")

	// ErrInvalidToken se retorna cuando el JWT o refresh_token es inválido o expirado.
	ErrInvalidToken = errors.New("token inválido")

	// ErrWAPermanentFailure se retorna cuando WhatsApp rechaza el envío de forma permanente
	// (ej: número no existe). No tiene sentido reintentar estos mensajes.
	ErrWAPermanentFailure = errors.New("whatsapp: fallo permanente de envío")

	// ErrInvalidStatusTransition se retorna cuando un cambio de estado no es valido.
	ErrInvalidStatusTransition = errors.New("transición de estado no válida")

	// ErrClinicalNoteExists se retorna cuando ya existe una nota clínica para la cita.
	ErrClinicalNoteExists = errors.New("ya existe una nota clínica para esta cita")

	// ErrAppointmentNotCompleted se retorna cuando se intenta crear una nota en una cita no completada.
	ErrAppointmentNotCompleted = errors.New("solo se pueden crear notas para citas completadas")

	// ErrFileTooLarge se retorna cuando un archivo excede el tamaño máximo.
	ErrFileTooLarge = errors.New("el archivo excede el tamaño máximo permitido")

	// ErrFileTypeNotAllowed se retorna cuando el tipo de archivo no está permitido.
	ErrFileTypeNotAllowed = errors.New("tipo de archivo no permitido")

	// ErrMaxFilesReached se retorna cuando se alcanzó el límite de archivos por nota.
	ErrMaxFilesReached = errors.New("se alcanzó el límite máximo de archivos por nota")

	// ErrRateLimited se retorna cuando se excede el límite de solicitudes.
	ErrRateLimited = errors.New("demasiadas solicitudes, intentá de nuevo más tarde")
)
