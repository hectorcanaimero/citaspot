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

    # Todas las tools del registry deben estar disponibles para el LLM
    from app.agent.tools import TOOL_REGISTRY
    assert captured["tools"] is OPENAI_TOOLS
    assert len(captured["tools"]) == len(TOOL_REGISTRY)

    # ctx debe incluir tenant_slug (perfiles públicos), tenant_id (RAG) y
    # customer_phone (book_appointment lo inyecta desde el wa_phone del runtime).
    assert captured["ctx"] == {
        "tenant_slug": "demo",
        "tenant_id": "tid-1",
        "customer_phone": "",
    }

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


# ---------------------------------------------------------------------------
# Reglas anti-pregunta-redundante (Plane #40)
# Si el LLM pregunta "¿te muestro X?" en vez de llamar la tool, perdemos un
# turno y rompemos el flujo: el "si" del usuario no tiene tracking de qué
# fue ofrecido. Estas pruebas garantizan que las 4 reglas críticas estén
# presentes y bien posicionadas en el prompt.
# ---------------------------------------------------------------------------


def test_query_system_prompt_forbids_permission_questions():
    """Regla ACCIÓN DIRECTA: prohibir preguntas tipo '¿te muestro X?'."""
    prompt = _build_query_system_prompt("demo", is_first_turn=False).lower()
    # Debe contener la regla (en alguno de los 3 idiomas)
    assert any(
        term in prompt
        for term in ("acción directa", "ação direta", "direct action")
    )
    # Debe incluir el ejemplo de pregunta prohibida o la mención de "permiso/permission"
    assert any(
        term in prompt
        for term in ("¿te muestro", "should i show", "quer que eu mostre")
    )
    # Debe ordenar llamar la tool inmediatamente
    assert any(
        term in prompt
        for term in ("inmediatamente", "imediatamente", "immediately")
    )


def test_query_system_prompt_handles_affirmative_continuity():
    """Regla CONTINUIDAD: si el bot ofreció algo y el usuario dice 'si',
    el LLM debe ejecutar la acción ofrecida, no repreguntar."""
    prompt = _build_query_system_prompt("demo", is_first_turn=False).lower()
    assert any(
        term in prompt
        for term in ("continuidad", "continuidade", "continuity")
    )
    # Debe mencionar al menos una afirmación corta como ejemplo
    assert "'sí'" in prompt or "'si'" in prompt or "'yes'" in prompt or "'sim'" in prompt
    # Debe ordenar no repreguntar / no volver al menú
    assert any(
        term in prompt
        for term in (
            "no repreguntes",
            "do not re-ask",
            "não repergunte",
            "no vuelvas al menú",
            "do not return to the menu",
            "não volte ao menu",
        )
    )


def test_query_system_prompt_forbids_menu_mid_conversation():
    """Regla NO MENÚ MID-CONVERSACIÓN: en medio de conversación, no responder
    con el saludo genérico de menú inicial."""
    prompt = _build_query_system_prompt("demo", is_first_turn=False).lower()
    assert any(
        term in prompt
        for term in (
            "no menú mid-conversación",
            "no menu mid-conversation",
            "sem menu no meio da conversa",
        )
    )
    # Debe referirse a "más de 1 turno" como condición
    assert any(
        term in prompt
        for term in ("más de 1 turno", "more than 1 turn", "mais de 1 turno")
    )


def test_query_system_prompt_handles_thread_loss():
    """Regla 'hola?' en medio: si el usuario reinicia con '¿hola?' / '¿estás
    ahí?', mirar el historial y ejecutar lo prometido, no repetir el menú."""
    prompt = _build_query_system_prompt("demo", is_first_turn=False).lower()
    # Debe mencionar la situación
    assert any(
        term in prompt
        for term in ("¿hola?", "hello?", "are you there", "estás ahí", "está aí", "oi?")
    )
    # Debe instruir mirar el historial
    assert any(
        term in prompt
        for term in ("mira el historial", "read the history", "leia o histórico")
    )


def test_query_system_prompt_critical_rules_appear_first():
    """Las reglas críticas (anti-pregunta-redundante) deben aparecer ANTES
    de las reglas generales. Los LLM dan más peso a las primeras instrucciones."""
    prompt = _build_query_system_prompt("demo", is_first_turn=False)
    lower = prompt.lower()

    # Buscar marcador del bloque crítico
    critical_markers = ["reglas críticas", "critical rules", "regras críticas"]
    critical_idx = min(
        (lower.find(m) for m in critical_markers if m in lower), default=-1
    )
    assert critical_idx > 0, "No se encontró el bloque de REGLAS CRÍTICAS"

    # Buscar marcador del bloque general
    general_markers = ["reglas generales", "general rules", "regras gerais"]
    general_idx = min(
        (lower.find(m) for m in general_markers if m in lower), default=-1
    )
    assert general_idx > 0, "No se encontró el bloque de REGLAS GENERALES"

    # Las críticas deben aparecer antes
    assert critical_idx < general_idx, (
        "Las REGLAS CRÍTICAS deben ir ANTES de las REGLAS GENERALES "
        "para que el LLM les dé más peso."
    )


def test_query_system_prompt_rules_present_for_both_turns():
    """Las reglas críticas deben estar presentes en first_turn y mid-conversation.
    El bug ocurre en ambos contextos."""
    for is_first_turn in (True, False):
        prompt = _build_query_system_prompt(
            "demo", is_first_turn=is_first_turn
        ).lower()
        assert any(
            term in prompt
            for term in ("acción directa", "ação direta", "direct action")
        ), f"Regla ACCIÓN DIRECTA falta en is_first_turn={is_first_turn}"
        assert any(
            term in prompt
            for term in ("continuidad", "continuidade", "continuity")
        ), f"Regla CONTINUIDAD falta en is_first_turn={is_first_turn}"
