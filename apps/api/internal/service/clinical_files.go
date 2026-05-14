package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/client/storage"
	"github.com/citaspot/api/internal/domain"
)

const (
	maxClinicalFileSize = 10 * 1024 * 1024 // 10 MB
	maxFilesPerNote     = 10
)

// MIME types permitidos para archivos clínicos.
var allowedClinicalMIME = map[string]string{
	"image/jpeg":      "jpg",
	"image/jpg":       "jpg",
	"image/png":       "png",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

// Categorías válidas para archivos clínicos.
var validCategories = map[string]bool{
	"xray":         true,
	"lab_result":   true,
	"photo":        true,
	"report":       true,
	"prescription": true,
	"other":        true,
}

type clinicalFileSvc struct {
	fileRepo domain.ClinicalFileRepository
	noteRepo domain.ClinicalNoteRepository
	storage  *storage.Client
}

func NewClinicalFileSvc(
	fileRepo domain.ClinicalFileRepository,
	noteRepo domain.ClinicalNoteRepository,
	s *storage.Client,
) domain.ClinicalFileSvc {
	return &clinicalFileSvc{
		fileRepo: fileRepo,
		noteRepo: noteRepo,
		storage:  s,
	}
}

func (s *clinicalFileSvc) Upload(ctx context.Context, tenantID, noteID uuid.UUID, input domain.ClinicalFileUploadInput) (*domain.ClinicalFile, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("clinicalFileSvc.Upload: storage no configurado")
	}

	// Validar tamaño
	if input.Size <= 0 || input.Size > maxClinicalFileSize {
		return nil, domain.ErrFileTooLarge
	}

	// Validar content type
	ext, ok := allowedClinicalMIME[strings.ToLower(input.ContentType)]
	if !ok {
		return nil, domain.ErrFileTypeNotAllowed
	}

	// Validar categoría
	cat := input.Category
	if cat == "" {
		cat = "other"
	}
	if !validCategories[cat] {
		return nil, fmt.Errorf("clinicalFileSvc.Upload: %w: categoría inválida '%s'", domain.ErrValidation, cat)
	}

	// Verificar que la nota existe y obtener customer_id
	note, err := s.noteRepo.GetByID(ctx, tenantID, noteID)
	if err != nil {
		return nil, fmt.Errorf("clinicalFileSvc.Upload: get note: %w", err)
	}

	// Verificar límite de archivos por nota
	existing, err := s.fileRepo.ListByNoteID(ctx, tenantID, noteID)
	if err != nil {
		return nil, fmt.Errorf("clinicalFileSvc.Upload: list existing: %w", err)
	}
	if len(existing) >= maxFilesPerNote {
		return nil, domain.ErrMaxFilesReached
	}

	// Subir a MinIO
	fileID := uuid.New()
	key := fmt.Sprintf("clinical/%s/%s/%s/%s.%s",
		tenantID.String(), note.CustomerID.String(), noteID.String(), fileID.String(), ext)

	url, err := s.storage.Put(ctx, key, input.Reader, input.Size, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("clinicalFileSvc.Upload: put: %w", err)
	}

	// Determinar nombre de archivo (usar el original o generar uno)
	fileName := input.FileName
	if fileName == "" {
		fileName = fileID.String() + "." + ext
	}

	file := &domain.ClinicalFile{
		ID:             fileID,
		TenantID:       tenantID,
		ClinicalNoteID: noteID,
		CustomerID:     note.CustomerID,
		FileName:       filepath.Base(fileName),
		FileKey:        key,
		FileURL:        url,
		ContentType:    input.ContentType,
		SizeBytes:      input.Size,
		Category:       cat,
		Description:    input.Description,
		UploadedBy:     input.UploadedBy,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		// Limpiar archivo huérfano de MinIO
		if delErr := s.storage.Delete(ctx, key); delErr != nil {
			slog.Warn("clinicalFileSvc.Upload: MinIO cleanup failed", "key", key, "error", delErr)
		}
		return nil, fmt.Errorf("clinicalFileSvc.Upload: create: %w", err)
	}

	return file, nil
}

func (s *clinicalFileSvc) ListByNote(ctx context.Context, tenantID, noteID uuid.UUID) ([]*domain.ClinicalFile, error) {
	files, err := s.fileRepo.ListByNoteID(ctx, tenantID, noteID)
	if err != nil {
		return nil, fmt.Errorf("clinicalFileSvc.ListByNote: %w", err)
	}
	return files, nil
}

func (s *clinicalFileSvc) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, category string, limit, offset int) ([]*domain.ClinicalFile, int, error) {
	files, total, err := s.fileRepo.ListByCustomer(ctx, tenantID, customerID, category, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("clinicalFileSvc.ListByCustomer: %w", err)
	}
	return files, total, nil
}

func (s *clinicalFileSvc) Delete(ctx context.Context, tenantID, fileID uuid.UUID) error {
	file, err := s.fileRepo.Delete(ctx, tenantID, fileID)
	if err != nil {
		return fmt.Errorf("clinicalFileSvc.Delete: %w", err)
	}

	// Limpiar de MinIO (best effort)
	if s.storage != nil {
		if err := s.storage.Delete(ctx, file.FileKey); err != nil {
			slog.Warn("clinicalFileSvc.Delete: MinIO cleanup failed",
				"file_key", file.FileKey, "error", err)
		}
	}
	return nil
}
