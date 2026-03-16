#!/bin/bash
# Crea múltiples bases de datos en el mismo servidor PostgreSQL
# Usado en desarrollo para crear agendai y evolution en el mismo contenedor
set -e

function create_user_and_database() {
    local database=$1
    echo "  Creando base de datos '$database'..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
        CREATE DATABASE $database;
        GRANT ALL PRIVILEGES ON DATABASE $database TO $POSTGRES_USER;
EOSQL
}

# Crear la DB de Evolution API si no existe
if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
    echo "Creando bases de datos adicionales: $POSTGRES_MULTIPLE_DATABASES"
    for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
        create_user_and_database $db
    done
    echo "Bases de datos creadas exitosamente."
fi
