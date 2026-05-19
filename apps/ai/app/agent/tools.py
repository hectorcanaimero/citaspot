# Tool registry para function-calling del agente IA.
# Cada herramienta envuelve una acción de `app.agent.actions` y expone:
#   - una función async que devuelve dict {"ok": bool, "data": ...} o {"ok": False, "error": ...}
#   - un schema JSON compatible con OpenAI tools / Gemini function declarations
#
# El LLM nunca debe inventar servicios, precios u horarios: si necesita info
# del catálogo del tenant, debe pedirla vía estas tools.
from __future__ import annotations

from contextvars import ContextVar
from dataclasses import dataclass
from typing import Any, Awaitable, Callable

import structlog

from app.agent.actions import (
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
}


# Listas pre-armadas para pasar directo al SDK
OPENAI_TOOLS: list[dict[str, Any]] = [
    spec.openai_schema() for spec in TOOL_REGISTRY.values()
]

GEMINI_TOOLS: list[dict[str, Any]] = [
    {"function_declarations": [spec.gemini_schema() for spec in TOOL_REGISTRY.values()]}
]


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
