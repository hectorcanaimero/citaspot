# Historial de conversación persistido en Redis (LIST por conversación).
#
# Key:  conv:hist:{tenant_id}:{conversation_id}  (Redis LIST)
# TTL:  24h, refrescado en cada append.
# Cap:  LTRIM a los últimos 20 mensajes para evitar crecimiento ilimitado.
# Read: devuelve los últimos N (default 10) en orden cronológico (oldest → newest).
#
# Cada entrada se serializa como JSON: {"role": "user|assistant", "content": "..."}.
# Las operaciones (LPUSH + LTRIM + EXPIRE) corren en pipeline para atomicidad.
# Cualquier error de Redis se loggea como warning y degrada a un default seguro
# (None / lista vacía) — el worker NUNCA debe crashear por un fallo de historial.
from __future__ import annotations

import json
import logging
from typing import Any

from app.core.redis import get_redis

log = logging.getLogger(__name__)

# TTL del historial en Redis: 24h de inactividad
_HIST_TTL = 86_400  # segundos

# Cap máximo de mensajes guardados (los más recientes)
_HIST_CAP = 20

# Roles permitidos en el historial
_VALID_ROLES = frozenset({"user", "assistant"})


def _key(tenant_id: str, conversation_id: str) -> str:
    return f"conv:hist:{tenant_id}:{conversation_id}"


async def append_message(
    tenant_id: str, conversation_id: str, role: str, content: str
) -> None:
    """Persiste un mensaje al historial (LPUSH + LTRIM + EXPIRE atómico).

    - role debe ser 'user' o 'assistant'.
    - content vacío se ignora (no aporta señal al LLM).
    - Errores de Redis se loggean como warning y NO se propagan.
    """
    if role not in _VALID_ROLES:
        log.warning("history.append: role inválido role=%s", role)
        return
    if not content:
        return

    entry = json.dumps({"role": role, "content": content}, ensure_ascii=False)
    key = _key(tenant_id, conversation_id)

    try:
        r = get_redis()
        pipe = r.pipeline(transaction=True)
        pipe.lpush(key, entry)
        # Mantener solo los últimos _HIST_CAP elementos (índices 0.._HIST_CAP-1)
        pipe.ltrim(key, 0, _HIST_CAP - 1)
        pipe.expire(key, _HIST_TTL)
        await pipe.execute()
        # NO loguear content (PII) — solo role y longitud
        log.debug(
            "history.append: ok tenant=%s conv=%s role=%s len=%d",
            tenant_id, conversation_id, role, len(content),
        )
    except Exception as e:  # noqa: BLE001 — defensivo: Redis caído no debe romper el worker
        log.warning(
            "history.append: redis error tenant=%s conv=%s role=%s err=%s",
            tenant_id, conversation_id, role, str(e),
        )


async def get_recent(
    tenant_id: str, conversation_id: str, limit: int = 10
) -> list[dict[str, str]]:
    """Retorna los últimos `limit` mensajes en orden cronológico (oldest → newest).

    Cada item es {"role": "...", "content": "..."}.
    En caso de error de Redis retorna lista vacía.
    """
    if limit <= 0:
        return []

    key = _key(tenant_id, conversation_id)
    try:
        r = get_redis()
        raw_list: list[Any] = await r.lrange(key, 0, limit - 1)
    except Exception as e:  # noqa: BLE001
        log.warning(
            "history.get_recent: redis error tenant=%s conv=%s err=%s",
            tenant_id, conversation_id, str(e),
        )
        return []

    # raw_list viene en orden newest → oldest (porque usamos LPUSH).
    # Invertimos para devolver cronológicamente.
    messages: list[dict[str, str]] = []
    for raw in reversed(raw_list):
        if raw is None:
            continue
        try:
            obj = json.loads(raw)
        except (TypeError, ValueError):
            # Entrada corrupta — saltarla, no romper la conversación
            continue
        role = obj.get("role")
        content = obj.get("content")
        if role in _VALID_ROLES and isinstance(content, str) and content:
            messages.append({"role": role, "content": content})

    return messages


async def clear(tenant_id: str, conversation_id: str) -> None:
    """Elimina el historial de una conversación (best-effort)."""
    key = _key(tenant_id, conversation_id)
    try:
        r = get_redis()
        await r.delete(key)
    except Exception as e:  # noqa: BLE001
        log.warning(
            "history.clear: redis error tenant=%s conv=%s err=%s",
            tenant_id, conversation_id, str(e),
        )
