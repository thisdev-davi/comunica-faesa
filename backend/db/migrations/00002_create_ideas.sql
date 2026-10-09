-- +goose Up
-- Curso e categoria são validados na API (precisa da mensagem por campo); status tem CHECK porque só o banco garante para qualquer escritor.
CREATE TABLE ideas (
    id          bigserial PRIMARY KEY,
    author_id   bigint NOT NULL REFERENCES users (id),
    title       text NOT NULL,
    description text NOT NULL,
    course      text NOT NULL,
    category    text NOT NULL,
    slots       int NOT NULL CHECK (slots BETWEEN 1 AND 20),
    status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'done')),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ideas_created_at_idx ON ideas (created_at DESC);

-- +goose Down
DROP TABLE ideas;
