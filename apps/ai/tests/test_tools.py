# Tests de app.agent.tools — registry, execute_tool, y wrappers individuales.
# Mockea actions (Core API + RAG) para evitar tráfico real.
from __future__ import annotations

from unittest.mock import AsyncMock

import pytest

from app.agent import tools as tools_module
from app.agent.tools import OPENAI_TOOLS, TOOL_REGISTRY, execute_tool


# ---------------------------------------------------------------------------
# Registry
# ---------------------------------------------------------------------------


def test_registry_has_expected_tools():
    expected = {
        "list_services",
        "list_professionals",
        "get_service_professionals",
        "check_availability",
        "search_knowledge",
        "get_business_info",
        "list_my_treatments",
        "book_appointment",
    }
    assert set(TOOL_REGISTRY.keys()) == expected


def test_openai_tools_shape():
    assert len(OPENAI_TOOLS) == len(TOOL_REGISTRY)
    names_in_registry = set(TOOL_REGISTRY.keys())
    seen = set()
    for entry in OPENAI_TOOLS:
        assert entry["type"] == "function"
        fn = entry["function"]
        assert "name" in fn
        assert "description" in fn
        assert "parameters" in fn
        assert fn["name"] in names_in_registry
        seen.add(fn["name"])
    assert seen == names_in_registry


# ---------------------------------------------------------------------------
# list_services
# ---------------------------------------------------------------------------


_PROFILE_OK = {
    "name": "Salón Demo",
    "timezone": "America/Caracas",
    "services": [
        {
            "id": "svc-1",
            "name": "Corte",
            "price": 50,
            "currency": "USD",
            "duration_min": 30,
            "description": "Corte clásico",
        },
        {
            "id": "svc-2",
            "name": "Tinte",
            "price": 80,
            "duration_min": 90,
        },
    ],
    "professionals": [
        {"id": "prof-1", "name": "Ana", "specialty": "color", "bio": ""},
        {"id": "prof-2", "name": "Bea", "specialty": "", "bio": "10 años"},
    ],
    "service_professionals": [
        {"service_id": "svc-1", "professional_id": "prof-1"},
        {"service_id": "svc-2", "professional_id": "prof-2"},
    ],
    "business_type": "salón",
    "city": "Caracas",
    "country": "VE",
    "description": "test",
    "phone": "+58000",
    "email": "x@y.com",
    "address": "calle 1",
    "bot_name": "Lia",
}


async def test_list_services_happy(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    res = await execute_tool("list_services", {}, {"tenant_slug": "demo"})
    assert res["ok"] is True
    data = res["data"]
    assert len(data) == 2
    assert data[0]["name"] == "Corte"
    assert data[0]["price"] == 50
    assert data[0]["currency"] == "USD"
    assert data[1]["currency"] == "USD"  # default cuando no viene


async def test_list_services_no_profile(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=None)
    )
    res = await execute_tool("list_services", {}, {"tenant_slug": "demo"})
    assert res["ok"] is False
    assert "error" in res


async def test_list_services_core_api_raises(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module,
        "get_tenant_profile",
        AsyncMock(side_effect=RuntimeError("boom")),
    )
    res = await execute_tool("list_services", {}, {"tenant_slug": "demo"})
    # _load_profile captura la excepción y devuelve None → _err
    assert res["ok"] is False


# ---------------------------------------------------------------------------
# list_professionals
# ---------------------------------------------------------------------------


async def test_list_professionals_happy(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    res = await execute_tool("list_professionals", {}, {"tenant_slug": "demo"})
    assert res["ok"] is True
    assert len(res["data"]) == 2
    assert res["data"][0]["name"] == "Ana"


async def test_list_professionals_no_profile(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=None)
    )
    res = await execute_tool("list_professionals", {}, {"tenant_slug": "demo"})
    assert res["ok"] is False


# ---------------------------------------------------------------------------
# get_service_professionals
# ---------------------------------------------------------------------------


async def test_get_service_professionals_happy(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    res = await execute_tool(
        "get_service_professionals",
        {"service_id": "svc-1"},
        {"tenant_slug": "demo"},
    )
    assert res["ok"] is True
    assert len(res["data"]) == 1
    assert res["data"][0]["id"] == "prof-1"


async def test_get_service_professionals_unknown_service(
    monkeypatch: pytest.MonkeyPatch,
):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    res = await execute_tool(
        "get_service_professionals",
        {"service_id": "non-existent"},
        {"tenant_slug": "demo"},
    )
    assert res["ok"] is True
    assert res["data"] == []


# ---------------------------------------------------------------------------
# check_availability
# ---------------------------------------------------------------------------


async def test_check_availability_happy(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    slots = [{"starts_at": "2026-05-20T10:00:00Z", "ends_at": "2026-05-20T10:30:00Z"}]
    monkeypatch.setattr(
        tools_module, "get_availability", AsyncMock(return_value=slots)
    )

    res = await execute_tool(
        "check_availability",
        {"date": "2026-05-20", "service_id": "svc-1"},
        {"tenant_slug": "demo"},
    )
    assert res["ok"] is True
    assert res["data"]["date"] == "2026-05-20"
    assert res["data"]["professional_id"] == "prof-1"
    assert res["data"]["slots"] == slots


async def test_check_availability_missing_service_id(monkeypatch: pytest.MonkeyPatch):
    res = await execute_tool(
        "check_availability",
        {"date": "2026-05-20"},
        {"tenant_slug": "demo"},
    )
    assert res["ok"] is False
    assert res["error"] == "missing_service_id"


async def test_check_availability_get_availability_raises(
    monkeypatch: pytest.MonkeyPatch,
):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    monkeypatch.setattr(
        tools_module,
        "get_availability",
        AsyncMock(side_effect=RuntimeError("api down")),
    )
    res = await execute_tool(
        "check_availability",
        {"date": "2026-05-20", "service_id": "svc-1"},
        {"tenant_slug": "demo"},
    )
    assert res["ok"] is False
    assert "availability_failed" in res["error"]


# ---------------------------------------------------------------------------
# search_knowledge
# ---------------------------------------------------------------------------


async def test_search_knowledge_happy(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "rag_query", AsyncMock(return_value="Estacionamiento gratis.")
    )
    res = await execute_tool(
        "search_knowledge",
        {"query": "tienen estacionamiento?"},
        {"tenant_id": "tid-1"},
    )
    assert res["ok"] is True
    assert "Estacionamiento" in res["data"]["context"]


async def test_search_knowledge_empty_query(monkeypatch: pytest.MonkeyPatch):
    res = await execute_tool(
        "search_knowledge", {"query": "   "}, {"tenant_id": "tid-1"}
    )
    assert res["ok"] is False
    assert res["error"] == "empty_query"


async def test_search_knowledge_rag_raises(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "rag_query", AsyncMock(side_effect=RuntimeError("vector db"))
    )
    res = await execute_tool(
        "search_knowledge", {"query": "horario"}, {"tenant_id": "tid-1"}
    )
    assert res["ok"] is False
    assert res["error"] == "rag_failed"


# ---------------------------------------------------------------------------
# get_business_info
# ---------------------------------------------------------------------------


async def test_get_business_info_happy(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    res = await execute_tool("get_business_info", {}, {"tenant_slug": "demo"})
    assert res["ok"] is True
    d = res["data"]
    assert d["name"] == "Salón Demo"
    assert d["city"] == "Caracas"
    assert d["country"] == "VE"
    assert d["timezone"] == "America/Caracas"


async def test_get_business_info_no_profile(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=None)
    )
    res = await execute_tool("get_business_info", {}, {"tenant_slug": "demo"})
    assert res["ok"] is False


# ---------------------------------------------------------------------------
# execute_tool — control de errores genéricos
# ---------------------------------------------------------------------------


async def test_execute_tool_unknown_name():
    res = await execute_tool("non_existent_tool", {}, {"tenant_slug": "demo"})
    assert res["ok"] is False
    assert "unknown_tool" in res["error"]


async def test_execute_tool_missing_context(monkeypatch: pytest.MonkeyPatch):
    """Si falta tenant_slug en ctx para una tool que lo requiere → error."""
    monkeypatch.setattr(
        tools_module, "get_tenant_profile", AsyncMock(return_value=_PROFILE_OK)
    )
    res = await execute_tool("list_services", {}, {})
    assert res["ok"] is False
    assert "missing_context" in res["error"]


async def test_execute_tool_filters_undeclared_args(
    monkeypatch: pytest.MonkeyPatch,
):
    """Args no declarados en el schema deben filtrarse — la tool no recibe basura."""
    captured = {}

    async def fake_profile(slug):
        captured["slug"] = slug
        return _PROFILE_OK

    monkeypatch.setattr(tools_module, "get_tenant_profile", fake_profile)

    # list_services no acepta argumentos del LLM, los inventados deben filtrarse.
    res = await execute_tool(
        "list_services",
        {"junk_arg": "ignored", "another": 123},
        {"tenant_slug": "demo"},
    )
    assert res["ok"] is True
    assert captured["slug"] == "demo"


async def test_execute_tool_never_raises_on_internal_exception(
    monkeypatch: pytest.MonkeyPatch,
):
    """Si la fn interna lanza una excepción no-TypeError, execute_tool la captura."""
    from dataclasses import replace

    async def boom(**kwargs):
        raise ValueError("inesperado")

    # ToolSpec es frozen → usamos dataclasses.replace para construir una copia
    # con `fn` reemplazada, y la inyectamos en una copia del registry.
    original = TOOL_REGISTRY["list_services"]
    patched_spec = replace(original, fn=boom)
    patched_registry = dict(TOOL_REGISTRY)
    patched_registry["list_services"] = patched_spec
    monkeypatch.setattr(tools_module, "TOOL_REGISTRY", patched_registry)

    res = await execute_tool("list_services", {}, {"tenant_slug": "demo"})
    assert res["ok"] is False
    assert "tool_failed" in res["error"]


# ---------------------------------------------------------------------------
# book_appointment — registro, schema, happy path y manejo de errores HTTP
# ---------------------------------------------------------------------------


_VALID_PROF_ID = "11111111-1111-4111-8111-111111111111"
_VALID_SVC_ID = "22222222-2222-4222-8222-222222222222"
_VALID_STARTS_AT = "2026-06-12T14:30:00-04:00"


def test_book_appointment_registered():
    """La tool book_appointment debe estar en el registry y en OPENAI_TOOLS."""
    assert "book_appointment" in TOOL_REGISTRY
    names_in_openai = {t["function"]["name"] for t in OPENAI_TOOLS}
    assert "book_appointment" in names_in_openai


def test_book_appointment_schema_required_params():
    """El schema OpenAI debe exigir los 4 params del LLM, y NO incluir los del ctx.

    customer_phone y tenant_slug se inyectan desde el runtime — el LLM NO debe
    poder controlarlos via tool args, eso permitiría alucinaciones peligrosas
    (booking en otro tenant, reemplazo de número).
    """
    spec = TOOL_REGISTRY["book_appointment"]
    props = spec.parameters["properties"]
    required = set(spec.parameters["required"])

    assert required == {"professional_id", "service_id", "starts_at", "customer_name"}
    assert set(props.keys()) == required
    # Los context_args declaran exactamente lo que la tool exige del runtime.
    assert set(spec.context_args) == {"tenant_slug", "customer_phone"}
    # Verificación explícita: el schema no expone customer_phone ni tenant_slug.
    assert "customer_phone" not in props
    assert "tenant_slug" not in props


async def test_book_appointment_happy_path(monkeypatch: pytest.MonkeyPatch):
    """Llamada exitosa: devuelve {ok: True, data: {appointment_id, ...}}."""
    fake_response = {
        "id": "appt-uuid-123",
        "starts_at": _VALID_STARTS_AT,
        "ends_at": "2026-06-12T15:00:00-04:00",
        "status": "scheduled",
    }
    mock_book = AsyncMock(return_value=fake_response)
    monkeypatch.setattr(tools_module, "book_appointment", mock_book)

    res = await execute_tool(
        "book_appointment",
        {
            "professional_id": _VALID_PROF_ID,
            "service_id": _VALID_SVC_ID,
            "starts_at": _VALID_STARTS_AT,
            "customer_name": "Juan Pérez",
        },
        {"tenant_slug": "demo", "customer_phone": "+584241234567"},
    )

    assert res["ok"] is True
    assert res["data"]["appointment_id"] == "appt-uuid-123"
    assert res["data"]["starts_at"] == _VALID_STARTS_AT
    # Verificar que el ctx se inyectó correctamente en la llamada subyacente.
    mock_book.assert_awaited_once()
    kwargs = mock_book.await_args.kwargs
    assert kwargs["slug"] == "demo"
    assert kwargs["customer_phone"] == "+584241234567"
    assert kwargs["source"] == "whatsapp"
    assert kwargs["return_error_details"] is True


async def test_book_appointment_slot_taken_409(monkeypatch: pytest.MonkeyPatch):
    """Si el endpoint responde 409, la tool devuelve un error accionable sin lanzar."""
    mock_book = AsyncMock(return_value={
        "_error": True,
        "status": 409,
        "message": "slot no disponible",
    })
    monkeypatch.setattr(tools_module, "book_appointment", mock_book)

    res = await execute_tool(
        "book_appointment",
        {
            "professional_id": _VALID_PROF_ID,
            "service_id": _VALID_SVC_ID,
            "starts_at": _VALID_STARTS_AT,
            "customer_name": "Juan",
        },
        {"tenant_slug": "demo", "customer_phone": "+584241234567"},
    )

    assert res["ok"] is False
    assert res["error"] == "slot_no_longer_available"


async def test_book_appointment_invalid_uuid_short_circuits(
    monkeypatch: pytest.MonkeyPatch,
):
    """Si el LLM alucina un UUID inválido, no llamamos al endpoint."""
    mock_book = AsyncMock()
    monkeypatch.setattr(tools_module, "book_appointment", mock_book)

    res = await execute_tool(
        "book_appointment",
        {
            "professional_id": "not-a-uuid",
            "service_id": _VALID_SVC_ID,
            "starts_at": _VALID_STARTS_AT,
            "customer_name": "Juan",
        },
        {"tenant_slug": "demo", "customer_phone": "+584241234567"},
    )

    assert res["ok"] is False
    assert res["error"] == "invalid_professional_id"
    mock_book.assert_not_awaited()


async def test_book_appointment_invalid_starts_at_no_tz(
    monkeypatch: pytest.MonkeyPatch,
):
    """starts_at sin timezone es ambiguo → rechazar antes del HTTP."""
    mock_book = AsyncMock()
    monkeypatch.setattr(tools_module, "book_appointment", mock_book)

    res = await execute_tool(
        "book_appointment",
        {
            "professional_id": _VALID_PROF_ID,
            "service_id": _VALID_SVC_ID,
            "starts_at": "2026-06-12T14:30:00",  # sin tz
            "customer_name": "Juan",
        },
        {"tenant_slug": "demo", "customer_phone": "+584241234567"},
    )

    assert res["ok"] is False
    assert res["error"] == "invalid_starts_at_format"
    mock_book.assert_not_awaited()


async def test_book_appointment_missing_customer_phone_in_ctx(
    monkeypatch: pytest.MonkeyPatch,
):
    """Sin customer_phone en ctx, execute_tool corta antes de ejecutar la tool."""
    mock_book = AsyncMock()
    monkeypatch.setattr(tools_module, "book_appointment", mock_book)

    res = await execute_tool(
        "book_appointment",
        {
            "professional_id": _VALID_PROF_ID,
            "service_id": _VALID_SVC_ID,
            "starts_at": _VALID_STARTS_AT,
            "customer_name": "Juan",
        },
        {"tenant_slug": "demo"},  # falta customer_phone
    )

    assert res["ok"] is False
    assert "missing_context" in res["error"]
    assert "customer_phone" in res["error"]
    mock_book.assert_not_awaited()


async def test_book_appointment_filters_phone_from_llm_args(
    monkeypatch: pytest.MonkeyPatch,
):
    """Aunque el LLM intente pasar customer_phone, se filtra y se usa el del ctx."""
    captured = {}

    async def fake_book(**kwargs):
        captured.update(kwargs)
        return {"id": "x", "starts_at": _VALID_STARTS_AT}

    monkeypatch.setattr(tools_module, "book_appointment", fake_book)

    await execute_tool(
        "book_appointment",
        {
            "professional_id": _VALID_PROF_ID,
            "service_id": _VALID_SVC_ID,
            "starts_at": _VALID_STARTS_AT,
            "customer_name": "Juan",
            # Si el LLM trata de inyectar un phone, debe filtrarse: solo entran
            # los args declarados en el schema.
            "customer_phone": "+10000000000",
        },
        {"tenant_slug": "demo", "customer_phone": "+584241234567"},
    )

    # El phone que llegó a la action HTTP es el del ctx, NO el del LLM.
    assert captured["customer_phone"] == "+584241234567"
