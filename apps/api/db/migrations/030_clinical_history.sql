-- Historia clínica: notas SOAP por cita + archivos médicos adjuntos.

CREATE TABLE clinical_notes (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  appointment_id  UUID NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
  customer_id     UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
  professional_id UUID NOT NULL REFERENCES professionals(id),

  -- Formato SOAP (estándar médico universal)
  subjective      TEXT NOT NULL DEFAULT '',
  objective       TEXT NOT NULL DEFAULT '',
  assessment      TEXT NOT NULL DEFAULT '',
  plan            TEXT NOT NULL DEFAULT '',

  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 1 nota por cita
ALTER TABLE clinical_notes
  ADD CONSTRAINT uq_clinical_notes_appointment UNIQUE (appointment_id);

ALTER TABLE clinical_notes ENABLE ROW LEVEL SECURITY;

CREATE POLICY clinical_notes_tenant_isolation ON clinical_notes
  USING (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE INDEX idx_clinical_notes_tenant_customer
  ON clinical_notes (tenant_id, customer_id, created_at DESC);
CREATE INDEX idx_clinical_notes_tenant_professional
  ON clinical_notes (tenant_id, professional_id, created_at DESC);

-- ──────────────────────────────────────────────────────────────────────────────

CREATE TABLE clinical_files (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  clinical_note_id UUID NOT NULL REFERENCES clinical_notes(id) ON DELETE CASCADE,
  customer_id      UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,

  file_name        VARCHAR(255) NOT NULL,
  file_key         VARCHAR(512) NOT NULL,
  file_url         TEXT NOT NULL,
  content_type     VARCHAR(100) NOT NULL,
  size_bytes       BIGINT NOT NULL,

  category         VARCHAR(50) NOT NULL DEFAULT 'other'
                   CHECK (category IN ('xray', 'lab_result', 'photo', 'report', 'prescription', 'other')),
  description      TEXT NOT NULL DEFAULT '',

  uploaded_by      UUID NOT NULL REFERENCES professionals(id),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE clinical_files ENABLE ROW LEVEL SECURITY;

CREATE POLICY clinical_files_tenant_isolation ON clinical_files
  USING (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE INDEX idx_clinical_files_note
  ON clinical_files (tenant_id, clinical_note_id);
CREATE INDEX idx_clinical_files_customer_category
  ON clinical_files (tenant_id, customer_id, category);
