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
    return_error_details: bool = False,
) -> dict[str, Any] | None:
    """
    Crea una cita vía el endpoint público del Core API.

    Args:
        return_error_details: Si True, en lugar de retornar None ante un fallo
            HTTP, retorna un dict {"_error": True, "status": int, "message": str}.
            Pensado para callers que necesitan distinguir 409 (slot tomado) de
            422 (validación) — p.ej. la tool `book_appointment` del LLM agent.
            Si False (default), preserva el contrato histórico: None en fallo.
            En éxito siempre retorna el dict de la cita creada.

    Returns:
        Dict con la cita creada en caso de éxito.
        None si falla y return_error_details=False.
        Dict con _error=True si falla y return_error_details=True.
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
            if r.status_code >= 400:
                # Loguear sin PII: solo status y slug
                log.error(
                    "actions.book_appointment: http %s en %s",
                    r.status_code, slug,
                )
                if return_error_details:
                    # Intentar leer el mensaje del body (Fiber retorna {"error": "..."})
                    err_msg = ""
                    try:
                        body_json = r.json()
                        if isinstance(body_json, dict):
                            err_msg = body_json.get("error") or body_json.get("message") or ""
                    except Exception:
                        err_msg = ""
                    return {
                        "_error": True,
                        "status": r.status_code,
                        "message": err_msg,
                    }
                return None
            return r.json()
        except Exception as e:
            log.error("actions.book_appointment: %s", e)
            if return_error_details:
                return {"_error": True, "status": 0, "message": str(e)}
            return None


async def get_my_appointments(slug: str, phone: str) -> list[dict[str, Any]]:
    """Obtiene citas futuras del cliente por teléfono."""
    params = {"phone": phone}
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.get(_core_url(f"/public/{slug}/my-appointments"), params=params)
            r.raise_for_status()
            return r.json().get("data", [])
        except Exception as e:
            log.error("actions.get_my_appointments: %s", e)
            return []


async def get_my_treatments(slug: str, phone: str) -> list[dict[str, Any]] | None:
    """Obtiene tratamientos activos del cliente por teléfono (Plane #34).
    Retorna None si el tenant no tiene módulo dental activo (HTTP 403)."""
    params = {"phone": phone}
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.get(_core_url(f"/public/{slug}/my-treatments"), params=params)
            if r.status_code == 403:
                return None  # módulo dental no activo para este tenant
            r.raise_for_status()
            return r.json().get("data", [])
        except Exception as e:
            log.error("actions.get_my_treatments: %s", e)
            return []


async def cancel_appointment(slug: str, appointment_id: str, phone: str) -> bool:
    """Cancela una cita existente. Retorna True si éxito."""
    body = {"phone": phone}
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.post(_core_url(f"/public/{slug}/appointments/{appointment_id}/cancel"), json=body)
            r.raise_for_status()
            return True
        except Exception as e:
            log.error("actions.cancel_appointment: %s", e)
            return False


async def reschedule_appointment(slug: str, appointment_id: str, phone: str, starts_at: str) -> bool:
    """Reagenda una cita existente. Retorna True si éxito."""
    body = {"phone": phone, "starts_at": starts_at}
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.post(_core_url(f"/public/{slug}/appointments/{appointment_id}/reschedule"), json=body)
            r.raise_for_status()
            return True
        except Exception as e:
            log.error("actions.reschedule_appointment: %s", e)
            return False


async def notify_handoff(
    slug: str,
    conversation_id: str,
    customer_phone: str,
    reason: str,
) -> bool:
    """
    Notifica al Core API que una conversación fue derivada a humano. El Core API
    crea un user_notification y lo publica al canal SSE del tenant para que la
    campana del dashboard lo muestre en tiempo real.

    Best-effort: si el POST falla, el handoff de Redis ya sucedió y el cliente
    igual ve "te conecto con el equipo". Solo perdemos la notificación interna.
    """
    body = {
        "conversation_id": conversation_id,
        "customer_phone": customer_phone,
        "reason": reason,
    }
    async with httpx.AsyncClient(timeout=_TIMEOUT) as client:
        try:
            r = await client.post(_core_url(f"/public/{slug}/handoff-notify"), json=body)
            r.raise_for_status()
            return True
        except Exception as e:
            log.error("actions.notify_handoff: %s", e)
            return False


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
