# Detección de intención del usuario usando el LLM Router.
from __future__ import annotations

import logging
from enum import Enum

from app.llm.router import chat_lite

log = logging.getLogger(__name__)


class Intent(str, Enum):
    BOOKING = "BOOKING"                # reservar cita
    QUERY = "QUERY"                    # pregunta sobre servicios/precios/etc.
    CONFIRM = "CONFIRM"                # confirmar selección (sí, acepto, ok, etc.)
    CANCEL = "CANCEL"                  # cancelar cita o salir del flujo
    MY_APPOINTMENTS = "MY_APPOINTMENTS"  # consultar citas existentes
    RESCHEDULE = "RESCHEDULE"          # reagendar cita existente
    HANDOFF = "HANDOFF"                # solicitar hablar con humano
    UNKNOWN = "UNKNOWN"                # no se entiende la intención


_SYSTEM_PROMPT = """Eres un clasificador de intenciones para un asistente de agenda.
Debes clasificar el mensaje del usuario en una de estas categorías:
- BOOKING: quiere agendar, reservar o pedir una cita
- QUERY: pregunta sobre servicios, precios, horarios, ubicación, equipo, políticas
- CONFIRM: confirma algo (sí, ok, perfecto, acepto, ese horario, la primera opción, etc.)
- CANCEL: quiere cancelar una cita existente o cancelar el flujo actual (no, cancelar, no gracias, etc.)
- MY_APPOINTMENTS: quiere consultar, ver o saber sobre sus citas existentes (¿cuándo es mi cita?, mis citas, qué tengo agendado, tengo algo agendado)
- RESCHEDULE: quiere cambiar fecha/hora de una cita existente (reagendar, mover, cambiar la cita, cambiar el horario)
- HANDOFF: quiere hablar con un humano (agente, persona, asesor, etc.)
- UNKNOWN: no puedes determinar la intención

Responde ÚNICAMENTE con una de las palabras clave: BOOKING, QUERY, CONFIRM, CANCEL, MY_APPOINTMENTS, RESCHEDULE, HANDOFF, UNKNOWN.
No incluyas explicaciones ni puntuación."""


async def detect(message: str, history: list[dict] | None = None) -> Intent:
    """
    Detecta la intención del mensaje del usuario.

    Args:
        message: Texto del usuario.
        history: Últimos mensajes de la conversación para contexto.

    Returns:
        Intent detectada.
    """
    messages = [{"role": "system", "content": _SYSTEM_PROMPT}]

    # Incluir últimos 3 mensajes para contexto
    if history:
        for h in history[-3:]:
            messages.append({"role": h["role"], "content": h["content"]})

    messages.append({"role": "user", "content": message})

    try:
        result = await chat_lite(messages, temperature=0.0)
        raw = result.strip().upper()
        if raw in Intent.__members__:
            return Intent(raw)
        log.warning("agent.intent: respuesta inesperada del LLM: %r", result)
        return Intent.UNKNOWN
    except Exception as e:
        log.error("agent.intent: error detectando intención: %s", e)
        return Intent.UNKNOWN
