#!/usr/bin/env bash
# backup.sh — Backup automatizado de PostgreSQL para AgendAI
#
# Uso:
#   ./backup.sh
#
# Variables de entorno requeridas:
#   DATABASE_URL          — postgresql://user:pass@host:5432/agendai
#   S3_BUCKET             — nombre del bucket (ej: agendai-backups)
#   AWS_ACCESS_KEY_ID     — credencial S3/R2
#   AWS_SECRET_ACCESS_KEY — credencial S3/R2
#
# Variables opcionales:
#   S3_ENDPOINT_URL       — para Cloudflare R2: https://<account>.r2.cloudflarestorage.com
#   BACKUP_RETENTION_DAYS — días a retener backups (default: 30)
#   BACKUP_DIR            — directorio temporal local (default: /tmp/agendai-backups)

set -euo pipefail

# ── Configuración ──────────────────────────────────────────────────────────────

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILENAME="agendai_${TIMESTAMP}.dump"
BACKUP_DIR="${BACKUP_DIR:-/tmp/agendai-backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILENAME}"

# ── Funciones ──────────────────────────────────────────────────────────────────

log() {
    echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] [BACKUP] $*"
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
    for cmd in pg_dump aws; do
        if ! command -v "$cmd" &>/dev/null; then
            error "Comando no encontrado: ${cmd}. Instalar antes de continuar."
            exit 1
        fi
    done
}

# ── Main ───────────────────────────────────────────────────────────────────────

log "Iniciando backup de AgendAI..."

check_required_vars
check_dependencies

# Crear directorio temporal
mkdir -p "${BACKUP_DIR}"

# Ejecutar pg_dump en formato custom (comprimido, restaurable con pg_restore)
log "Ejecutando pg_dump → ${BACKUP_FILENAME}"
if ! pg_dump \
    --format=custom \
    --compress=9 \
    --no-password \
    --verbose \
    "${DATABASE_URL}" \
    --file="${BACKUP_PATH}" 2>&1 | while IFS= read -r line; do log "  pg_dump: ${line}"; done; then
    error "pg_dump falló. Abortando."
    rm -f "${BACKUP_PATH}"
    exit 1
fi

BACKUP_SIZE=$(du -sh "${BACKUP_PATH}" | cut -f1)
log "Backup creado: ${BACKUP_PATH} (${BACKUP_SIZE})"

# Subir a S3/R2
log "Subiendo a S3: s3://${S3_BUCKET}/backups/${BACKUP_FILENAME}"

AWS_ARGS=(
    s3 cp
    "${BACKUP_PATH}"
    "s3://${S3_BUCKET}/backups/${BACKUP_FILENAME}"
    --storage-class STANDARD_IA
)

# Endpoint personalizado para Cloudflare R2 u otros proveedores S3-compatibles
if [[ -n "${S3_ENDPOINT_URL:-}" ]]; then
    AWS_ARGS+=(--endpoint-url "${S3_ENDPOINT_URL}")
fi

if ! aws "${AWS_ARGS[@]}" 2>&1 | while IFS= read -r line; do log "  aws: ${line}"; done; then
    error "Upload a S3 falló. El archivo local se conserva en ${BACKUP_PATH}."
    exit 1
fi

log "Upload completado exitosamente."

# Limpiar archivo local
rm -f "${BACKUP_PATH}"
log "Archivo local eliminado."

# Limpiar backups antiguos del bucket (más de BACKUP_RETENTION_DAYS días)
log "Eliminando backups con más de ${BACKUP_RETENTION_DAYS} días..."

CUTOFF_DATE=$(date -d "-${BACKUP_RETENTION_DAYS} days" +"%Y-%m-%dT%H:%M:%S" 2>/dev/null || \
              date -v "-${BACKUP_RETENTION_DAYS}d" +"%Y-%m-%dT%H:%M:%S")

LIST_ARGS=(
    s3api list-objects-v2
    --bucket "${S3_BUCKET}"
    --prefix "backups/"
    --query "Contents[?LastModified<='${CUTOFF_DATE}'].Key"
    --output text
)

if [[ -n "${S3_ENDPOINT_URL:-}" ]]; then
    LIST_ARGS+=(--endpoint-url "${S3_ENDPOINT_URL}")
fi

OLD_OBJECTS=$(aws "${LIST_ARGS[@]}" 2>/dev/null || echo "")

if [[ -n "${OLD_OBJECTS}" && "${OLD_OBJECTS}" != "None" ]]; then
    while IFS= read -r key; do
        [[ -z "$key" ]] && continue
        log "  Eliminando: ${key}"
        DELETE_ARGS=(s3 rm "s3://${S3_BUCKET}/${key}")
        if [[ -n "${S3_ENDPOINT_URL:-}" ]]; then
            DELETE_ARGS+=(--endpoint-url "${S3_ENDPOINT_URL}")
        fi
        aws "${DELETE_ARGS[@]}" &>/dev/null || error "  No se pudo eliminar: ${key}"
    done <<< "${OLD_OBJECTS}"
else
    log "  No hay backups antiguos que eliminar."
fi

log "Backup completado exitosamente. Archivo: backups/${BACKUP_FILENAME}"
