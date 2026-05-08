package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/client/storage"
	"github.com/citaspot/api/internal/domain"
)

// brandingSvc implementa domain.BrandingService.
type brandingSvc struct {
	storage  *storage.Client
	authRepo domain.AuthRepository
}

// NewBrandingSvc crea el servicio de branding.
func NewBrandingSvc(s *storage.Client, authRepo domain.AuthRepository) domain.BrandingService {
	return &brandingSvc{storage: s, authRepo: authRepo}
}

// Límites por tipo de asset.
var brandingLimits = map[domain.BrandingAssetKind]int64{
	domain.BrandingAssetLogo:  2 * 1024 * 1024, // 2 MB
	domain.BrandingAssetCover: 5 * 1024 * 1024, // 5 MB
}

// MIME types permitidos para imágenes.
var allowedImageMIME = map[string]string{
	"image/jpeg": "jpg",
	"image/jpg":  "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// UploadAsset valida, sube a MinIO y persiste la URL en TenantSettings.
func (s *brandingSvc) UploadAsset(ctx context.Context, tenantID uuid.UUID, in *domain.BrandingUploadInput) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("brandingSvc.UploadAsset: storage no configurado")
	}
	if in == nil || in.Reader == nil {
		return "", domain.ErrValidation
	}

	limit, ok := brandingLimits[in.Kind]
	if !ok {
		return "", fmt.Errorf("brandingSvc.UploadAsset: %w: kind inválido", domain.ErrValidation)
	}
	if in.Size <= 0 || in.Size > limit {
		return "", fmt.Errorf("brandingSvc.UploadAsset: %w: archivo excede %d bytes", domain.ErrValidation, limit)
	}

	ext, ok := allowedImageMIME[strings.ToLower(in.ContentType)]
	if !ok {
		return "", fmt.Errorf("brandingSvc.UploadAsset: %w: tipo no soportado (use JPG, PNG o WebP)", domain.ErrValidation)
	}

	settings, err := s.authRepo.GetTenantSettings(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("brandingSvc.UploadAsset: get settings: %w", err)
	}

	// Borrar el asset previo (best effort) para no acumular huérfanos.
	if prev := assetURL(settings, in.Kind); prev != "" {
		if key := s.storage.KeyFromURL(prev); key != "" {
			_ = s.storage.Delete(ctx, key)
		}
	}

	key := fmt.Sprintf("%s/%s-%d.%s", tenantID.String(), in.Kind, time.Now().Unix(), ext)
	url, err := s.storage.Put(ctx, key, in.Reader, in.Size, in.ContentType)
	if err != nil {
		return "", fmt.Errorf("brandingSvc.UploadAsset: put: %w", err)
	}

	setAssetURL(settings, in.Kind, url)
	if err := s.authRepo.UpdateTenantSettings(ctx, tenantID, settings); err != nil {
		// Si falla la persistencia, limpiar el asset recién subido para no dejar huérfanos.
		_ = s.storage.Delete(ctx, key)
		return "", fmt.Errorf("brandingSvc.UploadAsset: update settings: %w", err)
	}

	return url, nil
}

// RemoveAsset borra el asset de MinIO y limpia la URL en settings.
func (s *brandingSvc) RemoveAsset(ctx context.Context, tenantID uuid.UUID, kind domain.BrandingAssetKind) error {
	if _, ok := brandingLimits[kind]; !ok {
		return fmt.Errorf("brandingSvc.RemoveAsset: %w: kind inválido", domain.ErrValidation)
	}

	settings, err := s.authRepo.GetTenantSettings(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("brandingSvc.RemoveAsset: get settings: %w", err)
	}

	if prev := assetURL(settings, kind); prev != "" && s.storage != nil {
		if key := s.storage.KeyFromURL(prev); key != "" {
			_ = s.storage.Delete(ctx, key)
		}
	}

	setAssetURL(settings, kind, "")
	if err := s.authRepo.UpdateTenantSettings(ctx, tenantID, settings); err != nil {
		return fmt.Errorf("brandingSvc.RemoveAsset: update settings: %w", err)
	}
	return nil
}

func assetURL(s *domain.TenantSettings, kind domain.BrandingAssetKind) string {
	switch kind {
	case domain.BrandingAssetLogo:
		return s.LogoURL
	case domain.BrandingAssetCover:
		return s.CoverURL
	}
	return ""
}

func setAssetURL(s *domain.TenantSettings, kind domain.BrandingAssetKind, url string) {
	switch kind {
	case domain.BrandingAssetLogo:
		s.LogoURL = url
	case domain.BrandingAssetCover:
		s.CoverURL = url
	}
}
