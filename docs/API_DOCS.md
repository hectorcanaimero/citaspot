# Documentación de APIs — CitaSpot

## Core API (Go)

La API REST del Core está documentada con **OpenAPI 3**. Tienes dos interfaces:

| Interfaz     | URL                    | Uso principal                    |
|-------------|-------------------------|----------------------------------|
| **Swagger UI** | `http://localhost:3001/swagger` | Probar endpoints (Try it out)   |
| **ReDoc**      | `http://localhost:3001/redoc`   | Leer documentación en formato guía |

- **Spec en bruto:** `http://localhost:3001/openapi.json` (para clientes o herramientas).
- La spec se mantiene en `apps/api/docs/swagger.json`. Si añades rutas o cambias contratos, actualiza ese archivo.

En producción sustituye `localhost:3001` por la URL base de tu API.

---

## AI Service (Python / FastAPI)

El servicio AI expone una API HTTP **interna** (procesamiento, vectorizado, búsqueda). FastAPI genera OpenAPI y dos UIs por defecto:

| Interfaz     | URL (puerto 8001)   | Uso principal                    |
|-------------|---------------------|------------------------------------|
| **Swagger UI** | `http://localhost:8001/docs`   | Probar endpoints                  |
| **ReDoc**      | `http://localhost:8001/redoc`   | Leer documentación                |

- **Spec en bruto:** `http://localhost:8001/openapi.json`.

Endpoints principales del AI Service:

- `GET /health` — Health check.
- `POST /process` — Procesar mensaje WA (testing; en producción se usa RabbitMQ).
- `POST /vectorize` — Vectorizar documento de knowledge base.
- `POST /search` — Búsqueda semántica en la knowledge base.

El procesamiento normal de WhatsApp va por colas (Core API → RabbitMQ → AI Service), no por HTTP.
