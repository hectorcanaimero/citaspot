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


def test_registry_has_six_tools():
    expected = {
        "list_services",
        "list_professionals",
        "get_service_professionals",
        "check_availability",
        "search_knowledge",
        "get_business_info",
    }
    assert set(TOOL_REGISTRY.keys()) == expected


def test_openai_tools_shape():
    assert len(OPENAI_TOOLS) == 6
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
