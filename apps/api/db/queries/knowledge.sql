-- Queries para knowledge_documents y knowledge_chunks
-- Las queries se implementan en Fase 4, Tarea 13

-- name: ListKnowledgeDocuments :many
SELECT * FROM knowledge_documents
WHERE tenant_id = current_setting('app.tenant_id')::UUID
  AND is_active = TRUE
ORDER BY category ASC, title ASC;
