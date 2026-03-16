# Consumer RabbitMQ para knowledge.vectorize.
# Cada mensaje contiene un documento recién creado (texto o archivo base64).
# Este worker extrae el texto si es archivo, guarda el contenido en DB y vectoriza.
from __future__ import annotations

import asyncio
import base64
import json
import logging
import uuid as _uuid

import aio_pika
from aio_pika.abc import AbstractIncomingMessage

from app.core.config import settings
from app.core.database import get_pool
from app.rag.chunker import chunk_text
from app.rag.parser import extract_text
from app.rag.store import vectorize_document

log = logging.getLogger(__name__)

_QUEUE_NAME = "knowledge.vectorize"


async def _save_content(tenant_id: str, document_id: str, content: str, status: str) -> None:
    """Actualiza content y status del documento en knowledge_documents."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        await conn.execute("SELECT set_config('app.tenant_id', $1, true)", tenant_id)
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
    """Marca el documento como 'error' para que el frontend muestre el estado correcto."""
    try:
        await _save_content(tenant_id, document_id, "", "error")
        log.warning("worker.knowledge: error doc=%s razón=%s", document_id, reason)
    except Exception as e:
        log.error("worker.knowledge: no se pudo marcar error doc=%s: %s", document_id, e)


async def _process(message: AbstractIncomingMessage) -> None:
    """Procesa un mensaje de knowledge.vectorize."""
    async with message.process(requeue=False):
        try:
            payload = json.loads(message.body)
        except json.JSONDecodeError as e:
            log.error("worker.knowledge: JSON inválido: %s", e)
            return

        tenant_id = payload.get("tenant_id", "")
        document_id = payload.get("document_id", "")
        content = payload.get("content", "")
        file_bytes_b64 = payload.get("file_bytes", "")
        file_type = payload.get("file_type", "")

        if not tenant_id or not document_id:
            log.error("worker.knowledge: tenant_id o document_id faltante")
            return

        log.info("worker.knowledge: procesando doc=%s tenant=%s tipo=%s", document_id, tenant_id, file_type or "text")

        # ── Flujo archivo: extraer texto del binario ───────────────────────────
        if file_bytes_b64:
            if not file_type:
                await _mark_error(tenant_id, document_id, "file_type requerido cuando hay file_bytes")
                return

            try:
                raw_bytes = base64.b64decode(file_bytes_b64)
            except Exception as e:
                await _mark_error(tenant_id, document_id, f"base64 inválido: {e}")
                return

            try:
                content = extract_text(raw_bytes, file_type)
            except ValueError as e:
                await _mark_error(tenant_id, document_id, str(e))
                return
            except RuntimeError as e:
                await _mark_error(tenant_id, document_id, str(e))
                return

            # Persistir el texto extraído en la BD antes de vectorizar
            await _save_content(tenant_id, document_id, content, "ready")
            log.info("worker.knowledge: texto extraído doc=%s chars=%d", document_id, len(content))

        # ── Validación de contenido ────────────────────────────────────────────
        if not content.strip():
            await _mark_error(tenant_id, document_id, "contenido vacío tras parsear")
            return

        # ── Chunking y vectorización ───────────────────────────────────────────
        try:
            chunks = chunk_text(content)
            if not chunks:
                await _mark_error(tenant_id, document_id, "no se pudo generar chunks del contenido")
                return

            await vectorize_document(
                tenant_id=tenant_id,
                document_id=document_id,
                chunks=chunks,
            )
            log.info("worker.knowledge: vectorizado doc=%s chunks=%d", document_id, len(chunks))

            # Marcar como ready (texto manual llega sin status=processing, pero no hace daño actualizarlo)
            await _save_content(tenant_id, document_id, content, "ready")

        except Exception as e:
            log.exception("worker.knowledge: error vectorizando doc=%s: %s", document_id, e)
            await _mark_error(tenant_id, document_id, str(e))


async def start_consumer() -> None:
    """
    Inicia el consumer de knowledge.vectorize con reconexión automática.
    Llamar como asyncio.Task en el startup de FastAPI.
    """
    delays = [1, 2, 4, 8, 16]
    idx = 0

    while True:
        try:
            connection = await aio_pika.connect_robust(settings.rabbitmq_url)
            async with connection:
                channel = await connection.channel()
                await channel.set_qos(prefetch_count=4)  # procesar hasta 4 docs en paralelo
                queue = await channel.declare_queue(_QUEUE_NAME, durable=True)

                log.info("worker.knowledge: consumiendo %s", _QUEUE_NAME)
                idx = 0

                async with queue.iterator() as queue_iter:
                    async for message in queue_iter:
                        await _process(message)

        except asyncio.CancelledError:
            log.info("worker.knowledge: detenido")
            return
        except Exception as e:
            d = delays[min(idx, len(delays) - 1)]
            log.error("worker.knowledge: error de conexión (%s), reconectando en %ds", e, d)
            await asyncio.sleep(d)
            idx += 1
