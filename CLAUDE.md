# CLAUDE.md

Plataforma para alunos de TI da FAESA acharem parceiros de projeto (mural de ideias) e marcarem eventos. Projeto pessoal do Davi, que também é um projeto de aprendizado de Go.

## Leia antes de qualquer tarefa

- `docs/PRD.md` — escopo do MVP, modelo de dados, regras de negócio e o que está fora de escopo. É a fonte da verdade do produto.
- `docs/IDEIA.md` — visão, fila de fases futuras e a régua de decisão.
- `docs/contratos/` — contratos de API por fatia. Front e back codam contra o contrato; se precisar mudar, mude o contrato primeiro e avise.

Na dúvida sobre escopo, aplique a régua: "sem isso, a dor principal continua sem solução?". Se não, não entra — anote como sugestão e siga.

## Stack

- **backend/** — Go, API REST. Postgres via **sqlc + pgx** (SQL puro, sem ORM). Migrations com **goose**.
- **frontend/** — Next.js (React).
- **Auth** — Discord OAuth (`golang.org/x/oauth2`). Chat e conversa ficam no Discord, a plataforma não constrói chat.
- **Deploy** — Docker na VPS, Caddy como proxy reverso por subdomínio.

## Estrutura do backend

```
backend/
  cmd/api/main.go      # ponto de entrada: sobe o servidor
  internal/<domínio>/  # ideas, users, events... handler, serviço e queries juntos por domínio
  db/migrations/       # arquivos do goose
  db/queries/          # SQL que o sqlc lê
  db/seed/             # dados fictícios só para o banco local
```

- Nada de pasta `utils`, `helpers` ou `common`. Código mora no domínio a que pertence.
- Biblioteca padrão primeiro (`net/http` com o roteador do Go 1.22+, `encoding/json`, `log/slog`). Dependência nova só com justificativa no PR.
- Código gerado pelo sqlc não é editado à mão.

## Regras de trabalho (obrigatórias)

1. **Um ticket = um pull request = no máximo 500 linhas de código de lógica.**
   - Conta: handlers, serviços, regras de negócio, queries escritas à mão, componentes e lógica do front.
   - Não conta: código gerado pelo sqlc, migrations, YAML, configs, scripts, testes e documentação.
   - Se a tarefa não cabe em 500 linhas, pare e proponha como dividir antes de codar.
2. **O Davi revisa e faz o merge de tudo.** Nunca dar merge, push na `main` ou force push.
3. **Commits granulares**, separados por contexto lógico (ex.: migration, query, handler, teste — cada um no seu), seguindo o padrão de nomenclatura abaixo.
4. **Segredos nunca entram no repositório.** Credenciais (Discord client secret, string do banco) vivem em `.env`, que está no `.gitignore`. Mantenha um `.env.example` atualizado sem valores reais.
5. **Escreva o mínimo que funciona** (Ponytail ligado), mas **nunca** corte validação na entrada da API, tratamento de erro, segurança ou acessibilidade.
6. Toda rota nova tem teste cobrindo o caminho feliz e a validação.
7. **O repositório é público.** Além de segredos, nunca entram: dados pessoais reais (nomes, Discord IDs, e-mails de alunos), dumps ou backups de banco, logs com dados de usuário. Dado de teste é fictício, mora em `db/seed/` e roda só no banco local — nunca por migration. Rota que grava dados não vai para produção sem login.

## Fluxo de cada ticket

1. Davi roda o `grill-me` sobre a fatia; o resultado vira a especificação do ticket.
2. O agente implementa contra a especificação e o contrato, com Ponytail ativo.
3. Antes de abrir o PR, rodar `/ponytail-review` no diff e aplicar o que fizer sentido.
4. Abrir o PR com: o que foi feito, contagem aproximada de linhas de lógica e **as decisões de Go não óbvias explicadas em uma frase cada** — o Davi está aprendendo Go e o PR também é material de estudo.
5. Davi revisa e decide o merge.

## Skills

Skills em uso no projeto:

- **grill-me** (mattpocock/skills) — entrevista sobre a fatia antes do ticket; o resultado vira a especificação.
- **Ponytail** (DietrichGebert/ponytail) — ativo durante a implementação; `/ponytail-review` no diff antes do PR.
- **find-skills** (vercel-labs/skills) — busca skills prontas no ecossistema skills.sh.

Quando usar o find-skills: ao começar algo de domínio específico em que uma skill pronta pode trazer boas práticas (ex.: testes em Go, padrões do Next.js, OAuth, Docker/Caddy, Postgres), rode uma busca antes de improvisar — `npx skills find <termo>`.

Regras para skills novas:

- **Não instale nenhuma skill sem aprovação do Davi.** Skills executam comandos e entram no contexto de todo agente. Apresente as opções e espere a decisão.
- Ao apresentar, informe para cada opção: o que ela faz, a fonte (dono do repositório), o número de instalações e de estrelas, e o comando de instalação. Prefira fontes conhecidas e skills com 1K+ instalações; desconfie de repositórios com menos de 100 estrelas.
- Leia o SKILL.md da candidata e avise se ela pedir para rodar scripts ou instalar dependências.
- Se nada servir, siga sem skill. Não crie skill nova por conta própria; sugira ao Davi se o mesmo procedimento se repetir.

## Commits, branches e PRs

Sem Claude CO-Author

**Commit:** `<tipo>(<escopo>) - <descrição breve>`

```
feat(db) - cria tabela ideas
fix(api) - valida slots maior que zero
test(ideas) - cobre validacao do post
```

**Branch e título do PR:** `<tipo>/<descricao-breve>`

```
feat/mural-de-ideias
fix/validacao-slots
```

- Tipos: `feat` (funcionalidade), `fix` (correção), `refactor`, `test`, `docs`, `chore` (config, dependências, build).
- Escopo: a área tocada — `db`, `api`, `auth`, `front`, ou o domínio (`ideas`, `users`, `events`).
- Descrição curta, em minúsculas, no imperativo. Na branch, palavras separadas por hífen.
- Uma branch e um PR por ticket.

## Convenções

- Código, nomes de tabelas, colunas e chaves JSON em inglês, snake_case no banco e no JSON.
- Documentação e textos da interface em português.
- Erros da API no formato `{ "error": "<codigo>", "fields": { ... } }` (ver contratos).

## Comandos

Primeira vez: `cp .env.example .env` e troque a senha. A API não lê o `.env` sozinha — exporte antes, na raiz: `set -a; . ./.env; set +a`.

```sh
# banco (Postgres na porta 5450, cria também o comunica_test)
docker compose up -d

# migrations (goose instalado na máquina) — dev e teste
goose -dir backend/db/migrations postgres "$DATABASE_URL" up
goose -dir backend/db/migrations postgres "$TEST_DATABASE_URL" up

# usuário fictício de teste, só no banco local (id 1 = DEV_AUTHOR_ID)
docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql

# gerar código do sqlc (via Docker, versão fixa) depois de mudar migrations ou queries
cd backend && docker run --rm -u "$(id -u):$(id -g)" -v "$PWD":/src -w /src sqlc/sqlc:1.29.0 generate

# testes (sem TEST_DATABASE_URL, os que usam banco são pulados)
cd backend && go vet ./... && go test ./...

# subir a API em http://localhost:8090
cd backend && go run ./cmd/api
```

_Front: a preencher no ticket `feat/front-mural`._
