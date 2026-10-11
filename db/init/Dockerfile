### Dockerfile for init container, used for postgres migration

# Stage 1: готовый бинарник golang-migrate
FROM migrate/migrate:v4.19.1 AS migrate-bin

# Stage 2: PostgreSQL client + мигратор
FROM postgres:18-alpine3.24

# Копируем утилиту миграций
COPY --from=migrate-bin \
    /usr/local/bin/migrate \
    /usr/local/bin/migrate

# Скрипт запуска init-контейнера
COPY db/init/entrypoint.sh /usr/local/bin/db-init.sh

# SQL для управления ролью и правами
COPY db/init/create-role.sql /opt/db/create-role.sql
COPY db/init/grant-privileges.sql /opt/db/grant-privileges.sql

# Версионированные миграции
COPY db/migrations/ /migrations/

# Даём право выполнения стартовому скрипту
RUN chmod 755 /usr/local/bin/db-init.sh

# Запускаем процесс не от root
USER postgres

ENTRYPOINT ["/usr/local/bin/db-init.sh"]
CMD []
