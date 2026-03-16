# POST /vectorize — vectoriza un documento de la knowledge base.
# Llamado por el Core API (worker de knowledge.vectorize) cuando se crea/actualiza un documento.
# Soporta dos flujos:
#   - Texto manual: content presente, file_bytes vacío → chunk + embed directamente
#   - Archivo subido: file_bytes (base64) + file_type presentes → parsear → chunk + embed
from __future__ import annotations

import base64
import logging
import uuid as _uuid

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from app.core.database import get_pool
from app.rag.chunker import chunk_text
from app.rag.parser import extract_text
from app.rag.store import vectorize_document

log = logging.getLogger(__name__)
router = APIRouter(prefix="/vectorize", tags=["vectorize"])


class VectorizeRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant propietario")
    document_id: str = Field(..., description="UUID del knowledge_document")
    content: str = Field(default="", description="Contenido de texto directo (flujo manual)")
    file_bytes: str = Field(default="", description="Archivo en base64 (flujo upload)")
    file_type: str = Field(default="", description="Tipo de archivo: pdf | docx | xlsx")


class VectorizeResponse(BaseModel):
    document_id: str
    chunks_count: int


@router.post("", response_model=VectorizeResponse)
async def vectorize(req: VectorizeRequest) -> VectorizeResponse:
    """
    Vectoriza un documento de la knowledge base.
    - Si contiene file_bytes: extrae texto, actualiza content en DB, luego vectoriza.
    - Si contiene content: vectoriza directamente.
    Operación idempotente: elimina chunks previos antes de insertar.
    """
    content = req.content

    # ── Flujo archivo ──────────────────────────────────────────────────────────
    if req.file_bytes:
        if not req.file_type:
            raise HTTPException(status_code=422, detail="file_type es requerido cuando se envía file_bytes")

        # Decodificar base64
        try:
            raw_bytes = base64.b64decode(req.file_bytes)
        except Exception as e:
            await _mark_error(req.tenant_id, req.document_id, "Error decodificando el archivo")
            raise HTTPException(status_code=422, detail=f"file_bytes no es base64 válido: {e}")

        # Extraer texto según el tipo de archivo
        try:
            content = extract_text(raw_bytes, req.file_type)
        except ValueError as e:
            await _mark_error(req.tenant_id, req.document_id, str(e))
            raise HTTPException(status_code=422, detail=str(e))
        except RuntimeError as e:
            await _mark_error(req.tenant_id, req.document_id, str(e))
            raise HTTPException(status_code=500, detail=str(e))

        # Persistir el contenido extraído en knowledge_documents
        await _save_content(req.tenant_id, req.document_id, content, status="ready")
        log.info(
            "vectorize: archivo parseado doc=%s tipo=%s chars=%d",
            req.document_id, req.file_type, len(content),
        )

    # ── Validación de contenido ────────────────────────────────────────────────
    if not content.strip():
        raise HTTPException(status_code=422, detail="El contenido no puede estar vacío")

    chunks = chunk_text(content)
    if not chunks:
        raise HTTPException(status_code=422, detail="No se pudo dividir el contenido en chunks")

    await vectorize_document(
        tenant_id=req.tenant_id,
        document_id=req.document_id,
        chunks=chunks,
    )
    return VectorizeResponse(document_id=req.document_id, chunks_count=len(chunks))


async def _save_content(tenant_id: str, document_id: str, content: str, status: str) -> None:
    """Actualiza content y status en knowledge_documents tras extraer el texto del archivo."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            "SELECT set_config('app.tenant_id', $1, true)", tenant_id
        )
        await conn.execute(
            """
            UPDATE knowledge_documents
            SET content = $1, status = $2, updated_at = NOW()
            WHERE tenant_id = $3::uuid AND id = $4::uuid
            """,
            content,
            status,
            _uuid.UUID(tenant_id),
            _uuid.UUID(document_id),
        )


async def _mark_error(tenant_id: str, document_id: str, reason: str) -> None:
    """Marca el documento como 'error' para que el frontend informe al usuario."""
    try:
        await _save_content(tenant_id, document_id, "", "error")
        log.warning("vectorize: marcado como error doc=%s razón=%s", document_id, reason)
    except Exception as e:
        log.error("vectorize: no se pudo marcar error doc=%s: %s", document_id, e)
