# ── Stage 1: Builder ──────────────────────────────────────────────────────────
# Compilar wheels en el builder para no necesitar gcc en runtime
FROM python:3.12-slim AS builder

WORKDIR /app

RUN pip install --upgrade pip

COPY requirements.txt .
RUN pip wheel --no-cache-dir --wheel-dir /wheels -r requirements.txt

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM python:3.12-slim

WORKDIR /app

# Instalar wheels pre-compiladas (sin gcc ni herramientas de build)
COPY --from=builder /wheels /wheels
RUN pip install --no-cache /wheels/* && rm -rf /wheels

# Copiar código de la aplicación
COPY . .

# Crear usuario no root
RUN addgroup --system app && adduser --system --ingroup app app
USER app

EXPOSE 8001

# 2 workers en producción; ajustar según CPU disponible
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8001", "--workers", "2"]
