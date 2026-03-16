-- 013_knowledge_file_upload.sql
-- Añade soporte para subida de documentos (PDF, DOCX, XLSX) a la knowledge base.
-- Nuevos campos: source_type (text|file), file_name, status (ready|processing|error).

ALTER TABLE knowledge_documents
  ADD COLUMN source_type VARCHAR(10) NOT NULL DEFAULT 'text',
  ADD COLUMN file_name   VARCHAR(255),
  ADD COLUMN status      VARCHAR(20) NOT NULL DEFAULT 'ready';

-- Documentos existentes quedan como source_type='text', status='ready'
COMMENT ON COLUMN knowledge_documents.source_type IS 'text = escrito manualmente; file = extraído de PDF/DOCX/XLSX';
COMMENT ON COLUMN knowledge_documents.status      IS 'ready = listo para RAG; processing = extrayendo texto; error = fallo al parsear';
