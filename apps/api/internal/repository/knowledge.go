package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type knowledgeRepository struct {
	db *pgxpool.Pool
}

// NewKnowledgeRepository crea el repositorio de knowledge base.
func NewKnowledgeRepository(db *pgxpool.Pool) domain.KnowledgeRepository {
	return &knowledgeRepository{db: db}
}

// List retorna todos los documentos de la knowledge base de un tenant.
func (r *knowledgeRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.KnowledgeDocument, error) {
	// Inicializar como slice vacío (no nil) para que JSON serialice [] en vez de null.
	docs := make([]*domain.KnowledgeDocument, 0)
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, category, title, content, is_active,
			       source_type, file_name, status, created_at, updated_at
			FROM knowledge_documents
			WHERE tenant_id = $1
			ORDER BY category, title
		`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			d := &domain.KnowledgeDocument{}
			if err := rows.Scan(
				&d.ID, &d.TenantID, &d.Category, &d.Title, &d.Content,
				&d.IsActive, &d.SourceType, &d.FileName, &d.Status,
				&d.CreatedAt, &d.UpdatedAt,
			); err != nil {
				return err
			}
			docs = append(docs, d)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("knowledge.List: %w", err)
	}
	return docs, nil
}

// GetByID retorna un documento por ID.
func (r *knowledgeRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.KnowledgeDocument, error) {
	var doc *domain.KnowledgeDocument
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		doc = &domain.KnowledgeDocument{}
		return tx.QueryRow(ctx, `
			SELECT id, tenant_id, category, title, content, is_active,
			       source_type, file_name, status, created_at, updated_at
			FROM knowledge_documents
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, id).Scan(
			&doc.ID, &doc.TenantID, &doc.Category, &doc.Title, &doc.Content,
			&doc.IsActive, &doc.SourceType, &doc.FileName, &doc.Status,
			&doc.CreatedAt, &doc.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("knowledge.GetByID: %w", err)
	}
	return doc, nil
}

// Create inserta un nuevo documento en la knowledge base.
func (r *knowledgeRepository) Create(ctx context.Context, doc *domain.KnowledgeDocument) error {
	err := withTenant(ctx, r.db, doc.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO knowledge_documents
				(id, tenant_id, category, title, content, is_active, source_type, file_name, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, doc.ID, doc.TenantID, doc.Category, doc.Title, doc.Content,
			doc.IsActive, doc.SourceType, doc.FileName, doc.Status)
		return err
	})
	if err != nil {
		return fmt.Errorf("knowledge.Create: %w", err)
	}
	return nil
}

// UpdateContent actualiza el contenido extraído y el status de un documento de archivo.
// Llamado por el AI Service tras parsear el documento subido.
func (r *knowledgeRepository) UpdateContent(ctx context.Context, tenantID, id uuid.UUID, content, status string) error {
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE knowledge_documents
			SET content = $1, status = $2, updated_at = NOW()
			WHERE tenant_id = $3 AND id = $4
		`, content, status, tenantID, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("knowledge.UpdateContent: %w", err)
	}
	return nil
}

// Update actualiza un documento existente.
func (r *knowledgeRepository) Update(ctx context.Context, doc *domain.KnowledgeDocument) error {
	err := withTenant(ctx, r.db, doc.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE knowledge_documents
			SET category = $1, title = $2, content = $3, is_active = $4, updated_at = $5
			WHERE tenant_id = $6 AND id = $7
		`, doc.Category, doc.Title, doc.Content, doc.IsActive, time.Now().UTC(), doc.TenantID, doc.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("knowledge.Update: %w", err)
	}
	return nil
}

// Delete elimina un documento y sus chunks (ON DELETE CASCADE en la FK).
func (r *knowledgeRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM knowledge_documents WHERE tenant_id = $1 AND id = $2
		`, tenantID, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("knowledge.Delete: %w", err)
	}
	return nil
}
