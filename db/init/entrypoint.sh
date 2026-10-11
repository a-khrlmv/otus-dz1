#!/bin/sh

set -eu

# Проверяем обязательные переменные окружения.
: "${DB_HOST:?DB_HOST is required}"
: "${DB_NAME:?DB_NAME is required}"
: "${DB_ADMIN_USER:?DB_ADMIN_USER is required}"
: "${DB_ADMIN_PASSWORD:?DB_ADMIN_PASSWORD is required}"
: "${APP_DB_USER:?APP_DB_USER is required}"
: "${APP_DB_PASSWORD:?APP_DB_PASSWORD is required}"

# Сервис не должен использовать административную роль.
if [ "$APP_DB_USER" = "$DB_ADMIN_USER" ]; then
    echo "ERROR: App user cannot be DB admin" >&2
    exit 1
fi

# Настройки подключения для psql и golang-migrate.
export PGHOST="$DB_HOST"
export PGPORT="${DB_PORT:-5432}"
export PGDATABASE="$DB_NAME"
export PGUSER="$DB_ADMIN_USER"
export PGPASSWORD="$DB_ADMIN_PASSWORD"
export PGSSLMODE=disable

echo "[db-init] Creating application role..."

psql -X -v ON_ERROR_STOP=1 \
    -f /opt/db/create-role.sql

echo "[db-init] Applying SQL migrations..."

# Пароль берётся из PGPASSWORD, не из URL.
migrate \
    -path /migrations \
    -database "postgres://${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable&search_path=public" \
    up

echo "[db-init] Granting application privileges..."

psql -X -v ON_ERROR_STOP=1 \
    -f /opt/db/grant-privileges.sql

echo "[db-init] Initialization completed successfully."
