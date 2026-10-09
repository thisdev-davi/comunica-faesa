-- name: UpsertDiscordUser :one
-- Nome e foto acompanham o Discord: são regravados a cada login.
INSERT INTO users (discord_id, name, avatar_url)
VALUES ($1, $2, $3)
ON CONFLICT (discord_id) DO UPDATE SET name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
RETURNING id;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionUser :one
SELECT u.id, u.name, u.avatar_url, u.course
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;
