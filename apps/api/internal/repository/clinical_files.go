package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type clinicalFileRepository struct {
	db *pgxpool.Pool
}

func NewClinicalFileRepository(db *pgxpool.Pool) domain.ClinicalFileRepository {
	return &clinicalFileRepository{db: db}
}

func (r *clinicalFileRepository) Create(ctx context.Context, file *domain.ClinicalFile) error {
	return withTenant(ctx, r.db, file.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO clinical_files
				(id, tenant_id, clinical_note_id, customer_id,
				 file_name, file_key, file_url, content_type, size_bytes,
				 category, description, uploaded_by, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`, file.ID, file.TenantID, file.ClinicalNoteID, file.CustomerID,
			file.FileName, file.FileKey, file.FileURL, file.ContentType, file.SizeBytes,
			file.Category, file.Description, file.UploadedBy)
		if err != nil {
			return fmt.Errorf("clinicalFileRepository.Create: %w", err)
		}
		return nil
	})
}

func (r *clinicalFileRepository) ListByNoteID(ctx context.Context, tenantID, noteID uuid.UUID) ([]*domain.ClinicalFile, error) {
	var result []*domain.ClinicalFile
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, clinical_note_id, customer_id,
			       file_name, file_key, file_url, content_type, size_bytes,
			       category, description, uploaded_by, created_at
			FROM clinical_files
			WHERE tenant_id = $1 AND clinical_note_id = $2
			ORDER BY created_at
		`, tenantID, noteID)
		if err != nil {
			return fmt.Errorf("clinicalFileRepository.ListByNoteID: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			f := &domain.ClinicalFile{}
			if err := rows.Scan(
				&f.ID, &f.TenantID, &f.ClinicalNoteID, &f.CustomerID,
				&f.FileName, &f.FileKey, &f.FileURL, &f.ContentType, &f.SizeBytes,
				&f.Category, &f.Description, &f.UploadedBy, &f.CreatedAt,
			); err != nil {
				return fmt.Errorf("clinicalFileRepository.ListByNoteID: scan: %w", err)
			}
			result = append(result, f)
		}
		return rows.Err()
	})
	return result, err
}

func (r *clinicalFileRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, category string, limit, offset int) ([]*domain.ClinicalFile, int, error) {
	var result []*domain.ClinicalFile
	var total int
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := `
			SELECT id, tenant_id, clinical_note_id, customer_id,
			       file_name, file_key, file_url, content_type, size_bytes,
			       category, description, uploaded_by, created_at,
			       COUNT(*) OVER() AS total_count
			FROM clinical_files
			WHERE tenant_id = $1 AND customer_id = $2`
		args := []any{tenantID, customerID}
		argN := 3

		if category != "" {
			query += fmt.Sprintf(" AND category = $%d", argN)
			args = append(args, category)
			argN++
		}

		query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("clinicalFileRepository.ListByCustomer: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			f := &domain.ClinicalFile{}
			if err := rows.Scan(
				&f.ID, &f.TenantID, &f.ClinicalNoteID, &f.CustomerID,
				&f.FileName, &f.FileKey, &f.FileURL, &f.ContentType, &f.SizeBytes,
				&f.Category, &f.Description, &f.UploadedBy, &f.CreatedAt,
				&total,
			); err != nil {
				return fmt.Errorf("clinicalFileRepository.ListByCustomer: scan: %w", err)
			}
			result = append(result, f)
		}
		return rows.Err()
	})
	return result, total, err
}

func (r *clinicalFileRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) (*domain.ClinicalFile, error) {
	var file *domain.ClinicalFile
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		file = &domain.ClinicalFile{}
		err := tx.QueryRow(ctx, `
			DELETE FROM clinical_files
			WHERE tenant_id = $1 AND id = $2
			RETURNING id, tenant_id, clinical_note_id, customer_id,
			          file_name, file_key, file_url, content_type, size_bytes,
			          category, description, uploaded_by, created_at
		`, tenantID, id).Scan(
			&file.ID, &file.TenantID, &file.ClinicalNoteID, &file.CustomerID,
			&file.FileName, &file.FileKey, &file.FileURL, &file.ContentType, &file.SizeBytes,
			&file.Category, &file.Description, &file.UploadedBy, &file.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("clinicalFileRepository.Delete: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return file, nil
}
