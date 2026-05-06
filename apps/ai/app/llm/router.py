# LLM Router — Estrategia multi-modelo optimizada por costo:
#   Intent detection → Gemini 2.5 Flash Lite (barato, rápido)
#   Respuestas conversacionales → DeepSeek V4 Flash (mejor relación costo/calidad)
#   Fallback → GPT-4o-mini (safety net)
from __future__ import annotations

from typing import Any

import google.generativeai as genai
import structlog
from openai import AsyncOpenAI

from app.core.config import settings

log = structlog.get_logger(__name__)

# Inicializar clientes
genai.configure(api_key=settings.gemini_api_key)
_openai = AsyncOpenAI(api_key=settings.openai_api_key)
_deepseek = AsyncOpenAI(
    api_key=settings.deepseek_api_key,
    base_url="https://api.deepseek.com",
)


def _to_gemini_messages(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Convierte mensajes formato OpenAI a formato Gemini."""
    result = []
    for m in messages:
        role = m["role"]
        if role == "system":
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


async def chat_lite(messages: list[dict[str, Any]], temperature: float = 0.0) -> str:
    """
    Modelo ligero para tareas de clasificación (intent detection).
    Usa Gemini 2.5 Flash Lite ($0.10/M input, $0.40/M output).
    Fallback: GPT-4o-mini.
    """
    try:
        system_instruction = _extract_system(messages)
        gemini_msgs = _to_gemini_messages(messages)

        model = genai.GenerativeModel(
            model_name="gemini-2.5-flash-lite",
            system_instruction=system_instruction or None,
            generation_config=genai.types.GenerationConfig(
                temperature=temperature,
                max_output_tokens=32,
            ),
        )
        response = model.generate_content(
            [m["parts"][0] for m in gemini_msgs]
            if len(gemini_msgs) == 1
            else gemini_msgs
        )
        return response.text.strip()
    except Exception as e:
        log.warning("llm.router.lite: Gemini Flash Lite falló, usando GPT-4o-mini", error=str(e))

    # Fallback a GPT-4o-mini
    response = await _openai.chat.completions.create(
        model="gpt-4o-mini",
        messages=messages,  # type: ignore[arg-type]
        temperature=temperature,
        max_tokens=32,
    )
    return response.choices[0].message.content.strip()


async def chat(messages: list[dict[str, Any]], temperature: float = 0.3) -> str:
    """
    Modelo principal para respuestas conversacionales (RAG, queries).
    Usa DeepSeek V4 Flash ($0.14/M input, $0.28/M output).
    Fallback: GPT-4o-mini.
    """
    # --- DeepSeek V4 Flash (primario) ---
    try:
        response = await _deepseek.chat.completions.create(
            model="deepseek-chat",
            messages=messages,  # type: ignore[arg-type]
            temperature=temperature,
            max_tokens=1024,
        )
        return response.choices[0].message.content.strip()
    except Exception as e:
        log.warning("llm.router: DeepSeek falló, usando GPT-4o-mini como fallback", error=str(e))

    # --- GPT-4o-mini (fallback) ---
    response = await _openai.chat.completions.create(
        model="gpt-4o-mini",
        messages=messages,  # type: ignore[arg-type]
        temperature=temperature,
        max_tokens=1024,
    )
    return response.choices[0].message.content.strip()
