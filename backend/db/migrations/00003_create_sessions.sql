-- +goose Up
-- Guarda só o hash do token: quem ler o banco não consegue usar as sessões.
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE sessions;
