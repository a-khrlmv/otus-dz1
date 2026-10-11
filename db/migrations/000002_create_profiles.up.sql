
CREATE TABLE public.profiles (
    user_id BIGINT PRIMARY KEY
        REFERENCES public.auth(user_id)
        ON DELETE CASCADE,

    first_name VARCHAR(100),
    last_name VARCHAR(100),
    birth_date DATE,
    gender VARCHAR(30),
    interests TEXT,
    city VARCHAR(150)
);
