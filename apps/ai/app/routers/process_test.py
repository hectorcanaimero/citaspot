# POST /process-test — endpoint síncrono para test del chatbot desde el dashboard.
#
# OBJETIVO: ejecutar el MISMO code path que el worker/inbound real (tool-calling,
# intent state-aware, historial en Redis, formato WhatsApp) sin publicar en
# RabbitMQ y aislando el estado en keys prefijadas con `test:`.
#
# Notas:
#   - SÍNCRONO: retorna la respuesta directamente al caller (dashboard).
#   - AISLADO: usa get_state/save_state pero con conversation_id `test:{id}` así
#     el historial y la state machine no colisionan con conversaciones WA reales.
#   - OBSERVABLE: devuelve intent_detected, tools_called, processing_time_ms,
#     chunks_count, state_after y rag_sources_used (legacy, ya no se llena).
from __future__ import annotations

import time
from typing import Any

import structlog
from fastapi import APIRouter
from pydantic import BaseModel, Field

from app.agent import history as history_module
from app.agent import tools as tools_module
from app.agent.intent import Intent
from app.agent.intent import detect as detect_intent
from app.agent.orchestrator import _handle_query
from app.agent.output import format_for_whatsapp
from app.agent.state import ConvState, get_state, reset_test_state

log = structlog.get_logger(__name__)

router = APIRouter(prefix="/process-test", tags=["process-test"])

# Tope de chars por mensaje WA — mismo que el orquestador real (CLAUDE.md §7).
_WHATSAPP_MAX_CHARS = 400


def _test_conv_id(conversation_id: str) -> str:
    """Prefija la conversation_id para aislar estado/historial de test del de prod.

    El estado real vive en `conv:{tenant_id}:{conversation_id}` y el historial
    en `conv:hist:{tenant_id}:{conversation_id}`. Al prefijar con `test:`
    obtenemos keys disjuntas sin tocar la state.py existente.
    """
    if conversation_id.startswith("test:"):
        return conversation_id
    return f"test:{conversation_id}"


class ProcessTestRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant")
    tenant_slug: str = Field(..., description="Slug del tenant")
    conversation_id: str = Field(..., description="ID de conversación de test")
    message_text: str = Field(..., max_length=2000, description="Texto del mensaje de test")
    test: bool = Field(True, description="Flag de modo test")


class ProcessTestResponse(BaseModel):
    # Campos legacy (compatibilidad con el frontend actual)
    response: str = Field("", description="Respuesta del bot")
    intent_detected: str = Field("", description="Intención detectada")
    rag_sources_used: list[str] = Field(
        default_factory=list,
        description="(legacy) Solo se llena si se invocó search_knowledge en el futuro. Hoy queda vacío.",
    )
    processing_time_ms: int = Field(0, description="Tiempo de procesamiento en ms")
    # Campos nuevos (aditivos)
    tools_called: list[str] = Field(
        default_factory=list,
        description="Tools que el LLM invocó vía function calling",
    )
    chunks_count: int = Field(
        1,
        description="Cuántos mensajes WA serían en producción (post format_for_whatsapp)",
    )
    state_after: str = Field(
        "IDLE",
        description="Estado conversacional al finalizar el turno",
    )


@router.post("", response_model=ProcessTestResponse)
async def process_test(req: ProcessTestRequest) -> ProcessTestResponse:
    """
    Procesa un mensaje de test de forma SÍNCRONA.

    Reusa el mismo code path que el worker real (intent state-aware + handler
    de QUERY con tool-calling + format_for_whatsapp), pero:
      - NO publica en RabbitMQ (la respuesta se devuelve en el body).
      - Usa keys de Redis prefijadas con `test:` para no contaminar conversaciones reales.
      - Devuelve metadata de debug (intent, tools, chunks, tiempo).

    Para BOOKING/CANCEL/RESCHEDULE devuelve un mensaje "modo test" — esos
    flujos requieren todo el state machine + Core API real, fuera de scope
    para el chatbot del dashboard.
    """
    start = time.time()
    test_conv_id = _test_conv_id(req.conversation_id)

    # 1. Cargar historial desde Redis (con prefijo test:)
    history = await history_module.get_recent(req.tenant_id, test_conv_id, limit=10)

    # 2. Cargar estado de la conversación (también con prefijo test:)
    state = await get_state(req.tenant_id, test_conv_id)
    current_state = ConvState(state.get("state", ConvState.IDLE))
    pending_data: dict[str, Any] = {
        "service_id": state.get("pending_service_id"),
        "professional_id": state.get("pending_professional_id"),
        "date": state.get("pending_date"),
        "selected_slot": state.get("pending_selected_slot"),
        "customer_name": state.get("customer_name"),
        "cancel_id": state.get("pending_cancel_id"),
        "reschedule_id": state.get("pending_reschedule_id"),
        "reschedule_date": state.get("pending_reschedule_date"),
    }

    # 3. Detectar intención con el detector state-aware
    intent = await detect_intent(
        req.message_text,
        history=history,
        conv_state=current_state,
        pending_data=pending_data,
    )
    log.info(
        "process-test: intent detectado",
        tenant=req.tenant_id,
        intent=str(intent),
        state=str(current_state),
        history_len=len(history),
    )

    # 4. Routing por intent
    # Por ahora el endpoint solo soporta QUERY (y UNKNOWN cae al mismo handler).
    # TODO: extender a BOOKING/CANCEL/RESCHEDULE cuando queramos simular el
    # state machine completo en el dashboard.
    if intent in (Intent.BOOKING, Intent.CANCEL, Intent.RESCHEDULE):
        final_text = (
            f"[modo test] Para probar el flujo de {intent.value.lower()}, "
            f"conectá un número de WhatsApp real."
        )
        chunks_count = 1
        tools_called: list[str] = []
    else:
        # QUERY / UNKNOWN / CONFIRM / MY_APPOINTMENTS / HANDOFF caen al
        # handler de QUERY: es el único que aplica tool-calling puro y no
        # exige avanzar la state machine.
        tools_module.start_tracking()
        try:
            response_text = await _handle_query(
                req.tenant_id, req.tenant_slug, req.message_text, history,
            )
        except Exception as e:  # noqa: BLE001 — endpoint de test, no debe crashear
            log.error("process-test: _handle_query falló", error=str(e))
            response_text = "Tuvimos un problema procesando tu mensaje. Intentá de nuevo."

        tools_called = tools_module.get_tracked()

        # 5. Aplicar formato WhatsApp y unir chunks para mostrar todo en
        #    una sola burbuja del dashboard. chunks_count refleja cuántos
        #    mensajes serían en producción WA real.
        chunks = format_for_whatsapp(response_text, max_chars=_WHATSAPP_MAX_CHARS)
        chunks_count = len(chunks)
        final_text = "\n\n".join(chunks)

    # 6. Persistir ambos turnos en historial (mismo prefijo test:)
    await history_module.append_message(
        req.tenant_id, test_conv_id, "user", req.message_text,
    )
    if final_text:
        await history_module.append_message(
            req.tenant_id, test_conv_id, "assistant", final_text,
        )

    elapsed_ms = int((time.time() - start) * 1000)

    return ProcessTestResponse(
        response=final_text,
        intent_detected=intent.value if intent else "UNKNOWN",
        rag_sources_used=[],  # legacy — el tool search_knowledge no se introspecciona aquí
        processing_time_ms=elapsed_ms,
        tools_called=tools_called,
        chunks_count=chunks_count,
        state_after=str(current_state.value if hasattr(current_state, "value") else current_state),
    )


@router.delete("/{tenant_id}/state")
async def clear_test_state(tenant_id: str) -> dict[str, str]:
    """Borra el estado de test de un tenant."""
    await reset_test_state(tenant_id)
    return {"status": "ok"}
