# Tests para app.agent.output.format_for_whatsapp.
# Funciones puras (sin async, sin I/O) — chequean la normalización de markdown
# y el splitter de chunks <= 400 chars.
from __future__ import annotations

import pytest

from app.agent.output import format_for_whatsapp


# ---------------------------------------------------------------------------
# Casos triviales / parametrizados
# ---------------------------------------------------------------------------


def test_short_text_returns_single_chunk():
    """Texto corto bajo max_chars → lista de un solo elemento."""
    out = format_for_whatsapp("hola mundo")
    assert out == ["hola mundo"]


def test_empty_string_returns_empty_list_entry():
    """Input vacío → ['']."""
    assert format_for_whatsapp("") == [""]


def test_whitespace_only_returns_empty_entry():
    """Solo whitespace → ['']."""
    assert format_for_whatsapp("   \n\n  \t  ") == [""]


def test_none_input_returns_empty_entry():
    """None de defensa → ['']."""
    # type: ignore[arg-type]
    assert format_for_whatsapp(None) == [""]  # type: ignore[arg-type]


@pytest.mark.parametrize(
    "raw,expected",
    [
        # Headers ATX se eliminan, queda solo el texto
        ("# Título", "Título"),
        ("## Subtítulo", "Subtítulo"),
        ("### H3", "H3"),
        # **bold** → *bold*
        ("**hola**", "*hola*"),
        ("texto **bold** aquí", "texto *bold* aquí"),
        # __bold__ → *bold*
        ("__importante__", "*importante*"),
        # ***x*** → *_x_*
        ("***muy fuerte***", "*_muy fuerte_*"),
        # Markdown link → "text (url)"
        ("Ver [docs](https://example.com)", "Ver docs (https://example.com)"),
        # Code fence simple
        ("```\nplain\n```", "plain"),
    ],
)
def test_normalize_markdown_simple_cases(raw: str, expected: str):
    out = format_for_whatsapp(raw)
    assert out == [expected]


def test_code_fence_with_language_keeps_content():
    """Code fences con lenguaje → mantener contenido como texto plano."""
    raw = "```python\nprint('hi')\n```"
    out = format_for_whatsapp(raw)
    assert len(out) == 1
    assert "print('hi')" in out[0]
    assert "```" not in out[0]
    assert "python" not in out[0]


def test_collapse_three_or_more_newlines_to_two():
    """3+ newlines consecutivos → 2."""
    raw = "uno\n\n\n\n\ndos"
    out = format_for_whatsapp(raw)
    assert out == ["uno\n\ndos"]


# ---------------------------------------------------------------------------
# Splitter — texto largo
# ---------------------------------------------------------------------------


def test_long_paragraphs_split_into_chunks_under_max():
    """1000+ chars en varios párrafos → varios chunks, todos <= 400."""
    # ~50 chars por párrafo, 25 párrafos → ~1250 chars
    para = "Lorem ipsum dolor sit amet consectetur adipiscing."
    raw = "\n\n".join([para for _ in range(25)])
    assert len(raw) > 1000

    out = format_for_whatsapp(raw, max_chars=400)
    assert len(out) > 1, "long input should split into multiple chunks"
    for chunk in out:
        assert len(chunk) <= 400, f"chunk excedió 400 chars: len={len(chunk)}"


def test_long_single_sentence_splits_by_period():
    """Oración > 400 chars sin \\n\\n → parte en . cada chunk <= 400."""
    # Construir un párrafo de varias oraciones, en un solo bloque (sin \n\n).
    sentence = (
        "Esta es una oración informativa pero relativamente larga sobre el negocio."
    )
    raw = " ".join([sentence for _ in range(10)])
    assert len(raw) > 400

    out = format_for_whatsapp(raw, max_chars=400)
    assert len(out) >= 2
    for chunk in out:
        assert len(chunk) <= 400


def test_pathological_single_word_hard_split():
    """String monolítico de 500 chars sin espacios → hard split, chunks <= 400."""
    raw = "x" * 500
    out = format_for_whatsapp(raw, max_chars=400)
    assert len(out) >= 2
    for chunk in out:
        assert len(chunk) <= 400
    # No debe perder caracteres
    joined = "".join(out)
    assert joined.count("x") == 500


def test_chunks_filtered_when_empty():
    """Chunks vacíos deben filtrarse del resultado."""
    raw = "uno\n\n\n\n\n\ndos"
    out = format_for_whatsapp(raw)
    assert all(c.strip() for c in out)


def test_max_chars_respected_when_exact():
    """Texto justo en el límite → un solo chunk."""
    raw = "a" * 400
    out = format_for_whatsapp(raw, max_chars=400)
    assert len(out) == 1
    assert len(out[0]) == 400


def test_combined_markdown_normalization():
    """Caso combinado: headers + bold + link en un mismo input."""
    raw = "# Servicios\n\nTenemos **corte** y [reservas](https://x.com)"
    out = format_for_whatsapp(raw)
    assert len(out) == 1
    text = out[0]
    assert "Servicios" in text
    assert "#" not in text
    assert "*corte*" in text
    assert "**" not in text
    assert "reservas (https://x.com)" in text
