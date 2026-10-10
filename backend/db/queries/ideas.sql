-- Create, List e Get devolvem as mesmas colunas, na mesma ordem: assim as structs geradas são conversíveis entre si.

-- name: CreateIdea :one
WITH i AS (
    INSERT INTO ideas (author_id, title, description, course, category, slots)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING *
)
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name
FROM i
JOIN users u ON u.id = i.author_id;

-- name: ListIdeas :many
-- max NULL = sem limite (LIMIT NULL é o mesmo que não ter LIMIT); o visitante recebe só uma prévia.
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name
FROM ideas i
JOIN users u ON u.id = i.author_id
ORDER BY i.created_at DESC, i.id DESC
LIMIT sqlc.narg('max');

-- name: GetIdea :one
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name
FROM ideas i
JOIN users u ON u.id = i.author_id
WHERE i.id = $1;
