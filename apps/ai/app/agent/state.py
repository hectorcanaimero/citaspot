# State machine de conversación persistida en Redis.
from __future__ import annotations

import json
import logging
from enum import Enum
from typing import Any

from app.core.redis import get_redis

log = logging.getLogger(__name__)

# TTL de la conversación en Redis: 24 horas de inactividad
_CONV_TTL = 86_400  # segundos


class ConvState(str, Enum):
    IDLE = "IDLE"
    AWAITING_SLOT = "AWAITING_SLOT"       # usuario eligiendo horario
    AWAITING_CONFIRM = "AWAITING_CONFIRM"  # esperando confirmación de reserva
    AWAITING_NAME = "AWAITING_NAME"        # recolectando nombre del cliente nuevo
    HANDED_OFF = "HANDED_OFF"              # escalado a humano
    AWAITING_CANCEL_SELECT = "AWAITING_CANCEL_SELECT"
    AWAITING_CANCEL_CONFIRM = "AWAITING_CANCEL_CONFIRM"
    AWAITING_RESCHEDULE_SELECT = "AWAITING_RESCHEDULE_SELECT"
    AWAITING_RESCHEDULE_DATE = "AWAITING_RESCHEDULE_DATE"
    AWAITING_RESCHEDULE_SLOT = "AWAITING_RESCHEDULE_SLOT"
    AWAITING_RESCHEDULE_CONFIRM = "AWAITING_RESCHEDULE_CONFIRM"


def _key(tenant_id: str, conversation_id: str) -> str:
    return f"conv:{tenant_id}:{conversation_id}"


async def get_state(tenant_id: str, conversation_id: str) -> dict[str, Any]:
    """
    Retorna el estado completo de la conversación desde Redis.
    Si no existe, retorna estado inicial.
    """
    r = get_redis()
    raw = await r.get(_key(tenant_id, conversation_id))
    if raw:
        return json.loads(raw)
    return {
        "state": ConvState.IDLE,
        "pending_slots": [],       # slots disponibles mostrados al usuario
        "pending_service_id": None,
        "pending_professional_id": None,
        "pending_date": None,
        "customer_name": None,
        "customer_phone": None,
        "booking_turns_without_service": 0,  # contador para oferta suave del link
    }


async def save_state(tenant_id: str, conversation_id: str, data: dict[str, Any]) -> None:
    """Persiste el estado de la conversación en Redis con TTL de 24h."""
    r = get_redis()
    await r.setex(
        _key(tenant_id, conversation_id),
        _CONV_TTL,
        json.dumps(data, default=str),
    )
    log.debug("agent.state: saved tenant=%s conv=%s state=%s", tenant_id, conversation_id, data.get("state"))


async def reset_state(tenant_id: str, conversation_id: str) -> None:
    """Borra el estado de una conversación (después de completar o cancelar)."""
    r = get_redis()
    await r.delete(_key(tenant_id, conversation_id))


# TTL para conversaciones de test: 30 minutos
_TEST_TTL = 1_800  # segundos


def _test_key(tenant_id: str) -> str:
    return f"test:{tenant_id}"


async def get_test_state(tenant_id: str) -> dict[str, Any]:
    """Retorna el estado de la conversación de test desde Redis."""
    r = get_redis()
    raw = await r.get(_test_key(tenant_id))
    if raw:
        return json.loads(raw)
    return {
        "state": ConvState.IDLE,
        "pending_slots": [],
        "pending_service_id": None,
        "pending_professional_id": None,
        "pending_date": None,
        "customer_name": None,
        "customer_phone": None,
        "booking_turns_without_service": 0,
    }


async def save_test_state(tenant_id: str, data: dict[str, Any]) -> None:
    """Persiste el estado de test en Redis con TTL de 30 min."""
    r = get_redis()
    await r.setex(
        _test_key(tenant_id),
        _TEST_TTL,
        json.dumps(data, default=str),
    )


async def reset_test_state(tenant_id: str) -> None:
    """Borra el estado de test de un tenant."""
    r = get_redis()
    await r.delete(_test_key(tenant_id))
