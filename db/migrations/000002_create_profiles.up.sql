
CREATE TABLE public.profiles (
    user_id BIGINT PRIMARY KEY
        REFERENCES public.auth(user_id)
        ON DELETE CASCADE,

    first_name VARCHAR(64),
    last_name VARCHAR(64),
    birth_date DATE,
    gender VARCHAR(20) NOT NULL DEFAULT 'unspecified'
        CHECK (gender IN ('male', 'female', 'unspecified')),
    interests TEXT,
    city VARCHAR(150)
);
