# Orquestador del asistente IA: state machine → intent → action → respuesta.
from __future__ import annotations

import asyncio
import os
import re
from datetime import date, datetime
from typing import Any
from zoneinfo import ZoneInfo

import structlog

from app.agent.actions import (
    book_appointment,
    cancel_appointment,
    get_availability,
    get_my_appointments,
    get_tenant_profile,
    notify_handoff,
    reschedule_appointment,
)
from app.agent.guards import GuardKind, run_all_guards
from app.agent.intent import Intent
from app.agent.intent import detect as detect_intent
from app.agent.messages import get_messages
from app.agent.output import format_for_whatsapp
from app.agent.state import ConvState, get_state, reset_state, save_state
from app.agent.tools import OPENAI_TOOLS, openai_tools_for_mode
from app.core.config import settings
from app.core.rabbitmq import publish
from app.llm.router import chat_with_tools
from app.rag.store import search as rag_search

log = structlog.get_logger(__name__)

# Idioma del asistente — configurar con WHATSAPP_LANGUAGE (es | en | pt). Default: es.
_LANG = os.environ.get("WHATSAPP_LANGUAGE", "es")

# Timeout para publicar "Un momento..." al usuario
_SLOW_RESPONSE_SECS = 5.0

# Tope de chars por mensaje de WhatsApp (post-formateo del LLM).
_WHATSAPP_MAX_CHARS = 400
# Delay entre chunks consecutivos al partir un mensaje largo en varios envíos.
# Respeta el rate limit de WhatsApp (1 msg/seg/sesión, CLAUDE.md §7).
_INTER_CHUNK_DELAY_SECS = 1.0


def _m(key: str, **kwargs: Any) -> str:
    """Obtiene el mensaje traducido para la clave dada, con sustitución de parámetros."""
    msg = get_messages(_LANG).get(key, key)
    return msg.format(**kwargs) if kwargs else msg


def _booking_url(tenant_slug: str) -> str:
    """Construye la URL pública de reserva del tenant."""
    return f"{settings.booking_base_url.rstrip('/')}/book/{tenant_slug}"


def _should_offer_link_proactively(
    profile: dict[str, Any] | None,
    state: dict[str, Any],
) -> tuple[bool, str]:
    """Decide si proactivamente conviene ofrecer el link de reserva.

    Centraliza dos triggers:
      - Regla B: el flujo lleva >=3 turnos sin que el usuario elija servicio.
      - Regla C: muchas opciones (>4 servicios y >3 profesionales).

    Returns:
        (offer, reason) donde reason ∈ {"many_options", "stalled_3_turns", ""}.
    """
    if not profile:
        return False, ""
    services = profile.get("services", [])
    professionals = profile.get("professionals", [])
    if len(services) > 4 and len(professionals) > 3:
        return True, "many_options"
    if state.get("booking_turns_without_service", 0) >= 3:
        return True, "stalled_3_turns"
    return False, ""


def _first_turn_greeting(profile: dict[str, Any]) -> str:
    """Construye el saludo determinista del primer turno con lista de servicios.

    Trunca la lista a 5 servicios máximo para respetar el límite de 400 chars
    de WhatsApp. Formato: '1. {name} — ${price} ({duration_min} min)'.
    """
    # Resolver bot_name y business_name por separado: bot_name es el nombre
    # del asistente (ej: "SarAI") y business_name es el nombre del negocio
    # (ej: "Clínica Dental"). NUNCA mezclarlos.
    bot_name = profile.get("bot_name")
    business_name = profile.get("name") or "el negocio"
    services = profile.get("services", [])[:5]
    lines = []
    for i, s in enumerate(services):
        currency = s.get("currency", "USD")
        price = s.get("price")
        price_str = f"${price} {currency}" if price else ""
        duration = s.get("duration_min", 0)
        if price_str:
            lines.append(f"{i+1}. {s['name']} — {price_str} ({duration} min)")
        else:
            lines.append(f"{i+1}. {s['name']} ({duration} min)")
    services_list = "\n".join(lines)
    # Si tenemos bot_name, el asistente se presenta por su nombre.
    # Si no, fallback al saludo genérico "asistente de {business_name}".
    template_key = "first_turn_greeting_with_bot_name" if bot_name else "first_turn_greeting"
    return _m(
        template_key,
        bot_name=bot_name or "",
        business_name=business_name,
        services_list=services_list,
    )


def _parse_flexible_date(text: str) -> str | None:
    """Parsea fechas en formatos naturales y retorna YYYY-MM-DD o None.

    Formatos aceptados:
      - YYYY-MM-DD  (2025-01-15)
      - DD-MM-YYYY  (15-01-2025)
      - DD/MM/YYYY  (15/01/2025)
      - DD-MM       (15-01 → asume año actual)
      - DD/MM       (15/01 → asume año actual)
    """
    text = text.strip()
    current_year = date.today().year

    # YYYY-MM-DD (formato ISO completo)
    m = re.search(r"\b(\d{4})-(\d{1,2})-(\d{1,2})\b", text)
    if m:
        try:
            d = date(int(m.group(1)), int(m.group(2)), int(m.group(3)))
            return d.isoformat()
        except ValueError:
            return None

    # DD-MM-YYYY o DD/MM/YYYY
    m = re.search(r"\b(\d{1,2})[/-](\d{1,2})[/-](\d{4})\b", text)
    if m:
        try:
            d = date(int(m.group(3)), int(m.group(2)), int(m.group(1)))
            return d.isoformat()
        except ValueError:
            return None

    # DD-MM o DD/MM (sin año → año actual)
    m = re.search(r"\b(\d{1,2})[/-](\d{1,2})\b", text)
    if m:
        try:
            d = date(current_year, int(m.group(2)), int(m.group(1)))
            return d.isoformat()
        except ValueError:
            return None

    return None


# Nombres de días y meses por idioma para formato amigable
_DAY_NAMES = {
    "es": ["lunes", "martes", "miércoles", "jueves", "viernes", "sábado", "domingo"],
    "en": ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"],
    "pt": ["segunda", "terça", "quarta", "quinta", "sexta", "sábado", "domingo"],
}
_MONTH_NAMES = {
    "es": ["", "enero", "febrero", "marzo", "abril", "mayo", "junio",
           "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"],
    "en": ["", "January", "February", "March", "April", "May", "June",
           "July", "August", "September", "October", "November", "December"],
    "pt": ["", "janeiro", "fevereiro", "março", "abril", "maio", "junho",
           "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"],
}


def _utc_to_local(iso_str: str, tz_name: str) -> datetime:
    """Convierte un ISO timestamp (UTC) a datetime en la zona horaria del tenant."""
    utc = datetime.fromisoformat(iso_str.replace("Z", "+00:00"))
    try:
        return utc.astimezone(ZoneInfo(tz_name))
    except (KeyError, ValueError):
        return utc


def _format_friendly_date(date_str: str) -> str:
    """Convierte 'YYYY-MM-DD' a formato amigable: 'jueves 15 de mayo'."""
    try:
        d = date.fromisoformat(date_str)
        lang = _LANG if _LANG in _DAY_NAMES else "es"
        day_name = _DAY_NAMES[lang][d.weekday()]
        month_name = _MONTH_NAMES[lang][d.month]
        if lang == "en":
            return f"{day_name}, {month_name} {d.day}"
        return f"{day_name} {d.day} de {month_name}"
    except ValueError:
        return date_str


async def _publish_reply(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    wa_phone: str,
    text: str | list[str],
) -> None:
    """Publica la respuesta en wa.messages.outbound.

    Acepta un único string o una lista de strings (chunks). Cuando recibe lista,
    publica cada chunk como mensaje WA separado con un pequeño delay entre cada
    uno para respetar el rate limit (1 msg/seg, CLAUDE.md §7).
    """
    chunks = text if isinstance(text, list) else [text]
    chunks = [c for c in chunks if c]
    if not chunks:
        return

    for i, chunk in enumerate(chunks):
        if i > 0:
            await asyncio.sleep(_INTER_CHUNK_DELAY_SECS)
        await publish(
            "wa.messages.outbound",
            {
                "tenant_id": tenant_id,
                "tenant_slug": tenant_slug,
                "conversation_id": conversation_id,
                "wa_phone": wa_phone,
                "content": chunk,
            },
        )


# Mapeo de tone a instrucciones de comunicación
_TONE_INSTRUCTIONS: dict[str, dict[str, str]] = {
    "friendly": {
        "es": "Usa un tono amigable y cercano. Puedes usar emojis con moderación. Tutea al cliente.",
        "en": "Use a friendly and warm tone. You can use emojis sparingly. Address the client informally.",
        "pt": "Use um tom amigável e próximo. Pode usar emojis com moderação. Trate o cliente por você.",
    },
    "professional": {
        "es": "Mantén un tono profesional y respetuoso. Evita emojis. Usa usted.",
        "en": "Maintain a professional and respectful tone. Avoid emojis. Use formal address.",
        "pt": "Mantenha um tom profissional e respeitoso. Evite emojis. Trate o cliente por senhor(a).",
    },
    "premium": {
        "es": "Usa un tono cálido pero elegante. Transmite exclusividad y cuidado personalizado.",
        "en": "Use a warm but elegant tone. Convey exclusivity and personalized care.",
        "pt": "Use um tom caloroso mas elegante. Transmita exclusividade e cuidado personalizado.",
    },
    "casual": {
        "es": "Sé directo y relajado. Puedes usar expresiones coloquiales. Tutea al cliente.",
        "en": "Be direct and relaxed. You can use colloquial expressions. Address the client casually.",
        "pt": "Seja direto e descontraído. Pode usar expressões coloquiais. Trate o cliente por você.",
    },
}


def _build_system_prompt(
    profile: dict[str, Any] | None,
    rag_context: str,
    tone: str | None = None,
    custom_instructions: str | None = None,
    is_first_turn: bool = True,
) -> str:
    """Construye el system prompt del asistente con contexto del negocio."""
    msgs = get_messages(_LANG)
    business_name = profile.get("name", "el negocio") if profile else "el negocio"
    bot_name = (profile.get("bot_name") or "el asistente virtual") if profile else "el asistente virtual"

    # Servicios con descripción y moneda
    services_text = ""
    if profile and profile.get("services"):
        lines = []
        for s in profile["services"]:
            currency = s.get("currency", "USD")
            price_str = f"${s['price']} {currency}" if s.get("price") else "consultar en cita"
            desc = f" — {s['description']}" if s.get("description") else ""
            lines.append(f"- {s['name']} ({price_str}, {s['duration_min']} min){desc}")
        services_text = f"\n\n{msgs['system_services_header']}\n" + "\n".join(lines)

    # Profesionales con especialidad y bio
    professionals_text = ""
    if profile and profile.get("professionals"):
        prof_lines = []
        for p in profile["professionals"]:
            line = f"- {p['name']}"
            if p.get("specialty"):
                line += f" ({p['specialty']})"
            if p.get("bio"):
                line += f" — {p['bio']}"
            prof_lines.append(line)
        professionals_text = f"\n\n{msgs['system_professionals_header']}\n" + "\n".join(prof_lines)

    # Mapeo servicio → profesionales
    mapping_text = ""
    if profile and profile.get("service_professionals") and profile.get("services") and profile.get("professionals"):
        svc_map = {s["id"]: s["name"] for s in profile["services"]}
        prof_map = {p["id"]: p["name"] for p in profile["professionals"]}
        svc_to_profs: dict[str, list[str]] = {}
        for link in profile["service_professionals"]:
            svc_name = svc_map.get(link["service_id"], "")
            prof_name = prof_map.get(link["professional_id"], "")
            if svc_name and prof_name:
                svc_to_profs.setdefault(svc_name, []).append(prof_name)
        if svc_to_profs:
            mapping_lines = [f"- {svc}: {', '.join(profs)}" for svc, profs in svc_to_profs.items()]
            mapping_text = f"\n\n{msgs['system_mapping_header']}\n" + "\n".join(mapping_lines)

    # Contexto del negocio
    business_context = ""
    if profile:
        parts = []
        if profile.get("business_type"):
            parts.append(profile["business_type"])
        if profile.get("city"):
            loc = profile["city"]
            if profile.get("country"):
                loc += f", {profile['country']}"
            parts.append(loc)
        if profile.get("description"):
            parts.append(profile["description"])
        if parts:
            business_context = f"\n\nSobre el negocio: {'. '.join(parts)}."

    rag_section = (
        f"\n\n{msgs['system_rag_header']}\n{rag_context}" if rag_context else ""
    )

    # Guard anti-hallucination: si no hay servicios cargados, inyectar instrucción dura
    # para que el LLM no invente catálogo basado en el rubro del negocio.
    no_catalog_section = ""
    if not profile or not profile.get("services"):
        no_catalog_section = f"\n\n{msgs['system_no_catalog']}"

    greeting_instruction = (
        msgs["greeting_first_turn"] if is_first_turn else msgs["greeting_continuation"]
    )
    intro = msgs["system_intro"].format(
        business_name=business_name,
        bot_name=bot_name,
        greeting_instruction=greeting_instruction,
    )
    warning = msgs["system_warning"]

    # Instrucciones de tono (desde chatbot_configs o default del system_intro)
    tone_section = ""
    if tone and tone in _TONE_INSTRUCTIONS:
        lang = _LANG if _LANG in _TONE_INSTRUCTIONS[tone] else "es"
        tone_section = f"\n\nEstilo de comunicación: {_TONE_INSTRUCTIONS[tone][lang]}"

    # Instrucciones personalizadas del negocio
    custom_section = ""
    if custom_instructions and custom_instructions.strip():
        custom_section = f"\n\nInstrucciones adicionales del negocio: {custom_instructions.strip()}"

    return f"{intro}{tone_section}{custom_section}{business_context}{services_text}{professionals_text}{mapping_text}{rag_section}{no_catalog_section}\n\n{warning}"


async def process_message(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    wa_phone: str,
    message_text: str,
    history: list[dict[str, Any]],
) -> str | None:
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

    Returns:
        El texto de la respuesta enviada al usuario (o None si no hubo respuesta,
        p. ej. estado HANDED_OFF). El caller puede usarlo para persistir el
        historial.
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
        _send_slow_response(tenant_id, tenant_slug, conversation_id, wa_phone, _SLOW_RESPONSE_SECS)
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
        # Post-procesar para WhatsApp: corta markdown no soportado y parte mensajes
        # largos (>400 chars) en varios chunks. Los templates de messages.py ya
        # son cortos y WA-safe → se devuelven como una sola lista de 1 elemento.
        chunks = format_for_whatsapp(response, max_chars=_WHATSAPP_MAX_CHARS)
        await _publish_reply(tenant_id, tenant_slug, conversation_id, wa_phone, chunks)
        # Devolver el texto unido para que el historial vea la respuesta completa
        # como un solo turno del asistente (más útil para el LLM en próximos turnos).
        return "\n\n".join(chunks)

    return response


async def _send_slow_response(
    tenant_id: str, tenant_slug: str, conversation_id: str, wa_phone: str, delay: float,
) -> None:
    """Envía el mensaje de espera si el procesamiento supera el timeout."""
    await asyncio.sleep(delay)
    await _publish_reply(tenant_id, tenant_slug, conversation_id, wa_phone, _m("slow_response"))


async def _handle_guard_hit(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    wa_phone: str,
    state: dict[str, Any],
    hit,  # GuardHit
) -> str | None:
    """
    Resuelve un mensaje que disparó un guard. Devuelve el texto de respuesta
    (o None si el bot no debe responder, p.ej. tras prompt injection).

    Reglas:
    - PROMPT_INJECTION → HANDED_OFF inmediato + notify staff + bot mudo (None).
    - OFFENSIVE        → HANDED_OFF + notify staff + mensaje de cierre.
    - OFF_TOPIC        → strike counter. 1er strike → redirect breve.
                         2do strike consecutivo → HANDED_OFF + notify + redirect handoff.
    """
    state["last_guard_hit_at"] = datetime.utcnow().isoformat()
    profile = await get_tenant_profile(tenant_slug)
    business_name = (profile or {}).get("name", "")

    if hit.kind == GuardKind.PROMPT_INJECTION:
        log.warning(
            "orchestrator.guard: prompt injection detected",
            tenant=tenant_id, conv=conversation_id, pattern=hit.pattern,
        )
        state["state"] = ConvState.HANDED_OFF
        await save_state(tenant_id, conversation_id, state)
        # Notificación al staff (best-effort, no await en caliente).
        asyncio.create_task(notify_handoff(
            tenant_slug, conversation_id, wa_phone,
            f"prompt_injection:{hit.pattern}",
        ))
        return None  # Bot mudo, no revelar el bypass.

    if hit.kind == GuardKind.OFFENSIVE:
        log.warning(
            "orchestrator.guard: offensive content",
            tenant=tenant_id, conv=conversation_id, pattern=hit.pattern,
        )
        state["state"] = ConvState.HANDED_OFF
        await save_state(tenant_id, conversation_id, state)
        asyncio.create_task(notify_handoff(
            tenant_slug, conversation_id, wa_phone,
            f"offensive:{hit.pattern}",
        ))
        return _m("guard_offensive_close")

    # OFF_TOPIC con strike counter.
    strikes = int(state.get("off_topic_strikes", 0)) + 1
    state["off_topic_strikes"] = strikes
    log.info(
        "orchestrator.guard: off_topic",
        tenant=tenant_id, conv=conversation_id,
        pattern=hit.pattern, strikes=strikes,
    )

    if strikes >= 2:
        state["state"] = ConvState.HANDED_OFF
        await save_state(tenant_id, conversation_id, state)
        asyncio.create_task(notify_handoff(
            tenant_slug, conversation_id, wa_phone,
            f"off_topic_strikes:{hit.pattern}",
        ))
        return _m("guard_offtopic_handoff")

    # 1er strike: redirect breve sin gastar LLM.
    await save_state(tenant_id, conversation_id, state)
    return _m("guard_offtopic_redirect", business_name=business_name)


async def _maybe_dental_triage(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    state: dict[str, Any],
) -> str | None:
    """
    Si el tenant es dental (business_type='dental') y aún no entramos al flujo
    de booking, transiciona a AWAITING_URGENCY_TRIAGE y devuelve la pregunta de
    triaje. En tenants no-dental devuelve None (no afecta el flujo normal).

    Migración 044 garantiza `dental_assistant_mode` con default 'in_chat', así
    que aunque sea dental el triaje SIEMPRE corre primero — el modo solo afecta
    qué se responde después del "no" en _handle_urgency_triage.
    """
    profile = await get_tenant_profile(tenant_slug)
    if not profile:
        return None
    if profile.get("business_type") != "dental":
        return None

    state["state"] = ConvState.AWAITING_URGENCY_TRIAGE
    await save_state(tenant_id, conversation_id, state)
    return _m("urgency_triage_prompt")


async def _handle_urgency_triage(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    state: dict[str, Any],
    wa_phone: str,
    intent: Intent,
) -> str | None:
    """
    Resuelve la respuesta del usuario al prompt de triaje dental.

    - CONFIRM (sí, hay urgencia) → mostrar teléfono / mensaje de urgencia + IDLE.
    - CANCEL (no, no es urgencia) → según `dental_assistant_mode`:
        - send_link → mandar link de booking + IDLE.
        - in_chat   → iniciar state machine normal de booking.
        - hybrid    → ofrecer link Y opción de seguir.
    - Cualquier otro intent → re-preguntar (mantener estado).
    """
    profile = await get_tenant_profile(tenant_slug) or {}
    mode = profile.get("dental_assistant_mode") or "in_chat"
    urgency_phone = profile.get("urgency_phone") or ""
    urgency_msg = profile.get("urgency_message") or ""

    # Sí → urgencia.
    if intent == Intent.CONFIRM:
        await reset_state(tenant_id, conversation_id)
        if urgency_msg:
            return urgency_msg
        if urgency_phone:
            return _m("urgency_response_yes_default", urgency_phone=urgency_phone)
        # Sin teléfono configurado: derivar a humano.
        state["state"] = ConvState.HANDED_OFF
        await save_state(tenant_id, conversation_id, state)
        asyncio.create_task(notify_handoff(
            tenant_slug, conversation_id, wa_phone, "urgency_no_phone_configured",
        ))
        return _m("urgency_response_yes_no_phone")

    # No → no es urgencia. Rutear según modo configurado.
    if intent == Intent.CANCEL:
        url = _booking_url(tenant_slug)
        if mode == "send_link":
            await reset_state(tenant_id, conversation_id)
            return _m("urgency_response_no_send_link", url=url)
        if mode == "hybrid":
            await reset_state(tenant_id, conversation_id)
            return _m("urgency_response_no_hybrid", url=url)
        # in_chat: pasar al flujo determinista normal.
        state["state"] = ConvState.IDLE
        await save_state(tenant_id, conversation_id, state)
        return await _start_booking_flow(
            tenant_id, tenant_slug, conversation_id, state, wa_phone,
        )

    # Intent ambiguo — re-preguntar.
    return _m("urgency_triage_prompt")


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

    # --- Guardrails Layer 1: injection / offensive / off-topic ---
    # Corren antes del intent classifier para no gastar tokens del LLM en
    # mensajes que igualmente vamos a rechazar/derivar.
    guard_hit = run_all_guards(message_text)
    if guard_hit is not None:
        return await _handle_guard_hit(
            tenant_id, tenant_slug, conversation_id, wa_phone,
            state, guard_hit,
        )

    # Mensaje on-topic — reset del contador de off-topic strikes.
    if state.get("off_topic_strikes", 0) > 0:
        state["off_topic_strikes"] = 0

    # --- Detectar intención (sesgado por el estado actual) ---
    pending_data = {
        "service_id": state.get("pending_service_id"),
        "professional_id": state.get("pending_professional_id"),
        "date": state.get("pending_date"),
        "selected_slot": state.get("pending_selected_slot"),
        "customer_name": state.get("customer_name"),
        "cancel_id": state.get("pending_cancel_id"),
        "reschedule_id": state.get("pending_reschedule_id"),
        "reschedule_date": state.get("pending_reschedule_date"),
    }
    intent = await detect_intent(
        message_text,
        history,
        conv_state=current_state,
        pending_data=pending_data,
    )
    log.info("orchestrator: intent detectado", intent=str(intent), state=str(current_state), text_len=len(message_text))

    # --- Escape hatch: HANDOFF y CANCEL desde cualquier estado de selección ---
    _SELECTION_STATES = {
        ConvState.AWAITING_CANCEL_SELECT,
        ConvState.AWAITING_RESCHEDULE_SELECT,
        ConvState.AWAITING_RESCHEDULE_DATE,
        ConvState.AWAITING_RESCHEDULE_SLOT,
    }
    if current_state in _SELECTION_STATES:
        if intent == Intent.HANDOFF:
            await reset_state(tenant_id, conversation_id)
            state["state"] = ConvState.HANDED_OFF
            await save_state(tenant_id, conversation_id, state)
            return _m("handoff")
        if intent == Intent.CANCEL:
            await reset_state(tenant_id, conversation_id)
            return _m("booking_cancelled")

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

    # === AWAITING_CANCEL_SELECT: eligiendo qué cita cancelar ===
    if current_state == ConvState.AWAITING_CANCEL_SELECT:
        return await _handle_cancel_select(message_text, tenant_id, conversation_id, state)

    # === AWAITING_CANCEL_CONFIRM: confirmando cancelación ===
    if current_state == ConvState.AWAITING_CANCEL_CONFIRM:
        if intent == Intent.CONFIRM:
            return await _execute_cancel(tenant_id, tenant_slug, conversation_id, state, wa_phone)
        if intent in (Intent.CANCEL, Intent.HANDOFF):
            await reset_state(tenant_id, conversation_id)
            if intent == Intent.HANDOFF:
                state["state"] = ConvState.HANDED_OFF
                await save_state(tenant_id, conversation_id, state)
                return _m("handoff")
            return _m("booking_cancelled")
        return _m("cancel_confirm", service=state.get("pending_cancel_service", ""), datetime=state.get("pending_cancel_datetime", ""))

    # === AWAITING_RESCHEDULE_SELECT: eligiendo qué cita reagendar ===
    if current_state == ConvState.AWAITING_RESCHEDULE_SELECT:
        return await _handle_reschedule_select(message_text, tenant_id, conversation_id, state)

    # === AWAITING_RESCHEDULE_DATE: ingresando nueva fecha ===
    if current_state == ConvState.AWAITING_RESCHEDULE_DATE:
        return await _handle_reschedule_date(message_text, tenant_id, tenant_slug, conversation_id, state)

    # === AWAITING_RESCHEDULE_SLOT: eligiendo nuevo horario ===
    if current_state == ConvState.AWAITING_RESCHEDULE_SLOT:
        return await _handle_reschedule_slot(message_text, tenant_id, conversation_id, state)

    # === AWAITING_RESCHEDULE_CONFIRM: confirmando reagendamiento ===
    if current_state == ConvState.AWAITING_RESCHEDULE_CONFIRM:
        if intent == Intent.CONFIRM:
            return await _execute_reschedule(tenant_id, tenant_slug, conversation_id, state, wa_phone)
        if intent in (Intent.CANCEL, Intent.HANDOFF):
            await reset_state(tenant_id, conversation_id)
            if intent == Intent.HANDOFF:
                state["state"] = ConvState.HANDED_OFF
                await save_state(tenant_id, conversation_id, state)
                return _m("handoff")
            return _m("booking_cancelled")
        return _m("reschedule_confirm", service=state.get("pending_reschedule_service", ""), datetime=state.get("pending_reschedule_datetime", ""))

    # === AWAITING_URGENCY_TRIAGE: triaje dental antes de iniciar booking ===
    if current_state == ConvState.AWAITING_URGENCY_TRIAGE:
        return await _handle_urgency_triage(
            tenant_id, tenant_slug, conversation_id, state, wa_phone, intent,
        )

    # === IDLE: flujos nuevos ===

    # Saludo determinista del primer turno (sin LLM) — predecible y barato.
    if (
        current_state == ConvState.IDLE
        and not history
        and intent in (Intent.UNKNOWN, Intent.QUERY)
    ):
        profile = await get_tenant_profile(tenant_slug)
        if profile and profile.get("services"):
            return _first_turn_greeting(profile)

    if intent == Intent.HANDOFF:
        state["state"] = ConvState.HANDED_OFF
        await save_state(tenant_id, conversation_id, state)
        return _m("handoff")

    # Regla A: el usuario pide el link explícitamente.
    if intent == Intent.SEND_BOOKING_LINK:
        log.info("orchestrator: link offered", reason="explicit_request", tenant=tenant_id)
        return _m("booking_link_explicit", url=_booking_url(tenant_slug))

    if intent == Intent.BOOKING:
        # Gate dental: si el tenant es dental, antes de iniciar el flujo
        # de booking corremos un triaje breve de urgencia (¿dolor fuerte?).
        # El profile lo cargamos lazy solo cuando hace falta.
        triage = await _maybe_dental_triage(
            tenant_id, tenant_slug, conversation_id, state,
        )
        if triage is not None:
            return triage
        return await _start_booking_flow(
            tenant_id, tenant_slug, conversation_id, state, wa_phone
        )

    if intent == Intent.MY_APPOINTMENTS:
        return await _handle_my_appointments(tenant_slug, wa_phone, state)

    if intent == Intent.RESCHEDULE:
        return await _start_reschedule_flow(tenant_id, tenant_slug, conversation_id, state, wa_phone)

    if intent == Intent.CANCEL:
        return await _start_cancel_flow(tenant_id, tenant_slug, conversation_id, state, wa_phone)

    if intent == Intent.QUERY:
        return await _handle_query(tenant_id, tenant_slug, message_text, history, wa_phone=wa_phone)

    # UNKNOWN — respuesta general con RAG
    return await _handle_query(tenant_id, tenant_slug, message_text, history, wa_phone=wa_phone)


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

    # Smart booking link: si hay muchas opciones, compartir link directo (Regla C).
    offer, reason = _should_offer_link_proactively(profile, state)
    if offer and reason == "many_options":
        log.info("orchestrator: link offered", reason=reason, tenant=tenant_id)
        return _m("booking_link", url=_booking_url(tenant_slug))

    lines = [f"{i+1}. {s['name']} — ${s['price']} USD ({s['duration_min']} min)"
             for i, s in enumerate(services)]

    state["state"] = ConvState.AWAITING_SLOT
    state["profile"] = profile
    state["pending_date"] = None
    state["booking_turns_without_service"] = 0
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
        # Contar turnos sin elegir servicio para Regla B (oferta suave del link).
        state["booking_turns_without_service"] = (
            state.get("booking_turns_without_service", 0) + 1
        )
        try:
            idx = int(message_text.strip()) - 1
            if idx < 0 or idx >= len(services):
                raise ValueError
            service = services[idx]
        except (ValueError, IndexError):
            lines = [f"{i+1}. {s['name']}" for i, s in enumerate(services)]
            reply = _m("invalid_option") + "\n" + "\n".join(lines)
            offer, reason = _should_offer_link_proactively(profile, state)
            if offer and reason == "stalled_3_turns":
                log.info("orchestrator: link offered", reason=reason)
                reply += "\n\n" + _m("booking_link_soft", url=_booking_url(profile.get("slug", "")))
            await save_state(tenant_id, conversation_id, state)
            return reply

        state["pending_service_id"] = service["id"]
        state["pending_service_name"] = service["name"]
        state["booking_turns_without_service"] = 0

        # Filtrar profesionales por los que efectivamente ofrecen el servicio elegido.
        # El mapping vive en profile["service_professionals"] como
        # [{"service_id": ..., "professional_id": ...}, ...].
        # Si el mapping no está cargado (tenant viejo / payload incompleto),
        # caemos al comportamiento previo: todos los profesionales del tenant.
        all_professionals = profile.get("professionals", [])
        svc_prof_links = profile.get("service_professionals") or []
        if svc_prof_links:
            eligible_prof_ids = {
                link.get("professional_id")
                for link in svc_prof_links
                if link.get("service_id") == service["id"]
            }
            professionals = [p for p in all_professionals if p.get("id") in eligible_prof_ids]
        else:
            professionals = all_professionals

        # Si después de filtrar no queda nadie elegible, avisar y NO crashear.
        if not professionals:
            return _m("no_professionals_for_service", service=service["name"])

        # Si hay más de un profesional elegible, preguntar
        if len(professionals) > 1:
            lines = [f"{i+1}. {p['name']}" for i, p in enumerate(professionals)]
            state["professionals"] = professionals
            await save_state(tenant_id, conversation_id, state)
            return _m("choose_professional", service=service["name"]) + "\n\n" + "\n".join(lines)
        else:
            state["pending_professional_id"] = professionals[0]["id"]
            state["pending_professional_name"] = professionals[0]["name"]

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
        date_str = _parse_flexible_date(message_text)
        if not date_str:
            return _m("invalid_date_format")
        slug = profile.get("slug", "")
        tz = profile.get("timezone", "UTC")
        slots = await get_availability(
            slug,
            state["pending_professional_id"],
            state["pending_service_id"],
            date_str,
            timezone=tz,
        )
        if not slots:
            return _m("no_slots", date=_format_friendly_date(date_str))

        # Mostrar todos los slots disponibles (ya filtrados por la API)
        shown_slots = slots
        state["pending_date"] = date_str
        state["pending_slots"] = shown_slots
        await save_state(tenant_id, conversation_id, state)

        friendly_date = _format_friendly_date(date_str)
        lines = []
        for i, s in enumerate(shown_slots):
            try:
                t = _utc_to_local(s["starts_at"], tz)
                lines.append(f"{i+1}. {t.strftime('%I:%M %p')}")
            except Exception:
                lines.append(f"{i+1}. {s['starts_at']}")

        return (
            _m("available_slots_header", date=friendly_date) + "\n\n"
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
    tz = state.get("profile", {}).get("timezone", "UTC")

    try:
        t = _utc_to_local(slot["starts_at"], tz)
        friendly = _format_friendly_date(t.strftime("%Y-%m-%d"))
        hora = f"{friendly} a las {t.strftime('%I:%M %p')}"
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
    tz = state.get("profile", {}).get("timezone", "UTC")
    try:
        t = _utc_to_local(slot["starts_at"], tz)
        friendly = _format_friendly_date(t.strftime("%Y-%m-%d"))
        datetime_str = f"{friendly} a las {t.strftime('%I:%M %p')}"
    except Exception:
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


def _format_kb_context(results: list[dict[str, Any]]) -> str:
    """Formatea chunks RAG como bloque legible para inyectar en el system prompt.

    Cada chunk se separa con un divisor y mantiene título + categoría para que el
    LLM entienda qué tipo de información tiene a mano. Cierre con una directiva
    fuerte: usar esta info ANTES de responder "no tengo".
    """
    if not results:
        return ""

    if _LANG == "en":
        header = "Relevant info from the business knowledge base:"
        footer = (
            "Use this information FIRST to answer. If the answer is here, "
            "do NOT invent and do NOT reply with 'I don't have that info'."
        )
    elif _LANG == "pt":
        header = "Informações relevantes da base de conhecimento do negócio:"
        footer = (
            "Use esta informação PRIMEIRO para responder. Se a resposta estiver aqui, "
            "NÃO invente nem responda com 'não tenho essa informação'."
        )
    else:
        header = "Información relevante de la base de conocimiento del negocio:"
        footer = (
            "Usa esta información PRIMERO para responder. Si la respuesta está aquí, "
            "NO inventes ni respondas con 'no tengo esa info'."
        )

    parts: list[str] = [header, "---"]
    for r in results:
        title = r.get("title") or "Documento"
        category = r.get("category") or "general"
        content = (r.get("content") or "").strip()
        # Encabezado de chunk: título + categoría entre paréntesis
        parts.append(f"[{title}] (categoría: {category})")
        parts.append(content)
        parts.append("---")
    parts.append(footer)
    return "\n".join(parts)


def _build_query_system_prompt(
    tenant_slug: str,
    is_first_turn: bool,
    kb_context: str = "",
) -> str:
    """System prompt corto y duro para el handler de QUERY con tool-calling.

    Sin catálogo inyectado: el LLM debe pedir info via tools (list_services,
    get_business_info, search_knowledge, etc.) y NUNCA inventar datos.

    Si `kb_context` no está vacío, se inyecta como bloque adicional al final
    del prompt para pre-cargar contexto RAG y evitar alucinaciones cuando el
    LLM no encadena tools por su cuenta.
    """
    msgs = get_messages(_LANG)
    greeting_rule = (
        msgs.get("greeting_first_turn", "First message — you may greet briefly.")
        if is_first_turn
        else msgs.get("greeting_continuation", "Mid-conversation — no greeting.")
    )

    # Bloque de KB opcional: si hay contexto pre-cargado del RAG, lo agregamos al final
    kb_block = f"\n\n{kb_context}\n" if kb_context else ""

    if _LANG == "en":
        return (
            f"You are the WhatsApp assistant for business `{tenant_slug}`.\n\n"
            f"{greeting_rule}\n\n"
            "CRITICAL RULES — DO NOT VIOLATE:\n"
            "- DIRECT ACTION: NEVER ask 'should I show you X?', 'do you want to see Y?', "
            "  'should I send you the catalog?'. If you have a tool to fetch that info "
            "  (list_services, get_business_info, search_knowledge), CALL IT IMMEDIATELY "
            "  and reply with the result. A permission question is A WASTED REPLY.\n"
            "- CONTINUITY: If your last reply ended with an open question like 'how can "
            "  I help?' or 'want to see the services?', assume ANY short affirmative "
            "  ('yes', 'ok', 'sure', 'go ahead') means 'yes, do what you offered'. DO NOT "
            "  re-ask. DO NOT return to the menu. EXECUTE the action — call the tool.\n"
            "- NO MENU MID-CONVERSATION: If the conversation already has more than 1 turn, "
            "  NEVER reply with a generic greeting like 'Hi! How can I help? Want to see "
            "  services, book...?'. That's for the first turn and is handled elsewhere. "
            "  Mid-conversation, answer the user's specific message or call the relevant tool.\n"
            "- 'are you there?' / 'hello?' mid-conversation: it means you dropped the "
            "  thread. Read the history: if your last reply was a question or offer, "
            "  EXECUTE the action you promised. If there was no active thread, reply "
            "  briefly 'I'm here, how can I help?' WITHOUT repeating the full menu.\n\n"
            "GENERAL RULES:\n"
            "- ONLY answer with information returned by the available tools.\n"
            "- If a tool doesn't return the data, say 'I don't have that info' "
            "  and offer to escalate to a human.\n"
            "- NEVER invent services, prices, hours, professionals, or policies.\n"
            "- WhatsApp style: max 2-3 short sentences. No long paragraphs.\n"
            "- Don't repeat the business name in every reply.\n"
            "- Tool guide: list_services for prices/catalog, get_business_info for "
            "  location/contact, search_knowledge for FAQs/policies, check_availability "
            "  only when the user asks about a specific date.\n"
            "- BOOKING: Only call book_appointment AFTER you (1) got a real slot "
            "  from check_availability, (2) have service_id and professional_id "
            "  from the list tools, (3) asked the user for their name, AND "
            "  (4) showed them the summary 'service + professional + date/time + name' "
            "  and got an EXPLICIT yes. If anything is missing, ask — don't book.\n"
            "- Don't use markdown headers (#). Don't use **double asterisks** — "
            "  WhatsApp uses *single* asterisks for bold.\n"
            "- Don't use numbered lists with more than 3 items. "
            "  Prefer inline 'A, B and C'.\n"
            "- Max 400 characters total. Be conversational, not formal.\n"
            f"{kb_block}"
        )
    if _LANG == "pt":
        return (
            f"Você é o assistente de WhatsApp do negócio `{tenant_slug}`.\n\n"
            f"{greeting_rule}\n\n"
            "REGRAS CRÍTICAS — NÃO VIOLE:\n"
            "- AÇÃO DIRETA: NUNCA pergunte 'quer que eu mostre X?', 'quer ver Y?', "
            "  'quer que eu envie o catálogo?'. Se você tem uma tool para buscar essa "
            "  info (list_services, get_business_info, search_knowledge), CHAME-A "
            "  IMEDIATAMENTE e responda com o resultado. Uma pergunta de permissão "
            "  é UMA RESPOSTA PERDIDA.\n"
            "- CONTINUIDADE: Se sua última resposta terminou com uma pergunta aberta "
            "  tipo 'como posso ajudar?' ou 'quer ver os serviços?', assuma que QUALQUER "
            "  afirmação curta ('sim', 'ok', 'pode', 'claro') significa 'sim, faça o que "
            "  você ofereceu'. NÃO repergunte. NÃO volte ao menu. EXECUTE a ação — chame "
            "  a tool.\n"
            "- SEM MENU NO MEIO DA CONVERSA: Se a conversa já tem mais de 1 turno, "
            "  NUNCA responda com saudação genérica tipo 'Olá! Como posso ajudar? Quer "
            "  ver serviços, agendar...?'. Isso é para o primeiro turno e é tratado "
            "  em outro lugar. No meio da conversa, responda à mensagem específica do "
            "  usuário ou chame a tool relevante.\n"
            "- 'oi?' / 'está aí?' no meio da conversa: significa que você perdeu o fio. "
            "  Leia o histórico: se sua última resposta era uma pergunta ou oferta, "
            "  EXECUTE a ação que prometeu. Se não havia fio ativo, responda brevemente "
            "  'Estou aqui, como posso ajudar?' SEM repetir o menu completo.\n\n"
            "REGRAS GERAIS:\n"
            "- Responda APENAS com informações obtidas pelas tools.\n"
            "- Se uma tool não retornar o dado, diga 'Não tenho essa informação' "
            "  e ofereça escalar para um humano.\n"
            "- NUNCA invente serviços, preços, horários, profissionais ou políticas.\n"
            "- Estilo WhatsApp: máximo 2-3 frases curtas. Sem parágrafos longos.\n"
            "- Não repita o nome do negócio em cada resposta.\n"
            "- Guia de tools: list_services para preços/catálogo, get_business_info "
            "  para localização/contato, search_knowledge para FAQs/políticas, "
            "  check_availability apenas quando o cliente perguntar por uma data específica.\n"
            "- RESERVAS: Só chame book_appointment DEPOIS de (1) obter um slot real "
            "  via check_availability, (2) ter service_id e professional_id das tools "
            "  de listagem, (3) pedir o nome ao cliente, E (4) mostrar o resumo "
            "  'serviço + profissional + data/hora + nome' e obter um SIM EXPLÍCITO. "
            "  Se faltar algo, pergunte — não reserve.\n"
            "- Não use cabeçalhos markdown (#). Não use **asteriscos duplos** — "
            "  WhatsApp usa *asterisco simples* para negrito.\n"
            "- Não use listas numeradas com mais de 3 itens. "
            "  Prefira 'A, B e C' em linha.\n"
            "- Máximo 400 caracteres no total. Seja conversacional, não formal.\n"
            f"{kb_block}"
        )
    # Default: español LATAM neutro (tú)
    return (
        f"Eres el asistente de WhatsApp del negocio `{tenant_slug}`.\n\n"
        f"{greeting_rule}\n\n"
        "REGLAS CRÍTICAS — NO VIOLES:\n"
        "- ACCIÓN DIRECTA: NUNCA preguntes '¿te muestro X?', '¿quieres ver Y?', "
        "  '¿te paso el catálogo?'. Si tienes una tool para traer esa info "
        "  (list_services, get_business_info, search_knowledge), LLÁMALA "
        "  INMEDIATAMENTE y responde con el resultado. Una pregunta de permiso "
        "  es UNA RESPUESTA PERDIDA.\n"
        "- CONTINUIDAD: Si tu último mensaje terminó con una pregunta abierta "
        "  tipo '¿en qué te ayudo?' o '¿quieres ver los servicios?', asume que "
        "  CUALQUIER respuesta afirmativa breve ('sí', 'si', 'ok', 'dale', 'claro') "
        "  significa 'sí, haz lo que ofreciste'. NO repreguntes. NO vuelvas al menú. "
        "  EJECUTA la acción — llama a la tool.\n"
        "- NO MENÚ MID-CONVERSACIÓN: Si la conversación ya tiene más de 1 turno, "
        "  NUNCA respondas con un saludo genérico tipo '¡Hola! ¿En qué te ayudo? "
        "  ¿Quieres ver servicios, agendar...?'. Eso es para el primer turno y "
        "  lo maneja otro código. En medio de conversación, responde al mensaje "
        "  específico del usuario o llama a la tool relevante.\n"
        "- '¿hola?' / '¿estás ahí?' en medio de conversación: significa que "
        "  perdiste el hilo. Mira el historial: si tu último mensaje era una "
        "  pregunta o una oferta, EJECUTA la acción que prometiste. Si no había "
        "  hilo activo, responde brevemente 'Acá estoy, ¿en qué te ayudo?' SIN "
        "  repetir el menú completo.\n\n"
        "REGLAS GENERALES:\n"
        "- Solo responde con información que obtengas de las tools.\n"
        "- Si una tool no devuelve el dato, di 'No tengo esa información' "
        "  y ofrece escalar a un humano.\n"
        "- NUNCA inventes servicios, precios, horarios, profesionales ni políticas.\n"
        "- Estilo WhatsApp: máximo 2-3 oraciones cortas. Sin párrafos largos.\n"
        "- No repitas el nombre del negocio en cada respuesta.\n"
        "- Guía de tools: list_services para precios/catálogo, get_business_info "
        "  para ubicación/contacto, search_knowledge para FAQs/políticas, "
        "  check_availability solo cuando el cliente pregunte por una fecha concreta.\n"
        "- RESERVAS: Solo llama a book_appointment DESPUÉS de (1) obtener un slot real "
        "  vía check_availability, (2) tener service_id y professional_id de las tools "
        "  de listado, (3) preguntar el nombre del cliente, Y (4) mostrarle el resumen "
        "  'servicio + profesional + fecha/hora + nombre' y obtener un SÍ EXPLÍCITO. "
        "  Si falta algo, pregunta — no reserves todavía.\n"
        "- Si book_appointment devuelve 'slot_no_longer_available', pide disculpas "
        "  y llama de nuevo a check_availability para ofrecer otro horario.\n"
        "- No uses encabezados markdown (#). No uses **asteriscos dobles** — "
        "  WhatsApp usa *asterisco simple* para negrita.\n"
        "- No uses listas numeradas con más de 3 ítems. "
        "  Prefiere 'A, B y C' en línea.\n"
        "- Máximo 400 caracteres en total. Sé conversacional, no formal.\n"
        f"{kb_block}"
    )


async def _handle_query(
    tenant_id: str,
    tenant_slug: str,
    message_text: str,
    history: list[dict[str, Any]],
    wa_phone: str = "",
) -> str:
    """Responde preguntas vía tool-calling: el LLM decide qué tools invocar.

    Anti-hallucination: NO inyectamos catálogo en el prompt. Si el LLM necesita
    datos del tenant los pide via list_services / get_business_info /
    search_knowledge / check_availability.

    Pre-carga RAG: además, hacemos un search en la knowledge base ANTES del
    LLM y le pasamos los chunks relevantes en el system prompt. Esto evita
    el patrón observado donde DeepSeek invoca una sola tool y no encadena
    con search_knowledge. La tool sigue disponible por si quiere refinar.
    """
    is_first_turn = len(history) == 0

    # Pre-cargar contexto del RAG: hacemos best-effort. Si falla, seguimos sin KB.
    kb_context_block = ""
    try:
        kb_results = await rag_search(tenant_id, message_text)
        if kb_results:
            kb_context_block = _format_kb_context(kb_results)
            log.info(
                "orchestrator._handle_query: kb_context inyectado",
                chunks=len(kb_results),
                tenant=tenant_id,
            )
        else:
            log.info(
                "orchestrator._handle_query: kb sin matches",
                tenant=tenant_id,
            )
    except Exception as e:
        # No rompemos el flujo: si el RAG falla, el LLM aún tiene las tools.
        log.warning(
            "orchestrator._handle_query: kb search falló, continuando sin contexto",
            error=str(e),
            tenant=tenant_id,
        )

    system = _build_query_system_prompt(tenant_slug, is_first_turn, kb_context_block)
    messages: list[dict[str, Any]] = [{"role": "system", "content": system}]

    for h in history[-8:]:
        messages.append({"role": h["role"], "content": h["content"]})
    messages.append({"role": "user", "content": message_text})

    # En tenants dentales con modo 'send_link' desregistramos las tools que
    # agendan in-chat (check_availability + book_appointment): el LLM solo
    # informa y deriva al link de booking. Para no-dental o modos in_chat/hybrid
    # se devuelve OPENAI_TOOLS completo.
    profile_for_mode = await get_tenant_profile(tenant_slug) or {}
    dental_mode = profile_for_mode.get("dental_assistant_mode")
    tools_for_call = openai_tools_for_mode(dental_mode)

    try:
        result = await chat_with_tools(
            messages=messages,
            tools=tools_for_call,
            # customer_phone se inyecta para que la tool book_appointment
            # pueda reservar sin que el LLM lo controle. Si llega vacío
            # (p.ej. tests), la tool fallará con "missing_customer_phone".
            ctx={
                "tenant_slug": tenant_slug,
                "tenant_id": tenant_id,
                "customer_phone": wa_phone,
            },
            temperature=0.2,
            max_tokens=800,
            max_iterations=3,
        )
        # Si el LLM devuelve respuesta vacía o muy corta, ofrecer el link como fallback.
        if not result or len(result.strip()) < 5:
            log.info("orchestrator: link offered", reason="empty_tool_response", tenant=tenant_id)
            return _m("booking_link_soft", url=_booking_url(tenant_slug))
        return result
    except Exception as e:
        log.error("orchestrator._handle_query: tool-calling falló", error=str(e))
        return _m("booking_link_soft", url=_booking_url(tenant_slug))


# ---------------------------------------------------------------------------
# Helpers para appointment lifecycle
# ---------------------------------------------------------------------------


def _format_appointment_line(idx: int, appt: dict[str, Any], tz: str) -> str:
    """Formatea una línea de cita para mostrar al usuario."""
    service = appt.get("service_name", "Servicio")
    professional = appt.get("professional_name", "")
    try:
        t = _utc_to_local(appt["starts_at"], tz)
        friendly = _format_friendly_date(t.strftime("%Y-%m-%d"))
        datetime_str = f"{friendly} a las {t.strftime('%I:%M %p')}"
    except Exception:
        datetime_str = appt.get("starts_at", "")
    return _m("appointment_line", idx=idx, service=service, datetime=datetime_str, professional=professional)


async def _fetch_appointments_with_profile(
    tenant_slug: str, wa_phone: str, state: dict[str, Any]
) -> tuple[list[dict[str, Any]], str]:
    """Obtiene citas y timezone del perfil. Retorna (appointments, timezone)."""
    profile = state.get("profile") or await get_tenant_profile(tenant_slug)
    tz = (profile or {}).get("timezone", "UTC")
    appointments = await get_my_appointments(tenant_slug, wa_phone)
    return appointments, tz


async def _handle_my_appointments(
    tenant_slug: str, wa_phone: str, state: dict[str, Any]
) -> str:
    """Muestra las próximas citas del cliente."""
    appointments, tz = await _fetch_appointments_with_profile(tenant_slug, wa_phone, state)
    if not appointments:
        return _m("no_appointments")
    lines = [_format_appointment_line(i + 1, a, tz) for i, a in enumerate(appointments)]
    return _m("my_appointments_header") + "\n\n" + "\n".join(lines)


# ---------------------------------------------------------------------------
# Flujo de cancelación
# ---------------------------------------------------------------------------


async def _start_cancel_flow(
    tenant_id: str, tenant_slug: str, conversation_id: str,
    state: dict[str, Any], wa_phone: str,
) -> str:
    """Inicia flujo de cancelación: busca citas y pide selección."""
    appointments, tz = await _fetch_appointments_with_profile(tenant_slug, wa_phone, state)
    if not appointments:
        return _m("no_appointments_cancel")

    state["pending_appointments"] = appointments
    state["profile_tz"] = tz

    if len(appointments) == 1:
        # Auto-seleccionar la única cita
        appt = appointments[0]
        try:
            t = _utc_to_local(appt["starts_at"], tz)
            friendly = _format_friendly_date(t.strftime("%Y-%m-%d"))
            dt_str = f"{friendly} a las {t.strftime('%I:%M %p')}"
        except Exception:
            dt_str = appt.get("starts_at", "")
        state["pending_cancel_id"] = appt["id"]
        state["pending_cancel_service"] = appt.get("service_name", "")
        state["pending_cancel_datetime"] = dt_str
        state["state"] = ConvState.AWAITING_CANCEL_CONFIRM
        await save_state(tenant_id, conversation_id, state)
        return _m("cancel_confirm", service=appt.get("service_name", ""), datetime=dt_str)

    # Múltiples citas: mostrar lista
    lines = [_format_appointment_line(i + 1, a, tz) for i, a in enumerate(appointments)]
    state["state"] = ConvState.AWAITING_CANCEL_SELECT
    await save_state(tenant_id, conversation_id, state)
    return _m("cancel_which") + "\n\n" + "\n".join(lines)


async def _handle_cancel_select(
    message_text: str, tenant_id: str, conversation_id: str, state: dict[str, Any]
) -> str:
    """Usuario selecciona qué cita cancelar."""
    appointments = state.get("pending_appointments", [])
    tz = state.get("profile_tz", "UTC")
    try:
        idx = int(message_text.strip()) - 1
        appt = appointments[idx]
    except (ValueError, IndexError):
        return _m("invalid_option") + "\n" + "\n".join(
            _format_appointment_line(i + 1, a, tz) for i, a in enumerate(appointments)
        )

    try:
        t = _utc_to_local(appt["starts_at"], tz)
        friendly = _format_friendly_date(t.strftime("%Y-%m-%d"))
        dt_str = f"{friendly} a las {t.strftime('%I:%M %p')}"
    except Exception:
        dt_str = appt.get("starts_at", "")

    state["pending_cancel_id"] = appt["id"]
    state["pending_cancel_service"] = appt.get("service_name", "")
    state["pending_cancel_datetime"] = dt_str
    state["state"] = ConvState.AWAITING_CANCEL_CONFIRM
    await save_state(tenant_id, conversation_id, state)
    return _m("cancel_confirm", service=appt.get("service_name", ""), datetime=dt_str)


async def _execute_cancel(
    tenant_id: str, tenant_slug: str, conversation_id: str,
    state: dict[str, Any], wa_phone: str,
) -> str:
    """Ejecuta la cancelación de la cita seleccionada."""
    appt_id = state.get("pending_cancel_id")
    ok = await cancel_appointment(tenant_slug, appt_id, wa_phone)
    if ok:
        await reset_state(tenant_id, conversation_id)
        return _m("cancel_success")
    return _m("cancel_failed")


# ---------------------------------------------------------------------------
# Flujo de reagendamiento
# ---------------------------------------------------------------------------


async def _start_reschedule_flow(
    tenant_id: str, tenant_slug: str, conversation_id: str,
    state: dict[str, Any], wa_phone: str,
) -> str:
    """Inicia flujo de reagendamiento: busca citas y pide selección."""
    appointments, tz = await _fetch_appointments_with_profile(tenant_slug, wa_phone, state)
    if not appointments:
        return _m("no_appointments_reschedule")

    profile = state.get("profile") or await get_tenant_profile(tenant_slug)
    state["pending_appointments"] = appointments
    state["profile_tz"] = tz
    state["profile"] = profile

    if len(appointments) == 1:
        appt = appointments[0]
        state["pending_reschedule_id"] = appt["id"]
        state["pending_reschedule_service"] = appt.get("service_name", "")
        state["pending_reschedule_professional_id"] = appt.get("professional_id", "")
        state["pending_reschedule_service_id"] = appt.get("service_id", "")
        state["state"] = ConvState.AWAITING_RESCHEDULE_DATE
        await save_state(tenant_id, conversation_id, state)
        return _m("reschedule_date", service=appt.get("service_name", ""))

    lines = [_format_appointment_line(i + 1, a, tz) for i, a in enumerate(appointments)]
    state["state"] = ConvState.AWAITING_RESCHEDULE_SELECT
    await save_state(tenant_id, conversation_id, state)
    return _m("reschedule_which") + "\n\n" + "\n".join(lines)


async def _handle_reschedule_select(
    message_text: str, tenant_id: str, conversation_id: str, state: dict[str, Any]
) -> str:
    """Usuario selecciona qué cita reagendar."""
    appointments = state.get("pending_appointments", [])
    tz = state.get("profile_tz", "UTC")
    try:
        idx = int(message_text.strip()) - 1
        appt = appointments[idx]
    except (ValueError, IndexError):
        return _m("invalid_option") + "\n" + "\n".join(
            _format_appointment_line(i + 1, a, tz) for i, a in enumerate(appointments)
        )

    state["pending_reschedule_id"] = appt["id"]
    state["pending_reschedule_service"] = appt.get("service_name", "")
    state["pending_reschedule_professional_id"] = appt.get("professional_id", "")
    state["pending_reschedule_service_id"] = appt.get("service_id", "")
    state["state"] = ConvState.AWAITING_RESCHEDULE_DATE
    await save_state(tenant_id, conversation_id, state)
    return _m("reschedule_date", service=appt.get("service_name", ""))


async def _handle_reschedule_date(
    message_text: str, tenant_id: str, tenant_slug: str,
    conversation_id: str, state: dict[str, Any]
) -> str:
    """Usuario ingresa nueva fecha para reagendamiento."""
    date_str = _parse_flexible_date(message_text)
    if not date_str:
        return _m("invalid_date_format")

    tz = state.get("profile_tz", "UTC")
    prof_id = state.get("pending_reschedule_professional_id", "")
    svc_id = state.get("pending_reschedule_service_id", "")

    slots = await get_availability(tenant_slug, prof_id, svc_id, date_str, timezone=tz)
    if not slots:
        return _m("no_slots", date=_format_friendly_date(date_str))

    state["pending_reschedule_date"] = date_str
    state["pending_reschedule_slots"] = slots
    state["state"] = ConvState.AWAITING_RESCHEDULE_SLOT
    await save_state(tenant_id, conversation_id, state)

    friendly_date = _format_friendly_date(date_str)
    lines = []
    for i, s in enumerate(slots):
        try:
            t = _utc_to_local(s["starts_at"], tz)
            lines.append(f"{i+1}. {t.strftime('%I:%M %p')}")
        except Exception:
            lines.append(f"{i+1}. {s['starts_at']}")

    return (
        _m("available_slots_header", date=friendly_date) + "\n\n"
        + "\n".join(lines)
        + "\n\n" + _m("available_slots_footer")
    )


async def _handle_reschedule_slot(
    message_text: str, tenant_id: str, conversation_id: str, state: dict[str, Any]
) -> str:
    """Usuario elige nuevo horario para reagendamiento."""
    slots = state.get("pending_reschedule_slots", [])
    tz = state.get("profile_tz", "UTC")
    try:
        idx = int(message_text.strip()) - 1
        slot = slots[idx]
    except (ValueError, IndexError):
        return _m("invalid_slot", max=len(slots))

    try:
        t = _utc_to_local(slot["starts_at"], tz)
        friendly = _format_friendly_date(t.strftime("%Y-%m-%d"))
        dt_str = f"{friendly} a las {t.strftime('%I:%M %p')}"
    except Exception:
        dt_str = slot.get("starts_at", "")

    state["pending_reschedule_slot"] = slot
    state["pending_reschedule_datetime"] = dt_str
    state["state"] = ConvState.AWAITING_RESCHEDULE_CONFIRM
    await save_state(tenant_id, conversation_id, state)
    return _m("reschedule_confirm", service=state.get("pending_reschedule_service", ""), datetime=dt_str)


async def _execute_reschedule(
    tenant_id: str, tenant_slug: str, conversation_id: str,
    state: dict[str, Any], wa_phone: str,
) -> str:
    """Ejecuta el reagendamiento de la cita."""
    appt_id = state.get("pending_reschedule_id")
    slot = state.get("pending_reschedule_slot", {})
    dt_str = state.get("pending_reschedule_datetime", "")

    ok = await reschedule_appointment(tenant_slug, appt_id, wa_phone, slot["starts_at"])
    if ok:
        await reset_state(tenant_id, conversation_id)
        return _m("reschedule_success", datetime=dt_str)
    return _m("reschedule_failed")
