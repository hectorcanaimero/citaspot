# Publisher de RabbitMQ para wa.messages.outbound y knowledge.vectorize.
from __future__ import annotations

import json
import logging

import aio_pika

from app.core.config import settings

log = logging.getLogger(__name__)

_connection: aio_pika.Connection | None = None
_channel: aio_pika.Channel | None = None


async def get_channel() -> aio_pika.Channel:
    global _connection, _channel
    if _connection is None or _connection.is_closed:
        _connection = await aio_pika.connect_robust(settings.rabbitmq_url)
    if _channel is None or _channel.is_closed:
        _channel = await _connection.channel()
    return _channel


async def publish(queue_name: str, payload: dict) -> None:
    """Publica un mensaje JSON en la cola especificada."""
    try:
        ch = await get_channel()
        await ch.declare_queue(queue_name, durable=True)
        await ch.default_exchange.publish(
            aio_pika.Message(
                body=json.dumps(payload).encode(),
                delivery_mode=aio_pika.DeliveryMode.PERSISTENT,
                content_type="application/json",
            ),
            routing_key=queue_name,
        )
    except Exception as e:
        log.error("rabbitmq.publish error queue=%s: %s", queue_name, e)
        raise


async def close() -> None:
    global _connection
    if _connection and not _connection.is_closed:
        await _connection.close()
