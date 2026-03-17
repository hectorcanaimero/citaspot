# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Instalar certificados TLS (necesarios para llamadas HTTPS externas)
RUN apk add --no-cache ca-certificates

# Descargar dependencias primero para aprovechar el cache de Docker
COPY go.mod go.sum ./
RUN go mod download

# Compilar con optimizaciones: strip symbols (-s -w) reduce el tamaño del binario
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o migrate ./cmd/migrate

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# wget y ca-certificates para healthcheck y llamadas HTTPS
RUN apk add --no-cache ca-certificates wget \
 && addgroup -S appgroup \
 && adduser -S appuser -G appgroup

# Copiar binarios compilados y migraciones SQL
COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY --from=builder /app/db ./db

EXPOSE 3001

# Correr como usuario no root
USER appuser

CMD ["./server"]
