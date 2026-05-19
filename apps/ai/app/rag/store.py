# Vectorstore: persiste y consulta chunks en PostgreSQL + pgvector.
from __future__ import annotations

import logging
import uuid
from typing import Any

from app.core.database import get_pool
from app.llm.embeddings import embed, embed_batch

log = logging.getLogger(__name__)

# Número máximo de chunks a retornar en búsqueda semántica
_TOP_K = 5
# Umbral mínimo de similitud coseno (0–1)
_MIN_SIMILARITY = 0.55


async def vectorize_document(
    tenant_id: str,
    document_id: str,
    chunks: list[str],
) -> None:
    """
    Genera embeddings para una lista de chunks y los guarda en knowledge_chunks.
    Elimina chunks previos del documento antes de insertar los nuevos.

    Args:
        tenant_id: UUID del tenant propietario.
        document_id: UUID del knowledge_document padre.
        chunks: Lista de textos a vectorizar.
    """
    pool = await get_pool()
    if not chunks:
        return

    embeddings = await embed_batch(chunks)

    async with pool.acquire() as conn:
        async with conn.transaction():
            # RLS: establecer contexto del tenant
            await conn.execute(
                "SELECT set_config('app.tenant_id', $1, true)", tenant_id
            )
            # Eliminar chunks previos del documento
            await conn.execute(
                "DELETE FROM knowledge_chunks WHERE document_id = $1",
                uuid.UUID(document_id),
            )
            # Insertar chunks nuevos
            await conn.executemany(
                """
                INSERT INTO knowledge_chunks
                    (id, tenant_id, document_id, content, embedding, token_count, chunk_index)
                VALUES
                    ($1, $2, $3, $4, $5::vector, $6, $7)
                """,
                [
                    (
                        uuid.uuid4(),
                        uuid.UUID(tenant_id),
                        uuid.UUID(document_id),
                        text,
                        str(emb),  # pgvector acepta '[x,y,z,...]'
                        len(text.split()),  # aproximación de tokens por palabras
                        idx,
                    )
                    for idx, (text, emb) in enumerate(zip(chunks, embeddings))
                ],
            )
    log.info(
        "rag.store: vectorized document=%s tenant=%s chunks=%d",
        document_id, tenant_id, len(chunks),
    )


async def search(
    tenant_id: str,
    query: str,
    top_k: int = _TOP_K,
    min_similarity: float = _MIN_SIMILARITY,
) -> list[dict[str, Any]]:
    """
    Busca los chunks más similares a la query usando cosine similarity.

    Args:
        tenant_id: UUID del tenant (filtra por RLS).
        query: Texto de búsqueda.
        top_k: Número de resultados a retornar.
        min_similarity: Similitud mínima para incluir un resultado.

    Returns:
        Lista de dicts con keys: id, document_id, content, similarity, category, title.
    """
    query_embedding = await embed(query)
    query_vec_str = str(query_embedding)

    pool = await get_pool()
    async with pool.acquire() as conn:
        # RLS
        await conn.execute(
            "SELECT set_config('app.tenant_id', $1, true)", tenant_id
        )
        rows = await conn.fetch(
            """
            SELECT
                kc.id,
                kc.document_id,
                kc.content,
                1 - (kc.embedding <=> $2::vector) AS similarity,
                kd.category,
                kd.title
            FROM knowledge_chunks kc
            JOIN knowledge_documents kd ON kd.id = kc.document_id
            WHERE kc.tenant_id = $1::uuid
              AND kd.is_active = true
              AND 1 - (kc.embedding <=> $2::vector) >= $3
            ORDER BY kc.embedding <=> $2::vector
            LIMIT $4
            """,
            uuid.UUID(tenant_id),
            query_vec_str,
            min_similarity,
            top_k,
        )

    results = [
        {
            "id": str(r["id"]),
            "document_id": str(r["document_id"]),
            "content": r["content"],
            "similarity": float(r["similarity"]),
            "category": r["category"],
            "title": r["title"],
        }
        for r in rows
    ]
    log.debug(
        "rag.store.search: tenant=%s query_len=%d results=%d",
        tenant_id, len(query), len(results),
    )
    return results
