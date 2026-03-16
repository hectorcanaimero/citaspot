# Chunker: divide documentos largos en chunks de tamaño adecuado para embeddings.
from __future__ import annotations

_MAX_CHUNK_TOKENS = 400   # ~300 palabras, seguro para text-embedding-3-small
_OVERLAP_TOKENS = 50       # solapamiento para preservar contexto entre chunks


def chunk_text(text: str, max_tokens: int = _MAX_CHUNK_TOKENS, overlap: int = _OVERLAP_TOKENS) -> list[str]:
    """
    Divide un texto en chunks solapados por oraciones.
    Preserva oraciones completas: no corta en medio de una.

    Args:
        text: Texto a dividir.
        max_tokens: Tamaño máximo estimado en tokens por chunk (1 token ≈ 1 palabra).
        overlap: Palabras de solapamiento entre chunks consecutivos.

    Returns:
        Lista de strings, cada uno es un chunk.
    """
    # Separar por oraciones (punto, signo de interrogación, exclamación)
    import re
    sentences = re.split(r'(?<=[.!?])\s+', text.strip())

    chunks: list[str] = []
    current_words: list[str] = []

    for sentence in sentences:
        words = sentence.split()
        if not words:
            continue

        # Si agregar la oración supera el límite, guardamos el chunk actual
        if current_words and len(current_words) + len(words) > max_tokens:
            chunks.append(" ".join(current_words))
            # Overlap: conservar las últimas `overlap` palabras
            current_words = current_words[-overlap:] if overlap else []

        current_words.extend(words)

    # Último chunk
    if current_words:
        chunks.append(" ".join(current_words))

    return [c for c in chunks if c.strip()]
