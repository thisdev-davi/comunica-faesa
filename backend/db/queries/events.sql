-- List e Get devolvem as mesmas colunas, na mesma ordem: assim as structs geradas são conversíveis entre si.
-- attendee_count e going dizem quantos confirmaram presença e se quem pede (viewer) é um deles; viewer 0 = visitante.
-- "De hoje em diante" é sempre no horário de Brasília: o evento de hoje fica aberto até o fim do dia, mesmo depois de começar.

-- name: CreateEvent :one
-- Só grava se o autor já escolheu o curso; sem curso não volta linha (pgx.ErrNoRows), e o handler responde 403.
-- Devolve só o id: o handler busca o evento completo com o GetEvent.
INSERT INTO events (author_id, title, description, starts_at, location)
SELECT $1, $2, $3, $4, $5
FROM users WHERE id = $1 AND course IS NOT NULL
RETURNING id;

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
-- Se o evento ainda está na agenda: quando o AddAttendee não grava, é ele que diz o porquê.
SELECT starts_at >= date_trunc('day', now() AT TIME ZONE 'America/Sao_Paulo') AT TIME ZONE 'America/Sao_Paulo'
FROM events WHERE id = $1;

-- name: AddAttendee :execrows
-- Confere e grava na mesma instrução: só evento da agenda (a mesma regra do IsUpcoming). O FOR SHARE segura a linha
-- do evento até o fim: se ele é editado ou apagado ao mesmo tempo, um espera o outro, e o insert reconfere a linha nova.
-- Repetir não dá erro nem duplica: a chave (event_id, user_id) já existe e o insert vira nada.
-- 0 linhas = não gravou (não existe, já passou ou a pessoa já estava na lista); o handler descobre qual.
INSERT INTO event_attendees (event_id, user_id)
SELECT id, sqlc.arg('user_id') FROM events
WHERE id = sqlc.arg('event_id')
  AND starts_at >= date_trunc('day', now() AT TIME ZONE 'America/Sao_Paulo') AT TIME ZONE 'America/Sao_Paulo'
FOR SHARE
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

-- name: UpdateEvent :exec
-- Troca os campos do criar; autor, data de criação e a lista de presença não mudam. O handler busca o evento completo com o GetEvent.
UPDATE events SET title = $2, description = $3, starts_at = $4, location = $5
WHERE id = $1;

-- name: DeleteEvent :exec
-- A lista de presença vai junto (ON DELETE CASCADE).
DELETE FROM events WHERE id = $1;

-- name: GetEventAuthor :one
SELECT author_id FROM events WHERE id = $1;
