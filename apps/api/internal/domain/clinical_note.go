package domain

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// ── Clinical Notes (SOAP) ───────────────────────────────────────────────────

// ClinicalNote nota clínica SOAP vinculada a una cita completada.
type ClinicalNote struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	AppointmentID  uuid.UUID `json:"appointment_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	ProfessionalID uuid.UUID `json:"professional_id"`
	Subjective     string    `json:"subjective"`
	Objective      string    `json:"objective"`
	Assessment     string    `json:"assessment"`
	Plan           string    `json:"plan"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ClinicalNoteWithDetails nota con datos del appointment, profesional y archivos.
type ClinicalNoteWithDetails struct {
	ClinicalNote
	ProfessionalName string         `json:"professional_name"`
	ServiceName      string         `json:"service_name"`
	AppointmentDate  time.Time      `json:"appointment_date"`
	Files            []ClinicalFile `json:"files"`
}

// CreateClinicalNoteRequest datos para crear una nota clínica.
type CreateClinicalNoteRequest struct {
	ProfessionalID uuid.UUID `json:"professional_id" validate:"required"`
	Subjective     string    `json:"subjective"`
	Objective      string    `json:"objective"`
	Assessment     string    `json:"assessment"`
	Plan           string    `json:"plan"`
}

// UpdateClinicalNoteRequest datos para editar una nota clínica (parcial).
type UpdateClinicalNoteRequest struct {
	Subjective *string `json:"subjective"`
	Objective  *string `json:"objective"`
	Assessment *string `json:"assessment"`
	Plan       *string `json:"plan"`
}

// ── Clinical Files ──────────────────────────────────────────────────────────

// ClinicalFile archivo adjunto a una nota clínica.
type ClinicalFile struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ClinicalNoteID uuid.UUID `json:"clinical_note_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	FileName       string    `json:"file_name"`
	FileKey        string    `json:"-"`
	FileURL        string    `json:"file_url"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	Category       string    `json:"category"`
	Description    string    `json:"description"`
	UploadedBy     uuid.UUID `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// ClinicalFileUploadInput datos para subir un archivo clínico.
type ClinicalFileUploadInput struct {
	FileName    string
	ContentType string
	Reader      io.Reader
	Size        int64
	Category    string
	Description string
	UploadedBy  uuid.UUID
}
