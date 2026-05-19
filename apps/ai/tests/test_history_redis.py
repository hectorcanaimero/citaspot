# Tests de app.agent.history — usando fakeredis para evitar Redis real.
# Verifica:
#   - append_message + get_recent roundtrip
#   - LTRIM al cap de 20
#   - TTL seteado en cada append
#   - clear borra el historial
#   - rol inválido se filtra
#   - JSON corrupto se salta
#   - Errores de Redis → []
from __future__ import annotations

import json
from unittest.mock import AsyncMock, MagicMock

import fakeredis.aioredis
import pytest

from app.agent import history as history_module
from app.agent.history import _HIST_CAP, _HIST_TTL, append_message, clear, get_recent


@pytest.fixture
def fake_redis(monkeypatch: pytest.MonkeyPatch):
    """Reemplaza get_redis() por una instancia in-memory de fakeredis."""
    client = fakeredis.aioredis.FakeRedis(decode_responses=True)
    monkeypatch.setattr(history_module, "get_redis", lambda: client)
    return client


# ---------------------------------------------------------------------------
# Roundtrip happy path
# ---------------------------------------------------------------------------


async def test_append_and_get_roundtrip(fake_redis):
    await append_message("t1", "c1", "user", "hola")
    out = await get_recent("t1", "c1", limit=10)
    assert out == [{"role": "user", "content": "hola"}]


async def test_append_25_returns_last_10_in_chronological_order(fake_redis):
    """25 appends → get_recent(10) devuelve los últimos 10 en orden cronológico."""
    for i in range(25):
        await append_message("t1", "c1", "user", f"msg-{i}")
    out = await get_recent("t1", "c1", limit=10)
    # get_recent retorna oldest → newest, y LTRIM cortó a los últimos 20.
    # Por lo tanto los 10 que esperamos son msg-15..msg-24 (los más recientes).
    assert len(out) == 10
    assert out[0]["content"] == "msg-15"
    assert out[-1]["content"] == "msg-24"
    assert all(item["role"] == "user" for item in out)


async def test_ltrim_to_cap_20(fake_redis):
    """Tras 25 appends, la lista en Redis debe tener exactamente 20 elementos."""
    for i in range(25):
        await append_message("t1", "c1", "user", f"m{i}")
    key = f"conv:hist:t1:c1"
    length = await fake_redis.llen(key)
    assert length == _HIST_CAP == 20


async def test_ttl_set_on_each_append(fake_redis):
    """EXPIRE debe estar seteado tras cada append (ttl > 0 y <= _HIST_TTL)."""
    await append_message("t1", "c1", "user", "hola")
    ttl = await fake_redis.ttl(f"conv:hist:t1:c1")
    # ttl puede ser un poco menor que _HIST_TTL si pasó algún ms
    assert 0 < ttl <= _HIST_TTL


async def test_clear_removes_history(fake_redis):
    await append_message("t1", "c1", "user", "uno")
    await append_message("t1", "c1", "assistant", "dos")
    await clear("t1", "c1")
    out = await get_recent("t1", "c1", limit=10)
    assert out == []


# ---------------------------------------------------------------------------
# Roles y validación
# ---------------------------------------------------------------------------


async def test_invalid_role_is_rejected_silently(fake_redis):
    """role='system' (no permitido) → no se persiste, no se levanta error."""
    await append_message("t1", "c1", "system", "system prompt")
    out = await get_recent("t1", "c1", limit=10)
    assert out == []


async def test_empty_content_is_ignored(fake_redis):
    """content vacío → no se persiste."""
    await append_message("t1", "c1", "user", "")
    out = await get_recent("t1", "c1", limit=10)
    assert out == []


async def test_assistant_role_accepted(fake_redis):
    await append_message("t1", "c1", "assistant", "hola, ¿en qué te ayudo?")
    out = await get_recent("t1", "c1", limit=10)
    assert out == [{"role": "assistant", "content": "hola, ¿en qué te ayudo?"}]


# ---------------------------------------------------------------------------
# Resiliencia
# ---------------------------------------------------------------------------


async def test_corrupt_json_is_skipped(fake_redis):
    """Entrada corrupta en Redis → get_recent la salta y retorna las válidas."""
    key = f"conv:hist:t1:c1"
    # Inyectar manualmente una entrada corrupta + una válida (más reciente)
    await fake_redis.lpush(key, "not-json-at-all")
    await fake_redis.lpush(
        key, json.dumps({"role": "user", "content": "soy válido"})
    )
    out = await get_recent("t1", "c1", limit=10)
    assert out == [{"role": "user", "content": "soy válido"}]


async def test_entry_missing_role_is_skipped(fake_redis):
    """JSON válido pero con role inválido → se salta."""
    key = f"conv:hist:t1:c1"
    await fake_redis.lpush(key, json.dumps({"role": "system", "content": "x"}))
    await fake_redis.lpush(
        key, json.dumps({"role": "user", "content": "válido"})
    )
    out = await get_recent("t1", "c1", limit=10)
    assert out == [{"role": "user", "content": "válido"}]


async def test_redis_error_returns_empty_list(monkeypatch: pytest.MonkeyPatch):
    """Si Redis levanta excepción, get_recent retorna [] y no propaga."""
    broken = MagicMock()
    broken.lrange = AsyncMock(side_effect=ConnectionError("redis down"))
    monkeypatch.setattr(history_module, "get_redis", lambda: broken)
    out = await get_recent("t1", "c1", limit=10)
    assert out == []


async def test_redis_error_on_append_does_not_raise(monkeypatch: pytest.MonkeyPatch):
    """Si Redis levanta en el pipeline, append no debe propagar."""
    broken = MagicMock()
    pipe = MagicMock()
    pipe.lpush = MagicMock(return_value=pipe)
    pipe.ltrim = MagicMock(return_value=pipe)
    pipe.expire = MagicMock(return_value=pipe)
    pipe.execute = AsyncMock(side_effect=ConnectionError("redis down"))
    broken.pipeline = MagicMock(return_value=pipe)
    monkeypatch.setattr(history_module, "get_redis", lambda: broken)

    # No debe levantar
    await append_message("t1", "c1", "user", "hola")


async def test_limit_zero_returns_empty(fake_redis):
    """limit <= 0 → retorna [] sin tocar Redis."""
    await append_message("t1", "c1", "user", "hola")
    assert await get_recent("t1", "c1", limit=0) == []
    assert await get_recent("t1", "c1", limit=-1) == []
