#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# CitaSpot — Reset de base de datos a "estado de fábrica"
#
# Qué hace:
#   1. Carga el .env de la raíz del repo
#   2. Aplica DOBLE CANDADO anti-prod (host check + confirmación interactiva)
#   3. DROP SCHEMA public CASCADE + CREATE SCHEMA public en el contenedor
#   4. Re-aplica todas las migraciones via `make migrate`
#
# Uso:
#   make db-reset                      # vía Makefile
#   bash scripts/db-reset.sh           # directo
#   I_KNOW_WHAT_IM_DOING=yes bash scripts/db-reset.sh   # bypass del candado anti-prod
#
# ⚠️  ESTE SCRIPT BORRA TODOS LOS DATOS — incluyendo todos los tenants.
# ──────────────────────────────────────────────────────────────────────────────

set -euo pipefail

# ── Colores ANSI ──────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # reset

info()    { printf "${BLUE}▶${NC} %s\n" "$*"; }
ok()      { printf "${GREEN}✅${NC} %s\n" "$*"; }
warn()    { printf "${YELLOW}⚠${NC}  %s\n" "$*"; }
err()     { printf "${RED}✖${NC}  %s\n" "$*" >&2; }
banner()  { printf "${RED}${BOLD}%s${NC}\n" "$*"; }

# ── Resolver paths relativos al script ────────────────────────────────────────
# Esto permite correr el script desde cualquier CWD sin romper.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Cargar .env desde la raíz del repo ────────────────────────────────────────
if [[ -f "${REPO_ROOT}/.env" ]]; then
  info "Cargando variables desde ${REPO_ROOT}/.env"
  # set -a exporta todo lo que se source mientras esté activo
  set -a
  # shellcheck disable=SC1091
  source "${REPO_ROOT}/.env"
  set +a
else
  warn "No encontré ${REPO_ROOT}/.env — usando defaults del docker-compose"
fi

# ── Defaults coherentes con docker-compose.yml ────────────────────────────────
# Si .env no define DATABASE_URL, usamos los del docker-compose.yml local.
: "${DATABASE_URL:=postgresql://citaspot:citaspot_dev@localhost:5432/citaspot}"

# Servicio del docker-compose.yml
COMPOSE_SERVICE="postgres"

# ── Parsear DATABASE_URL para extraer host, user y dbname ─────────────────────
# Formato esperado: postgresql://USER:PASS@HOST:PORT/DBNAME?...
# Truco: aislar el segmento entre @ y / o ? para sacar el host.
extract_host() {
  local url="$1"
  # quitar el prefijo postgresql:// (o postgres://)
  local rest="${url#*://}"
  # si tiene credenciales, quitarlas (cortar por el último @ por si el password tiene @)
  if [[ "${rest}" == *"@"* ]]; then
    rest="${rest#*@}"
  fi
  # quedarse con la parte antes del primer / o ?
  local hostport="${rest%%/*}"
  hostport="${hostport%%\?*}"
  # quitar :PORT si existe
  echo "${hostport%%:*}"
}

# extract_user: extrae el USER de postgresql://USER:PASS@HOST:PORT/DBNAME
# Si no hay credenciales (no hay @), devuelve string vacío.
extract_user() {
  local url="$1"
  local rest="${url#*://}"
  # si no hay @, no hay credenciales
  if [[ "${rest}" != *"@"* ]]; then
    echo ""
    return
  fi
  # parte de credenciales: todo lo que está antes del primer @
  local creds="${rest%%@*}"
  # si tiene :, el user es lo que está antes; si no, todo es user
  echo "${creds%%:*}"
}

# extract_db: extrae el DBNAME (parte después del último / y antes del ? si existe)
extract_db() {
  local url="$1"
  local rest="${url#*://}"
  # si tiene credenciales, quitarlas
  if [[ "${rest}" == *"@"* ]]; then
    rest="${rest#*@}"
  fi
  # quitar host:port (todo hasta el primer /)
  if [[ "${rest}" != *"/"* ]]; then
    echo ""
    return
  fi
  local dbpart="${rest#*/}"
  # cortar query string si existe
  echo "${dbpart%%\?*}"
}

DB_HOST="$(extract_host "${DATABASE_URL}")"
DB_USER="$(extract_user "${DATABASE_URL}")"
DB_NAME="$(extract_db "${DATABASE_URL}")"

# Fallback a defaults si el parsing falla
if [[ -z "${DB_USER}" ]]; then
  warn "No pude parsear el USER desde DATABASE_URL — usando default 'citaspot'"
  DB_USER="citaspot"
fi
if [[ -z "${DB_NAME}" ]]; then
  warn "No pude parsear el DBNAME desde DATABASE_URL — usando default 'citaspot'"
  DB_NAME="citaspot"
fi

# ── Candado #1: detectar si el host es local o remoto ─────────────────────────
is_local_host() {
  local host="$1"
  case "${host}" in
    localhost|127.0.0.1|::1|postgres|db|citaspot_postgres)
      return 0
      ;;
    *.local)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

info "Host detectado en DATABASE_URL: ${BOLD}${DB_HOST}${NC}"
info "Conectando como: ${BOLD}${DB_USER}${NC} a base: ${BOLD}${DB_NAME}${NC}"

if is_local_host "${DB_HOST}"; then
  ok "Host parece LOCAL — modo seguro activo."
else
  banner "════════════════════════════════════════════════════════════════"
  banner " ⛔ HOST NO ES LOCAL: ${DB_HOST}"
  banner "════════════════════════════════════════════════════════════════"
  warn "Este script está pensado SOLO para ambientes de desarrollo locales."
  warn "Si realmente sabés lo que hacés y querés ejecutarlo contra ${DB_HOST},"
  warn "exportá la variable de entorno: ${BOLD}I_KNOW_WHAT_IM_DOING=yes${NC}"

  if [[ "${I_KNOW_WHAT_IM_DOING:-no}" != "yes" ]]; then
    err "Abortado: candado anti-prod activado."
    err "Ejecutá: I_KNOW_WHAT_IM_DOING=yes bash scripts/db-reset.sh"
    exit 1
  fi

  banner ""
  banner " 🚨 BYPASS ACTIVADO — vas a destruir datos en ${DB_HOST} 🚨 "
  banner ""
fi

# ── Candado #2: confirmación interactiva escribiendo "RESET" ──────────────────
echo
warn "Esto va a:"
warn "  • DROP SCHEMA public CASCADE en la base ${DB_NAME}"
warn "  • Borrar TODOS los datos de TODOS los tenants"
warn "  • Re-correr todas las migraciones desde cero"
echo
printf "${YELLOW}Para continuar escribí literalmente la palabra ${BOLD}RESET${NC}${YELLOW} (en mayúsculas): ${NC}"
# Si no hay TTY (CI por ejemplo), abortamos para no quedarnos colgados
if [[ ! -t 0 ]]; then
  err "No hay TTY interactivo — abortando para evitar destrucción accidental."
  exit 1
fi
read -r CONFIRM
if [[ "${CONFIRM}" != "RESET" ]]; then
  err "Confirmación inválida (escribiste: '${CONFIRM}'). Abortado."
  exit 1
fi

# ── Verificar que el contenedor de postgres esté corriendo ────────────────────
info "Verificando que el servicio '${COMPOSE_SERVICE}' esté corriendo..."
if ! docker compose -f "${REPO_ROOT}/docker-compose.yml" ps --services --filter "status=running" | grep -qx "${COMPOSE_SERVICE}"; then
  err "El servicio '${COMPOSE_SERVICE}' no está corriendo."
  err "Levantalo primero con: ${BOLD}make dev${NC} o ${BOLD}docker compose up -d ${COMPOSE_SERVICE}${NC}"
  exit 1
fi
ok "Servicio '${COMPOSE_SERVICE}' está activo."

# ── Ejecutar el reset del schema ──────────────────────────────────────────────
info "Ejecutando DROP SCHEMA public CASCADE..."
docker compose -f "${REPO_ROOT}/docker-compose.yml" exec -T "${COMPOSE_SERVICE}" \
  psql -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 <<SQL
-- Tirar todo el schema público (cascada borra tablas, índices, FKs, etc.)
DROP SCHEMA IF EXISTS public CASCADE;
-- Recrear schema vacío
CREATE SCHEMA public;
-- Restaurar grants estándar de Postgres
GRANT ALL ON SCHEMA public TO ${DB_USER};
GRANT ALL ON SCHEMA public TO public;
SQL
ok "Schema 'public' recreado vacío."

# ── Re-aplicar migraciones via make migrate ───────────────────────────────────
# Usamos el target del Makefile para mantener una sola fuente de verdad.
# El runner Go (apps/api/cmd/migrate/main.go) corre desde el host y reaplica
# todas las migraciones SQL incluyendo extensions (001) y RLS.
info "Re-aplicando migraciones con 'make migrate'..."
if ! (cd "${REPO_ROOT}" && make migrate); then
  err "Falló la aplicación de migraciones."
  err "Revisá el output anterior. La DB quedó vacía pero sin schema aplicado."
  exit 1
fi

# ── Mensaje final ─────────────────────────────────────────────────────────────
echo
ok "${BOLD}Reset completo.${NC}"
echo
info "La base ${DB_NAME} quedó como recién instalada:"
info "  • Schema 'public' recreado"
info "  • Migraciones 001-014 reaplicadas"
info "  • Tabla 'schema_migrations' regenerada por el runner"
echo
info "Próximos pasos sugeridos:"
info "  ${BOLD}make seed${NC}         # insertar datos de prueba (test-salon, admin@test.com)"
info "  ${BOLD}make migrate-status${NC}  # verificar que todas las migraciones figuran como aplicadas"
echo
