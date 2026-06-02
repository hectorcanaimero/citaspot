# Tool registry para function-calling del agente IA.
# Cada herramienta envuelve una acción de `app.agent.actions` y expone:
#   - una función async que devuelve dict {"ok": bool, "data": ...} o {"ok": False, "error": ...}
#   - un schema JSON compatible con OpenAI tools / Gemini function declarations
#
# El LLM nunca debe inventar servicios, precios u horarios: si necesita info
# del catálogo del tenant, debe pedirla vía estas tools.
from __future__ import annotations

import re
from contextvars import ContextVar
from dataclasses import dataclass
from datetime import datetime
from typing import Any, Awaitable, Callable

import structlog

from app.agent.actions import (
    book_appointment,
    get_availability,
    get_tenant_profile,
    rag_query,
)

log = structlog.get_logger(__name__)

# ---------------------------------------------------------------------------
# Tool-call tracking (opt-in, isolated per task via ContextVar)
# ---------------------------------------------------------------------------
# Usado por el endpoint /process-test para devolver al dashboard la lista de
# tools que el LLM invocó. NUNCA se activa por defecto en producción para
# evitar overhead. El consumer/inbound worker ignora estas funciones.
_tools_called: ContextVar[list[str] | None] = ContextVar("_tools_called", default=None)


def start_tracking() -> None:
    """Habilita el tracking de tools en el contexto actual.

    Reemplaza la lista anterior (si la había) por una nueva vacía. Es seguro
    llamarlo múltiples veces. Las llamadas posteriores a `execute_tool`
    registrarán el nombre de la tool en esta lista.
    """
    _tools_called.set([])


def get_tracked() -> list[str]:
    """Devuelve los nombres de las tools invocadas desde la última `start_tracking`.

    Si no hay tracking activo, retorna lista vacía. Devuelve una copia para
    que el caller no mute el estado interno.
    """
    tracked = _tools_called.get()
    if tracked is None:
        return []
    return list(tracked)

# ---------------------------------------------------------------------------
# Tipos del registro
# ---------------------------------------------------------------------------

ToolFn = Callable[..., Awaitable[dict[str, Any]]]


@dataclass(frozen=True)
class ToolSpec:
    """Especificación de una tool: nombre, callable y schemas para cada provider."""

    name: str
    description: str
    fn: ToolFn
    # Schema JSON crudo del parámetro `parameters` (formato OpenAI)
    parameters: dict[str, Any]
    # Argumentos del ctx (no del LLM) que se inyectan al ejecutar
    context_args: tuple[str, ...] = ()

    def openai_schema(self) -> dict[str, Any]:
        """Formato OpenAI tools: {type, function: {name, description, parameters}}."""
        return {
            "type": "function",
            "function": {
                "name": self.name,
                "description": self.description,
                "parameters": self.parameters,
            },
        }

    def gemini_schema(self) -> dict[str, Any]:
        """Formato Gemini function_declaration: {name, description, parameters}."""
        return {
            "name": self.name,
            "description": self.description,
            "parameters": self.parameters,
        }


# ---------------------------------------------------------------------------
# Helpers internos
# ---------------------------------------------------------------------------


def _ok(data: Any) -> dict[str, Any]:
    return {"ok": True, "data": data}


def _err(msg: str) -> dict[str, Any]:
    return {"ok": False, "error": msg}


async def _load_profile(tenant_slug: str) -> dict[str, Any] | None:
    """Carga el perfil del tenant. Cero PII en logs (slug es público)."""
    try:
        return await get_tenant_profile(tenant_slug)
    except Exception as e:
        log.error("tools._load_profile: error cargando perfil", error=str(e))
        return None


# ---------------------------------------------------------------------------
# Implementaciones de tools
# ---------------------------------------------------------------------------


async def _tool_list_services(tenant_slug: str) -> dict[str, Any]:
    """Lista servicios activos del negocio: id, name, price, duration_min, description."""
    profile = await _load_profile(tenant_slug)
    if not profile:
        return _err("no_profile")
    services = profile.get("services") or []
    out = [
        {
            "id": s.get("id"),
            "name": s.get("name"),
            "price": s.get("price"),
            "currency": s.get("currency", "USD"),
            "duration_min": s.get("duration_min"),
            "description": s.get("description") or "",
        }
        for s in services
    ]
    return _ok(out)


async def _tool_list_professionals(tenant_slug: str) -> dict[str, Any]:
    """Lista profesionales/personal del negocio: id, name, specialty, bio."""
    profile = await _load_profile(tenant_slug)
    if not profile:
        return _err("no_profile")
    pros = profile.get("professionals") or []
    out = [
        {
            "id": p.get("id"),
            "name": p.get("name"),
            "specialty": p.get("specialty") or "",
            "bio": p.get("bio") or "",
        }
        for p in pros
    ]
    return _ok(out)


async def _tool_get_service_professionals(
    tenant_slug: str, service_id: str
) -> dict[str, Any]:
    """Devuelve los profesionales que ofrecen un servicio específico."""
    profile = await _load_profile(tenant_slug)
    if not profile:
        return _err("no_profile")

    prof_map = {p["id"]: p for p in (profile.get("professionals") or []) if p.get("id")}
    links = profile.get("service_professionals") or []
    out = []
    for link in links:
        if link.get("service_id") != service_id:
            continue
        p = prof_map.get(link.get("professional_id"))
        if not p:
            continue
        out.append({
            "id": p.get("id"),
            "name": p.get("name"),
            "specialty": p.get("specialty") or "",
            "bio": p.get("bio") or "",
        })
    return _ok(out)


async def _tool_check_availability(
    tenant_slug: str,
    date: str,
    service_id: str | None = None,
    professional_id: str | None = None,
) -> dict[str, Any]:
    """Devuelve slots disponibles para fecha/servicio/profesional.

    Si no se especifica `professional_id`, intenta usar el primer profesional
    asociado al servicio. Si tampoco se especifica `service_id`, retorna error.
    """
    if not service_id:
        return _err("missing_service_id")

    profile = await _load_profile(tenant_slug)
    if not profile:
        return _err("no_profile")
    tz = profile.get("timezone", "UTC")

    # Resolver profesional si no vino
    resolved_prof = professional_id
    if not resolved_prof:
        prof_ids = [
            link.get("professional_id")
            for link in (profile.get("service_professionals") or [])
            if link.get("service_id") == service_id and link.get("professional_id")
        ]
        if prof_ids:
            resolved_prof = prof_ids[0]
        else:
            pros = profile.get("professionals") or []
            if pros:
                resolved_prof = pros[0].get("id")
    if not resolved_prof:
        return _err("no_professional_available")

    try:
        slots = await get_availability(
            tenant_slug, resolved_prof, service_id, date, timezone=tz,
        )
    except Exception as e:
        log.error("tools.check_availability: error", error=str(e))
        return _err("availability_failed")

    return _ok({
        "date": date,
        "professional_id": resolved_prof,
        "timezone": tz,
        "slots": slots,
    })


async def _tool_search_knowledge(tenant_id: str, query: str) -> dict[str, Any]:
    """Busca en la base de conocimiento del tenant via RAG."""
    if not query or not query.strip():
        return _err("empty_query")
    try:
        text = await rag_query(tenant_id, query)
    except Exception as e:
        log.error("tools.search_knowledge: error", error=str(e))
        return _err("rag_failed")
    return _ok({"context": text or ""})


async def _tool_get_business_info(tenant_slug: str) -> dict[str, Any]:
    """Devuelve info pública del negocio: nombre, tipo, ubicación, descripción, contacto."""
    profile = await _load_profile(tenant_slug)
    if not profile:
        return _err("no_profile")
    return _ok({
        "name": profile.get("name") or "",
        "business_type": profile.get("business_type") or "",
        "city": profile.get("city") or "",
        "country": profile.get("country") or "",
        "timezone": profile.get("timezone") or "UTC",
        "description": profile.get("description") or "",
        "phone": profile.get("phone") or "",
        "email": profile.get("email") or "",
        "address": profile.get("address") or "",
        "bot_name": profile.get("bot_name") or "",
    })


async def _tool_list_my_treatments(tenant_slug: str, phone: str) -> dict[str, Any]:
    """Lista tratamientos activos del cliente identificado por teléfono (Plane #34).
    Solo funciona para tenants con módulo dental activo — devuelve no_dental_module
    si el backend retorna 403.
    """
    if not phone or not phone.strip():
        return _err("empty_phone")
    from .actions import get_my_treatments
    data = await get_my_treatments(tenant_slug, phone.strip())
    if data is None:
        return _err("no_dental_module")
    return _ok(data)


# Regex de UUID v4 (case-insensitive). Validamos antes de llamar al endpoint
# para no quemar una llamada HTTP si el LLM alucinó un id mal formado.
_UUID_RE = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
    re.IGNORECASE,
)


def _is_valid_iso8601(s: str) -> bool:
    """True si `s` es ISO 8601 parseable por datetime.fromisoformat.

    Acepta el sufijo 'Z' (UTC) que fromisoformat históricamente no soportaba
    antes de Python 3.11, normalizándolo a '+00:00'. Rechaza strings sin info
    de zona horaria — el endpoint Core espera timestamp con timezone.
    """
    if not s or not isinstance(s, str):
        return False
    try:
        normalized = s.replace("Z", "+00:00") if s.endswith("Z") else s
        dt = datetime.fromisoformat(normalized)
    except ValueError:
        return False
    # Exigir tzinfo: starts_at sin zona horaria es ambiguo y produce
    # comportamiento inconsistente entre tenants en distintos timezones.
    return dt.tzinfo is not None


async def _tool_book_appointment(
    tenant_slug: str,
    customer_phone: str,
    professional_id: str,
    service_id: str,
    starts_at: str,
    customer_name: str,
) -> dict[str, Any]:
    """Crea una cita vía el endpoint público del Core API.

    Diseñada para que el LLM la invoque en el handler QUERY UNA VEZ que ya
    confirmó verbalmente con el usuario los 4 datos (servicio, profesional,
    slot, nombre). El LLM NO controla customer_phone ni tenant_slug — esos
    se inyectan desde el contexto del runtime.

    Validaciones defensivas:
      - UUIDs bien formados (regex) → evita una llamada HTTP fallida.
      - starts_at ISO 8601 con timezone → el endpoint Core lo exige.
      - customer_name no vacío → el endpoint también lo requiere.

    Mapeo de errores HTTP a strings que el LLM puede razonar:
      - 409 conflict → "slot_no_longer_available" (sugerir otro horario).
      - 400/422 → "invalid_data" (revisar inputs antes de reintentar).
      - 404 → "tenant_not_found" (no debería ocurrir si el slug viene del ctx).
      - 0/network → "network_error" (best-effort retry o handoff).

    Returns:
        dict {"ok": True, "data": {"appointment_id": str, "starts_at": str,
              "status": str}} en éxito.
        dict {"ok": False, "error": <code>} en fallo. Nunca lanza excepción.
    """
    # Validaciones de input — fallar rápido sin llamar al API
    if not _UUID_RE.match(professional_id or ""):
        return _err("invalid_professional_id")
    if not _UUID_RE.match(service_id or ""):
        return _err("invalid_service_id")
    if not _is_valid_iso8601(starts_at):
        return _err("invalid_starts_at_format")
    if not customer_name or not customer_name.strip():
        return _err("missing_customer_name")
    if not customer_phone or not customer_phone.strip():
        return _err("missing_customer_phone")

    # Loguear sin PII: nada de nombre/teléfono en logs
    log.info(
        "tools.book_appointment: invocando action",
        tenant_slug=tenant_slug,
        service_id=service_id,
        professional_id=professional_id,
    )

    result = await book_appointment(
        slug=tenant_slug,
        professional_id=professional_id,
        service_id=service_id,
        starts_at=starts_at,
        customer_name=customer_name.strip(),
        customer_phone=customer_phone.strip(),
        source="whatsapp",
        return_error_details=True,
    )

    if result is None:
        # No debería ocurrir con return_error_details=True, pero por seguridad.
        return _err("booking_failed")

    if result.get("_error"):
        status = result.get("status", 0)
        if status == 409:
            return _err("slot_no_longer_available")
        if status in (400, 422):
            return _err("invalid_data")
        if status == 404:
            return _err("tenant_not_found")
        if status == 0:
            return _err("network_error")
        return _err(f"booking_failed_http_{status}")

    # Éxito — devolver al LLM solo lo esencial para confirmar al usuario.
    # No filtramos `result` completo para no exponer campos internos por error.
    return _ok({
        "appointment_id": result.get("id") or result.get("appointment_id"),
        "starts_at": result.get("starts_at"),
        "ends_at": result.get("ends_at"),
        "status": result.get("status"),
    })


# ---------------------------------------------------------------------------
# Schemas (parameters) — JSON Schema subset común OpenAI/Gemini
# ---------------------------------------------------------------------------

_TENANT_SLUG_PROP = {
    "type": "string",
    "description": "Slug interno del tenant. SIEMPRE inyectado desde el contexto, no inventar.",
}

_LIST_SERVICES_PARAMS = {
    "type": "object",
    "properties": {},
    "required": [],
}

_LIST_PROFS_PARAMS = {
    "type": "object",
    "properties": {},
    "required": [],
}

_SVC_PROFS_PARAMS = {
    "type": "object",
    "properties": {
        "service_id": {
            "type": "string",
            "description": "UUID del servicio (obtenido previamente con list_services).",
        },
    },
    "required": ["service_id"],
}

_AVAILABILITY_PARAMS = {
    "type": "object",
    "properties": {
        "date": {
            "type": "string",
            "description": "Fecha objetivo en formato YYYY-MM-DD.",
        },
        "service_id": {
            "type": "string",
            "description": "UUID del servicio. Obtener con list_services.",
        },
        "professional_id": {
            "type": "string",
            "description": "UUID del profesional (opcional). Si se omite, usa el primero disponible.",
        },
    },
    "required": ["date", "service_id"],
}

_SEARCH_KB_PARAMS = {
    "type": "object",
    "properties": {
        "query": {
            "type": "string",
            "description": "Pregunta o keywords a buscar en la base de conocimiento.",
        },
    },
    "required": ["query"],
}

_BUSINESS_INFO_PARAMS = {
    "type": "object",
    "properties": {},
    "required": [],
}

_LIST_MY_TREATMENTS_PARAMS = {
    "type": "object",
    "properties": {
        "phone": {
            "type": "string",
            "description": "Teléfono del cliente en formato internacional (ej: +584241234567). Obtener del context de la conversación, no inventar.",
        },
    },
    "required": ["phone"],
}

_BOOK_APPOINTMENT_PARAMS = {
    "type": "object",
    "properties": {
        "professional_id": {
            "type": "string",
            "description": (
                "UUID del profesional. DEBES haberlo obtenido previamente vía "
                "list_professionals o get_service_professionals. NUNCA inventar."
            ),
        },
        "service_id": {
            "type": "string",
            "description": (
                "UUID del servicio. DEBES haberlo obtenido previamente vía "
                "list_services. NUNCA inventar."
            ),
        },
        "starts_at": {
            "type": "string",
            "description": (
                "Inicio de la cita en ISO 8601 CON timezone (ej: "
                "'2026-06-12T14:30:00-04:00' o '2026-06-12T18:30:00Z'). "
                "DEBES tomarlo de un slot devuelto por check_availability. "
                "NUNCA inventar un horario."
            ),
        },
        "customer_name": {
            "type": "string",
            "description": (
                "Nombre del cliente tal como lo dijo en la conversación. "
                "Si no lo dio, primero preguntarlo — no completarlo con el "
                "push_name de WhatsApp sin confirmación."
            ),
        },
    },
    "required": ["professional_id", "service_id", "starts_at", "customer_name"],
    "additionalProperties": False,
}


# ---------------------------------------------------------------------------
# Registry
# ---------------------------------------------------------------------------

TOOL_REGISTRY: dict[str, ToolSpec] = {
    "list_services": ToolSpec(
        name="list_services",
        description=(
            "Returns the catalog of services offered by the business: id, name, price, "
            "currency, duration_min, description. Call this when the user asks about "
            "services, prices, what we offer, or before booking."
        ),
        fn=_tool_list_services,
        parameters=_LIST_SERVICES_PARAMS,
        context_args=("tenant_slug",),
    ),
    "list_professionals": ToolSpec(
        name="list_professionals",
        description=(
            "Returns the list of professionals/staff at the business: id, name, "
            "specialty, bio. Call this when the user asks about staff, who works there, "
            "specialists, or wants to pick a professional."
        ),
        fn=_tool_list_professionals,
        parameters=_LIST_PROFS_PARAMS,
        context_args=("tenant_slug",),
    ),
    "get_service_professionals": ToolSpec(
        name="get_service_professionals",
        description=(
            "Returns the professionals who can perform a specific service. Use after "
            "list_services when the user picks a service and you need to know who can do it."
        ),
        fn=_tool_get_service_professionals,
        parameters=_SVC_PROFS_PARAMS,
        context_args=("tenant_slug",),
    ),
    "check_availability": ToolSpec(
        name="check_availability",
        description=(
            "Returns available time slots for a given date and service. Use ONLY when the "
            "user wants to know if there are openings on a specific date. Requires "
            "service_id (from list_services). professional_id is optional; if omitted, the "
            "first eligible professional is used."
        ),
        fn=_tool_check_availability,
        parameters=_AVAILABILITY_PARAMS,
        context_args=("tenant_slug",),
    ),
    "search_knowledge": ToolSpec(
        name="search_knowledge",
        description=(
            "Searches the business knowledge base (FAQs, policies, location details, "
            "promotions, special notes) via semantic search. Call this when the user asks "
            "anything not covered by services/professionals/availability — e.g. parking, "
            "payment methods, opening hours, cancellation policy, address details."
        ),
        fn=_tool_search_knowledge,
        parameters=_SEARCH_KB_PARAMS,
        context_args=("tenant_id",),
    ),
    "get_business_info": ToolSpec(
        name="get_business_info",
        description=(
            "Returns general business info: name, type, location (city/country), "
            "timezone, description, contact (phone/email/address). Call this when the user "
            "asks where you are, what kind of business, contact details, or basic info."
        ),
        fn=_tool_get_business_info,
        parameters=_BUSINESS_INFO_PARAMS,
        context_args=("tenant_slug",),
    ),
    "list_my_treatments": ToolSpec(
        name="list_my_treatments",
        description=(
            "DENTAL ONLY. Returns the client's active treatment plans (orthodontics, "
            "implants, endodontics, etc.) with progress and next scheduled session. "
            "Call ONLY when the user asks about their treatment, sessions remaining, "
            "treatment progress, or next dental session. Returns no_dental_module if "
            "the business is not a dental clinic with the module activated."
        ),
        fn=_tool_list_my_treatments,
        parameters=_LIST_MY_TREATMENTS_PARAMS,
        context_args=("tenant_slug",),
    ),
    "book_appointment": ToolSpec(
        name="book_appointment",
        description=(
            "Books an appointment for the customer in ONE shot. Use this ONLY when "
            "ALL of the following are true:\n"
            "1. You already called check_availability and have a real slot "
            "   (starts_at) returned by it — NEVER invent a time slot.\n"
            "2. You already know the service_id (from list_services) and the "
            "   professional_id (from list_professionals or get_service_professionals).\n"
            "3. You asked the customer for their name and they gave it to you.\n"
            "4. You presented the 4 data points (service, professional, date/time, "
            "   name) back to the customer in your previous turn AND got an "
            "   EXPLICIT confirmation (e.g. 'sí', 'dale', 'confirmo', 'ok agenda').\n"
            "If ANY of those is missing, DO NOT call this tool — ask the user or "
            "call the appropriate read tool first.\n"
            "On 'slot_no_longer_available' error, apologize and call "
            "check_availability again to offer alternative slots."
        ),
        fn=_tool_book_appointment,
        parameters=_BOOK_APPOINTMENT_PARAMS,
        # customer_phone y tenant_slug se inyectan desde el ctx; el LLM nunca
        # los controla. Ver execute_tool() — orden de inyección.
        context_args=("tenant_slug", "customer_phone"),
    ),
}


# Listas pre-armadas para pasar directo al SDK
OPENAI_TOOLS: list[dict[str, Any]] = [
    spec.openai_schema() for spec in TOOL_REGISTRY.values()
]

GEMINI_TOOLS: list[dict[str, Any]] = [
    {"function_declarations": [spec.gemini_schema() for spec in TOOL_REGISTRY.values()]}
]

# Tools que mutan o gestionan el booking in-chat. En modo dental 'send_link'
# las desregistramos del LLM para que NUNCA intente agendar — el flujo dirige
# al link de booking en vez de tomar el slot por chat.
_BOOKING_MUTATION_TOOLS = {"check_availability", "book_appointment"}


def openai_tools_for_mode(dental_mode: str | None) -> list[dict[str, Any]]:
    """
    Devuelve el subset de OPENAI_TOOLS apropiado según el modo del asistente.

    - 'send_link': sin check_availability ni book_appointment (el bot no agenda).
    - cualquier otro valor o None: todas las tools (comportamiento legacy).
    """
    if dental_mode == "send_link":
        return [
            spec.openai_schema()
            for name, spec in TOOL_REGISTRY.items()
            if name not in _BOOKING_MUTATION_TOOLS
        ]
    return OPENAI_TOOLS


# ---------------------------------------------------------------------------
# Executor
# ---------------------------------------------------------------------------


async def execute_tool(name: str, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    """Ejecuta una tool por nombre, inyectando ctx (tenant_slug, tenant_id).

    Args:
        name: nombre de la tool registrada.
        args: argumentos provistos por el LLM (sólo los del schema).
        ctx: contexto del runtime (tenant_slug, tenant_id, etc).

    Returns:
        dict con {"ok": bool, "data": ...} o {"ok": False, "error": "..."}.
        Nunca lanza excepción.
    """
    spec = TOOL_REGISTRY.get(name)
    if not spec:
        log.warning("tools.execute_tool: tool desconocida", name=name)
        return _err(f"unknown_tool:{name}")

    # Inyectar ctx_args (tenant_slug, tenant_id) antes de los args del LLM
    call_kwargs: dict[str, Any] = {}
    for key in spec.context_args:
        if key not in ctx:
            log.warning("tools.execute_tool: falta contexto", name=name, key=key)
            return _err(f"missing_context:{key}")
        call_kwargs[key] = ctx[key]

    # Args del LLM — sólo los declarados en el schema
    declared = set((spec.parameters.get("properties") or {}).keys())
    for k, v in (args or {}).items():
        if k in declared:
            call_kwargs[k] = v

    # Log seguro: sólo nombres de args, nunca valores (evita PII)
    log.info(
        "tool_call",
        name=name,
        args_keys=sorted(call_kwargs.keys()),
    )

    # Tracking opt-in (ver start_tracking/get_tracked). Si no hay tracking
    # activo, _tools_called es None y no hacemos nada.
    tracked = _tools_called.get()
    if tracked is not None:
        tracked.append(name)

    try:
        return await spec.fn(**call_kwargs)
    except TypeError as e:
        log.error("tools.execute_tool: args inválidos", name=name, error=str(e))
        return _err(f"bad_args:{e}")
    except Exception as e:
        log.error("tools.execute_tool: error ejecutando tool", name=name, error=str(e))
        return _err(f"tool_failed:{name}")
