# POST /process-test — endpoint síncrono para test del chatbot desde el dashboard.
# NO publica en RabbitMQ. Usa Redis key prefix test:{tenant_id}. Retorna respuesta directa.
from __future__ import annotations

import time
from typing import Any

import structlog
from fastapi import APIRouter
from pydantic import BaseModel, Field

from app.agent.actions import get_tenant_profile, rag_query
from app.agent.intent import Intent, detect as detect_intent
from app.agent.orchestrator import _build_system_prompt
from app.agent.state import reset_test_state
from app.llm.router import chat

log = structlog.get_logger(__name__)

router = APIRouter(prefix="/process-test", tags=["process-test"])


class ProcessTestRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant")
    tenant_slug: str = Field(..., description="Slug del tenant")
    conversation_id: str = Field(..., description="ID de conversación de test")
    message_text: str = Field(..., max_length=2000, description="Texto del mensaje de test")
    test: bool = Field(True, description="Flag de modo test")


class ProcessTestResponse(BaseModel):
    response: str = Field("", description="Respuesta del bot")
    intent_detected: str = Field("", description="Intención detectada")
    rag_sources_used: list[str] = Field(default_factory=list, description="Fuentes RAG usadas")
    processing_time_ms: int = Field(0, description="Tiempo de procesamiento en ms")


@router.post("", response_model=ProcessTestResponse)
async def process_test(req: ProcessTestRequest) -> ProcessTestResponse:
    """
    Procesa un mensaje de test de forma SÍNCRONA.
    No publica en RabbitMQ. Usa estado de test separado en Redis.
    Retorna la respuesta del bot directamente.
    """
    start = time.time()

    # Detectar intención
    intent = await detect_intent(req.message_text, [])
    log.info(
        "process-test: intent detectado",
        tenant=req.tenant_id,
        intent=str(intent),
    )

    # Obtener perfil del tenant
    profile = await get_tenant_profile(req.tenant_slug)

    tone = profile.get("tone") if profile else None
    custom_instructions = profile.get("custom_instructions") if profile else None

    # RAG search
    rag_context = ""
    rag_sources: list[str] = []
    if intent in (Intent.QUERY, Intent.UNKNOWN):
        rag_context = await rag_query(req.tenant_id, req.message_text)
        if rag_context:
            for line in rag_context.split("\n---\n"):
                line = line.strip()
                if line.startswith("[") and "]" in line:
                    rag_sources.append(line[1:line.index("]")])

    # Generar respuesta
    system = _build_system_prompt(profile, rag_context, tone=tone, custom_instructions=custom_instructions)
    messages: list[dict[str, Any]] = [{"role": "system", "content": system}]
    messages.append({"role": "user", "content": req.message_text})

    response_text = await chat(messages, temperature=0.3)

    elapsed_ms = int((time.time() - start) * 1000)

    return ProcessTestResponse(
        response=response_text,
        intent_detected=str(intent.value) if intent else "unknown",
        rag_sources_used=rag_sources,
        processing_time_ms=elapsed_ms,
    )


@router.delete("/{tenant_id}/state")
async def clear_test_state(tenant_id: str) -> dict[str, str]:
    """Borra el estado de test de un tenant."""
    await reset_test_state(tenant_id)
    return {"status": "ok"}
