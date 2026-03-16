-- 010_knowledge_base.sql
-- Base de conocimiento por tenant para el asistente IA (RAG)

CREATE TABLE knowledge_documents (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  category    VARCHAR(50) NOT NULL,
              -- 'services','pricing','faq','policies','team','location','promotions'
  title       VARCHAR(255) NOT NULL,
  content     TEXT NOT NULL,
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Chunks vectorizados para búsqueda semántica
CREATE TABLE knowledge_chunks (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id   UUID NOT NULL,
  document_id UUID NOT NULL REFERENCES knowledge_documents(id) ON DELETE CASCADE,
  content     TEXT NOT NULL,
  embedding   vector(1536),       -- OpenAI text-embedding-3-small dimensión
  token_count INTEGER,
  chunk_index INTEGER,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_knowledge_docs_tenant   ON knowledge_documents(tenant_id, category);
CREATE INDEX idx_knowledge_chunks_tenant ON knowledge_chunks(tenant_id);
CREATE INDEX idx_knowledge_chunks_doc    ON knowledge_chunks(document_id);

-- Índice HNSW para búsqueda semántica eficiente
-- NOTA: crear DESPUÉS de poblar datos iniciales para mayor velocidad
CREATE INDEX idx_knowledge_chunks_vector
  ON knowledge_chunks USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);

ALTER TABLE knowledge_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge_chunks    ENABLE ROW LEVEL SECURITY;

CREATE POLICY knowledge_docs_isolation ON knowledge_documents
  USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY knowledge_chunks_isolation ON knowledge_chunks
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
