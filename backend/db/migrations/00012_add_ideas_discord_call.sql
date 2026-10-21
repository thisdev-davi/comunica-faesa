-- +goose Up
-- A call do grupo no Discord (docs/contratos/fatia-19-call-no-discord.md): nulas até o bot criar o cargo e o canal.
-- Guardar os ids é o que faz cada disparo criar só o que falta.
ALTER TABLE ideas
    ADD COLUMN discord_role_id text,
    ADD COLUMN discord_channel_id text;

-- +goose Down
ALTER TABLE ideas DROP COLUMN discord_role_id, DROP COLUMN discord_channel_id;
