-- Só para o banco LOCAL. Dados fictícios; nunca coloque dados reais aqui (repo público).
-- Rodar uma vez depois do goose up (da raiz): docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql
-- Autor sem discord_id (não entra pelo Discord, só pela sessão fixa logo abaixo) e uma ideia de exemplo, para o mural não começar vazio.
WITH u AS (INSERT INTO users (name, course) VALUES ('Usuário de Teste', 'cc') RETURNING id)
INSERT INTO ideas (author_id, title, description, course, category, slots)
SELECT id, 'App de carona para o campus', 'Ideia fictícia de exemplo, criada pelo seed local.', 'cc', 'mobile', 3 FROM u;

-- Sessão fixa do "Usuário de Teste", para o Playwright CLI entrar logado no front local (cookie __Host-session=sessao-local-de-teste).
-- O token é público (está no repositório) e só vale neste banco: o seed nunca roda na VPS.
INSERT INTO sessions (token_hash, user_id, expires_at)
SELECT sha256(convert_to('sessao-local-de-teste', 'UTF8')), id, now() + interval '1 year'
FROM users WHERE name = 'Usuário de Teste' AND discord_id IS NULL
ORDER BY id LIMIT 1
ON CONFLICT (token_hash) DO UPDATE SET expires_at = EXCLUDED.expires_at; -- rodar o seed de novo renova a sessão

-- Um evento fictício daqui a uma semana, às 19h de Brasília (22h UTC), para a agenda não começar vazia.
INSERT INTO events (author_id, title, description, starts_at, location)
SELECT id, 'Noite de trabalho do app de carona', 'Evento fictício de exemplo, criado pelo seed local.',
       date_trunc('day', now()) + interval '7 days 22 hours', 'Bloco B, sala 204'
FROM users WHERE name = 'Usuário de Teste' AND discord_id IS NULL
ORDER BY id LIMIT 1;
