#!/usr/bin/env bash
# restore.sh — Restauración de backup de PostgreSQL para AgendAI
#
# Uso:
#   ./restore.sh                         # restaura el backup más reciente
#   ./restore.sh agendai_20260314_030000  # restaura un backup específico (sin .dump)
#
# Variables de entorno requeridas:
#   DATABASE_URL          — postgresql://user:pass@host:5432/agendai
#   S3_BUCKET             — nombre del bucket (ej: agendai-backups)
#   AWS_ACCESS_KEY_ID     — credencial S3/R2
#   AWS_SECRET_ACCESS_KEY — credencial S3/R2
#
# Variables opcionales:
#   S3_ENDPOINT_URL       — para Cloudflare R2: https://<account>.r2.cloudflarestorage.com
#   RESTORE_DIR           — directorio temporal local (default: /tmp/agendai-restore)

set -euo pipefail

# ── Configuración ──────────────────────────────────────────────────────────────

RESTORE_DIR="${RESTORE_DIR:-/tmp/agendai-restore}"
REQUESTED_BACKUP="${1:-}"

# ── Funciones ──────────────────────────────────────────────────────────────────

log() {
    echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] [RESTORE] $*"
}

error() {
    echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] [ERROR] $*" >&2
}

check_required_vars() {
    local missing=0
    for var in DATABASE_URL S3_BUCKET AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY; do
        if [[ -z "${!var:-}" ]]; then
            error "Variable de entorno requerida no configurada: ${var}"
            missing=1
        fi
    done
    if [[ $missing -eq 1 ]]; then
        exit 1
    fi
}

check_dependencies() {
    for cmd in pg_restore aws; do
        if ! command -v "$cmd" &>/dev/null; then
            error "Comando no encontrado: ${cmd}. Instalar antes de continuar."
            exit 1
        fi
    done
}

aws_cmd() {
    local args=("$@")
    if [[ -n "${S3_ENDPOINT_URL:-}" ]]; then
        aws "${args[@]}" --endpoint-url "${S3_ENDPOINT_URL}"
    else
        aws "${args[@]}"
    fi
}

list_backups() {
    aws_cmd s3api list-objects-v2 \
        --bucket "${S3_BUCKET}" \
        --prefix "backups/" \
        --query "sort_by(Contents, &LastModified)[-10:].{Key:Key,Date:LastModified,Size:Size}" \
        --output table 2>/dev/null || echo "(no se pudo listar backups)"
}

get_latest_backup() {
    aws_cmd s3api list-objects-v2 \
        --bucket "${S3_BUCKET}" \
        --prefix "backups/" \
        --query "sort_by(Contents, &LastModified)[-1].Key" \
        --output text 2>/dev/null
}

# ── Main ───────────────────────────────────────────────────────────────────────

log "=== HERRAMIENTA DE RESTAURACIÓN DE AGENDAI ==="
log ""

check_required_vars
check_dependencies

# Mostrar backups disponibles
log "Últimos 10 backups disponibles en s3://${S3_BUCKET}/backups/:"
list_backups
log ""

# Determinar qué backup restaurar
if [[ -n "${REQUESTED_BACKUP}" ]]; then
    S3_KEY="backups/${REQUESTED_BACKUP}.dump"
    log "Backup solicitado: ${S3_KEY}"
else
    S3_KEY=$(get_latest_backup)
    if [[ -z "${S3_KEY}" || "${S3_KEY}" == "None" ]]; then
        error "No se encontraron backups en el bucket."
        exit 1
    fi
    log "Backup más reciente: ${S3_KEY}"
fi

BACKUP_FILENAME=$(basename "${S3_KEY}")
LOCAL_PATH="${RESTORE_DIR}/${BACKUP_FILENAME}"

# Advertencia de seguridad — requiere confirmación explícita
echo ""
echo "⚠️  ADVERTENCIA: Esta operación es DESTRUCTIVA ⚠️"
echo ""
echo "  Se va a restaurar: ${BACKUP_FILENAME}"
echo "  Destino DB:        ${DATABASE_URL//:*@/:***@}"
echo ""
echo "  Esto sobreescribirá todos los datos actuales en la base de datos."
echo "  Esta acción NO se puede deshacer."
echo ""
read -r -p "¿Confirmar restauración? Escribir 'RESTAURAR' para continuar: " CONFIRM

if [[ "${CONFIRM}" != "RESTAURAR" ]]; then
    log "Restauración cancelada por el usuario."
    exit 0
fi

echo ""
log "Confirmación recibida. Procediendo..."

# Crear directorio temporal
mkdir -p "${RESTORE_DIR}"

# Descargar backup desde S3
log "Descargando s3://${S3_BUCKET}/${S3_KEY} → ${LOCAL_PATH}"
if ! aws_cmd s3 cp "s3://${S3_BUCKET}/${S3_KEY}" "${LOCAL_PATH}"; then
    error "Descarga falló. Verificar que el backup existe en el bucket."
    exit 1
fi

BACKUP_SIZE=$(du -sh "${LOCAL_PATH}" | cut -f1)
log "Descarga completada: ${BACKUP_SIZE}"

# Restaurar con pg_restore
# --clean elimina objetos existentes antes de recrearlos
# --if-exists evita errores si un objeto no existe al limpiar
# --no-owner no restaura ownership (para entornos donde el usuario cambia)
log "Ejecutando pg_restore..."
if ! pg_restore \
    --clean \
    --if-exists \
    --no-owner \
    --no-acl \
    --verbose \
    --dbname="${DATABASE_URL}" \
    "${LOCAL_PATH}" 2>&1 | while IFS= read -r line; do log "  pg_restore: ${line}"; done; then
    error "pg_restore reportó errores. Verificar los logs anteriores."
    log "El archivo local se conserva en: ${LOCAL_PATH}"
    exit 1
fi

# Limpiar archivo local
rm -f "${LOCAL_PATH}"
log "Archivo local eliminado."

log ""
log "✅ Restauración completada exitosamente desde: ${BACKUP_FILENAME}"
log ""
log "Próximos pasos recomendados:"
log "  1. Verificar integridad: psql \$DATABASE_URL -c 'SELECT COUNT(*) FROM tenants;'"
log "  2. Reanudar el servicio API si estaba detenido"
log "  3. Verificar health check: curl https://tu-dominio/api/v1/health/ready"
