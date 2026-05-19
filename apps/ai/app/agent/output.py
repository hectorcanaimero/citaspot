"""WhatsApp-friendly output formatting.

WhatsApp supports a SUBSET of markdown:
  *bold*   _italic_   ~strikethrough~   ```code```   > quote
It does NOT support: # headers, ## headers, **bold** (double asterisk),
__italic__, [link](url), tables, complex lists.

Funciones puras, sin side-effects, solo stdlib `re`.
"""
from __future__ import annotations

import re

# --- Patrones de normalización de markdown (compilados una sola vez) ---

# Headers ATX al inicio de línea: `#`, `##`, `###`, etc. (con espacio opcional).
_RE_ATX_HEADER = re.compile(r"^[ \t]*#{1,6}[ \t]*", flags=re.MULTILINE)

# Triple-asterisco/underscore: ***bold-italic*** → *_bold_*
_RE_TRIPLE_AST = re.compile(r"\*{3}([^*\n]+?)\*{3}")
_RE_TRIPLE_UND = re.compile(r"_{3}([^_\n]+?)_{3}")

# Doble-asterisco/underscore: **bold** | __bold__ → *bold*
_RE_DOUBLE_AST = re.compile(r"\*{2}([^*\n]+?)\*{2}")
_RE_DOUBLE_UND = re.compile(r"_{2}([^_\n]+?)_{2}")

# Code fences: ```lang ... ``` → mantener el contenido como texto plano.
# DOTALL para que . también matchee newlines.
_RE_CODE_FENCE = re.compile(r"```[a-zA-Z0-9_\-]*\n?(.*?)\n?```", flags=re.DOTALL)

# Markdown links: [text](url) → "text (url)"
_RE_MD_LINK = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")

# 3+ newlines consecutivos → 2 newlines.
_RE_MANY_NEWLINES = re.compile(r"\n{3,}")

# Trailing whitespace por línea.
_RE_TRAILING_WS = re.compile(r"[ \t]+$", flags=re.MULTILINE)


def _normalize_markdown(text: str) -> str:
    """Aplica todas las reglas de normalización de markdown para WhatsApp.

    Orden importa: code fences primero (para no tocar contenido dentro de ```),
    luego triples antes que dobles para no consumir mal los asteriscos.
    """
    # 1. Code fences: extraer contenido, quitar las marcas ```
    text = _RE_CODE_FENCE.sub(lambda m: m.group(1), text)

    # 2. Links markdown → "text (url)"
    text = _RE_MD_LINK.sub(r"\1 (\2)", text)

    # 3. Triples ANTES que dobles
    text = _RE_TRIPLE_AST.sub(r"*_\1_*", text)
    text = _RE_TRIPLE_UND.sub(r"*_\1_*", text)

    # 4. Dobles → simple (formato WhatsApp)
    text = _RE_DOUBLE_AST.sub(r"*\1*", text)
    text = _RE_DOUBLE_UND.sub(r"*\1*", text)

    # 5. Headers ATX → quitar los # marks al inicio de línea
    text = _RE_ATX_HEADER.sub("", text)

    # 6. Trailing whitespace por línea
    text = _RE_TRAILING_WS.sub("", text)

    # 7. Colapsar 3+ newlines a 2
    text = _RE_MANY_NEWLINES.sub("\n\n", text)

    return text.strip()


def _split_paragraphs(text: str, max_chars: int) -> list[str]:
    """Intenta partir por párrafos (\\n\\n). Si un párrafo supera max_chars,
    cae a _split_sentences. Acumula párrafos hasta que el chunk se llena."""
    paragraphs = [p for p in text.split("\n\n") if p.strip()]
    chunks: list[str] = []
    current = ""

    for para in paragraphs:
        para = para.strip()
        if len(para) > max_chars:
            # Flush current y partir este párrafo por oraciones
            if current:
                chunks.append(current)
                current = ""
            chunks.extend(_split_sentences(para, max_chars))
            continue

        # ¿Cabe en el chunk actual?
        if current and len(current) + 2 + len(para) <= max_chars:
            current = f"{current}\n\n{para}"
        else:
            if current:
                chunks.append(current)
            current = para

    if current:
        chunks.append(current)

    return chunks


def _split_sentences(text: str, max_chars: int) -> list[str]:
    """Parte un bloque largo en oraciones (.!?) acumulando hasta max_chars."""
    # Encontrar finales de oración: `. ` `! ` `? ` (con espacio o fin de string).
    # Usamos un regex con lookbehind simple para preservar el separador.
    parts = re.split(r"(?<=[.!?])\s+", text)
    chunks: list[str] = []
    current = ""

    for sentence in parts:
        sentence = sentence.strip()
        if not sentence:
            continue
        if len(sentence) > max_chars:
            # Oración demasiado larga incluso sola: flush y hard-split.
            if current:
                chunks.append(current)
                current = ""
            chunks.extend(_hard_split(sentence, max_chars))
            continue
        if current and len(current) + 1 + len(sentence) <= max_chars:
            current = f"{current} {sentence}"
        else:
            if current:
                chunks.append(current)
            current = sentence

    if current:
        chunks.append(current)

    return chunks


def _hard_split(text: str, max_chars: int) -> list[str]:
    """Último recurso: corta a max_chars buscando un espacio en los últimos 30
    chars para no romper palabras."""
    chunks: list[str] = []
    remaining = text
    while len(remaining) > max_chars:
        # Buscar último espacio dentro de los últimos 30 chars del límite.
        window_start = max(0, max_chars - 30)
        cut = remaining.rfind(" ", window_start, max_chars)
        if cut == -1:
            cut = max_chars  # no hay espacio, cortamos a lo bruto
        chunks.append(remaining[:cut].rstrip())
        remaining = remaining[cut:].lstrip()
    if remaining:
        chunks.append(remaining)
    return chunks


def format_for_whatsapp(text: str, max_chars: int = 400) -> list[str]:
    """Convierte texto crudo del LLM a una lista de mensajes WhatsApp-friendly.

    Pipeline:
      1. Normalizar markdown (headers, bold, links, code fences, newlines).
      2. Si len <= max_chars: devolver [text].
      3. Sino: partir en chunks priorizando párrafo → oración → hard-split.
      4. Filtrar chunks vacíos.
      5. Si la lista quedó vacía, devolver [""].

    El caller envía los chunks en orden con un pequeño delay entre cada uno
    para respetar el rate limit de WhatsApp (1 msg/seg por sesión, CLAUDE.md §7).
    """
    if text is None:
        return [""]

    normalized = _normalize_markdown(text)

    if not normalized:
        return [""]

    if len(normalized) <= max_chars:
        return [normalized]

    chunks = _split_paragraphs(normalized, max_chars)
    chunks = [c.strip() for c in chunks if c and c.strip()]

    if not chunks:
        return [""]

    return chunks


def estimate_chars(text: str) -> int:
    """Contador barato de chars para logging/métricas."""
    return len(text) if text else 0
