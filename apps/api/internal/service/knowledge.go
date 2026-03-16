// Package service — lógica de negocio para la base de conocimiento.
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// maxUploadBytes es el tamaño máximo de archivo permitido (8 MB).
const maxUploadBytes = 8 * 1024 * 1024

type knowledgeSvc struct {
	repo      domain.KnowledgeRepository
	publisher domain.MessagePublisher
}

// NewKnowledgeSvc crea el servicio de knowledge base.
func NewKnowledgeSvc(repo domain.KnowledgeRepository, publisher domain.MessagePublisher) domain.KnowledgeSvc {
	return &knowledgeSvc{repo: repo, publisher: publisher}
}

func (s *knowledgeSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.KnowledgeDocument, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *knowledgeSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.KnowledgeDocument, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// Create inserta un nuevo documento de texto y dispara la vectorización en background.
func (s *knowledgeSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error) {
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	doc := &domain.KnowledgeDocument{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Category:   input.Category,
		Title:      input.Title,
		Content:    input.Content,
		IsActive:   isActive,
		SourceType: "text",
		Status:     "ready",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("knowledgeSvc.Create: %w", err)
	}

	// Publicar en knowledge.vectorize para generar embeddings en background
	s.publishVectorize(ctx, doc.TenantID.String(), doc.ID.String(), doc.Content, "", "")

	return doc, nil
}

// Update actualiza un documento y re-vectoriza si el contenido cambió.
func (s *knowledgeSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error) {
	doc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	contentChanged := doc.Content != input.Content

	doc.Category = input.Category
	doc.Title = input.Title
	doc.Content = input.Content
	if input.IsActive != nil {
		doc.IsActive = *input.IsActive
	}
	doc.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("knowledgeSvc.Update: %w", err)
	}

	// Solo re-vectorizar si el contenido cambió
	if contentChanged {
		s.publishVectorize(ctx, doc.TenantID.String(), doc.ID.String(), doc.Content, "", "")
	}

	return doc, nil
}

// Upload crea un documento a partir de un archivo (PDF/DOCX/XLSX).
// El documento se crea con status='processing' y content vacío.
// El AI Service extrae el texto, lo guarda y vectoriza de forma asíncrona.
func (s *knowledgeSvc) Upload(
	ctx context.Context,
	tenantID uuid.UUID,
	input *domain.KnowledgeUploadInput,
	fileName string,
	fileBytes []byte,
	fileType string,
) (*domain.KnowledgeDocument, error) {
	if len(fileBytes) > maxUploadBytes {
		return nil, fmt.Errorf("el archivo supera el límite de 8 MB")
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	doc := &domain.KnowledgeDocument{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Category:   input.Category,
		Title:      input.Title,
		Content:    "", // el AI Service lo llenará tras parsear
		IsActive:   isActive,
		SourceType: "file",
		FileName:   &fileName,
		Status:     "processing",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("knowledgeSvc.Upload: %w", err)
	}

	// Publicar con el archivo en base64 para que el AI Service lo parsee
	encoded := base64.StdEncoding.EncodeToString(fileBytes)
	s.publishVectorize(ctx, doc.TenantID.String(), doc.ID.String(), "", encoded, fileType)

	return doc, nil
}

func (s *knowledgeSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// publishVectorize publica en knowledge.vectorize para que el AI Service genere embeddings.
// Se ejecuta de forma best-effort: un error en la publicación no falla la operación principal.
// Si el publisher es nil (modo degradado sin RabbitMQ), se omite silenciosamente.
func (s *knowledgeSvc) publishVectorize(ctx context.Context, tenantID, documentID, content, fileBytes, fileType string) {
	if s.publisher == nil {
		return
	}
	payload := domain.KnowledgeVectorizePayload{
		TenantID:   tenantID,
		DocumentID: documentID,
		Content:    content,
		FileBytes:  fileBytes,
		FileType:   fileType,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("knowledgeSvc.publishVectorize: marshal error", "error", err)
		return
	}
	if err := s.publisher.Publish(ctx, "knowledge.vectorize", body); err != nil {
		slog.Error("knowledgeSvc.publishVectorize: publish error", "doc", documentID, "error", err)
	}
}

// inferFileType determina el tipo de archivo por extensión.
func inferFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".doc":
		return "docx"
	case ".xlsx", ".xls":
		return "xlsx"
	case ".csv":
		return "csv"
	default:
		return ""
	}
}
