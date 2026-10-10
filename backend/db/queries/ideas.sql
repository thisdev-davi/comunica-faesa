-- Create, List e Get devolvem as mesmas colunas, na mesma ordem: assim as structs geradas são conversíveis entre si.
-- interest_count e interested dizem quantos marcaram "quero participar" e se quem pede (viewer) é um deles; viewer 0 = visitante.

-- name: CreateIdea :one
-- Ideia recém-criada ainda não tem interessados: as duas colunas saem fixas.
WITH i AS (
    INSERT INTO ideas (author_id, title, description, course, category, slots)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING *
)
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name,
       0::bigint AS interest_count, false AS interested
FROM i
JOIN users u ON u.id = i.author_id;

-- name: ListIdeas :many
-- max NULL = sem limite (LIMIT NULL é o mesmo que não ter LIMIT); o visitante recebe só uma prévia.
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name,
       (SELECT count(*) FROM interests n WHERE n.idea_id = i.id) AS interest_count,
       EXISTS (SELECT 1 FROM interests n WHERE n.idea_id = i.id AND n.user_id = sqlc.arg('viewer')) AS interested
FROM ideas i
JOIN users u ON u.id = i.author_id
ORDER BY i.created_at DESC, i.id DESC
LIMIT sqlc.narg('max');

-- name: GetIdea :one
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name,
       (SELECT count(*) FROM interests n WHERE n.idea_id = i.id) AS interest_count,
       EXISTS (SELECT 1 FROM interests n WHERE n.idea_id = i.id AND n.user_id = sqlc.arg('viewer')) AS interested
FROM ideas i
JOIN users u ON u.id = i.author_id
WHERE i.id = sqlc.arg('id');

-- name: GetIdeaAuthor :one
SELECT author_id FROM ideas WHERE id = $1;

-- name: AddInterest :exec
-- Repetir não dá erro nem duplica: a chave (idea_id, user_id) já existe e o insert vira nada.
INSERT INTO interests (idea_id, user_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveInterest :exec
DELETE FROM interests WHERE idea_id = $1 AND user_id = $2;

-- name: ListInterests :many
-- Em ordem de chegada: quem marcou primeiro aparece primeiro para o autor.
SELECT u.id, u.name, u.avatar_url, u.discord_id
FROM interests n
JOIN users u ON u.id = n.user_id
WHERE n.idea_id = $1
ORDER BY n.created_at, n.user_id;
