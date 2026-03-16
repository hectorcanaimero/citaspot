# Entrypoint del AI Service de CitaSpot.
# Setup, registro de routers y startup/shutdown de recursos.
import asyncio

import structlog
import uvicorn
from fastapi import FastAPI

from app.core.config import settings
from app.core.logging_config import configure_logging
from app.core.database import close_pool
from app.core.rabbitmq import close as close_rabbitmq
from app.core.redis import close_redis
from app.routers import health
from app.routers import process, vectorize, search

configure_logging(env=settings.app_env)
log = structlog.get_logger(__name__)

app = FastAPI(
    title=settings.app_name,
    debug=settings.debug,
)

# Registrar routers
app.include_router(health.router)
app.include_router(process.router)
app.include_router(vectorize.router)
app.include_router(search.router)

# Tarea background del consumer RabbitMQ
_consumer_task: asyncio.Task | None = None


@app.on_event("startup")
async def startup() -> None:
    """Inicia los consumers de RabbitMQ en background al arrancar el servicio."""
    global _consumer_task
    from app.worker.inbound import start_consumer as start_wa_consumer
    from app.worker.knowledge import start_consumer as start_knowledge_consumer
    _consumer_task = asyncio.create_task(start_wa_consumer())
    asyncio.create_task(start_knowledge_consumer())
    log.info("AI Service iniciado — consumers RabbitMQ activos (wa.inbound + knowledge.vectorize)")


@app.on_event("shutdown")
async def shutdown() -> None:
    """Cierra conexiones de forma ordenada al detener el servicio."""
    global _consumer_task
    if _consumer_task:
        _consumer_task.cancel()
        try:
            await _consumer_task
        except asyncio.CancelledError:
            pass

    await close_rabbitmq()
    await close_redis()
    await close_pool()
    log.info("AI Service detenido — conexiones cerradas")


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8001,
        reload=True,
    )
