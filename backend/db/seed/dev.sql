-- Só para o banco LOCAL. Dados fictícios; nunca coloque dados reais aqui (repo público).
-- Rodar uma vez depois do goose up (da raiz): docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql
-- Autor sem discord_id (ninguém consegue logar como ele) e uma ideia de exemplo, para o mural não começar vazio.
WITH u AS (INSERT INTO users (name, course) VALUES ('Usuário de Teste', 'cc') RETURNING id)
INSERT INTO ideas (author_id, title, description, course, category, slots)
SELECT id, 'App de carona para o campus', 'Ideia fictícia de exemplo, criada pelo seed local.', 'cc', 'mobile', 3 FROM u;
