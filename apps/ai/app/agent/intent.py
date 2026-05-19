# Detección de intención del usuario usando el LLM Router.
from __future__ import annotations

import logging
import re
from enum import Enum
from typing import TYPE_CHECKING, Any

from app.llm.router import chat_lite

if TYPE_CHECKING:
    from app.agent.state import ConvState

log = logging.getLogger(__name__)


class Intent(str, Enum):
    BOOKING = "BOOKING"                # reservar cita
    QUERY = "QUERY"                    # pregunta sobre servicios/precios/etc.
    CONFIRM = "CONFIRM"                # confirmar selección (sí, acepto, ok, etc.)
    CANCEL = "CANCEL"                  # cancelar cita o salir del flujo
    MY_APPOINTMENTS = "MY_APPOINTMENTS"  # consultar citas existentes
    RESCHEDULE = "RESCHEDULE"          # reagendar cita existente
    HANDOFF = "HANDOFF"                # solicitar hablar con humano
    SEND_BOOKING_LINK = "SEND_BOOKING_LINK"  # pide explícitamente el link de reserva web
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
- SEND_BOOKING_LINK: pide explícitamente el link de reserva por la web (mándame el link, pásame el link, mejor por la página, send me the booking link, etc.)
- UNKNOWN: no puedes determinar la intención

Responde ÚNICAMENTE con una de las palabras clave: BOOKING, QUERY, CONFIRM, CANCEL, MY_APPOINTMENTS, RESCHEDULE, HANDOFF, SEND_BOOKING_LINK, UNKNOWN.
No incluyas explicaciones ni puntuación."""


# --- Pattern matching: respuestas cortas que no necesitan LLM ---
# Token positivo: una palabra de confirmación (sí/ok/dale/listo/yes/sim/etc.) o emoji.
# El mensaje completo debe ser una secuencia de 1 a 4 tokens positivos separados por
# espacios/puntuación leve, por ej.: "dale", "ok", "dale ok", "sí, perfecto", "ok listo".
_POSITIVE_TOKEN = (
    r"(?:s[ií]|ok(?:ay|ey)?|dale|listo|confirmo|confirmado|"
    r"de\s+acuerdo|perfecto|por\s+supuesto|claro|vale|"
    r"yes|yeah|yep|yup|sure|"
    r"sim|"
    r"\U0001F44D|✅)"
)
_POSITIVE_PATTERN = re.compile(
    rf"^\s*{_POSITIVE_TOKEN}(?:[\s.!¡,]+{_POSITIVE_TOKEN}){{0,3}}[\s.!¡,]*$",
    re.IGNORECASE | re.UNICODE,
)

# Negativas explícitas en es/en/pt y emojis. Mismo patrón compositivo.
_NEGATIVE_TOKEN = (
    r"(?:no|nope|nah|"
    r"cancelar|cancela|cancelado|"
    r"nada|mejor\s+no|"
    r"n[ãa]o|"
    r"❌)"
)
_NEGATIVE_PATTERN = re.compile(
    rf"^\s*{_NEGATIVE_TOKEN}(?:[\s.!¡,]+{_NEGATIVE_TOKEN}){{0,3}}[\s.!¡,]*$",
    re.IGNORECASE | re.UNICODE,
)

# Pedido explícito del link de reserva web (es/en/pt).
_LINK_PATTERN = re.compile(
    r"\b(?:m[aá]ndame\s+el\s+link|p[aá]same\s+el\s+link|link\s+de\s+reserva|"
    r"por\s+(?:la\s+)?(?:p[aá]gina|web)|mejor\s+por\s+(?:la\s+)?web|"
    r"send\s+(?:me\s+)?the\s+link|booking\s+link|"
    r"me\s+manda\s+o\s+link|pela\s+(?:p[aá]gina|web)|link\s+de\s+agendamento)\b",
    re.IGNORECASE | re.UNICODE,
)


def _pattern_match(message: str, conv_state: "ConvState | None") -> Intent | None:
    """
    Layer 1 sin LLM: si el mensaje es una confirmación/negación inequívoca,
    devuelve la intención directamente. Retorna None si no hay match.
    """
    # Importar acá para evitar ciclos en tiempo de carga (solo se necesita en runtime).
    from app.agent.state import ConvState as _ConvState

    if not message:
        return None

    if _POSITIVE_PATTERN.match(message):
        # En estados que esperan confirmación, "sí/ok/dale" => CONFIRM.
        # Fuera de flujo, un "sí" suelto suele significar "sí quiero reservar".
        confirm_states = {
            _ConvState.AWAITING_CONFIRM,
            _ConvState.AWAITING_NAME,
            _ConvState.AWAITING_CANCEL_CONFIRM,
            _ConvState.AWAITING_RESCHEDULE_CONFIRM,
        }
        if conv_state in confirm_states:
            return Intent.CONFIRM
        return Intent.BOOKING

    if _NEGATIVE_PATTERN.match(message):
        return Intent.CANCEL

    if _LINK_PATTERN.search(message):
        return Intent.SEND_BOOKING_LINK

    return None


def _state_hint(conv_state: "ConvState | None", pending_data: dict[str, Any] | None) -> str:
    """Construye una pista para el LLM según el estado actual del flujo."""
    from app.agent.state import ConvState as _ConvState

    if conv_state is None or conv_state == _ConvState.IDLE:
        return ""

    pending_keys = sorted(k for k, v in (pending_data or {}).items() if v)
    pending_str = ", ".join(pending_keys) if pending_keys else "ninguno"

    base = (
        f"\n\nCONTEXTO DE CONVERSACIÓN: el usuario está actualmente en el estado "
        f"`{conv_state.value if hasattr(conv_state, 'value') else conv_state}`. "
        f"Datos pendientes: {pending_str}. "
        f"Sesgá la clasificación hacia intenciones que tengan sentido en este estado."
    )

    if conv_state in (_ConvState.AWAITING_CONFIRM, _ConvState.AWAITING_CANCEL_CONFIRM, _ConvState.AWAITING_RESCHEDULE_CONFIRM):
        return base + " Lo más probable es CONFIRM o CANCEL."
    if conv_state in (
        _ConvState.AWAITING_SLOT,
        _ConvState.AWAITING_NAME,
        _ConvState.AWAITING_CANCEL_SELECT,
        _ConvState.AWAITING_RESCHEDULE_SELECT,
        _ConvState.AWAITING_RESCHEDULE_DATE,
        _ConvState.AWAITING_RESCHEDULE_SLOT,
    ):
        return base + (
            " El usuario está en medio de un flujo: trátalo como nueva intención "
            "(QUERY/BOOKING/CANCEL/HANDOFF) solo si el mensaje es claramente off-topic."
        )
    return base


async def detect(
    message: str,
    history: list[dict[str, Any]] | None = None,
    conv_state: "ConvState | None" = None,
    pending_data: dict[str, Any] | None = None,
) -> Intent:
    """
    Detecta la intención del mensaje del usuario, sesgando por el estado de la conversación.

    Args:
        message: Texto del usuario.
        history: Últimos mensajes de la conversación para contexto.
        conv_state: Estado actual de la conversación (Redis) — afecta clasificación.
        pending_data: Datos parciales ya recolectados en el flujo (service_id, etc.).

    Returns:
        Intent detectada.
    """
    # --- Layer 1: pattern matching (sin LLM) ---
    matched = _pattern_match(message, conv_state)
    if matched is not None:
        log.debug("agent.intent: pattern match → %s (state=%s)", matched, conv_state)
        return matched

    # --- Layer 2 / 3: LLM (con o sin hint de estado) ---
    system_prompt = _SYSTEM_PROMPT + _state_hint(conv_state, pending_data)
    messages = [{"role": "system", "content": system_prompt}]

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
