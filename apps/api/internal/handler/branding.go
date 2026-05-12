package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

// BrandingHandler endpoints para uploads de logo/portada del tenant.
type BrandingHandler struct {
	svc domain.BrandingService
}

// NewBrandingHandler crea el handler de branding.
func NewBrandingHandler(svc domain.BrandingService) *BrandingHandler {
	return &BrandingHandler{svc: svc}
}

// UploadLogo POST /api/v1/tenant/branding/logo
func (h *BrandingHandler) UploadLogo(c *fiber.Ctx) error {
	return h.upload(c, domain.BrandingAssetLogo)
}

// UploadCover POST /api/v1/tenant/branding/cover
func (h *BrandingHandler) UploadCover(c *fiber.Ctx) error {
	return h.upload(c, domain.BrandingAssetCover)
}

// Remove DELETE /api/v1/tenant/branding/:kind  (kind = logo | cover)
func (h *BrandingHandler) Remove(c *fiber.Ctx) error {
	if h.svc == nil {
		return fiber.NewError(http.StatusServiceUnavailable, "Almacenamiento de imágenes no disponible — configurá las variables MINIO_*")
	}

	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	kind, err := parseAssetKind(c.Params("kind"))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	if err := h.svc.RemoveAsset(c.Context(), tenantID, kind); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

func (h *BrandingHandler) upload(c *fiber.Ctx, kind domain.BrandingAssetKind) error {
	if h.svc == nil {
		return fiber.NewError(http.StatusServiceUnavailable, "Almacenamiento de imágenes no disponible — configurá las variables MINIO_*")
	}

	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "Se requiere un archivo (field: file)")
	}

	f, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, "Error al leer el archivo")
	}
	defer f.Close()

	// Algunos browsers omiten el Content-Type del part multipart.
	// Si no está, se detecta leyendo los primeros 512 bytes (sniffing RFC 9110).
	contentType := fileHeader.Header.Get("Content-Type")
	var reader io.Reader = f
	if contentType == "" {
		sniff := make([]byte, 512)
		n, _ := io.ReadFull(f, sniff)
		contentType = http.DetectContentType(sniff[:n])
		reader = io.MultiReader(bytes.NewReader(sniff[:n]), f)
	}

	url, err := h.svc.UploadAsset(c.Context(), tenantID, &domain.BrandingUploadInput{
		Kind:        kind,
		Filename:    fileHeader.Filename,
		ContentType: contentType,
		Reader:      reader,
		Size:        fileHeader.Size,
	})
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"url": url})
}

func parseAssetKind(raw string) (domain.BrandingAssetKind, error) {
	switch domain.BrandingAssetKind(raw) {
	case domain.BrandingAssetLogo:
		return domain.BrandingAssetLogo, nil
	case domain.BrandingAssetCover:
		return domain.BrandingAssetCover, nil
	default:
		return "", fiber.NewError(http.StatusBadRequest, "kind inválido (use 'logo' o 'cover')")
	}
}
