# Contrato — Fatia 5: quero participar

Objetivo único: o aluno logado clica em "Quero participar" e entra direto na lista de interessados da ideia, sem aprovação. O autor vê a lista e chama cada pessoa no Discord, com a primeira mensagem já pronta.

Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

Aviso ao autor quando alguém entra (o bot do Discord é fase futura), travar novos interessados e trocar o status (fatia de status), o @ do autor para o interessado chamar, vagas preenchidas automaticamente, perfil com "ideias que participei".

## Regras

- Sem aprovação: marcar o interesse já põe a pessoa na lista.
- O interesse **não ocupa vaga** e não trava a ideia: "precisa de N pessoas" continua só informativo.
- Dá para desistir: o botão alterna.
- O autor não participa da própria ideia.
- A lista de interessados (com o `discord_id`) só vai para o autor; os outros veem só o número.
- O status não é conferido nesta fatia: hoje toda ideia está aberta. Travar entra com a troca de status.

## Banco

Tabela `interests` (nova):

| coluna | tipo | regra |
|---|---|---|
| idea_id | bigint | FK → ideas.id, `ON DELETE CASCADE` |
| user_id | bigint | FK → users.id, `ON DELETE CASCADE` |
| created_at | timestamptz | default now() |

Chave primária `(idea_id, user_id)`: a mesma pessoa não entra duas vezes, e o índice da chave serve para contar os interessados de cada ideia.

## Endpoints

### `PUT /api/ideas/{id}/interest` — quero participar

Sem corpo. Responde `204 No Content`. Repetir não muda nada (idempotente).

Erros:

- `401` — sem sessão válida: `{ "error": "unauthenticated" }`.
- `404` — a ideia não existe ou o `id` não é inteiro positivo: `{ "error": "not_found" }`.
- `409` — quem chama é o autor da ideia: `{ "error": "own_idea" }`.
- `500` — `{ "error": "internal" }`.

### `DELETE /api/ideas/{id}/interest` — desistir

Responde `204` sempre que há sessão, mesmo se o interesse ou a ideia não existem (o resultado é o mesmo: a pessoa não está na lista). Sem sessão: `401 { "error": "unauthenticated" }`.

### Mudança na fatia 1: `GET /api/ideas`

Cada item ganha:

```json
{ "interest_count": 2, "interested": false }
```

- `interest_count`: quantas pessoas marcaram interesse.
- `interested`: se quem chama está na lista (`false` para visitante).

### Mudança na fatia 4: `GET /api/ideas/{id}`

Ganha `interest_count` e `interested`, como no mural. Quando quem chama é o **autor**, ganha também a lista, em ordem de chegada:

```json
{
  "interests": [
    { "id": 7, "name": "Ana Ribeiro", "avatar_url": "https://cdn.discordapp.com/avatars/…png", "discord_id": "…" }
  ]
}
```

- Para quem não é o autor, `interests` não vem no JSON.
- Autor sem interessados recebe `"interests": []`.
- `avatar_url` e `discord_id` podem ser `null`.

## Front

- O card do mural **não tem ação**: para participar, o aluno abre a ideia (o card inteiro leva à página da [fatia 4](fatia-4-pagina-da-ideia.md)). No card, quem já está na ideia vê o selo "Você participa" ao lado do status e a borda `accent`.
- Botão "Quero participar" (`accent`) só na página da ideia. Depois do clique, vira "Interessado" (`secondary`, `aria-pressed="true"`); clicar de novo desiste. Não aparece para visitante nem para o autor.
- Depois de marcar ou desmarcar, o front pede a página de novo ao servidor (`router.refresh()`), que traz o contador e o estado atualizados. Sessão expirada (`401`) vai para `/login`.
- Card e página mostram "Precisa de N pessoas · M interessados" numa linha e "postada em DD/MM" na de baixo.
- Página da ideia, para quem é interessado: "Você está na lista. O autor vai te chamar no Discord."
- Página da ideia, para o autor: seção "Interessados (N)", com foto, nome e o link "Chamar no Discord" de cada pessoa. Sem interessados: "Ninguém se interessou ainda."
- "Chamar no Discord" abre `https://discord.com/users/<discord_id>` numa nova aba e, no mesmo clique, copia a mensagem `Oi, <primeiro nome>! Vi que você quer participar da ideia “<título>” no Comunica FAESA. Bora conversar?`, com o aviso "Mensagem copiada, cole na conversa." O Discord não tem link que abra uma conversa já escrita; o perfil é o mais perto disso. Se a cópia falhar, o perfil abre do mesmo jeito.

## Pronto quando

- [ ] `goose up` cria `interests`.
- [ ] Na página da ideia, "Quero participar" põe o aluno na lista; clicar de novo tira.
- [ ] No mural, o card de quem participa mostra "Você participa"; o card não tem botão.
- [ ] O autor não vê o botão na própria ideia, e a API responde `409`.
- [ ] O autor vê a lista na página da ideia e o link abre o perfil no Discord com a mensagem copiada.
- [ ] Quem não é autor vê só o número de interessados.
- [ ] Testes dos handlers cobrindo `PUT`, `DELETE`, a lista só para o autor e o contador no mural.
