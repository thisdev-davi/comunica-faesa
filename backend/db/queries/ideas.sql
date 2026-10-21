-- List, ListAuthor, ListParticipating e Get devolvem as mesmas colunas, na mesma ordem: assim as structs geradas são conversíveis entre si.
-- interest_count e interested dizem quantos marcaram "quero participar" e se quem pede (viewer) é um deles; viewer 0 = visitante.
-- interest_faces são os 8 primeiros interessados (id, nome e foto) em JSON, na ordem de chegada; o visitante recebe [].

-- name: CreateIdea :one
-- Só grava se o autor já escolheu o curso; sem curso não volta linha (pgx.ErrNoRows), e o handler responde 403.
-- Devolve só o id: o handler busca a ideia completa com o GetIdea.
INSERT INTO ideas (author_id, title, description, course, category, slots, skills)
SELECT $1, $2, $3, $4, $5, $6, $7
FROM users WHERE id = $1 AND course IS NOT NULL
RETURNING id;

-- name: ListIdeas :many
-- status NULL é o mural (abertas e em andamento); 'done' é a vitrine; 'open' e 'in_progress' filtram o mural.
-- Os outros filtros NULL não filtram (docs/contratos/fatia-17-filtros-do-mural.md).
-- search é o padrão do ILIKE já montado no Go ('%texto%', com % _ \ escapados): acha no título ou na descrição,
-- sem acento dos dois lados (docs/contratos/fatia-18-busca-por-texto.md).
-- A vitrine ordena pela conclusão; no mural done_at é sempre NULL, e a ordem cai na data de postagem.
-- max NULL = sem limite (LIMIT NULL é o mesmo que não ter LIMIT); o visitante recebe só uma prévia.
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at, i.done_at, i.skills,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM interests n WHERE n.idea_id = i.id) AS interest_count,
       EXISTS (SELECT 1 FROM interests n WHERE n.idea_id = i.id AND n.user_id = sqlc.arg('viewer')) AS interested,
       (SELECT coalesce(json_agg(json_build_object('id', f.id, 'name', f.name, 'avatar_url', f.avatar_url) ORDER BY f.created_at, f.id), '[]')
        FROM (SELECT u2.id, u2.name, u2.avatar_url, n.created_at FROM interests n JOIN users u2 ON u2.id = n.user_id
              WHERE n.idea_id = i.id AND sqlc.arg('viewer') <> 0 ORDER BY n.created_at, n.user_id LIMIT 8) f)::json AS interest_faces
FROM ideas i
JOIN users u ON u.id = i.author_id
WHERE CASE WHEN sqlc.narg('status')::text IS NULL THEN i.status <> 'done' ELSE i.status = sqlc.narg('status') END
  AND i.course = coalesce(sqlc.narg('course'), i.course)
  AND i.category = coalesce(sqlc.narg('category'), i.category)
  AND (sqlc.narg('skill')::text IS NULL OR sqlc.narg('skill') = ANY (i.skills))
  AND (sqlc.narg('search')::text IS NULL
       OR unaccent(i.title) ILIKE unaccent(sqlc.narg('search'))
       OR unaccent(i.description) ILIKE unaccent(sqlc.narg('search')))
ORDER BY i.done_at DESC NULLS LAST, i.created_at DESC, i.id DESC
LIMIT sqlc.narg('max');

-- name: GetIdea :one
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at, i.done_at, i.skills,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM interests n WHERE n.idea_id = i.id) AS interest_count,
       EXISTS (SELECT 1 FROM interests n WHERE n.idea_id = i.id AND n.user_id = sqlc.arg('viewer')) AS interested,
       (SELECT coalesce(json_agg(json_build_object('id', f.id, 'name', f.name, 'avatar_url', f.avatar_url) ORDER BY f.created_at, f.id), '[]')
        FROM (SELECT u2.id, u2.name, u2.avatar_url, n.created_at FROM interests n JOIN users u2 ON u2.id = n.user_id
              WHERE n.idea_id = i.id ORDER BY n.created_at, n.user_id LIMIT 8) f)::json AS interest_faces
FROM ideas i
JOIN users u ON u.id = i.author_id
WHERE i.id = sqlc.arg('id');

-- name: GetIdeaAuthor :one
-- O status vem junto: quando o AddInterest não grava, é ele que diz o porquê.
SELECT author_id, status FROM ideas WHERE id = $1;

-- name: AddInterest :execrows
-- Confere e grava na mesma instrução: só ideia aberta, e nunca o próprio autor. O FOR SHARE segura a linha da ideia
-- até o fim: se o autor troca o status ou apaga a ideia ao mesmo tempo, um espera o outro, e o insert reconfere a linha nova.
-- Repetir não dá erro nem duplica: a chave (idea_id, user_id) já existe e o insert vira nada.
-- 0 linhas = não gravou (não existe, não está aberta, é do autor ou já estava na lista); o handler descobre qual.
INSERT INTO interests (idea_id, user_id)
SELECT id, sqlc.arg('user_id') FROM ideas
WHERE id = sqlc.arg('idea_id') AND status = 'open' AND author_id <> sqlc.arg('user_id')
FOR SHARE
ON CONFLICT DO NOTHING;

-- name: RemoveInterest :exec
DELETE FROM interests WHERE idea_id = $1 AND user_id = $2;

-- name: ListInterests :many
-- Em ordem de chegada: quem marcou primeiro aparece primeiro para o autor.
SELECT u.id, u.name, u.avatar_url, u.discord_id, u.skills
FROM interests n
JOIN users u ON u.id = n.user_id
WHERE n.idea_id = $1
ORDER BY n.created_at, n.user_id;

-- name: UpdateIdea :exec
-- Troca os campos do postar; o handler busca a ideia completa com o GetIdea.
UPDATE ideas SET title = $2, description = $3, course = $4, category = $5, slots = $6, skills = $7
WHERE id = $1;

-- name: DeleteIdea :exec
-- Os interessados vão junto: interests tem ON DELETE CASCADE.
DELETE FROM ideas WHERE id = $1;

-- name: SetIdeaStatus :exec
-- done_at acompanha o status: preenchido ao concluir (repetir done mantém a data), null ao sair de done.
UPDATE ideas
SET status = sqlc.arg('status'),
    done_at = CASE WHEN sqlc.arg('status') = 'done' THEN COALESCE(done_at, now()) END
WHERE id = sqlc.arg('id');

-- name: ListAuthorIdeas :many
-- Todas as ideias de um autor, em qualquer status: "Minhas ideias" (viewer = autor) e o perfil.
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at, i.done_at, i.skills,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM interests n WHERE n.idea_id = i.id) AS interest_count,
       EXISTS (SELECT 1 FROM interests n WHERE n.idea_id = i.id AND n.user_id = sqlc.arg('viewer')) AS interested,
       (SELECT coalesce(json_agg(json_build_object('id', f.id, 'name', f.name, 'avatar_url', f.avatar_url) ORDER BY f.created_at, f.id), '[]')
        FROM (SELECT u2.id, u2.name, u2.avatar_url, n.created_at FROM interests n JOIN users u2 ON u2.id = n.user_id
              WHERE n.idea_id = i.id ORDER BY n.created_at, n.user_id LIMIT 8) f)::json AS interest_faces
FROM ideas i
JOIN users u ON u.id = i.author_id
WHERE i.author_id = sqlc.arg('author')
ORDER BY i.created_at DESC, i.id DESC;

-- name: ListParticipating :many
-- As ideias em que um aluno marcou "quero participar", em qualquer status, da marcação mais recente para a mais antiga.
SELECT i.id, i.title, i.description, i.course, i.category, i.slots, i.status, i.created_at, i.done_at, i.skills,
       u.id AS author_id, u.name AS author_name, u.avatar_url AS author_avatar_url,
       (SELECT count(*) FROM interests n WHERE n.idea_id = i.id) AS interest_count,
       EXISTS (SELECT 1 FROM interests n WHERE n.idea_id = i.id AND n.user_id = sqlc.arg('viewer')) AS interested,
       (SELECT coalesce(json_agg(json_build_object('id', f.id, 'name', f.name, 'avatar_url', f.avatar_url) ORDER BY f.created_at, f.id), '[]')
        FROM (SELECT u2.id, u2.name, u2.avatar_url, n.created_at FROM interests n JOIN users u2 ON u2.id = n.user_id
              WHERE n.idea_id = i.id ORDER BY n.created_at, n.user_id LIMIT 8) f)::json AS interest_faces
FROM interests p
JOIN ideas i ON i.id = p.idea_id
JOIN users u ON u.id = i.author_id
WHERE p.user_id = sqlc.arg('member')
ORDER BY p.created_at DESC, i.id DESC;

-- name: GetUser :one
SELECT id, name, avatar_url, course, discord_id, skills FROM users WHERE id = $1;
