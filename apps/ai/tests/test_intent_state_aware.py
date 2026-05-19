# Tests de la capa de pattern-matching de intent.py (sin LLM).
# Comprueba que los mensajes cortos resuelven sin llamar al LLM,
# y que detect() llama (o no) al LLM según el patrón.
from __future__ import annotations

from unittest.mock import AsyncMock

import pytest

from app.agent import intent as intent_module
from app.agent.intent import Intent, _pattern_match, detect
from app.agent.state import ConvState


# ---------------------------------------------------------------------------
# _pattern_match — confirmaciones
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    "msg,state,expected",
    [
        # Confirmaciones en estado AWAITING_CONFIRM → CONFIRM
        ("sí", ConvState.AWAITING_CONFIRM, Intent.CONFIRM),
        ("dale ok", ConvState.AWAITING_CONFIRM, Intent.CONFIRM),
        ("OK!", ConvState.AWAITING_CONFIRM, Intent.CONFIRM),
        ("👍", ConvState.AWAITING_CONFIRM, Intent.CONFIRM),
        # AWAITING_NAME también es estado de confirmación → CONFIRM
        ("perfecto", ConvState.AWAITING_NAME, Intent.CONFIRM),
        # Confirmación fuera de flujo (IDLE) → BOOKING (positivo = quiero reservar)
        ("sí", ConvState.IDLE, Intent.BOOKING),
        # Negativas en cualquier estado → CANCEL
        ("no", ConvState.AWAITING_CONFIRM, Intent.CANCEL),
        ("no", ConvState.IDLE, Intent.CANCEL),
        ("cancela", ConvState.IDLE, Intent.CANCEL),
        ("cancela", ConvState.AWAITING_SLOT, Intent.CANCEL),
        # Mensajes que NO matchean → None (fallthrough al LLM)
        ("hola", ConvState.IDLE, None),
        ("hola", ConvState.AWAITING_CONFIRM, None),
        ("¿tienen masajes?", ConvState.AWAITING_CONFIRM, None),
        ("quiero agendar un corte", ConvState.IDLE, None),
        # Edge: empty
        ("", ConvState.IDLE, None),
    ],
)
def test_pattern_match_cases(msg: str, state: ConvState | None, expected):
    assert _pattern_match(msg, state) == expected


def test_pattern_match_none_state():
    """conv_state=None → positivo cae a BOOKING (no en confirm_states)."""
    assert _pattern_match("dale", None) == Intent.BOOKING


def test_pattern_match_emoji_thumbs_up():
    """👍 (U+1F44D) → CONFIRM en AWAITING_CONFIRM."""
    assert _pattern_match("👍", ConvState.AWAITING_CONFIRM) == Intent.CONFIRM


def test_pattern_match_check_mark():
    """✅ → CONFIRM en AWAITING_CONFIRM."""
    assert _pattern_match("✅", ConvState.AWAITING_CONFIRM) == Intent.CONFIRM


# ---------------------------------------------------------------------------
# detect() — verifica que el LLM se llama (o no) según el pattern match
# ---------------------------------------------------------------------------


async def test_detect_skips_llm_when_pattern_matches(monkeypatch: pytest.MonkeyPatch):
    """Si _pattern_match resuelve, el LLM NO debe ser llamado."""
    fake_llm = AsyncMock(return_value="BOOKING")
    monkeypatch.setattr(intent_module, "chat_lite", fake_llm)

    result = await detect("dale ok", history=[], conv_state=ConvState.AWAITING_CONFIRM)

    assert result == Intent.CONFIRM
    fake_llm.assert_not_called()


async def test_detect_calls_llm_when_pattern_does_not_match(
    monkeypatch: pytest.MonkeyPatch,
):
    """Si _pattern_match retorna None, el LLM SÍ debe ser llamado."""
    fake_llm = AsyncMock(return_value="QUERY")
    monkeypatch.setattr(intent_module, "chat_lite", fake_llm)

    result = await detect(
        "¿tienen masajes?", history=[], conv_state=ConvState.AWAITING_CONFIRM
    )

    assert result == Intent.QUERY
    fake_llm.assert_called_once()


async def test_detect_includes_state_hint_in_prompt(monkeypatch: pytest.MonkeyPatch):
    """El system prompt enviado al LLM debe incluir el estado actual."""
    captured: dict = {}

    async def fake_chat_lite(messages, temperature=0.0):
        captured["messages"] = messages
        return "QUERY"

    monkeypatch.setattr(intent_module, "chat_lite", fake_chat_lite)

    await detect(
        "consulta rara",
        history=[],
        conv_state=ConvState.AWAITING_SLOT,
        pending_data={"service_id": "abc"},
    )

    msgs = captured.get("messages") or []
    assert len(msgs) >= 1
    system = msgs[0]
    assert system["role"] == "system"
    assert "AWAITING_SLOT" in system["content"]


async def test_detect_returns_unknown_on_unexpected_llm_response(
    monkeypatch: pytest.MonkeyPatch,
):
    """Respuesta del LLM fuera de Intent.__members__ → UNKNOWN."""
    monkeypatch.setattr(
        intent_module, "chat_lite", AsyncMock(return_value="BANANA_INTENT")
    )
    result = await detect("hola que tal", history=[], conv_state=ConvState.IDLE)
    assert result == Intent.UNKNOWN


async def test_detect_returns_unknown_on_llm_exception(
    monkeypatch: pytest.MonkeyPatch,
):
    """Si chat_lite lanza, detect retorna UNKNOWN (no propaga)."""
    monkeypatch.setattr(
        intent_module, "chat_lite", AsyncMock(side_effect=RuntimeError("api down"))
    )
    result = await detect("buenas", history=[], conv_state=ConvState.IDLE)
    assert result == Intent.UNKNOWN


async def test_detect_passes_history_to_llm(monkeypatch: pytest.MonkeyPatch):
    """Los últimos 3 mensajes del historial deben llegar al LLM."""
    captured: dict = {}

    async def fake_chat_lite(messages, temperature=0.0):
        captured["messages"] = messages
        return "QUERY"

    monkeypatch.setattr(intent_module, "chat_lite", fake_chat_lite)

    history = [
        {"role": "user", "content": "primero"},
        {"role": "assistant", "content": "respuesta"},
        {"role": "user", "content": "segundo"},
        {"role": "assistant", "content": "respuesta 2"},
    ]
    await detect("nuevo mensaje", history=history, conv_state=ConvState.IDLE)

    # system + (3 últimos del history) + user actual = 5
    msgs = captured["messages"]
    contents = [m["content"] for m in msgs]
    # Los 3 últimos del history están
    assert "respuesta" in contents
    assert "segundo" in contents
    assert "respuesta 2" in contents
    # Y el mensaje actual cierra la lista
    assert msgs[-1]["role"] == "user"
    assert msgs[-1]["content"] == "nuevo mensaje"
