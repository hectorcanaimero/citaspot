# LLM Router — Gemini 1.5 Flash (primario) con fallback a GPT-4o-mini.
from __future__ import annotations

from typing import Any

import google.generativeai as genai
import structlog
from openai import AsyncOpenAI

from app.core.config import settings

log = structlog.get_logger(__name__)

# Inicializar clientes una sola vez
genai.configure(api_key=settings.gemini_api_key)
_openai = AsyncOpenAI(api_key=settings.openai_api_key)


def _to_gemini_messages(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Convierte mensajes formato OpenAI a formato Gemini."""
    result = []
    for m in messages:
        role = m["role"]
        if role == "system":
            # Gemini no tiene rol system — lo prepend como user
            continue
        gemini_role = "user" if role == "user" else "model"
        result.append({"role": gemini_role, "parts": [m["content"]]})
    return result


def _extract_system(messages: list[dict[str, Any]]) -> str:
    """Extrae el contenido del mensaje system si existe."""
    for m in messages:
        if m["role"] == "system":
            return m["content"]
    return ""


async def chat(messages: list[dict[str, Any]], temperature: float = 0.3) -> str:
    """
    Genera una respuesta de chat usando el LLM primario (Gemini) con fallback a OpenAI.

    Args:
        messages: Lista de mensajes en formato OpenAI [{role, content}].
        temperature: Temperatura de generación (0–1).

    Returns:
        Texto de respuesta del LLM.
    """
    # --- Gemini 1.5 Flash (primario) ---
    try:
        system_instruction = _extract_system(messages)
        gemini_msgs = _to_gemini_messages(messages)

        model = genai.GenerativeModel(
            model_name="gemini-2.5-flash",
            system_instruction=system_instruction or None,
            generation_config=genai.types.GenerationConfig(
                temperature=temperature,
                max_output_tokens=1024,
            ),
        )
        response = model.generate_content(
            [m["parts"][0] for m in gemini_msgs]
            if len(gemini_msgs) == 1
            else gemini_msgs
        )
        return response.text.strip()
    except Exception as e:
        log.warning("llm.router: Gemini falló, usando GPT-4o-mini como fallback", error=str(e))

    # --- GPT-4o-mini (fallback) ---
    response = await _openai.chat.completions.create(
        model="gpt-4o-mini",
        messages=messages,  # type: ignore[arg-type]
        temperature=temperature,
        max_tokens=1024,
    )
    return response.choices[0].message.content.strip()
