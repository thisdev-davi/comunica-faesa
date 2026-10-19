# Comunica FAESA

Plataforma para alunos de TI da FAESA (Ciência da Computação, Engenharia e ADS) acharem parceiros de projeto e terem mais oportunidades de botar a mão na massa juntos.

- **Mural de ideias:** o aluno posta uma ideia de projeto e encontra gente para participar.
- **Eventos:** a comunidade organiza e confirma presença em encontros, dias de trabalho e hackathons.

A conversa dos grupos acontece no Discord da comunidade; a plataforma não tem chat, só aponta para lá.

## Status

**MVP concluído** em 19 de outubro de 2026, construído em fatias verticais (tela → API → banco). A próxima fase sai da fila do [IDEIA](docs/IDEIA.md).

- [x] Fatia 1, back: postar e listar ideias (`POST` / `GET /api/ideas`)
- [x] Fatia 1, front: mural e formulário de postar ideia
- [x] Fatia 2: login com Discord (só membros do servidor da comunidade)
- [x] Fatia 3: escolha de curso no primeiro login e prévia do mural para visitante
- [x] Fatia 4: página da ideia (`GET /api/ideas/{id}`), aberta pelo card do mural
- [x] Fatia 5: "Quero participar" sem aprovação; o autor vê os interessados e chama no Discord
- [x] Fatia 6: o autor edita e apaga a própria ideia
- [x] Fatia 7: o autor troca o status da ideia; só ideia aberta recebe novos interessados
- [x] Fatia 8: vitrine de projetos concluídos (`/concluidas`) e navegação no cabeçalho
- [x] Fatia 9: "Minhas ideias", com todas as ideias do aluno em qualquer status
- [x] Fatia 10: perfil do aluno (`/alunos/<id>`), com as ideias que postou e as de que participa
- [x] Fatia 11: habilidades no perfil (até 5, de um catálogo fixo)
- [x] Fatia 12: habilidades procuradas na ideia, com destaque das que você tem
- [x] Fatia 13: agenda de eventos (`/eventos`) e criar evento
- [x] Fatia 14: página do evento, "Vou participar" e lista de quem vai
- [x] Fatia 15: cargo de moderação lido do Discord; o moderador apaga qualquer ideia
- [x] Fatia 16: o criador e o moderador editam e apagam o evento
- [x] Fatia 17: filtros do mural (status, curso, categoria e habilidade), para quem está logado
- [x] Fatia 18: busca por texto no título e na descrição, sem diferenciar maiúscula nem acento

Depois do fechamento, uma revisão do código virou correções sem funcionalidade nova: cookies `__Host-` e headers de segurança (PR #37), regras de participação e curso obrigatório na API (PR #38). A habilidade "Maratona de Programação" entrou no catálogo (PR #39).

Comentários nas ideias saíram do MVP e foram para a fila ([IDEIA](docs/IDEIA.md)).

Cada fatia tem o contrato em [`docs/contratos/`](docs/contratos/); as fases de eventos (13 a 16, concluída) e de descoberta (17 e 18) estão detalhadas no [PRD](docs/PRD.md).

## Stack

| Parte | Tecnologia |
|---|---|
| API | Go (`net/http`), REST/JSON |
| Banco | PostgreSQL, sqlc + pgx, migrations com goose |
| Front | Next.js (React) |
| Login | Discord OAuth |
| Deploy | Docker numa VPS, Caddy como proxy reverso |

## Rodando localmente

Pré-requisitos: Go 1.27+, Node.js 20.9+, Docker com Compose e [goose](https://github.com/pressly/goose).

```sh
cp .env.example .env              # troque a senha e preencha as DISCORD_* (abaixo)
set -a; . ./.env; set +a          # exporta as variáveis no shell

docker compose up -d              # Postgres em localhost:5450
goose -dir backend/db/migrations postgres "$DATABASE_URL" up
goose -dir backend/db/migrations postgres "$TEST_DATABASE_URL" up
docker compose exec -T db psql -U comunica -d comunica < backend/db/seed/dev.sql   # autor, ideia, evento e sessão de teste fictícios

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
4. Opcional, para a moderação: em **Configurações do servidor → Cargos**, clique com o botão direito nos cargos Admin e Moderação → **Copiar ID** e cole os dois em `DISCORD_MOD_ROLE_IDS`, separados por vírgula. Quem tem um desses cargos vira moderador no próximo login (ver abaixo).

#### Cargo de moderação

O cargo mora no Discord: quem dá e tira são os admins do servidor. A API relê os cargos a cada login e grava `users.is_moderator`. Tirar o cargo só vale no próximo login da pessoa, até 30 dias depois (a duração da sessão). O mesmo vale para quem é removido ou banido do servidor: a API só confere se a pessoa é membro no login, então ela continua com acesso até a sessão expirar. Nos dois casos, para valer na hora, apague as sessões dela, o que força um novo login (e quem saiu do servidor não consegue entrar de novo):

```sh
# local; na VPS, o mesmo DELETE dentro do container do banco (docker compose -f compose.prod.yaml exec db psql -U comunica -d comunica)
docker compose exec db psql -U comunica -d comunica -c "DELETE FROM sessions WHERE user_id = <id do usuário>;"
```

O secret nunca vai para o repositório: ele vive só no `.env`.

## Deploy na VPS

Em https://comunica.5-183-8-106.sslip.io. A VPS é dividida com outros projetos: o Comunica tem compose e Postgres próprios e nenhuma porta aberta no host. O HTTPS vem do Caddy compartilhado em `/opt/proxy`, que manda `/api/*` para `comunica-api` e o resto para `comunica-web` pela rede Docker `proxy`. A cada subida, o serviço `migrate` roda o goose antes da API.

Na VPS:

```
/opt/comunica-faesa/       root, 755: o usuário deploy não cria nem troca nada aqui
  compose.prod.yaml        root, 644: copiado à mão de app/
  deploy.sh                root, 644: copiado à mão de app/
  .env                     root:deploy, 640: segredos (o compose roda como deploy e precisa ler)
  app/                     deploy: clone do repositório, onde o deploy automático faz git pull
```

Por que o compose e o `deploy.sh` ficam fora do git: o usuário `deploy` está no grupo `docker`, que na prática vale root. Se esses dois arquivos viessem do `git pull`, um commit na `main` (o repositório é público) bastaria para virar root na VPS, por exemplo montando `/` num container. Fora do alcance do git, um commit na `main` só muda o código que roda dentro dos containers: o build do Docker é isolado, e volumes e privilégios vêm só do compose do root.

Primeira vez:

1. No Discord Developer Portal, adicione o redirect `https://comunica.5-183-8-106.sslip.io/api/auth/discord/callback` (o de localhost continua).
2. Na VPS, como root:

   ```sh
   useradd -m -s /bin/sh -G docker deploy
   install -d -m 755 -o root -g root /opt/comunica-faesa
   git clone https://github.com/thisdev-davi/comunica-faesa.git /opt/comunica-faesa/app
   chown -R deploy:deploy /opt/comunica-faesa/app
   cd /opt/comunica-faesa
   install -o root -g root -m 644 app/compose.prod.yaml app/deploy.sh .
   # .env só com estas chaves; a senha em hex porque vai dentro da URL do banco
   cat > .env <<EOF
   POSTGRES_PASSWORD=$(openssl rand -hex 24)
   DISCORD_CLIENT_ID=
   DISCORD_CLIENT_SECRET=
   DISCORD_REDIRECT_URL=https://comunica.5-183-8-106.sslip.io/api/auth/discord/callback
   DISCORD_GUILD_ID=
   DISCORD_INVITE_URL=
   DISCORD_MOD_ROLE_IDS=
   EOF
   chown root:deploy .env && chmod 640 .env && nano .env      # preencha as DISCORD_*
   docker compose -f compose.prod.yaml up -d --build
   ```

3. No fim do `/opt/proxy/Caddyfile` (sem mexer nos outros blocos), depois de guardar uma cópia com `cp Caddyfile Caddyfile.antes-comunica`:

   ```
   # Comunica FAESA
   comunica.5-183-8-106.sslip.io {
   	encode gzip
   	handle /api/* {
   		reverse_proxy comunica-api:8090
   	}
   	handle {
   		reverse_proxy comunica-web:3000
   	}
   }
   ```

   E recarregue: `cd /opt/proxy && docker compose exec caddy caddy reload -c /etc/caddy/Caddyfile`.

### Deploy automático

Todo push na `main` (menos os só de `docs/` e `.md`) roda o workflow `.github/workflows/deploy.yml`: testes da API e checagens do front e, se passarem, o deploy na VPS. O mesmo workflow roda as checagens em todo PR. Para refazer o deploy sem commit novo (ex.: depois de trocar o `.env`): aba **Actions** → **CI e deploy** → **Run workflow**, ou `gh workflow run deploy.yml`.

O GitHub entra na VPS como o usuário `deploy`, nunca como root, com uma chave que só serve para isso: o `authorized_keys` força o comando, então quem tiver a chave não abre shell nem faz túnel, só atualiza `app/` e roda o `deploy.sh` do root. Os segredos ficam no environment `production` do GitHub, liberado só para a `main`:

- `DEPLOY_SSH_KEY`: chave privada (gerada só para isso, nunca a de outro projeto)
- `DEPLOY_KNOWN_HOSTS`: a linha `<host> ssh-ed25519 ...` da chave do servidor, para não aceitar um servidor falso
- `DEPLOY_HOST`: endereço da VPS

A chave na VPS (uma vez, como root):

```sh
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
echo 'restrict,command="cd /opt/comunica-faesa/app && git pull --ff-only -q && exec sh /opt/comunica-faesa/deploy.sh" <chave pública>' > /home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys && chmod 600 /home/deploy/.ssh/authorized_keys
```

Se o deploy falhar, o job fica vermelho na aba Actions. Se foi no build, os containers antigos continuam no ar. Logs: `cd /opt/comunica-faesa && docker compose -f compose.prod.yaml logs --tail 50 comunica-api`.

### Mudou o `compose.prod.yaml` ou o `deploy.sh`?

O deploy automático não leva essas mudanças: depois do merge, ele já atualizou `app/`, mas a VPS continua usando as cópias do root. Revise o que mudou e copie à mão, como root:

```sh
cd /opt/comunica-faesa
diff -u compose.prod.yaml app/compose.prod.yaml; diff -u deploy.sh app/deploy.sh   # leia antes de copiar
install -o root -g root -m 644 app/compose.prod.yaml app/deploy.sh .
```

Depois, rode o deploy pelo **Run workflow**. Se o código do merge depende da mudança (ex.: uma variável nova no compose), o deploy automático do merge pode falhar até a cópia; o Run workflow resolve.

O seed nunca roda na VPS: lá só entram dados reais, criados pelo próprio site.

## Estrutura

```
backend/
  cmd/api/           ponto de entrada da API
  internal/<domínio> handler e queries por domínio (auth, ideas, events)
  db/migrations/     migrations do goose
  db/queries/        SQL lido pelo sqlc
  db/seed/           dados fictícios, só para uso local
frontend/
  app/               páginas e componentes do Next.js (App Router)
docs/
  PRD.md             escopo do MVP e regras de negócio
  IDEIA.md           visão e fases futuras
  contratos/         contrato de API de cada fatia
  DESIGN.md          regras de interface: visual, telas, texto e acessibilidade
```

## Como o projeto é tocado

Cada funcionalidade vira uma issue com a especificação, uma branch e um pull request pequeno, com no máximo 500 linhas de lógica. Front e back codam contra o contrato da fatia em `docs/contratos/`. As regras completas estão no [CLAUDE.md](CLAUDE.md).
