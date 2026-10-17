-- Create, List, Get e Update devolvem as mesmas colunas, na mesma ordem: assim as structs geradas são conversíveis entre si.
-- attendee_count e going dizem quantos confirmaram presença e se quem pede (viewer) é um deles; viewer 0 = visitante.
-- "De hoje em diante" é sempre no horário de Brasília: o evento de hoje fica aberto até o fim do dia, mesmo depois de começar.

-- name: CreateEvent :one
-- Evento recém-criado ainda não tem confirmados: as duas colunas saem fixas.
WITH e AS (
    INSERT INTO events (author_id, title, description, starts_at, location)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT e.id, e.title, e.description, e.starts_at, e.location, e.created_at,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       0::bigint AS attendee_count, false AS going
FROM e
JOIN users u ON u.id = e.author_id;

-- name: ListEvents :many
SELECT e.id, e.title, e.description, e.starts_at, e.location, e.created_at,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM event_attendees a WHERE a.event_id = e.id) AS attendee_count,
       EXISTS (SELECT 1 FROM event_attendees a WHERE a.event_id = e.id AND a.user_id = sqlc.arg('viewer')) AS going
FROM events e
JOIN users u ON u.id = e.author_id
WHERE e.starts_at >= date_trunc('day', now() AT TIME ZONE 'America/Sao_Paulo') AT TIME ZONE 'America/Sao_Paulo'
ORDER BY e.starts_at, e.id;

-- name: GetEvent :one
-- Também os que já passaram: a página continua no link.
SELECT e.id, e.title, e.description, e.starts_at, e.location, e.created_at,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM event_attendees a WHERE a.event_id = e.id) AS attendee_count,
       EXISTS (SELECT 1 FROM event_attendees a WHERE a.event_id = e.id AND a.user_id = sqlc.arg('viewer')) AS going
FROM events e
JOIN users u ON u.id = e.author_id
WHERE e.id = sqlc.arg('id');

-- name: IsUpcoming :one
-- Se o evento ainda está na agenda: só esses recebem confirmação.
SELECT starts_at >= date_trunc('day', now() AT TIME ZONE 'America/Sao_Paulo') AT TIME ZONE 'America/Sao_Paulo'
FROM events WHERE id = $1;

-- name: AddAttendee :exec
-- Repetir não dá erro nem duplica: a chave (event_id, user_id) já existe e o insert vira nada.
INSERT INTO event_attendees (event_id, user_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveAttendee :exec
DELETE FROM event_attendees WHERE event_id = $1 AND user_id = $2;

-- name: ListAttendees :many
-- Em ordem de chegada: quem confirmou primeiro aparece primeiro.
SELECT u.id, u.name, u.avatar_url
FROM event_attendees a
JOIN users u ON u.id = a.user_id
WHERE a.event_id = $1
ORDER BY a.created_at, a.user_id;

-- name: UpdateEvent :one
-- Troca os campos do criar; autor, data de criação e a lista de presença não mudam. viewer é quem edita.
WITH e AS (
    UPDATE events
    SET title = sqlc.arg('title'), description = sqlc.arg('description'), starts_at = sqlc.arg('starts_at'), location = sqlc.arg('location')
    WHERE events.id = sqlc.arg('id')
    RETURNING *
)
SELECT e.id, e.title, e.description, e.starts_at, e.location, e.created_at,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM event_attendees a WHERE a.event_id = e.id) AS attendee_count,
       EXISTS (SELECT 1 FROM event_attendees a WHERE a.event_id = e.id AND a.user_id = sqlc.arg('viewer')) AS going
FROM e
JOIN users u ON u.id = e.author_id;

-- name: DeleteEvent :exec
-- A lista de presença vai junto (ON DELETE CASCADE).
DELETE FROM events WHERE id = $1;

-- name: GetEventAuthor :one
SELECT author_id FROM events WHERE id = $1;

-- name: IsModerator :one
-- O moderador edita e apaga qualquer evento (docs/contratos/fatia-16-editar-e-apagar-evento.md).
SELECT is_moderator FROM users WHERE id = $1;
