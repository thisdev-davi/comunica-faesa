-- +goose Up
CREATE TABLE users (
    id         bigserial PRIMARY KEY,
    discord_id text UNIQUE,
    name       text NOT NULL,
    avatar_url text,
    course     text,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
