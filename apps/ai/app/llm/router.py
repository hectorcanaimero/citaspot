# LLM Router — Estrategia multi-modelo optimizada por costo:
#   Intent detection → Gemini 2.5 Flash Lite (barato, rápido)
#   Respuestas conversacionales → DeepSeek V4 Flash (mejor relación costo/calidad)
#   Fallback → GPT-4o-mini (safety net)
#   Tool-calling     → OpenAI-compatible (DeepSeek y GPT-4o-mini soportan el formato)
from __future__ import annotations

import asyncio
import json
from typing import Any, Awaitable, Callable

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


# ---------------------------------------------------------------------------
# chat_with_tools — loop de function-calling (formato OpenAI-compat)
# ---------------------------------------------------------------------------
#
# Decisión: usamos formato OpenAI tools tanto en DeepSeek (primario) como en
# GPT-4o-mini (fallback). DeepSeek expone una API compatible 1:1 con OpenAI,
# así que un solo loop sirve para ambos. Gemini queda fuera del path principal
# de tool-calling porque su API de function-calling tiene shape distinto y
# no aporta ventaja de costo aquí. Si en el futuro queremos Gemini con tools,
# habría que mapear `parts[*].function_call` ↔ OpenAI `tool_calls`.


async def _openai_chat_with_tools(
    client: AsyncOpenAI,
    model: str,
    messages: list[dict[str, Any]],
    tools: list[dict[str, Any]],
    executor: Callable[[str, dict[str, Any]], Awaitable[dict[str, Any]]],
    temperature: float,
    max_tokens: int,
    max_iterations: int,
) -> str:
    """Loop de tool-calling compatible OpenAI (sirve también para DeepSeek)."""
    # Copia local mutable de messages
    convo: list[dict[str, Any]] = list(messages)

    for iteration in range(max_iterations):
        response = await client.chat.completions.create(
            model=model,
            messages=convo,  # type: ignore[arg-type]
            tools=tools,  # type: ignore[arg-type]
            tool_choice="auto",
            temperature=temperature,
            max_tokens=max_tokens,
        )
        msg = response.choices[0].message

        tool_calls = getattr(msg, "tool_calls", None) or []

        # Sin tool_calls → respuesta final
        if not tool_calls:
            content = msg.content or ""
            return content.strip()

        # Agregar el assistant message con sus tool_calls al historial
        convo.append({
            "role": "assistant",
            "content": msg.content or "",
            "tool_calls": [
                {
                    "id": tc.id,
                    "type": "function",
                    "function": {
                        "name": tc.function.name,
                        "arguments": tc.function.arguments,
                    },
                }
                for tc in tool_calls
            ],
        })

        # Ejecutar todas las tool_calls en paralelo
        async def _run(tc: Any) -> tuple[str, str, dict[str, Any]]:
            try:
                args = json.loads(tc.function.arguments or "{}")
            except json.JSONDecodeError:
                args = {}
            try:
                result = await executor(tc.function.name, args)
            except Exception as e:
                # executor ya captura, pero por si acaso:
                log.error(
                    "llm.router.tools: executor raised",
                    name=tc.function.name,
                    error=str(e),
                )
                result = {"ok": False, "error": f"tool_exception:{e}"}
            return tc.id, tc.function.name, result

        results = await asyncio.gather(*[_run(tc) for tc in tool_calls])

        # Inyectar resultados como tool messages
        for tc_id, tc_name, result in results:
            convo.append({
                "role": "tool",
                "tool_call_id": tc_id,
                "name": tc_name,
                "content": json.dumps(result, ensure_ascii=False),
            })

        log.info(
            "llm.router.tools: iteración completada",
            iteration=iteration + 1,
            tool_calls=len(tool_calls),
        )

    # Si llegamos acá agotamos iteraciones — pedirle al LLM una respuesta final
    # sin tools (forzar texto)
    log.warning(
        "llm.router.tools: max_iterations alcanzado, forzando respuesta final",
        max_iterations=max_iterations,
    )
    final = await client.chat.completions.create(
        model=model,
        messages=convo,  # type: ignore[arg-type]
        temperature=temperature,
        max_tokens=max_tokens,
    )
    return (final.choices[0].message.content or "").strip()


async def chat_with_tools(
    messages: list[dict[str, Any]],
    tools: list[dict[str, Any]],
    ctx: dict[str, Any],
    temperature: float = 0.2,
    max_tokens: int = 800,
    max_iterations: int = 3,
) -> str:
    """
    Genera una respuesta permitiendo al LLM invocar tools (function-calling).

    Estrategia:
      1. DeepSeek (primario) usando formato OpenAI tools.
      2. Si DeepSeek falla, fallback a GPT-4o-mini con el mismo formato.

    El executor se importa lazy para evitar ciclos (tools → actions → ...).

    Args:
        messages: historial OpenAI-format (system, user, assistant).
        tools: lista de tools en formato OpenAI (ver app.agent.tools.OPENAI_TOOLS).
        ctx: contexto del runtime con `tenant_slug` y/o `tenant_id`.
        temperature: temperatura del modelo.
        max_tokens: tokens máximos por respuesta del LLM.
        max_iterations: máx número de ciclos LLM→tool→LLM (anti-loop infinito).

    Returns:
        Texto final del LLM.
    """
    # Import lazy para evitar ciclo: tools → actions → (potencialmente) router
    from app.agent.tools import execute_tool

    async def _executor(name: str, args: dict[str, Any]) -> dict[str, Any]:
        return await execute_tool(name, args, ctx)

    # --- DeepSeek (primario) ---
    try:
        return await _openai_chat_with_tools(
            client=_deepseek,
            model="deepseek-chat",
            messages=messages,
            tools=tools,
            executor=_executor,
            temperature=temperature,
            max_tokens=max_tokens,
            max_iterations=max_iterations,
        )
    except Exception as e:
        log.warning(
            "llm.router.tools: DeepSeek falló, usando GPT-4o-mini como fallback",
            error=str(e),
        )

    # --- GPT-4o-mini (fallback) ---
    return await _openai_chat_with_tools(
        client=_openai,
        model="gpt-4o-mini",
        messages=messages,
        tools=tools,
        executor=_executor,
        temperature=temperature,
        max_tokens=max_tokens,
        max_iterations=max_iterations,
    )
