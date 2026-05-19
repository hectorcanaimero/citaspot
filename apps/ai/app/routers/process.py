# POST /process — endpoint interno llamado por el Core API para procesar mensajes WA.
# En producción, el procesamiento ocurre vía RabbitMQ; este endpoint es para testing/debug.
from __future__ import annotations

from fastapi import APIRouter
from pydantic import BaseModel, Field

from app.agent import history as history_module
from app.agent.orchestrator import process_message

router = APIRouter(prefix="/process", tags=["process"])


class ProcessRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant")
    tenant_slug: str = Field(..., description="Slug del tenant para llamadas al Core API")
    conversation_id: str = Field(..., description="UUID de la conversación")
    wa_phone: str = Field(..., description="Número WhatsApp del cliente")
    message_text: str = Field(..., max_length=2000, description="Texto del mensaje")
    # Deprecated: el historial ahora vive en Redis (ver app.agent.history).
    # Campo aceptado por compatibilidad pero ignorado.
    history: list[dict] = Field(default_factory=list, description="DEPRECATED — ignorado")


class ProcessResponse(BaseModel):
    status: str = "accepted"


async def _process_and_persist(
    tenant_id: str,
    tenant_slug: str,
    conversation_id: str,
    wa_phone: str,
    message_text: str,
) -> None:
    """Lee historial de Redis, procesa el mensaje y persiste ambos turnos."""
    history = await history_module.get_recent(tenant_id, conversation_id, limit=10)
    reply = await process_message(
        tenant_id=tenant_id,
        tenant_slug=tenant_slug,
        conversation_id=conversation_id,
        wa_phone=wa_phone,
        message_text=message_text,
        history=history,
    )
    await history_module.append_message(tenant_id, conversation_id, "user", message_text)
    if reply:
        await history_module.append_message(
            tenant_id, conversation_id, "assistant", reply
        )


@router.post("", response_model=ProcessResponse)
async def process(req: ProcessRequest) -> ProcessResponse:
    """
    Procesa un mensaje entrante de WhatsApp de forma asíncrona.
    La respuesta se publica en wa.messages.outbound vía RabbitMQ.
    Siempre retorna 202 Accepted — el resultado llega al cliente por WA.

    Nota: el campo `history` del request se acepta por compatibilidad pero
    se ignora. El historial canónico vive en Redis (app.agent.history).
    """
    import asyncio
    asyncio.create_task(
        _process_and_persist(
            tenant_id=req.tenant_id,
            tenant_slug=req.tenant_slug,
            conversation_id=req.conversation_id,
            wa_phone=req.wa_phone,
            message_text=req.message_text,
        )
    )
    return ProcessResponse()
