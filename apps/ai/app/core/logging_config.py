"""Configuración centralizada de logging con structlog.

En producción (APP_ENV=production): JSON estructurado.
En desarrollo: texto con colores legible para humanos.
"""
import logging
import sys

import structlog


def configure_logging(env: str = "development") -> None:
    """Configura structlog como logger global del servicio AI.

    Llama una sola vez al inicio de la aplicación, antes de importar
    cualquier módulo que use logging.
    """
    shared_processors: list[structlog.types.Processor] = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
    ]

    if env == "production":
        renderer: structlog.types.Processor = structlog.processors.JSONRenderer()
    else:
        renderer = structlog.dev.ConsoleRenderer(colors=True)

    structlog.configure(
        processors=shared_processors + [renderer],
        wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(sys.stdout),
        cache_logger_on_first_use=True,
    )

    # Redirigir stdlib logging → structlog para módulos que usen logging.getLogger()
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=logging.INFO,
    )
