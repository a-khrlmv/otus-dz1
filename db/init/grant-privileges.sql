
\getenv app_user APP_DB_USER

-- Разрешаем подключение к текущей базе.
SELECT format(
    'GRANT CONNECT ON DATABASE %I TO %I',
    current_database(),
    :'app_user'
)
\gexec

-- Разрешаем доступ к схеме public.
SELECT format(
    'GRANT USAGE ON SCHEMA public TO %I',
    :'app_user'
)
\gexec

-- Разрешаем основные операции с данными.
SELECT format(
    'GRANT SELECT, INSERT, UPDATE, DELETE
     ON TABLE public.auth, public.profiles TO %I',
    :'app_user'
)
\gexec

-- Разрешаем использование последовательностей
-- для генерации идентификаторов.
SELECT format(
    'GRANT USAGE, SELECT
     ON ALL SEQUENCES IN SCHEMA public TO %I',
    :'app_user'
)
\gexec
