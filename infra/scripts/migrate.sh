#!/usr/bin/env bash
# migrate.sh — Aplica migraciones SQL usando psql (alternativa al runner Go)
# Uso:
#   ./migrate.sh up              # Aplica todas las migraciones pendientes
#   ./migrate.sh down [N]        # Revierte las últimas N migraciones (default: 1)
#   ./migrate.sh status          # Muestra el estado de todas las migraciones
#
# Variables de entorno:
#   DATABASE_URL  — URL de conexión PostgreSQL (pgx format o libpq format)
#                   Ejemplo: postgresql://citaspot:citaspot_dev@localhost:5432/citaspot
#   MIGRATIONS_DIR — Directorio con los archivos .sql (default: apps/api/db/migrations)

set -euo pipefail

# ── Configuración ───────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DATABASE_URL="${DATABASE_URL:-postgresql://citaspot:citaspot_dev@localhost:5432/citaspot}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-${REPO_ROOT}/apps/api/db/migrations}"

COMMAND="${1:-up}"
COUNT="${2:-1}"

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ── Helpers ─────────────────────────────────────────────────────────────────

log_info()    { echo -e "${BLUE}▶${NC} $*"; }
log_success() { echo -e "${GREEN}✅${NC} $*"; }
log_warning() { echo -e "${YELLOW}⚠️ ${NC} $*"; }
log_error()   { echo -e "${RED}❌${NC} $*" >&2; }

# Verifica que psql esté disponible
check_psql() {
  if ! command -v psql &>/dev/null; then
    log_error "psql no encontrado. Instala PostgreSQL client tools."
    exit 1
  fi
}

# Ejecuta SQL contra la base de datos
run_sql() {
  psql "${DATABASE_URL}" --no-psqlrc -v ON_ERROR_STOP=1 -q "$@"
}

# Crea tabla de tracking si no existe
ensure_migrations_table() {
  run_sql <<'EOF'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ  DEFAULT NOW()
);
EOF
}

# Retorna 0 si la migración ya fue aplicada, 1 si no
is_applied() {
  local version="$1"
  local result
  result=$(run_sql -t -c "SELECT COUNT(*) FROM schema_migrations WHERE version = '${version}';")
  result=$(echo "${result}" | tr -d '[:space:]')
  [[ "${result}" == "1" ]]
}

# Lista archivos .sql del directorio de migraciones en orden
get_migration_files() {
  find "${MIGRATIONS_DIR}" -maxdepth 1 -name "*.sql" ! -name "*.down.sql" \
    | sort
}

# ── Comandos ─────────────────────────────────────────────────────────────────

cmd_up() {
  log_info "Aplicando migraciones pendientes..."
  ensure_migrations_table

  local pending=0
  while IFS= read -r filepath; do
    local filename
    filename="$(basename "${filepath}")"

    if is_applied "${filename}"; then
      echo "  [✓] ${filename} (ya aplicada)"
      continue
    fi

    echo -n "  [ ] ${filename} ... "

    # Aplica la migración + registra en schema_migrations dentro de una transacción
    run_sql <<EOF
BEGIN;
\i ${filepath}
INSERT INTO schema_migrations (version) VALUES ('${filename}');
COMMIT;
EOF

    echo -e "${GREEN}OK${NC}"
    (( pending++ )) || true
  done < <(get_migration_files)

  if [[ ${pending} -eq 0 ]]; then
    log_success "Sin migraciones pendientes — la DB está actualizada."
  else
    log_success "${pending} migración(es) aplicada(s) exitosamente."
  fi
}

cmd_down() {
  local n="${COUNT}"
  log_warning "Revirtiendo las últimas ${n} migración(es)..."
  ensure_migrations_table

  # Obtener las últimas N versiones en orden inverso
  local versions
  mapfile -t versions < <(run_sql -t -c \
    "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT ${n};" \
    | tr -d ' ')

  if [[ ${#versions[@]} -eq 0 ]]; then
    log_info "Sin migraciones aplicadas para revertir."
    return
  fi

  for version in "${versions[@]}"; do
    [[ -z "${version}" ]] && continue

    local down_file="${MIGRATIONS_DIR}/${version%.sql}.down.sql"
    if [[ ! -f "${down_file}" ]]; then
      log_error "Archivo de reversión no encontrado: ${down_file}"
      exit 1
    fi

    echo -n "  Revirtiendo ${version} ... "
    run_sql <<EOF
BEGIN;
\i ${down_file}
DELETE FROM schema_migrations WHERE version = '${version}';
COMMIT;
EOF
    echo -e "${GREEN}OK${NC}"
  done

  log_success "Reversión completada."
}

cmd_status() {
  ensure_migrations_table

  echo ""
  echo "Estado de migraciones CitaSpot:"
  echo "────────────────────────────────────────────────────────"
  while IFS= read -r filepath; do
    local filename
    filename="$(basename "${filepath}")"

    if is_applied "${filename}"; then
      echo "  [✓] ${filename}"
    else
      echo "  [ ] ${filename}  ← pendiente"
    fi
  done < <(get_migration_files)
  echo "────────────────────────────────────────────────────────"
}

# ── Main ─────────────────────────────────────────────────────────────────────

check_psql

case "${COMMAND}" in
  up)     cmd_up     ;;
  down)   cmd_down   ;;
  status) cmd_status ;;
  *)
    log_error "Comando desconocido: '${COMMAND}'"
    echo "Uso: $0 [up|down [N]|status]"
    exit 1
    ;;
esac
