# Generación de embeddings con OpenAI text-embedding-3-small (1536 dims).
from __future__ import annotations

import logging

from openai import AsyncOpenAI

from app.core.config import settings

log = logging.getLogger(__name__)

_EMBEDDING_MODEL = "text-embedding-3-small"
_DIMS = 1536

_client = AsyncOpenAI(api_key=settings.openai_api_key)


async def embed(text: str) -> list[float]:
    """
    Genera embedding para un texto.

    Returns:
        Vector de 1536 floats.
    """
    text = text.replace("\n", " ").strip()
    response = await _client.embeddings.create(
        model=_EMBEDDING_MODEL,
        input=text,
        dimensions=_DIMS,
    )
    return response.data[0].embedding


async def embed_batch(texts: list[str]) -> list[list[float]]:
    """
    Genera embeddings para múltiples textos en una sola llamada a la API.

    Returns:
        Lista de vectores en el mismo orden que texts.
    """
    cleaned = [t.replace("\n", " ").strip() for t in texts]
    response = await _client.embeddings.create(
        model=_EMBEDDING_MODEL,
        input=cleaned,
        dimensions=_DIMS,
    )
    # Ordenar por índice (la API puede devolver en cualquier orden)
    sorted_data = sorted(response.data, key=lambda d: d.index)
    return [d.embedding for d in sorted_data]
