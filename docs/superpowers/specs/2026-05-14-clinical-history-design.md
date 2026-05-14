# Clinical History — Design Spec

**Date:** 2026-05-14
**Status:** Draft
**Scope:** Notas clínicas SOAP + archivos médicos por cita

---

## Context

CitaSpot necesita una pierna de historial clínico para que los profesionales lleven registro de cada visita: qué observaron, qué indicaron, y adjuntar archivos (RX, exámenes, fotos). Esto aplica a TODOS los verticales (dentistas, psicólogos, nutricionistas, estilistas, médicos generales).

**Decisiones tomadas:**
- Alcance v1: notas clínicas + archivos (sin datos estructurados ni anamnesis)
- Notas siempre vinculadas a una cita (appointment)
- Archivos siempre vinculados a una nota (clinical_note_id NOT NULL)
- Tipos de archivo: imágenes (JPEG, PNG, WebP) + PDF
- Permisos: todos los profesionales del tenant pueden ver/editar
- Sin acceso IA/WhatsApp al historial por ahora

---

## 1. Database Schema

### Tabla `clinical_notes`

```sql
CREATE TABLE clinical_notes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    appointment_id  UUID NOT NULL REFERENCES appointments(id),
    customer_id     UUID NOT NULL REFERENCES customers(id),
    professional_id UUID NOT NULL REFERENCES professionals(id),

    -- Formato SOAP (estándar médico universal)
    subjective      TEXT NOT NULL DEFAULT '',  -- Lo que reporta el paciente
    objective       TEXT NOT NULL DEFAULT '',  -- Hallazgos del profesional
    assessment      TEXT NOT NULL DEFAULT '',  -- Diagnóstico / evaluación
    plan            TEXT NOT NULL DEFAULT '',  -- Indicaciones / plan de acción

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Constraint: 1 nota por cita
ALTER TABLE clinical_notes ADD CONSTRAINT uq_clinical_notes_appointment
    UNIQUE (appointment_id);

-- Índices
CREATE INDEX idx_clinical_notes_tenant_customer
    ON clinical_notes (tenant_id, customer_id, created_at DESC);
CREATE INDEX idx_clinical_notes_tenant_professional
    ON clinical_notes (tenant_id, professional_id, created_at DESC);

-- RLS
ALTER TABLE clinical_notes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON clinical_notes
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Tabla `clinical_files`

```sql
CREATE TABLE clinical_files (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    clinical_note_id UUID NOT NULL REFERENCES clinical_notes(id) ON DELETE CASCADE,
    customer_id      UUID NOT NULL REFERENCES customers(id),

    file_name        VARCHAR(255) NOT NULL,
    file_key         VARCHAR(512) NOT NULL,   -- MinIO object key
    file_url         TEXT NOT NULL,
    content_type     VARCHAR(100) NOT NULL,
    size_bytes       BIGINT NOT NULL,

    category         VARCHAR(50) NOT NULL DEFAULT 'other',
    -- Valores válidos: xray, lab_result, photo, report, prescription, other
    description      TEXT NOT NULL DEFAULT '',

    uploaded_by      UUID NOT NULL REFERENCES professionals(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Índices
CREATE INDEX idx_clinical_files_note
    ON clinical_files (tenant_id, clinical_note_id);
CREATE INDEX idx_clinical_files_customer_category
    ON clinical_files (tenant_id, customer_id, category);

-- RLS
ALTER TABLE clinical_files ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON clinical_files
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Migración

Archivo: `apps/api/db/migrations/030_clinical_history.sql`
(Siguiente número después de la última migración existente — verificar antes de crear)

---

## 2. Domain Layer (Go)

### Structs

```go
// ClinicalNote — nota clínica SOAP vinculada a una cita
type ClinicalNote struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    AppointmentID  uuid.UUID
    CustomerID     uuid.UUID
    ProfessionalID uuid.UUID
    Subjective     string
    Objective      string
    Assessment     string
    Plan           string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// ClinicalNoteWithDetails — nota con datos del appointment y professional
type ClinicalNoteWithDetails struct {
    ClinicalNote
    ProfessionalName string
    ServiceName      string
    AppointmentDate  time.Time
    Files            []ClinicalFile
}

// ClinicalFile — archivo adjunto a una nota clínica
type ClinicalFile struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    ClinicalNoteID uuid.UUID
    CustomerID     uuid.UUID
    FileName       string
    FileKey        string
    FileURL        string
    ContentType    string
    SizeBytes      int64
    Category       string // xray, lab_result, photo, report, prescription, other
    Description    string
    UploadedBy     uuid.UUID
    CreatedAt      time.Time
}
```

### Interfaces

```go
type ClinicalNoteRepository interface {
    Create(ctx context.Context, note *ClinicalNote) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*ClinicalNoteWithDetails, error)
    GetByAppointmentID(ctx context.Context, tenantID, appointmentID uuid.UUID) (*ClinicalNoteWithDetails, error)
    ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit, offset int) ([]*ClinicalNoteWithDetails, int, error)
    Update(ctx context.Context, note *ClinicalNote) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type ClinicalFileRepository interface {
    Create(ctx context.Context, file *ClinicalFile) error
    ListByNoteID(ctx context.Context, tenantID, noteID uuid.UUID) ([]*ClinicalFile, error)
    ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, category string, limit, offset int) ([]*ClinicalFile, int, error)
    Delete(ctx context.Context, tenantID, id uuid.UUID) (*ClinicalFile, error) // Returns deleted file for cleanup
}

type ClinicalNoteSvc interface {
    Create(ctx context.Context, tenantID uuid.UUID, req *CreateClinicalNoteRequest) (*ClinicalNote, error)
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*ClinicalNoteWithDetails, error)
    GetByAppointmentID(ctx context.Context, tenantID, appointmentID uuid.UUID) (*ClinicalNoteWithDetails, error)
    ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit, offset int) ([]*ClinicalNoteWithDetails, int, error)
    Update(ctx context.Context, tenantID, id uuid.UUID, req *UpdateClinicalNoteRequest) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type ClinicalFileSvc interface {
    Upload(ctx context.Context, tenantID, noteID uuid.UUID, input ClinicalFileUploadInput) (*ClinicalFile, error)
    ListByNote(ctx context.Context, tenantID, noteID uuid.UUID) ([]*ClinicalFile, error)
    ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, category string, limit, offset int) ([]*ClinicalFile, int, error)
    Delete(ctx context.Context, tenantID, fileID uuid.UUID) error
}
```

### Request types

```go
type CreateClinicalNoteRequest struct {
    AppointmentID  uuid.UUID
    ProfessionalID uuid.UUID
    Subjective     string
    Objective      string
    Assessment     string
    Plan           string
}

type UpdateClinicalNoteRequest struct {
    Subjective *string
    Objective  *string
    Assessment *string
    Plan       *string
}

type ClinicalFileUploadInput struct {
    FileName    string
    ContentType string
    Reader      io.Reader
    Size        int64
    Category    string
    Description string
    UploadedBy  uuid.UUID
}
```

---

## 3. API Endpoints

### Notas clínicas

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/appointments/:appointmentId/clinical-note` | Crear nota para una cita |
| `GET` | `/api/v1/customers/:customerId/clinical-notes` | Timeline del paciente (paginado) |
| `GET` | `/api/v1/customers/:customerId/clinical-notes/:noteId` | Nota individual con archivos |
| `GET` | `/api/v1/appointments/:appointmentId/clinical-note` | Nota de una cita específica |
| `PATCH` | `/api/v1/clinical-notes/:noteId` | Editar nota |
| `DELETE` | `/api/v1/clinical-notes/:noteId` | Eliminar nota (cascade elimina archivos) |

### Archivos clínicos

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/clinical-notes/:noteId/files/upload` | Subir archivo (multipart) |
| `GET` | `/api/v1/clinical-notes/:noteId/files` | Archivos de una nota |
| `GET` | `/api/v1/customers/:customerId/clinical-files` | Todos los archivos del paciente |
| `GET` | `/api/v1/customers/:customerId/clinical-files?category=xray` | Filtrar por categoría |
| `DELETE` | `/api/v1/clinical-files/:fileId` | Eliminar archivo (también de MinIO) |

### Validaciones

- Solo citas con `status = 'completed'` pueden tener nota clínica
- Máximo 1 nota por cita (UNIQUE constraint)
- Archivos: máx 10 MB por archivo, máx 10 archivos por nota
- Content types permitidos: `image/jpeg`, `image/png`, `image/webp`, `application/pdf`
- Categorías válidas: `xray`, `lab_result`, `photo`, `report`, `prescription`, `other`

---

## 4. Storage (MinIO)

- **Bucket:** `clinical-files` (nuevo, separado del bucket de branding)
- **Key pattern:** `{tenant_id}/{customer_id}/{note_id}/{uuid}.{ext}`
- **Reutiliza:** el client MinIO existente (`internal/client/storage/minio.go`) — métodos `Put` y `Delete`
- **Limpieza:** al eliminar una nota (CASCADE), el service debe iterar los files y llamar `storage.Delete` para cada key antes de borrar el registro

---

## 5. Frontend

### Ubicación en el dashboard

Tab **"Historial Clínico"** en `/dashboard/clients/[id]` — nueva tab junto a las existentes.

### Componentes principales

1. **ClinicalTimeline** — lista cronológica de notas, más reciente primero
   - Cada item muestra: fecha, profesional, servicio, preview de Assessment
   - Click expande la nota completa con los 4 campos SOAP
   - Thumbnails de archivos adjuntos (imágenes inline, icono PDF)

2. **ClinicalNoteForm** — formulario para crear/editar nota
   - 4 textareas para S/O/A/P con labels descriptivos en español
   - Aparece desde la vista de cita completada (botón "Agregar nota clínica")

3. **FileUploader** — drag & drop + file picker
   - Preview de imagen antes de subir
   - Selector de categoría (RX, laboratorio, foto, reporte, receta, otro)
   - Campo de descripción opcional
   - Progress bar durante upload

4. **FileGallery** — galería filtrable de archivos del paciente
   - Grid de thumbnails con filtro por categoría
   - Click abre lightbox para imágenes / nueva tab para PDFs
   - Accessible desde la tab de historial

### Flujo de uso

```
1. Profesional completa una cita → status = 'completed'
2. En la vista de cita, aparece botón "Agregar nota clínica"
3. Se abre el formulario SOAP → llena campos → guarda
4. Opcionalmente sube archivos (RX, fotos, reportes)
5. En la ficha del paciente, tab "Historial" muestra el timeline
6. Puede filtrar archivos por categoría
```

---

## 6. Archivos a crear/modificar

### Nuevos archivos

| Path | Descripción |
|------|-------------|
| `apps/api/db/migrations/030_clinical_history.sql` | Migración con ambas tablas |
| `apps/api/internal/domain/clinical_note.go` | Structs y request types |
| `apps/api/internal/repository/clinical_note.go` | Queries SQL para notas |
| `apps/api/internal/repository/clinical_file.go` | Queries SQL para archivos |
| `apps/api/internal/service/clinical_note.go` | Lógica de negocio notas |
| `apps/api/internal/service/clinical_file.go` | Lógica de negocio archivos + MinIO |
| `apps/api/internal/handler/clinical_note.go` | HTTP handlers notas |
| `apps/api/internal/handler/clinical_file.go` | HTTP handlers archivos |
| `apps/web/app/dashboard/clients/[id]/clinical-history/` | Página frontend |
| `apps/web/components/clinical/` | Componentes React |
| `apps/web/lib/api/clinical.ts` | API client functions |

### Archivos a modificar

| Path | Cambio |
|------|--------|
| `apps/api/internal/domain/interfaces.go` | Agregar interfaces ClinicalNote/File Repository y Svc |
| `apps/api/internal/domain/errors.go` | Agregar errores: `ErrClinicalNoteExists`, `ErrAppointmentNotCompleted`, `ErrFileTooLarge`, `ErrFileTypeNotAllowed` |
| `apps/api/cmd/api/main.go` (o router setup) | Registrar nuevas rutas |
| `apps/web/app/dashboard/clients/[id]/` | Agregar tab de historial clínico |

---

## 7. Verificación

### Tests backend
- Unit tests para service layer (crear nota, validar appointment completado, upload file)
- Repository tests contra DB real (RLS, unique constraint, cascade delete)
- Handler tests (validación de input, multipart upload)

### Tests frontend
- Renderizado del timeline
- Formulario SOAP (validación, submit)
- File upload (tipos permitidos, tamaño máximo)

### Manual
1. Crear una cita y completarla
2. Agregar nota clínica con los 4 campos SOAP
3. Subir imagen y PDF
4. Verificar que aparecen en el timeline del paciente
5. Filtrar archivos por categoría
6. Eliminar nota → verificar que archivos se borran de MinIO
7. Intentar crear segunda nota en misma cita → error esperado
8. Intentar crear nota en cita no completada → error esperado
