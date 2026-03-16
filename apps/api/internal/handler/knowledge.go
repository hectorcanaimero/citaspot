package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// KnowledgeHandler maneja los endpoints CRUD de la knowledge base.
type KnowledgeHandler struct {
	svc      domain.KnowledgeSvc
	validate *validator.Validate
}

// NewKnowledgeHandler crea el handler de knowledge base.
func NewKnowledgeHandler(svc domain.KnowledgeSvc) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc, validate: validator.New()}
}

// List GET /api/v1/knowledge
func (h *KnowledgeHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	docs, err := h.svc.List(c.Context(), tenantID)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": docs})
}

// Get GET /api/v1/knowledge/:id
func (h *KnowledgeHandler) Get(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	doc, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(doc)
}

// Create POST /api/v1/knowledge
func (h *KnowledgeHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	var input domain.KnowledgeDocumentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{Error: err.Error()})
	}

	doc, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(doc)
}

// Upload POST /api/v1/knowledge/upload — crea un documento desde un archivo (PDF/DOCX/XLSX).
func (h *KnowledgeHandler) Upload(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	// Leer campos de texto del formulario multipart
	input := domain.KnowledgeUploadInput{
		Category: c.FormValue("category"),
		Title:    c.FormValue("title"),
	}
	if v := c.FormValue("is_active"); v == "false" {
		f := false
		input.IsActive = &f
	}

	if err := h.validate.Struct(&input); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{Error: err.Error()})
	}

	// Obtener el archivo del formulario
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Se requiere un archivo (field: file)"})
	}

	// Verificar tipo de archivo por extensión
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExt := map[string]string{
		".pdf":  "pdf",
		".docx": "docx",
		".doc":  "docx",
		".xlsx": "xlsx",
		".xls":  "xlsx",
		".csv":  "csv",
	}
	fileType, ok := allowedExt[ext]
	if !ok {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{
			Error: "Tipo de archivo no soportado. Use PDF, DOCX, XLSX o CSV",
		})
	}

	// Leer bytes del archivo
	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al leer el archivo"})
	}
	defer f.Close()

	fileBytes := make([]byte, fileHeader.Size)
	if _, err := f.Read(fileBytes); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "Error al leer el archivo"})
	}

	doc, err := h.svc.Upload(c.Context(), tenantID, &input, fileHeader.Filename, fileBytes, fileType)
	if err != nil {
		if err.Error() == "el archivo supera el límite de 8 MB" {
			return c.Status(http.StatusRequestEntityTooLarge).JSON(errorResponse{Error: err.Error()})
		}
		return handleServiceError(c, err)
	}

	// 202 Accepted — el documento existe pero el contenido aún se está procesando
	return c.Status(http.StatusAccepted).JSON(doc)
}

// Update PUT /api/v1/knowledge/:id
func (h *KnowledgeHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.KnowledgeDocumentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(&input); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(errorResponse{Error: err.Error()})
	}

	doc, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(doc)
}

// Delete DELETE /api/v1/knowledge/:id
func (h *KnowledgeHandler) Delete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "No autorizado"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	if err := h.svc.Delete(c.Context(), tenantID, id); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
