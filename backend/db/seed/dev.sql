-- Só para o banco LOCAL. Dados fictícios; nunca coloque dados reais aqui (repo público).
-- Rodar uma vez depois do goose up (da raiz): docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql
-- Num banco limpo o usuário recebe id 1, que é o DEV_AUTHOR_ID do .env.example.
INSERT INTO users (name, course) VALUES ('Usuário de Teste', 'cc');
