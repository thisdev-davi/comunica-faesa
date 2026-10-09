-- Create e List devolvem as mesmas colunas, na mesma ordem: assim as structs geradas são conversíveis entre si.

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
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at,
       u.id AS author_id, u.name AS author_name
FROM ideas i
JOIN users u ON u.id = i.author_id
ORDER BY i.created_at DESC, i.id DESC;
