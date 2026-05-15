# Acciones del orquestador: llaman al Core API y al RAG store.
from __future__ import annotations

import logging
from typing import Any

import httpx

from app.core.config import settings
from app.rag.store import search as rag_search

log = logging.getLogger(__name__)

_TIMEOUT = 8.0  # segundos — si supera esto, el LLM ya envió "Un momento..."


def _core_url(path: str) -> str:
    return f"{settings.core_api_url.rstrip('/')}{path}"


async def get_tenant_profile(slug: str) -> dict[str, Any] | None:
    """
    Retorna el perfil público del tenant: nombre, servicios, profesionales.
    """
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.get(_core_url(f"/public/{slug}"))
            r.raise_for_status()
            return r.json()
        except Exception as e:
            log.error("actions.get_tenant_profile: %s", e)
            return None


async def get_availability(
    slug: str,
    professional_id: str,
    service_id: str,
    date: str,
    timezone: str = "UTC",
) -> list[dict[str, Any]]:
    """
    Consulta slots disponibles para un profesional/servicio/fecha.

    Returns:
        Lista de slots [{starts_at, ends_at}].
    """
    params = {
        "professional_id": professional_id,
        "service_id": service_id,
        "date": date,
        "timezone": timezone,
    }
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.get(_core_url(f"/public/{slug}/availability"), params=params)
            r.raise_for_status()
            return r.json().get("data", [])
        except Exception as e:
            log.error("actions.get_availability: %s", e)
            return []


async def book_appointment(
    slug: str,
    professional_id: str,
    service_id: str,
    starts_at: str,  # ISO 8601
    customer_name: str,
    customer_phone: str,
    source: str = "whatsapp",
) -> dict[str, Any] | None:
    """
    Crea una cita vía el endpoint público del Core API.

    Returns:
        Dict con la cita creada o None si falla.
    """
    body = {
        "professional_id": professional_id,
        "service_id": service_id,
        "starts_at": starts_at,
        "customer_name": customer_name,
        "customer_phone": customer_phone,
        "source": source,
    }
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.post(_core_url(f"/public/{slug}/book"), json=body)
            r.raise_for_status()
            return r.json()
        except Exception as e:
            log.error("actions.book_appointment: %s", e)
            return None


async def rag_query(tenant_id: str, query: str, min_similarity: float | None = None) -> str:
    """
    Busca en la base de conocimiento del tenant y retorna texto de contexto
    listo para incluir en el prompt del LLM.

    Args:
        tenant_id: UUID del tenant.
        query: Texto de búsqueda.
        min_similarity: Umbral mínimo de similitud (default: 0.70 del store).

    Returns:
        String con el contexto RAG o "" si no hay resultados.
    """
    kwargs: dict = {"tenant_id": tenant_id, "query": query}
    if min_similarity is not None:
        kwargs["min_similarity"] = min_similarity
    results = await rag_search(**kwargs)
    if not results:
        return ""

    parts = []
    for r in results:
        parts.append(f"[{r['category'].upper()} — {r['title']}]\n{r['content']}")

    return "\n\n---\n\n".join(parts)
