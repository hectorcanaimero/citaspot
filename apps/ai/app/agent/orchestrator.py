# Orquestador del asistente IA: state machine → intent → action → respuesta.
from __future__ import annotations

import asyncio
import os
from typing import Any

import structlog

from app.agent.actions import (
    book_appointment,
    get_availability,
    get_tenant_profile,
    rag_query,
)
from app.agent.intent import Intent, detect as detect_intent
from app.agent.messages import get_messages
from app.agent.state import ConvState, get_state, reset_state, save_state
from app.core.rabbitmq import publish
from app.llm.router import chat

log = structlog.get_logger(__name__)

# Idioma del asistente — configurar con WHATSAPP_LANGUAGE (es | en | pt). Default: es.
_LANG = os.environ.get("WHATSAPP_LANGUAGE", "es")

# Timeout para publicar "Un momento..." al usuario
_SLOW_RESPONSE_SECS = 5.0


def _m(key: str, **kwargs: Any) -> str:
    """Obtiene el mensaje traducido para la clave dada, con sustitución de parámetros."""
    msg = get_messages(_LANG).get(key, key)
    return msg.format(**kwargs) if kwargs else msg


async def _publish_reply(tenant_id: str, tenant_slug: str, wa_phone: str, text: str) -> None:
    """Publica la respuesta en wa.messages.outbound."""
    await publish(
        "wa.messages.outbound",
        {
            "tenant_id": tenant_id,
            "tenant_slug": tenant_slug,
            "wa_phone": wa_phone,
            "content": text,
        },
    )


def _build_system_prompt(profile: dict[str, Any] | None, rag_context: str) -> str:
    """Construye el system prompt del asistente con contexto del negocio."""
    msgs = get_messages(_LANG)
    business_name = profile.get("name", "el negocio") if profile else "el negocio"

    services_text = ""
    if profile and profile.get("services"):
        lines = [
            f"- {s['name']} (${s['price']} USD, {s['duration_min']} min)"
            for s in profile["services"]
        ]
        services_text = f"\n\n{msgs['system_services_header']}\n" + "\n".join(lines)

    rag_section = (
        f"\n\n{msgs['system_rag_header']}\n{rag_context}" if rag_context else ""
    )

    intro = msgs["system_intro"].format(business_name=business_name)
    warning = msgs["system_warning"]

    return f"{intro}{services_text}{rag_section}\n\n{warning}"


async def process_message(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    wa_phone: str,
    message_text: str,
    history: list[dict[str, Any]],
) -> None:
    """
    Punto de entrada principal del orquestador.
    Lee el estado de Redis, detecta intención, ejecuta acción y publica respuesta.

    Args:
        tenant_id: UUID del tenant.
        tenant_slug: Slug del tenant (para llamadas al Core API).
        conversation_id: UUID de la conversación en DB.
        wa_phone: Número WhatsApp del cliente.
        message_text: Texto del mensaje recibido.
        history: Últimos mensajes de la conversación [{role, content}].
    """
    state = await get_state(tenant_id, conversation_id)
    current_state = ConvState(state.get("state", ConvState.IDLE))

    log.info(
        "orchestrator: procesando mensaje",
        tenant=tenant_id,
        conv=conversation_id,
        state=str(current_state),
        msg_len=len(message_text),
    )

    # --- Timeout: avisar "Un momento..." si el procesamiento es lento ---
    slow_task = asyncio.create_task(
        _send_slow_response(tenant_id, tenant_slug, wa_phone, _SLOW_RESPONSE_SECS)
    )

    try:
        response = await _handle(
            tenant_id=tenant_id,
            tenant_slug=tenant_slug,
            conversation_id=conversation_id,
            wa_phone=wa_phone,
            message_text=message_text,
            history=history,
            state=state,
            current_state=current_state,
        )
    finally:
        slow_task.cancel()

    if response:
        await _publish_reply(tenant_id, tenant_slug, wa_phone, response)


async def _send_slow_response(tenant_id: str, tenant_slug: str, wa_phone: str, delay: float) -> None:
    """Envía el mensaje de espera si el procesamiento supera el timeout."""
    await asyncio.sleep(delay)
    await _publish_reply(tenant_id, tenant_slug, wa_phone, _m("slow_response"))


async def _handle(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    wa_phone: str,
    message_text: str,
    history: list[dict[str, Any]],
    state: dict[str, Any],
    current_state: ConvState,
) -> str | None:
    """Lógica interna del orquestador según el estado actual."""

    # --- Estado HANDED_OFF: no responder automáticamente ---
    if current_state == ConvState.HANDED_OFF:
        return None

    # --- Detectar intención ---
    intent = await detect_intent(message_text, history)

    # === AWAITING_CONFIRM: el usuario está confirmando o cancelando la reserva ===
    if current_state == ConvState.AWAITING_CONFIRM:
        if intent == Intent.CONFIRM:
            return await _confirm_booking(tenant_id, tenant_slug, wa_phone, conversation_id, state)
        elif intent in (Intent.CANCEL, Intent.HANDOFF):
            await reset_state(tenant_id, conversation_id)
            if intent == Intent.HANDOFF:
                state["state"] = ConvState.HANDED_OFF
                await save_state(tenant_id, conversation_id, state)
                return _m("handoff")
            return _m("booking_cancelled")
        # Cualquier otra cosa: re-preguntar
        return _confirm_prompt(state)

    # === AWAITING_SLOT: el usuario debe elegir un horario ===
    if current_state == ConvState.AWAITING_SLOT:
        return await _handle_slot_selection(
            message_text, tenant_id, conversation_id, state, wa_phone
        )

    # === AWAITING_NAME: recolectando nombre del cliente ===
    if current_state == ConvState.AWAITING_NAME:
        return await _handle_name_collection(
            message_text, tenant_id, tenant_slug, conversation_id, state, wa_phone
        )

    # === IDLE: flujos nuevos ===
    if intent == Intent.HANDOFF:
        state["state"] = ConvState.HANDED_OFF
        await save_state(tenant_id, conversation_id, state)
        return _m("handoff")

    if intent == Intent.BOOKING:
        return await _start_booking_flow(
            tenant_id, tenant_slug, conversation_id, state, wa_phone
        )

    if intent == Intent.QUERY:
        return await _handle_query(tenant_id, tenant_slug, message_text, history)

    if intent == Intent.CANCEL:
        return _m("cancel_cta")

    # UNKNOWN — respuesta general con RAG
    return await _handle_query(tenant_id, tenant_slug, message_text, history)


async def _start_booking_flow(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    state: dict[str, Any],
    wa_phone: str,
) -> str:
    """Inicia el flujo de reserva: muestra servicios disponibles."""
    profile = await get_tenant_profile(tenant_slug)
    if not profile or not profile.get("services"):
        return _m("no_services")

    services = profile["services"]
    lines = [f"{i+1}. {s['name']} — ${s['price']} USD ({s['duration_min']} min)"
             for i, s in enumerate(services)]

    state["state"] = ConvState.AWAITING_SLOT
    state["profile"] = profile
    state["pending_date"] = None
    await save_state(tenant_id, conversation_id, state)

    return (
        _m("choose_service_header") + "\n\n"
        + "\n".join(lines)
        + "\n\n" + _m("choose_service_footer")
    )


async def _handle_slot_selection(
    message_text: str,
    tenant_id: str,
    conversation_id: str,
    state: dict[str, Any],
    wa_phone: str,
) -> str:
    """Procesa la selección del usuario en el flujo de slots."""
    profile = state.get("profile", {})
    services = profile.get("services", [])

    # Si no hay servicio seleccionado aún, el usuario está eligiendo servicio
    if not state.get("pending_service_id"):
        try:
            idx = int(message_text.strip()) - 1
            if idx < 0 or idx >= len(services):
                raise ValueError
            service = services[idx]
        except (ValueError, IndexError):
            lines = [f"{i+1}. {s['name']}" for i, s in enumerate(services)]
            return _m("invalid_option") + "\n" + "\n".join(lines)

        state["pending_service_id"] = service["id"]
        state["pending_service_name"] = service["name"]

        # Si hay más de un profesional, preguntar
        professionals = profile.get("professionals", [])
        if len(professionals) > 1:
            lines = [f"{i+1}. {p['name']}" for i, p in enumerate(professionals)]
            state["professionals"] = professionals
            await save_state(tenant_id, conversation_id, state)
            return _m("choose_professional", service=service["name"]) + "\n\n" + "\n".join(lines)
        elif professionals:
            state["pending_professional_id"] = professionals[0]["id"]
            state["pending_professional_name"] = professionals[0]["name"]
        else:
            return _m("no_professionals")

        await save_state(tenant_id, conversation_id, state)
        return _m(
            "choose_date_with_prof",
            service=service["name"],
            professional=state["pending_professional_name"],
        )

    # Si hay profesionales pendientes de selección
    if "professionals" in state and not state.get("pending_professional_id"):
        professionals = state.get("professionals", [])
        try:
            idx = int(message_text.strip()) - 1
            prof = professionals[idx]
        except (ValueError, IndexError):
            lines = [f"{i+1}. {p['name']}" for i, p in enumerate(professionals)]
            return _m("choose_professional_prompt") + "\n" + "\n".join(lines)

        state["pending_professional_id"] = prof["id"]
        state["pending_professional_name"] = prof["name"]
        await save_state(tenant_id, conversation_id, state)
        return _m("choose_date_short", professional=prof["name"])

    # Si hay fecha pero no slots mostrados aún
    if not state.get("pending_date"):
        import re
        date_match = re.search(r"\d{4}-\d{2}-\d{2}", message_text)
        if not date_match:
            return _m("invalid_date_format")

        date_str = date_match.group()
        slug = profile.get("slug", "")
        slots = await get_availability(
            slug,
            state["pending_professional_id"],
            state["pending_service_id"],
            date_str,
        )
        if not slots:
            return _m("no_slots", date=date_str)

        # Mostrar máximo 5 slots
        shown_slots = slots[:5]
        state["pending_date"] = date_str
        state["pending_slots"] = shown_slots
        await save_state(tenant_id, conversation_id, state)

        from datetime import datetime
        lines = []
        for i, s in enumerate(shown_slots):
            try:
                t = datetime.fromisoformat(s["starts_at"].replace("Z", "+00:00"))
                lines.append(f"{i+1}. {t.strftime('%I:%M %p')}")
            except Exception:
                lines.append(f"{i+1}. {s['starts_at']}")

        return (
            _m("available_slots_header", date=date_str) + "\n\n"
            + "\n".join(lines)
            + "\n\n" + _m("available_slots_footer")
        )

    # Usuario eligiendo slot horario
    slots = state.get("pending_slots", [])
    try:
        idx = int(message_text.strip()) - 1
        slot = slots[idx]
    except (ValueError, IndexError):
        return _m("invalid_slot", max=len(slots))

    state["pending_selected_slot"] = slot
    state["state"] = ConvState.AWAITING_CONFIRM
    await save_state(tenant_id, conversation_id, state)
    return _confirm_prompt(state)


def _confirm_prompt(state: dict[str, Any]) -> str:
    """Genera el mensaje de confirmación de reserva."""
    slot = state.get("pending_selected_slot", {})
    service_name = state.get("pending_service_name", "el servicio")
    prof_name = state.get("pending_professional_name", "el profesional")

    from datetime import datetime
    try:
        t = datetime.fromisoformat(slot["starts_at"].replace("Z", "+00:00"))
        hora = t.strftime("%d/%m/%Y a las %I:%M %p")
    except Exception:
        hora = slot.get("starts_at", "la hora seleccionada")

    return _m("confirm_prompt", service=service_name, professional=prof_name, datetime=hora)


async def _confirm_booking(
    tenant_id: str,
    tenant_slug: str,
    wa_phone: str,
    conversation_id: str,
    state: dict[str, Any],
) -> str:
    """Confirma y crea la cita en el Core API."""
    customer_name = state.get("customer_name")

    # Si no tenemos nombre del cliente, pedirlo
    if not customer_name:
        state["state"] = ConvState.AWAITING_NAME
        await save_state(tenant_id, conversation_id, state)
        return _m("ask_name")

    slot = state.get("pending_selected_slot", {})
    appt = await book_appointment(
        slug=tenant_slug,
        professional_id=state["pending_professional_id"],
        service_id=state["pending_service_id"],
        starts_at=slot["starts_at"],
        customer_name=customer_name,
        customer_phone=wa_phone,
        source="whatsapp",
    )

    if not appt:
        return _m("booking_failed")

    await reset_state(tenant_id, conversation_id)
    service_name = state.get("pending_service_name", "tu cita")
    datetime_str = slot.get("starts_at", "")
    return _m("booking_confirmed", service=service_name, datetime=datetime_str)


async def _handle_name_collection(
    message_text: str,
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    state: dict[str, Any],
    wa_phone: str,
) -> str:
    """Recolecta el nombre del cliente y completa la reserva."""
    name = message_text.strip()
    if len(name) < 2:
        return _m("invalid_name")

    state["customer_name"] = name
    state["state"] = ConvState.AWAITING_CONFIRM
    await save_state(tenant_id, conversation_id, state)
    return await _confirm_booking(tenant_id, tenant_slug, wa_phone, conversation_id, state)


async def _handle_query(
    tenant_id: str,
    tenant_slug: str,
    message_text: str,
    history: list[dict[str, Any]],
) -> str:
    """Responde preguntas usando RAG + LLM."""
    profile = await get_tenant_profile(tenant_slug)
    context = await rag_query(tenant_id, message_text)

    system = _build_system_prompt(profile, context)
    messages = [{"role": "system", "content": system}]

    for h in history[-8:]:
        messages.append({"role": h["role"], "content": h["content"]})
    messages.append({"role": "user", "content": message_text})

    return await chat(messages, temperature=0.3)
