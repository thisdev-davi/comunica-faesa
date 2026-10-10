# Comunica FAESA

Plataforma para alunos de TI da FAESA (Ciência da Computação, Engenharia e ADS) acharem parceiros de projeto e terem mais oportunidades de botar a mão na massa juntos.

- **Mural de ideias:** o aluno posta uma ideia de projeto e encontra gente para participar.
- **Eventos:** a comunidade organiza e confirma presença em encontros, dias de trabalho e hackathons.

A conversa dos grupos acontece no Discord da comunidade; a plataforma não tem chat, só aponta para lá.

## Status

Em desenvolvimento, construído em fatias verticais (tela → API → banco).

- [x] Fatia 1, back: postar e listar ideias (`POST` / `GET /api/ideas`)
- [x] Fatia 1, front: tela provisória do mural e formulário (a identidade visual vem depois)
- [x] Fatia 2: login com Discord (só membros do servidor da comunidade)
- [x] Fatia 3: escolha de curso no primeiro login e prévia do mural para visitante
- [x] Fatia 4: página da ideia (`GET /api/ideas/{id}`), aberta pelo card do mural
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

Pré-requisitos: Go, Node.js 20.9+, Docker com Compose e [goose](https://github.com/pressly/goose).

```sh
cp .env.example .env              # troque a senha e preencha as DISCORD_* (abaixo)
set -a; . ./.env; set +a          # exporta as variáveis no shell

docker compose up -d              # Postgres em localhost:5450
goose -dir backend/db/migrations postgres "$DATABASE_URL" up
goose -dir backend/db/migrations postgres "$TEST_DATABASE_URL" up
docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql   # ideia fictícia

cd backend
go test ./...                     # testes
go run ./cmd/api                  # API em http://localhost:8090

# em outro terminal, na raiz
cd frontend && npm install && npm run dev   # mural em http://localhost:3000
```

Teste rápido: `curl localhost:8090/api/ideas` lista o mural (sem login, só as 3 ideias mais recentes). Postar exige login: entre por http://localhost:3000/login.

### App do Discord

O login usa OAuth do Discord e só deixa entrar quem está no servidor da comunidade.

1. No [Discord Developer Portal](https://discord.com/developers/applications), crie uma aplicação. Em **OAuth2**, adicione o redirect `http://localhost:3000/api/auth/discord/callback` e copie o **Client ID** e o **Client Secret** para `DISCORD_CLIENT_ID` e `DISCORD_CLIENT_SECRET` no `.env`.
2. No Discord, ative o **Modo desenvolvedor** (Configurações → Avançado), clique com o botão direito no servidor da comunidade → **Copiar ID** e cole em `DISCORD_GUILD_ID`.
3. Crie um convite sem expiração do servidor e cole em `DISCORD_INVITE_URL`.

O secret nunca vai para o repositório: ele vive só no `.env`.

## Estrutura

```
backend/
  cmd/api/           ponto de entrada da API
  internal/<domínio> handler e queries por domínio (ideas, ...)
  db/migrations/     migrations do goose
  db/queries/        SQL lido pelo sqlc
  db/seed/           dados fictícios, só para uso local
frontend/
  app/               páginas e componentes do Next.js (App Router)
docs/
  PRD.md             escopo do MVP e regras de negócio
  IDEIA.md           visão e fases futuras
  contratos/         contrato de API de cada fatia
```

## Como o projeto é tocado

Cada funcionalidade vira uma issue com a especificação, uma branch e um pull request pequeno, com no máximo 500 linhas de lógica. Front e back codam contra o contrato da fatia em `docs/contratos/`. As regras completas estão no [CLAUDE.md](CLAUDE.md).
