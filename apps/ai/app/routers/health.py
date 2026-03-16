# Router de health check — sin dependencias externas.
from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health_check() -> dict[str, str]:
    """Endpoint de verificación de salud del servicio AI."""
    return {"status": "ok", "service": "citaspot-ai"}
