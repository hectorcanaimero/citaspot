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
# distroless: sin shell, sin package manager, sin utilidades — superficie de ataque mínima
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copiar binarios compilados y migraciones SQL
COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY --from=builder /app/db ./db

# Certificados TLS para llamadas HTTPS (Evolution API, Stripe, Supabase)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 3001

# Correr como usuario no root
USER nonroot:nonroot

CMD ["./server"]
