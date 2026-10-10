-- +goose Up
-- Quem clicou em "Quero participar". Sem aprovação: existir a linha já é estar na lista.
-- A chave (idea_id, user_id) impede entrar duas vezes e serve de índice para contar os interessados de cada ideia.
CREATE TABLE interests (
    idea_id    bigint NOT NULL REFERENCES ideas (id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (idea_id, user_id)
);

-- +goose Down
DROP TABLE interests;
