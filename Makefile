# Makefile — CitaSpot
# Uso: make <comando>

.PHONY: dev dev-down test test-api test-ai test-web \
        migrate migrate-down migrate-status db-reset lint gen seed logs clean \
        build deploy help

# ── Colores para output ────────────────────────────────────
BLUE   = \033[0;34m
GREEN  = \033[0;32m
YELLOW = \033[0;33m
RED    = \033[0;31m
NC     = \033[0m

# ── Development ────────────────────────────────────────────

dev: ## Levantar todos los servicios en modo desarrollo
	@echo "$(BLUE)▶ Iniciando CitaSpot en modo desarrollo...$(NC)"
	docker compose up -d
	@echo "$(GREEN)✅ Servicios corriendo:$(NC)"
	@echo "   Web:        http://localhost:3000"
	@echo "   API:        http://localhost:3001"
	@echo "   API Docs:   http://localhost:3001/swagger  (ReDoc: http://localhost:3001/redoc)"
	@echo "   AI:         http://localhost:8001"
	@echo "   Evolution:  http://localhost:8080"
	@echo "   RabbitMQ:   http://localhost:15672 (citaspot/citaspot_dev)"

dev-down: ## Detener todos los servicios
	@echo "$(YELLOW)▶ Deteniendo servicios...$(NC)"
	docker compose down

dev-restart: ## Reiniciar servicios SIN perder datos (down + up)
	@echo "$(BLUE)▶ Reiniciando servicios (los datos se conservan)...$(NC)"
	docker compose down
	docker compose up -d
	@echo "$(GREEN)✅ Servicios reiniciados$(NC)"

dev-reset: ## ⚠️  Detener, limpiar volúmenes y volver a levantar (BORRA DATOS)
	@echo "$(RED)▶ Reset completo del ambiente...$(NC)"
	docker compose down -v
	$(MAKE) dev
	$(MAKE) migrate
	$(MAKE) seed

logs: ## Ver logs de todos los servicios (Ctrl+C para salir)
	docker compose logs -f

logs-api: ## Ver logs solo del Core API
	docker compose logs -f api

logs-ai: ## Ver logs solo del AI Service
	docker compose logs -f ai

# ── Testing ────────────────────────────────────────────────

test: test-api test-ai test-web ## Correr todos los tests
	@echo "$(GREEN)✅ Todos los tests pasaron$(NC)"

test-api: ## Tests del Core API (Go)
	@echo "$(BLUE)▶ Corriendo tests Go...$(NC)"
	cd apps/api && go test ./... -v -cover -coverprofile=coverage.out
	cd apps/api && go tool cover -html=coverage.out -o coverage.html

test-ai: ## Tests del AI Service (Python)
	@echo "$(BLUE)▶ Corriendo tests Python...$(NC)"
	cd apps/ai && python -m pytest tests/ -v --tb=short

test-web: ## Tests del frontend (Next.js)
	@echo "$(BLUE)▶ Corriendo tests Next.js...$(NC)"
	cd apps/web && npm run test

test-integration: ## Tests de integración (requiere DB corriendo)
	@echo "$(BLUE)▶ Corriendo tests de integración...$(NC)"
	cd apps/api && go test ./... -tags=integration -v

# ── Database ────────────────────────────────────────────────

migrate: ## Aplicar todas las migraciones pendientes
	@echo "$(BLUE)▶ Aplicando migraciones...$(NC)"
	cd apps/api && go run cmd/migrate/main.go up
	@echo "$(GREEN)✅ Migraciones aplicadas$(NC)"

migrate-down: ## Revertir la última migración
	@echo "$(YELLOW)▶ Revirtiendo última migración...$(NC)"
	cd apps/api && go run cmd/migrate/main.go down 1

migrate-status: ## Ver estado de migraciones
	cd apps/api && go run cmd/migrate/main.go status

db-reset: ## ⚠️  Resetear DB a estado de fábrica (DROP SCHEMA + re-migrate). BORRA TODO.
	@bash scripts/db-reset.sh

seed: ## Insertar datos de prueba en desarrollo
	@echo "$(BLUE)▶ Insertando datos de prueba...$(NC)"
	cd apps/api && go run cmd/seed/main.go
	@echo "$(GREEN)✅ Datos de prueba insertados$(NC)"
	@echo "   Tenant: test-salon (slug: test-salon)"
	@echo "   Usuario: admin@test.com / password: test123"

# ── Code Quality ────────────────────────────────────────────

lint: ## Correr linters en todos los servicios
	@echo "$(BLUE)▶ Lint Go...$(NC)"
	cd apps/api && golangci-lint run ./...
	@echo "$(BLUE)▶ Lint Python...$(NC)"
	cd apps/ai && ruff check .
	@echo "$(BLUE)▶ Lint TypeScript...$(NC)"
	cd apps/web && npm run lint
	@echo "$(GREEN)✅ Sin errores de lint$(NC)"

format: ## Formatear código en todos los servicios
	cd apps/api && gofmt -w .
	cd apps/ai && ruff format .
	cd apps/web && npm run format

# ── Code Generation ─────────────────────────────────────────

gen: gen-sqlc gen-mocks gen-types ## Generar todo el código auto-generado

gen-sqlc: ## Generar código Go desde queries SQL (sqlc)
	@echo "$(BLUE)▶ Generando código sqlc...$(NC)"
	cd apps/api && sqlc generate

gen-mocks: ## Generar mocks con mockery (Go)
	@echo "$(BLUE)▶ Generando mocks...$(NC)"
	cd apps/api && go generate ./...

gen-types: ## Generar TypeScript types desde el schema (para frontend)
	@echo "$(BLUE)▶ Generando TypeScript types...$(NC)"
	cd packages/shared-types && npm run generate

# ── Build ────────────────────────────────────────────────────

build: ## Build de producción de todos los servicios
	@echo "$(BLUE)▶ Building para producción...$(NC)"
	docker compose -f docker-compose.prod.yml build

# ── Utils ────────────────────────────────────────────────────

clean: ## ⚠️  BORRA TODOS LOS DATOS — containers, volúmenes y archivos generados
	@echo "$(RED)⚠️  ADVERTENCIA: Esto borrará TODA la base de datos, sesiones de WhatsApp y datos.$(NC)"
	@echo "$(RED)   Usa 'make dev-down' para solo detener los servicios sin perder datos.$(NC)"
	@printf "$(YELLOW)¿Continuar? Escribe 'si' para confirmar: $(NC)" && read ans && [ "$$ans" = "si" ]
	docker compose down -v --remove-orphans
	cd apps/api && rm -f coverage.out coverage.html
	cd apps/web && rm -rf .next

install: ## Instalar todas las dependencias
	cd apps/api && go mod download
	cd apps/ai && pip install -r requirements.txt -r requirements-dev.txt
	cd apps/web && npm install
	cd packages/shared-types && npm install

help: ## Mostrar esta ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "$(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
