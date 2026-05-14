package handler

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// ClinicalFileHandler maneja endpoints de archivos clínicos.
type ClinicalFileHandler struct {
	svc domain.ClinicalFileSvc
}

func NewClinicalFileHandler(svc domain.ClinicalFileSvc) *ClinicalFileHandler {
	return &ClinicalFileHandler{svc: svc}
}

// Upload POST /api/v1/clinical-notes/:noteId/files/upload
func (h *ClinicalFileHandler) Upload(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	noteID, err := uuid.Parse(c.Params("noteId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de nota inválido"})
	}

	// Obtener archivo del formulario multipart
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Se requiere un archivo (field: file)"})
	}

	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al leer el archivo"})
	}
	defer f.Close()

	// Resolver professional_id del usuario autenticado o del form
	var uploadedBy uuid.UUID
	if v := c.FormValue("professional_id"); v != "" {
		uploadedBy, err = uuid.Parse(v)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id inválido"})
		}
	} else {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Se requiere professional_id"})
	}

	input := domain.ClinicalFileUploadInput{
		FileName:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Reader:      f,
		Size:        fileHeader.Size,
		Category:    c.FormValue("category", "other"),
		Description: c.FormValue("description"),
		UploadedBy:  uploadedBy,
	}

	file, err := h.svc.Upload(c.Context(), tenantID, noteID, input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(file)
}

// ListByNote GET /api/v1/clinical-notes/:noteId/files
func (h *ClinicalFileHandler) ListByNote(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	noteID, err := uuid.Parse(c.Params("noteId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de nota inválido"})
	}

	files, err := h.svc.ListByNote(c.Context(), tenantID, noteID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": files})
}

// ListByCustomer GET /api/v1/customers/:id/clinical-files
func (h *ClinicalFileHandler) ListByCustomer(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	customerID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de cliente inválido"})
	}

	category := c.Query("category")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	files, total, err := h.svc.ListByCustomer(c.Context(), tenantID, customerID, category, limit, offset)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": files, "total": total})
}

// Delete DELETE /api/v1/clinical-files/:fileId
func (h *ClinicalFileHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	fileID, err := uuid.Parse(c.Params("fileId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID de archivo inválido"})
	}

	if err := h.svc.Delete(c.Context(), tenantID, fileID); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
