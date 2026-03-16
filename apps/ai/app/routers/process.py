# POST /process — endpoint interno llamado por el Core API para procesar mensajes WA.
# En producción, el procesamiento ocurre vía RabbitMQ; este endpoint es para testing/debug.
from __future__ import annotations

from pydantic import BaseModel, Field

from fastapi import APIRouter

from app.agent.orchestrator import process_message

router = APIRouter(prefix="/process", tags=["process"])


class ProcessRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant")
    tenant_slug: str = Field(..., description="Slug del tenant para llamadas al Core API")
    conversation_id: str = Field(..., description="UUID de la conversación")
    wa_phone: str = Field(..., description="Número WhatsApp del cliente")
    message_text: str = Field(..., max_length=2000, description="Texto del mensaje")
    history: list[dict] = Field(default_factory=list, description="Historial de mensajes")


class ProcessResponse(BaseModel):
    status: str = "accepted"


@router.post("", response_model=ProcessResponse)
async def process(req: ProcessRequest) -> ProcessResponse:
    """
    Procesa un mensaje entrante de WhatsApp de forma asíncrona.
    La respuesta se publica en wa.messages.outbound vía RabbitMQ.
    Siempre retorna 202 Accepted — el resultado llega al cliente por WA.
    """
    import asyncio
    asyncio.create_task(
        process_message(
            tenant_id=req.tenant_id,
            tenant_slug=req.tenant_slug,
            conversation_id=req.conversation_id,
            wa_phone=req.wa_phone,
            message_text=req.message_text,
            history=req.history,
        )
    )
    return ProcessResponse()
