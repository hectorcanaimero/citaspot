package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/client/storage"
	"github.com/citaspot/api/internal/domain"
)

type clinicalNoteSvc struct {
	noteRepo domain.ClinicalNoteRepository
	fileRepo domain.ClinicalFileRepository
	apptRepo domain.AppointmentRepository
	storage  *storage.Client
}

func NewClinicalNoteSvc(
	noteRepo domain.ClinicalNoteRepository,
	fileRepo domain.ClinicalFileRepository,
	apptRepo domain.AppointmentRepository,
	s *storage.Client,
) domain.ClinicalNoteSvc {
	return &clinicalNoteSvc{
		noteRepo: noteRepo,
		fileRepo: fileRepo,
		apptRepo: apptRepo,
		storage:  s,
	}
}

func (s *clinicalNoteSvc) Create(ctx context.Context, tenantID, appointmentID uuid.UUID, req *domain.CreateClinicalNoteRequest) (*domain.ClinicalNote, error) {
	// Verificar que la cita existe y está completada
	appt, err := s.apptRepo.GetByID(ctx, tenantID, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("clinicalNoteSvc.Create: get appointment: %w", err)
	}
	if appt.Status != "completed" {
		return nil, domain.ErrAppointmentNotCompleted
	}

	note := &domain.ClinicalNote{
		ID:             uuid.New(),
		TenantID:       tenantID,
		AppointmentID:  appointmentID,
		CustomerID:     appt.CustomerID,
		ProfessionalID: req.ProfessionalID,
		Subjective:     req.Subjective,
		Objective:      req.Objective,
		Assessment:     req.Assessment,
		Plan:           req.Plan,
	}

	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("clinicalNoteSvc.Create: %w", err)
	}
	return note, nil
}

func (s *clinicalNoteSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
	note, err := s.noteRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("clinicalNoteSvc.GetByID: %w", err)
	}
	files, err := s.fileRepo.ListByNoteID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("clinicalNoteSvc.GetByID: list files: %w", err)
	}
	note.Files = make([]domain.ClinicalFile, len(files))
	for i, f := range files {
		note.Files[i] = *f
	}
	return note, nil
}

func (s *clinicalNoteSvc) GetByAppointmentID(ctx context.Context, tenantID, appointmentID uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
	note, err := s.noteRepo.GetByAppointmentID(ctx, tenantID, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("clinicalNoteSvc.GetByAppointmentID: %w", err)
	}
	files, err := s.fileRepo.ListByNoteID(ctx, tenantID, note.ID)
	if err != nil {
		return nil, fmt.Errorf("clinicalNoteSvc.GetByAppointmentID: list files: %w", err)
	}
	note.Files = make([]domain.ClinicalFile, len(files))
	for i, f := range files {
		note.Files[i] = *f
	}
	return note, nil
}

func (s *clinicalNoteSvc) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit, offset int) ([]*domain.ClinicalNoteWithDetails, int, error) {
	notes, total, err := s.noteRepo.ListByCustomer(ctx, tenantID, customerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("clinicalNoteSvc.ListByCustomer: %w", err)
	}
	return notes, total, nil
}

func (s *clinicalNoteSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateClinicalNoteRequest) error {
	existing, err := s.noteRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("clinicalNoteSvc.Update: %w", err)
	}

	note := &existing.ClinicalNote
	if req.Subjective != nil {
		note.Subjective = *req.Subjective
	}
	if req.Objective != nil {
		note.Objective = *req.Objective
	}
	if req.Assessment != nil {
		note.Assessment = *req.Assessment
	}
	if req.Plan != nil {
		note.Plan = *req.Plan
	}

	if err := s.noteRepo.Update(ctx, note); err != nil {
		return fmt.Errorf("clinicalNoteSvc.Update: %w", err)
	}
	return nil
}

func (s *clinicalNoteSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// Obtener archivos ANTES de borrar (CASCADE eliminará los registros pero no MinIO)
	files, err := s.fileRepo.ListByNoteID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("clinicalNoteSvc.Delete: list files: %w", err)
	}

	// Borrar nota (CASCADE borra archivos en DB)
	if err := s.noteRepo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("clinicalNoteSvc.Delete: %w", err)
	}

	// Limpiar archivos de MinIO (best effort)
	if s.storage != nil {
		for _, f := range files {
			if err := s.storage.Delete(ctx, f.FileKey); err != nil {
				slog.Warn("clinicalNoteSvc.Delete: MinIO cleanup failed",
					"file_key", f.FileKey, "error", err)
			}
		}
	}
	return nil
}
