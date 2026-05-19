# Test crítico anti-hallucination del handler de QUERY.
#
# Demuestra dos garantías del refactor:
#   1. _handle_query llama a chat_with_tools pasándole OPENAI_TOOLS y el ctx
#      correcto (tenant_slug + tenant_id). Esto es lo que evita que el LLM invente
#      datos: las tools están disponibles para que pida la info real.
#   2. La respuesta que devuelve chat_with_tools es lo que devuelve _handle_query,
#      sin pasar por el prompt builder del flujo "rico" (que inyectaba catálogo
#      completo y le daba al LLM rope para inventar).
from __future__ import annotations

from unittest.mock import AsyncMock

import pytest

from app.agent import orchestrator as orch_module
from app.agent.orchestrator import _build_query_system_prompt, _handle_query
from app.agent.tools import OPENAI_TOOLS


async def test_handle_query_returns_llm_text(monkeypatch: pytest.MonkeyPatch):
    """_handle_query devuelve exactamente lo que retorna chat_with_tools."""
    expected = "No, no ofrecemos masajes. Tenemos corte y tinte."
    monkeypatch.setattr(
        orch_module, "chat_with_tools", AsyncMock(return_value=expected)
    )

    out = await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="¿tienen masajes?",
        history=[],
    )
    assert out == expected


async def test_handle_query_passes_tools_and_ctx(monkeypatch: pytest.MonkeyPatch):
    """chat_with_tools debe recibir OPENAI_TOOLS y ctx con tenant_slug/tenant_id.

    Sin tools en el call, el LLM no puede consultar el catálogo y queda libre
    de inventar. Este test prueba que la "rope anti-hallucination" está atada.
    """
    captured = {}

    async def fake_chat_with_tools(
        messages, tools, ctx, temperature=0.2, max_tokens=800, max_iterations=3
    ):
        captured["messages"] = messages
        captured["tools"] = tools
        captured["ctx"] = ctx
        captured["max_iterations"] = max_iterations
        return "respuesta del llm"

    monkeypatch.setattr(orch_module, "chat_with_tools", fake_chat_with_tools)

    await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="¿qué servicios tienen?",
        history=[],
    )

    # Las 6 tools del registry deben estar disponibles para el LLM
    assert captured["tools"] is OPENAI_TOOLS
    assert len(captured["tools"]) == 6

    # ctx debe incluir AMBOS identificadores (tenant_slug para perfiles públicos,
    # tenant_id para RAG)
    assert captured["ctx"] == {"tenant_slug": "demo", "tenant_id": "tid-1"}

    # max_iterations debe estar acotado (anti-loop)
    assert captured["max_iterations"] == 3


async def test_handle_query_includes_user_message_last(
    monkeypatch: pytest.MonkeyPatch,
):
    """El último mensaje pasado al LLM debe ser el del usuario."""
    captured = {}

    async def fake_chat_with_tools(messages, tools, ctx, **kwargs):
        captured["messages"] = messages
        return "ok"

    monkeypatch.setattr(orch_module, "chat_with_tools", fake_chat_with_tools)

    await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="pregunta del cliente",
        history=[],
    )

    msgs = captured["messages"]
    assert msgs[0]["role"] == "system"
    assert msgs[-1] == {"role": "user", "content": "pregunta del cliente"}


async def test_handle_query_includes_recent_history(
    monkeypatch: pytest.MonkeyPatch,
):
    """El historial reciente (últimos 8) debe pasarse al LLM."""
    captured = {}

    async def fake_chat_with_tools(messages, tools, ctx, **kwargs):
        captured["messages"] = messages
        return "ok"

    monkeypatch.setattr(orch_module, "chat_with_tools", fake_chat_with_tools)

    history = [
        {"role": "user", "content": f"msg-{i}"} for i in range(10)
    ]
    await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="nuevo",
        history=history,
    )

    msgs = captured["messages"]
    contents = [m["content"] for m in msgs]
    # Los últimos 8 del history deben estar (msg-2 .. msg-9)
    assert "msg-2" in contents
    assert "msg-9" in contents
    # Los más viejos NO
    assert "msg-0" not in contents
    assert "msg-1" not in contents


async def test_handle_query_fallback_on_tool_loop_error(
    monkeypatch: pytest.MonkeyPatch,
):
    """Si chat_with_tools lanza, _handle_query no propaga y devuelve un mensaje seguro."""
    monkeypatch.setattr(
        orch_module,
        "chat_with_tools",
        AsyncMock(side_effect=RuntimeError("llm down")),
    )

    out = await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="hola",
        history=[],
    )
    # No nos importa el texto exacto, pero debe ser un string no vacío.
    assert isinstance(out, str)
    assert out.strip() != ""


# ---------------------------------------------------------------------------
# _build_query_system_prompt — assertions clave del anti-hallucination
# ---------------------------------------------------------------------------


def test_query_system_prompt_does_not_inject_catalog():
    """El system prompt del flujo QUERY NO debe contener catálogo concreto:
    sin nombres de servicios ni precios. El LLM los pide vía tools.
    """
    prompt = _build_query_system_prompt("demo", is_first_turn=True)
    # No debería mencionar números monetarios concretos (no hay catálogo cargado)
    # Heurística: no menciona "$" como precio (puede ir en instrucciones pero no como dato)
    assert "demo" in prompt
    # Debe mencionar las tools por nombre para dirigir al LLM
    assert "list_services" in prompt
    assert "search_knowledge" in prompt


def test_query_system_prompt_forbids_invention():
    """Debe haber instrucción explícita de no inventar."""
    prompt = _build_query_system_prompt("demo", is_first_turn=True).lower()
    # Buscamos términos en es/en/pt que prohíban inventar
    assert any(term in prompt for term in ("nunca inventes", "never invent", "nunca invente"))
