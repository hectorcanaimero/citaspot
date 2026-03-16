# Runbook de Operaciones — CitaSpot

> Guía de referencia rápida para operaciones de producción.
> Audiencia: desarrolladores y DevOps con acceso al servidor.

---

## Índice

1. [Información de acceso](#1-información-de-acceso)
2. [Deploy](#2-deploy)
3. [Rollback](#3-rollback)
4. [Backups](#4-backups)
5. [Incidentes comunes](#5-incidentes-comunes)
6. [Escalar servicios](#6-escalar-servicios)
7. [Rotar secretos](#7-rotar-secretos)
8. [Mantenimiento de base de datos](#8-mantenimiento-de-base-de-datos)
9. [Monitoreo](#9-monitoreo)
10. [Contactos de emergencia](#10-contactos-de-emergencia)

---

## 1. Información de acceso

| Recurso | URL | Notas |
|---------|-----|-------|
| API Health | `https://api.citaspot.com/api/v1/health` | público |
| API Health Ready | `https://api.citaspot.com/api/v1/health/ready` | verifica DB + Redis |
| AI Health | `https://ai.citaspot.com/health` | interno |
| Grafana | `https://grafana.citaspot.com` | requiere login |
| Coolify | `https://coolify.citaspot.com` | panel de deploy |
| Supabase | `https://app.supabase.com` | auth + dashboard |
| RabbitMQ | interno, puerto 15672 solo en dev | |

**Logs en producción:**
```bash
# Ver logs de todos los servicios
docker compose -f docker-compose.prod.yml logs -f --tail=100

# Por servicio
docker compose -f docker-compose.prod.yml logs -f api
docker compose -f docker-compose.prod.yml logs -f ai
docker compose -f docker-compose.prod.yml logs -f web
docker compose -f docker-compose.prod.yml logs -f postgres
```

---

## 2. Deploy

### Deploy normal (automático)

1. Hacer merge a `main` en GitHub
2. GitHub Actions ejecuta CI (tests → build → push a GHCR)
3. Webhook dispara deploy en Coolify
4. Coolify descarga las imágenes `:latest` y reinicia los servicios

**Verificar que el deploy fue exitoso:**
```bash
# Health check del API
curl -s https://api.citaspot.com/api/v1/health/ready | jq .

# Respuesta esperada:
# { "status": "ok", "db": "ok", "redis": "ok" }

# Ver imagen corriendo (SHA debe coincidir con el último commit)
docker inspect citaspot-api --format '{{.Config.Image}}'
```

### Deploy manual (emergencia)

```bash
# En el servidor de producción
cd /opt/citaspot

# Pull de las imágenes más recientes
docker compose -f docker-compose.prod.yml pull api web ai

# Reiniciar solo los servicios de aplicación (sin tocar DB/Redis/RabbitMQ)
docker compose -f docker-compose.prod.yml up -d api web ai

# Verificar que levantaron
docker compose -f docker-compose.prod.yml ps
```

### Aplicar migraciones en producción

```bash
# Siempre hacer backup ANTES de migrar
./infra/backup/backup.sh

# Aplicar migraciones
docker compose -f docker-compose.prod.yml run --rm api /app/migrate up

# Verificar estado
docker compose -f docker-compose.prod.yml run --rm api /app/migrate status
```

---

## 3. Rollback

### Rollback de aplicación (volver a imagen anterior)

```bash
# 1. Encontrar el SHA del commit anterior estable
# En GitHub → Actions → buscar el último deploy exitoso → copiar el SHA

# 2. Editar docker-compose.prod.yml o setear en .env:
# CITASPOT_API_IMAGE=ghcr.io/TU_ORG/citaspot-api:SHA_ANTERIOR

# 3. Redeployar
docker compose -f docker-compose.prod.yml up -d api web ai

# 4. Verificar health
curl -s https://api.citaspot.com/api/v1/health/ready | jq .
```

### Rollback de migración de base de datos

```bash
# ADVERTENCIA: Hacer backup primero
./infra/backup/backup.sh

# Revertir la última migración
docker compose -f docker-compose.prod.yml run --rm api /app/migrate down 1

# Verificar estado
docker compose -f docker-compose.prod.yml run --rm api /app/migrate status
```

---

## 4. Backups

### Backup manual inmediato

```bash
cd /opt/citaspot

# Cargar variables de entorno
source .env

# Ejecutar backup
./infra/backup/backup.sh
```

### Configurar cron de backup automático

```bash
# Editar crontab del servidor
crontab -e

# Añadir línea: backup todos los días a las 3am UTC
0 3 * * * cd /opt/citaspot && source .env && ./infra/backup/backup.sh >> /var/log/citaspot-backup.log 2>&1
```

### Verificar backups en S3/R2

```bash
source .env
aws s3 ls s3://${S3_BUCKET}/backups/ \
    --endpoint-url "${S3_ENDPOINT_URL}" \
    --human-readable \
    --summarize \
    | tail -20
```

### Restaurar desde backup

```bash
# Ver backups disponibles y restaurar el más reciente
./infra/backup/restore.sh

# Restaurar un backup específico
./infra/backup/restore.sh citaspot_20260314_030000
```

---

## 5. Incidentes comunes

### API caída (5xx o sin respuesta)

```bash
# 1. Verificar que el contenedor esté corriendo
docker compose -f docker-compose.prod.yml ps api

# 2. Ver últimas líneas de logs
docker compose -f docker-compose.prod.yml logs --tail=50 api

# 3. Health check detallado
curl -v https://api.citaspot.com/api/v1/health/ready

# 4. Verificar conectividad con DB
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "SELECT 1;"

# 5. Verificar Redis
docker compose -f docker-compose.prod.yml exec redis redis-cli PING

# 6. Si todo parece bien, reiniciar el servicio
docker compose -f docker-compose.prod.yml restart api
```

### WhatsApp desconectado

```bash
# 1. Verificar estado en Evolution API
curl -s -H "apikey: ${EVOLUTION_API_KEY}" \
    "${EVOLUTION_API_URL}/instance/fetchInstances" | jq '.[].state'

# Estado esperado: "open"
# Si dice "close" o "connecting": reconectar

# 2. Reconectar instancia (genera nuevo QR)
curl -s -X DELETE -H "apikey: ${EVOLUTION_API_KEY}" \
    "${EVOLUTION_API_URL}/instance/logout/NOMBRE_INSTANCIA"

curl -s -X GET -H "apikey: ${EVOLUTION_API_KEY}" \
    "${EVOLUTION_API_URL}/instance/connect/NOMBRE_INSTANCIA"

# 3. Escanear QR desde el dashboard en:
#    https://tu-dominio.com/dashboard/whatsapp
```

### LLM timeout / IA no responde

```bash
# 1. Verificar logs del AI service
docker compose -f docker-compose.prod.yml logs --tail=50 ai

# 2. Verificar quota de Gemini
# → https://console.cloud.google.com/apis/api/generativelanguage/quotas

# 3. Verificar quota de OpenAI (fallback)
# → https://platform.openai.com/usage

# 4. El LLM Router cambia a GPT-4o-mini si Gemini tarda >5s
# Si ambos fallan, el asistente responde con mensaje de disculpa genérico

# 5. Verificar que RabbitMQ está procesando mensajes
curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" \
    http://localhost:15672/api/queues/%2F/wa.messages.inbound | jq '.messages_ready'

# 6. Si hay mensajes acumulados, reiniciar el AI service
docker compose -f docker-compose.prod.yml restart ai
```

### Base de datos lenta

```bash
# 1. Ver queries activas
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    SELECT pid, now() - pg_stat_activity.query_start AS duration,
           query, state
    FROM pg_stat_activity
    WHERE state != 'idle'
    ORDER BY duration DESC
    LIMIT 10;"

# 2. Ver queries lentas (requiere pg_stat_statements)
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    SELECT query, calls, mean_exec_time, total_exec_time
    FROM pg_stat_statements
    ORDER BY mean_exec_time DESC
    LIMIT 10;"

# 3. Terminar una query específica (pid del paso 1)
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "SELECT pg_terminate_backend(PID);"

# 4. Ver locks
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    SELECT l.locktype, l.relation::regclass, l.mode, l.granted, a.query
    FROM pg_locks l
    JOIN pg_stat_activity a ON l.pid = a.pid
    WHERE NOT l.granted;"
```

### Inspeccionar un tenant específico (sin cruzar RLS)

```bash
# IMPORTANTE: Siempre setear app.tenant_id antes de cualquier query
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    SET app.tenant_id = 'UUID_DEL_TENANT';
    SELECT id, name, plan, status FROM tenants WHERE id = 'UUID_DEL_TENANT';"

# Buscar tenant por slug
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    SELECT id, name, slug, plan, status FROM tenants WHERE slug = 'nombre-del-negocio';"
```

### RabbitMQ — cola acumulada

```bash
# Ver estado de todas las colas
curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" \
    http://localhost:15672/api/queues | jq '.[] | {name, messages_ready, consumers}'

# Si una cola tiene mensajes pero 0 consumers → el worker está caído
# Reiniciar el servicio correspondiente:
docker compose -f docker-compose.prod.yml restart ai   # wa.messages.inbound, knowledge.vectorize
docker compose -f docker-compose.prod.yml restart api  # wa.messages.outbound, notifications.*
```

---

## 6. Escalar servicios

En Coolify, el escalado se hace desde el panel UI ajustando réplicas.

Para escalar manualmente en el servidor:

```bash
# Escalar el API a 3 réplicas (requiere load balancer configurado)
docker compose -f docker-compose.prod.yml up -d --scale api=3

# Nota: Solo escalar api y ai — postgres, redis, rabbitmq son servicios únicos
```

**Límites de recursos (docker-compose.prod.yml):**
| Servicio | Memoria máx | CPU máx |
|----------|-------------|---------|
| postgres | 512 MB | 1.0 |
| redis | 192 MB | 0.5 |
| rabbitmq | 256 MB | 0.5 |
| api | 256 MB | 1.0 |
| ai | 512 MB | 1.0 |
| web | 256 MB | 0.5 |

Ajustar en `docker-compose.prod.yml` si hay OOM killers.

---

## 7. Rotar secretos

### Rotar JWT_SECRET (sin downtime)

1. Generar nuevo secreto: `openssl rand -hex 64`
2. Actualizar `.env` con el nuevo valor
3. Reiniciar el API: `docker compose -f docker-compose.prod.yml restart api`
4. **Efecto**: Todos los tokens existentes quedan inválidos → usuarios deben re-login

```bash
# Generar nuevo JWT_SECRET
openssl rand -hex 64

# Actualizar en .env
# JWT_SECRET=NUEVO_VALOR

# Reiniciar
docker compose -f docker-compose.prod.yml restart api
```

### Rotar API keys de LLM

```bash
# 1. Generar nueva key en Gemini/OpenAI console
# 2. Actualizar .env
# GEMINI_API_KEY=NUEVA_KEY

# 3. Reiniciar AI service
docker compose -f docker-compose.prod.yml restart ai

# 4. Verificar en logs que la nueva key funciona
docker compose -f docker-compose.prod.yml logs -f ai | head -20
```

### Rotar Evolution API Key

```bash
# 1. Generar nueva key en Evolution API
# 2. Actualizar .env
# EVOLUTION_API_KEY=NUEVA_KEY

# 3. Reiniciar API (usa la key para llamar a Evolution)
docker compose -f docker-compose.prod.yml restart api

# 4. Verificar webhook sigue funcionando
# Enviar mensaje de prueba por WhatsApp y verificar logs
```

---

## 8. Mantenimiento de base de datos

### VACUUM y estadísticas (ejecutar mensualmente)

```bash
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    VACUUM ANALYZE appointments;
    VACUUM ANALYZE customers;
    VACUUM ANALYZE knowledge_documents;
    VACUUM ANALYZE conversations;"
```

### Reindexar pgvector (si búsqueda semántica está lenta)

```bash
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    REINDEX INDEX CONCURRENTLY idx_knowledge_embedding;"
```

### Limpiar Redis — sesiones expiradas

Redis expira TTLs automáticamente. Para forzar limpieza:

```bash
docker compose -f docker-compose.prod.yml exec redis \
    redis-cli --scan --pattern "conv:*" | xargs redis-cli del
```

**Precaución**: Esto elimina todas las conversaciones activas. Solo en mantenimiento programado.

### Ver tamaño de tablas

```bash
docker compose -f docker-compose.prod.yml exec postgres \
    psql -U "${POSTGRES_USER}" -d citaspot -c "
    SELECT schemaname, tablename,
           pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"
```

---

## 9. Monitoreo

### Grafana — dashboards clave

| Dashboard | URL | Qué mirar en incidente |
|-----------|-----|------------------------|
| SaaS Overview | `/d/saas-overview` | Tenants activos, revenue, uptime |
| Appointments | `/d/appointments` | Citas creadas, tasa de error |
| WhatsApp & AI | `/d/whatsapp-ai` | Mensajes procesados, latencia LLM |
| Customers | `/d/customers` | Usuarios nuevos, churn |

### Health checks rápidos

```bash
# API completa
curl -s https://api.citaspot.com/api/v1/health/ready | jq .

# AI service (interno)
curl -s http://localhost:8000/health | jq .

# Postgres
docker compose -f docker-compose.prod.yml exec postgres \
    pg_isready -U "${POSTGRES_USER}" -d citaspot

# Redis
docker compose -f docker-compose.prod.yml exec redis redis-cli PING

# RabbitMQ
curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" \
    http://localhost:15672/api/healthchecks/node | jq .status
```

### Script de check de salud global

```bash
#!/usr/bin/env bash
# Ejecutar para verificar que todo está verde antes/después de un deploy

echo "=== CitaSpot Health Check ==="
echo ""

API=$(curl -sf https://api.citaspot.com/api/v1/health/ready 2>/dev/null | jq -r .status)
echo "API:      ${API:-ERROR}"

POSTGRES=$(docker compose -f docker-compose.prod.yml exec -T postgres \
    pg_isready -U "${POSTGRES_USER}" -d citaspot -q 2>/dev/null && echo "ok" || echo "ERROR")
echo "Postgres: ${POSTGRES}"

REDIS=$(docker compose -f docker-compose.prod.yml exec -T redis redis-cli PING 2>/dev/null)
echo "Redis:    ${REDIS:-ERROR}"

RMQ=$(curl -sf -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" \
    http://localhost:15672/api/healthchecks/node 2>/dev/null | jq -r .status)
echo "RabbitMQ: ${RMQ:-ERROR}"

echo ""
echo "=== Done ==="
```

---

## 10. Contactos de emergencia

> Completar con los datos reales del equipo antes del launch.

| Rol | Nombre | Canal |
|-----|--------|-------|
| On-call técnico | — | WhatsApp: +X |
| Evolution API support | — | https://github.com/EvolutionAPI/evolution-api/issues |
| Supabase support | — | https://supabase.com/dashboard/support |
| Gemini/Google Cloud | — | https://cloud.google.com/support |
| Stripe support | — | https://support.stripe.com |
| Cloudflare R2 | — | https://community.cloudflare.com |

---

*Runbook generado para CitaSpot v1.0 — actualizar con cada cambio significativo de infraestructura.*
