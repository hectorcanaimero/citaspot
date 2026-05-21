# Tests de inyección de contexto RAG en el handler de QUERY.
#
# Garantías que cubren:
#   1. Si rag_search devuelve chunks, el system prompt enviado a chat_with_tools
#      contiene el bloque KB formateado con título/categoría/contenido.
#   2. Si rag_search devuelve [], el system prompt NO contiene el bloque KB y
#      el flujo de tool-calling sigue normal.
#   3. Si rag_search lanza excepción, se loguea warning, el flujo continúa
#      sin contexto KB y la respuesta del LLM se devuelve igual.
from __future__ import annotations

from unittest.mock import AsyncMock

import pytest

from app.agent import orchestrator as orch_module
from app.agent.orchestrator import _format_kb_context, _handle_query


# ---------------------------------------------------------------------------
# _format_kb_context — formato del bloque inyectado
# ---------------------------------------------------------------------------


def test_format_kb_context_empty_returns_empty_string():
    """Sin resultados, devuelve string vacío (no header, no footer)."""
    assert _format_kb_context([]) == ""


def test_format_kb_context_includes_title_category_and_content():
    """Cada chunk debe aparecer con título, categoría y contenido legible."""
    results = [
        {
            "id": "1",
            "document_id": "doc-1",
            "content": "Estamos en el C.C. MAMI, local 5, Puerto Ordaz.",
            "similarity": 0.82,
            "category": "ubicacion",
            "title": "Dirección de la sede",
        },
        {
            "id": "2",
            "document_id": "doc-2",
            "content": "Horario: L-V 9am-6pm, sábados 9am-1pm.",
            "similarity": 0.71,
            "category": "horario",
            "title": "Horario de atención",
        },
    ]
    block = _format_kb_context(results)

    # Contenido de ambos chunks debe estar presente
    assert "C.C. MAMI" in block
    assert "Horario: L-V" in block
    # Títulos como encabezados de chunk
    assert "Dirección de la sede" in block
    assert "Horario de atención" in block
    # Categorías
    assert "ubicacion" in block
    assert "horario" in block
    # Directiva anti-alucinación al cierre
    assert "PRIMERO" in block or "PRIMEIRO" in block or "FIRST" in block


# ---------------------------------------------------------------------------
# _handle_query — inyección de contexto en el system prompt
# ---------------------------------------------------------------------------


async def test_handle_query_injects_kb_block_when_search_returns_results(
    monkeypatch: pytest.MonkeyPatch,
):
    """Cuando rag_search devuelve chunks, el system prompt debe contenerlos."""
    captured: dict = {}

    async def fake_chat_with_tools(messages, tools, ctx, **kwargs):
        captured["messages"] = messages
        return "respuesta"

    fake_results = [
        {
            "id": "1",
            "document_id": "doc-1",
            "content": "Estamos en el C.C. MAMI, local 5, Puerto Ordaz.",
            "similarity": 0.82,
            "category": "ubicacion",
            "title": "Dirección de la sede",
        },
    ]
    monkeypatch.setattr(
        orch_module, "rag_search", AsyncMock(return_value=fake_results)
    )
    monkeypatch.setattr(orch_module, "chat_with_tools", fake_chat_with_tools)

    await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="donde se ubican?",
        history=[],
    )

    system_msg = captured["messages"][0]
    assert system_msg["role"] == "system"
    # El contenido del chunk debe estar embebido en el system prompt
    assert "C.C. MAMI" in system_msg["content"]
    assert "Dirección de la sede" in system_msg["content"]


async def test_handle_query_omits_kb_block_when_no_results(
    monkeypatch: pytest.MonkeyPatch,
):
    """Cuando rag_search devuelve [], el system prompt NO debe tener bloque KB."""
    captured: dict = {}

    async def fake_chat_with_tools(messages, tools, ctx, **kwargs):
        captured["messages"] = messages
        return "ok"

    monkeypatch.setattr(orch_module, "rag_search", AsyncMock(return_value=[]))
    monkeypatch.setattr(orch_module, "chat_with_tools", fake_chat_with_tools)

    await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="hola",
        history=[],
    )

    system_content = captured["messages"][0]["content"]
    # No debe aparecer el header del bloque KB
    assert "Información relevante de la base de conocimiento" not in system_content
    assert "Relevant info from the business knowledge base" not in system_content
    # El resto del prompt sigue ahí (reglas anti-alucinación)
    assert "demo" in system_content


async def test_handle_query_continues_when_rag_search_raises(
    monkeypatch: pytest.MonkeyPatch,
):
    """Si rag_search lanza, _handle_query loguea warning y sigue sin KB."""
    captured: dict = {}

    async def fake_chat_with_tools(messages, tools, ctx, **kwargs):
        captured["messages"] = messages
        return "respuesta sin kb"

    monkeypatch.setattr(
        orch_module,
        "rag_search",
        AsyncMock(side_effect=RuntimeError("pgvector down")),
    )
    monkeypatch.setattr(orch_module, "chat_with_tools", fake_chat_with_tools)

    out = await _handle_query(
        tenant_id="tid-1",
        tenant_slug="demo",
        message_text="donde se ubican?",
        history=[],
    )

    # Flujo no rompe: devolvemos lo que dijo el LLM
    assert out == "respuesta sin kb"
    # System prompt NO contiene bloque KB (porque el search falló)
    system_content = captured["messages"][0]["content"]
    assert "Información relevante de la base de conocimiento" not in system_content


async def test_handle_query_calls_rag_search_with_tenant_and_message(
    monkeypatch: pytest.MonkeyPatch,
):
    """rag_search debe recibir tenant_id y el texto del mensaje del usuario."""
    rag_calls: dict = {}

    async def fake_rag_search(tenant_id, query, *args, **kwargs):
        rag_calls["tenant_id"] = tenant_id
        rag_calls["query"] = query
        return []

    monkeypatch.setattr(orch_module, "rag_search", fake_rag_search)
    monkeypatch.setattr(
        orch_module, "chat_with_tools", AsyncMock(return_value="ok")
    )

    await _handle_query(
        tenant_id="tid-xyz",
        tenant_slug="demo",
        message_text="cual es el horario?",
        history=[],
    )

    assert rag_calls["tenant_id"] == "tid-xyz"
    assert rag_calls["query"] == "cual es el horario?"
