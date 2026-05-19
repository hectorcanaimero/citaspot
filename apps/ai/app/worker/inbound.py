# Consumer RabbitMQ para wa.messages.inbound.
# Cada mensaje dispara el orquestador del asistente IA.
from __future__ import annotations

import asyncio
import json

import aio_pika
import structlog
from aio_pika.abc import AbstractIncomingMessage

from app.agent import history as history_module
from app.agent.orchestrator import process_message
from app.core.config import settings

log = structlog.get_logger(__name__)

_QUEUE_NAME = "wa.messages.inbound"


async def _process(message: AbstractIncomingMessage) -> None:
    """Procesa un mensaje entrante del queue y llama al orquestador."""
    async with message.process(requeue=True):
        try:
            payload = json.loads(message.body)
            tenant_id = payload["tenant_id"]
            conversation_id = payload["conversation_id"]
            message_text = payload["content"]

            log.info(
                "worker.inbound: recibido",
                tenant=tenant_id,
                conv=conversation_id,
            )

            # El historial es propiedad del AI service: lo leemos de Redis y
            # NO confiamos en lo que envíe el Core API en el payload.
            history = await history_module.get_recent(
                tenant_id, conversation_id, limit=10
            )
            log.info(
                "worker.inbound: history_loaded_from_redis",
                count=len(history),
                tenant_id=str(tenant_id),
            )

            reply = await process_message(
                tenant_id=tenant_id,
                tenant_slug=payload["tenant_slug"],
                conversation_id=conversation_id,
                wa_phone=payload["wa_phone"],
                message_text=message_text,
                history=history,
            )

            # Persistir AMBOS turnos en orden cronológico:
            #   1) mensaje del usuario (después de process_message para que
            #      el LLM haya visto el historial SIN el turno actual)
            #   2) respuesta del asistente (si la hubo)
            await history_module.append_message(
                tenant_id, conversation_id, "user", message_text
            )
            if reply:
                await history_module.append_message(
                    tenant_id, conversation_id, "assistant", reply
                )
        except json.JSONDecodeError as e:
            log.error("worker.inbound: JSON inválido", error=str(e))
            # No reencolar — mensaje malformado
            await message.nack(requeue=False)
        except KeyError as e:
            log.error("worker.inbound: campo requerido faltante", field=str(e))
            await message.nack(requeue=False)
        except Exception as e:
            log.exception("worker.inbound: error procesando mensaje", error=str(e))
            # Reencolar para retry (lo maneja el context manager con requeue=True)
            raise


async def start_consumer() -> None:
    """
    Inicia el consumer de RabbitMQ con reconexión automática.
    Llamar como task en el startup de FastAPI.
    """
    delays = [1, 2, 4, 8, 16]
    idx = 0

    while True:
        try:
            connection = await aio_pika.connect_robust(settings.rabbitmq_url)
            async with connection:
                channel = await connection.channel()
                await channel.set_qos(prefetch_count=1)  # 1 mensaje a la vez
                queue = await channel.declare_queue(_QUEUE_NAME, durable=True)

                log.info("worker.inbound: consumiendo", queue=_QUEUE_NAME)
                idx = 0  # reset backoff al reconectar exitosamente

                async with queue.iterator() as queue_iter:
                    async for message in queue_iter:
                        await _process(message)

        except asyncio.CancelledError:
            log.info("worker.inbound: detenido")
            return
        except Exception as e:
            d = delays[min(idx, len(delays) - 1)]
            log.error("worker.inbound: error de conexión", error=str(e), delay_s=d)
            await asyncio.sleep(d)
            idx += 1
