# Tests del endpoint /process-test (chatbot de prueba del dashboard).
#
# Verifica que el endpoint:
#   - Usa el MISMO code path que el worker real (intent + _handle_query).
#   - Persiste historial bajo el prefijo `test:` (no contamina prod).
#   - Detecta intents correctamente y rutea a _handle_query o al mensaje "modo test".
#   - Devuelve los campos nuevos (tools_called, chunks_count, state_after).
#   - Aplica format_for_whatsapp (chunks_count refleja cuántos mensajes WA serían).
from __future__ import annotations

from unittest.mock import AsyncMock

import fakeredis.aioredis
import pytest
from httpx import ASGITransport, AsyncClient

from app.agent import history as history_module
from app.agent import tools as tools_module
from app.agent.intent import Intent
from app.routers import process_test as endpoint_module
from main import app


@pytest.fixture
def fake_redis(monkeypatch: pytest.MonkeyPatch):
    """Reemplaza get_redis() en TODOS los módulos que lo usan por una instancia
    in-memory de fakeredis. Esto cubre history, state y cualquier otra cosa
    que vaya contra Redis durante el test.
    """
    from app.agent import state as state_module

    client = fakeredis.aioredis.FakeRedis(decode_responses=True)
    monkeypatch.setattr(history_module, "get_redis", lambda: client)
    monkeypatch.setattr(state_module, "get_redis", lambda: client)
    return client


async def _post(payload: dict) -> tuple[int, dict]:
    """Helper: POST al endpoint y devuelve (status, body)."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
        r = await ac.post("/process-test", json=payload)
    return r.status_code, r.json()


# ---------------------------------------------------------------------------
# Routing por intent
# ---------------------------------------------------------------------------


async def test_query_intent_routes_to_handle_query(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """`hola, ¿qué servicios tienen?` → QUERY → _handle_query."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )
    fake_response = "Ofrecemos corte ($50), tinte ($80) y manicura ($25)."
    handle_mock = AsyncMock(return_value=fake_response)
    monkeypatch.setattr(endpoint_module, "_handle_query", handle_mock)

    status, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "hola, ¿qué servicios tienen?",
    })

    assert status == 200
    assert body["response"] == fake_response
    assert body["intent_detected"] == "QUERY"
    assert body["chunks_count"] == 1
    assert body["state_after"] == "IDLE"
    assert isinstance(body["tools_called"], list)
    # _handle_query fue invocado con los args correctos
    handle_mock.assert_awaited_once()
    args = handle_mock.call_args.args
    assert args[0] == "tid-1"          # tenant_id
    assert args[1] == "demo"           # tenant_slug
    assert args[2] == "hola, ¿qué servicios tienen?"
    assert args[3] == []               # history vacío en primer turno


async def test_unknown_intent_also_routes_to_handle_query(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """UNKNOWN cae al mismo handler que QUERY."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.UNKNOWN),
    )
    handle_mock = AsyncMock(return_value="No estoy seguro de lo que pedís.")
    monkeypatch.setattr(endpoint_module, "_handle_query", handle_mock)

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "asdf qwer zxcv",
    })

    assert body["intent_detected"] == "UNKNOWN"
    handle_mock.assert_awaited_once()


async def test_booking_intent_returns_test_mode_message(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """BOOKING devuelve mensaje '[modo test]' sin invocar _handle_query."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.BOOKING),
    )
    handle_mock = AsyncMock(return_value="NO DEBE LLAMARSE")
    monkeypatch.setattr(endpoint_module, "_handle_query", handle_mock)

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "quiero agendar una cita",
    })

    assert body["intent_detected"] == "BOOKING"
    assert "[modo test]" in body["response"]
    assert "booking" in body["response"].lower()
    handle_mock.assert_not_awaited()


async def test_cancel_intent_returns_test_mode_message(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.CANCEL),
    )
    monkeypatch.setattr(
        endpoint_module, "_handle_query", AsyncMock(return_value=""),
    )

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "cancelar mi cita",
    })
    assert body["intent_detected"] == "CANCEL"
    assert "[modo test]" in body["response"]


async def test_reschedule_intent_returns_test_mode_message(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.RESCHEDULE),
    )
    monkeypatch.setattr(
        endpoint_module, "_handle_query", AsyncMock(return_value=""),
    )

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "mover mi cita para mañana",
    })
    assert body["intent_detected"] == "RESCHEDULE"
    assert "[modo test]" in body["response"]


# ---------------------------------------------------------------------------
# Historial: persistencia con prefijo test:
# ---------------------------------------------------------------------------


async def test_history_persisted_under_test_prefix(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """Después del POST, los turnos deben estar en la key test:{conv} y
    NO en la conv real (sin prefijo)."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )
    monkeypatch.setattr(
        endpoint_module, "_handle_query", AsyncMock(return_value="hola, ¿en qué te ayudo?"),
    )

    await _post({
        "tenant_id": "tid-X",
        "tenant_slug": "demo",
        "conversation_id": "conv-X",
        "message_text": "hola",
    })

    # Historial bajo prefijo test:
    hist_test = await history_module.get_recent("tid-X", "test:conv-X", limit=10)
    assert len(hist_test) == 2
    assert hist_test[0] == {"role": "user", "content": "hola"}
    assert hist_test[1] == {"role": "assistant", "content": "hola, ¿en qué te ayudo?"}

    # Historial sin prefijo (donde vive prod) debe estar vacío
    hist_prod = await history_module.get_recent("tid-X", "conv-X", limit=10)
    assert hist_prod == []


async def test_history_loaded_on_subsequent_turns(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """En el segundo POST, _handle_query debe recibir el historial del primer turno."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )
    handle_mock = AsyncMock(return_value="segunda respuesta")
    monkeypatch.setattr(endpoint_module, "_handle_query", handle_mock)

    # Pre-poblar historial bajo test:conv-Y para simular un turno previo
    await history_module.append_message("tid-Y", "test:conv-Y", "user", "primer mensaje")
    await history_module.append_message(
        "tid-Y", "test:conv-Y", "assistant", "primera respuesta",
    )

    await _post({
        "tenant_id": "tid-Y",
        "tenant_slug": "demo",
        "conversation_id": "conv-Y",
        "message_text": "segundo mensaje",
    })

    # El historial pasado a _handle_query debe tener los 2 turnos previos
    args = handle_mock.call_args.args
    history_arg = args[3]
    assert len(history_arg) == 2
    assert history_arg[0]["content"] == "primer mensaje"
    assert history_arg[1]["content"] == "primera respuesta"


# ---------------------------------------------------------------------------
# Tools tracking
# ---------------------------------------------------------------------------


async def test_tools_called_is_present_and_empty_when_handle_query_mocked(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """Con _handle_query mockeado, no se invoca ninguna tool real → lista vacía."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )
    monkeypatch.setattr(
        endpoint_module, "_handle_query", AsyncMock(return_value="respuesta"),
    )

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "hola",
    })

    assert "tools_called" in body
    assert body["tools_called"] == []


async def test_tools_called_records_executions(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """Si _handle_query (mock) invoca execute_tool, los nombres aparecen en tools_called."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )

    async def fake_handle(tenant_id, tenant_slug, msg, history):
        # Simular que el LLM invocó 2 tools durante el handler
        await tools_module.execute_tool(
            "get_business_info", {}, {"tenant_slug": tenant_slug},
        )
        await tools_module.execute_tool(
            "list_services", {}, {"tenant_slug": tenant_slug},
        )
        return "respuesta basada en tools"

    # Mockear la dependencia interna de las tools (get_tenant_profile)
    # para que execute_tool no haga HTTP real.
    monkeypatch.setattr(
        tools_module,
        "get_tenant_profile",
        AsyncMock(return_value={
            "name": "Demo",
            "services": [{"id": "s1", "name": "Corte", "price": 10, "duration_min": 30}],
            "professionals": [],
        }),
    )
    monkeypatch.setattr(endpoint_module, "_handle_query", fake_handle)

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "qué servicios y dónde están",
    })

    assert body["tools_called"] == ["get_business_info", "list_services"]


# ---------------------------------------------------------------------------
# format_for_whatsapp / chunks_count
# ---------------------------------------------------------------------------


async def test_long_response_reports_multiple_chunks(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    """Una respuesta larga → chunks_count > 1 pero el body llega completo."""
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )
    # Texto suficientemente largo para forzar split (>400 chars), con párrafos.
    long_text = "\n\n".join(["Párrafo " + ("x" * 300) for _ in range(3)])
    monkeypatch.setattr(
        endpoint_module, "_handle_query", AsyncMock(return_value=long_text),
    )

    _, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "contame todo",
    })

    assert body["chunks_count"] >= 2
    # El dashboard ve la respuesta unida con \n\n entre chunks
    assert "\n\n" in body["response"]


# ---------------------------------------------------------------------------
# Resiliencia: _handle_query falla → no crashea el endpoint
# ---------------------------------------------------------------------------


async def test_handle_query_exception_returns_fallback(
    fake_redis, monkeypatch: pytest.MonkeyPatch,
):
    monkeypatch.setattr(
        endpoint_module,
        "detect_intent",
        AsyncMock(return_value=Intent.QUERY),
    )
    monkeypatch.setattr(
        endpoint_module,
        "_handle_query",
        AsyncMock(side_effect=RuntimeError("LLM down")),
    )

    status, body = await _post({
        "tenant_id": "tid-1",
        "tenant_slug": "demo",
        "conversation_id": "conv-1",
        "message_text": "hola",
    })

    assert status == 200
    assert body["response"]  # algo de texto, no vacío
    assert body["intent_detected"] == "QUERY"


# ---------------------------------------------------------------------------
# DELETE /process-test/{tenant_id}/state
# ---------------------------------------------------------------------------


async def test_clear_test_state_endpoint(fake_redis):
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
        r = await ac.delete("/process-test/tid-1/state")
    assert r.status_code == 200
    assert r.json() == {"status": "ok"}
