
-- Получаем значения переменных окружения.
\getenv app_user APP_DB_USER
\getenv app_password APP_DB_PASSWORD

-- Создаем пользователя, если он еще не существует.
SELECT format(
    'CREATE ROLE %I WITH LOGIN',
    :'app_user'
)
WHERE NOT EXISTS (
    SELECT 1
    FROM pg_catalog.pg_roles
    WHERE rolname = :'app_user'
)
\gexec

-- Устанавливаем (или обновляем) пароль.
SELECT format(
    'ALTER ROLE %I WITH LOGIN PASSWORD %L',
    :'app_user',
    :'app_password'
)
\gexec
