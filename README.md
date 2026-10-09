# Comunica FAESA

Plataforma para alunos de TI da FAESA (Ciência da Computação, Engenharia e ADS) acharem parceiros de projeto e terem mais oportunidades de botar a mão na massa juntos.

- **Mural de ideias:** o aluno posta uma ideia de projeto e encontra gente para participar.
- **Eventos:** a comunidade organiza e confirma presença em encontros, dias de trabalho e hackathons.

A conversa dos grupos acontece no Discord da comunidade; a plataforma não tem chat, só aponta para lá.

## Status

Em desenvolvimento, construído em fatias verticais (tela → API → banco).

- [x] Fatia 1, back: postar e listar ideias (`POST` / `GET /api/ideas`)
- [ ] Fatia 1, front: página do mural e formulário ([#2](https://github.com/thisdev-davi/comunica-faesa/issues/2))
- [ ] Login com Discord
- [ ] Demais funcionalidades do MVP (ver [PRD](docs/PRD.md))

## Stack

| Parte | Tecnologia |
|---|---|
| API | Go (`net/http`), REST/JSON |
| Banco | PostgreSQL, sqlc + pgx, migrations com goose |
| Front | Next.js (React) |
| Login | Discord OAuth |
| Deploy | Docker numa VPS, Caddy como proxy reverso |

## Rodando localmente

Pré-requisitos: Go, Docker com Compose e [goose](https://github.com/pressly/goose).

```sh
cp .env.example .env              # troque a senha
set -a; . ./.env; set +a          # exporta as variáveis no shell

docker compose up -d              # Postgres em localhost:5450
goose -dir backend/db/migrations postgres "$DATABASE_URL" up
goose -dir backend/db/migrations postgres "$TEST_DATABASE_URL" up
docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql   # usuário fictício

cd backend
go test ./...                     # testes
go run ./cmd/api                  # API em http://localhost:8090
```

Teste rápido:

```sh
curl -X POST localhost:8090/api/ideas \
  -d '{"title":"App de carona","description":"Juntar quem mora perto","course":"cc","category":"mobile","slots":3}'
curl localhost:8090/api/ideas
```

Enquanto não há login, as ideias são criadas pelo usuário fictício do seed (`DEV_AUTHOR_ID`). Sem essa variável, a API recusa criar ideias.

## Estrutura

```
backend/
  cmd/api/           ponto de entrada da API
  internal/<domínio> handler e queries por domínio (ideas, ...)
  db/migrations/     migrations do goose
  db/queries/        SQL lido pelo sqlc
  db/seed/           dados fictícios, só para uso local
docs/
  PRD.md             escopo do MVP e regras de negócio
  IDEIA.md           visão e fases futuras
  contratos/         contrato de API de cada fatia
```

## Como o projeto é tocado

Cada funcionalidade vira uma issue com a especificação, uma branch e um pull request pequeno, com no máximo 500 linhas de lógica. Front e back codam contra o contrato da fatia em `docs/contratos/`. As regras completas estão no [CLAUDE.md](CLAUDE.md).
