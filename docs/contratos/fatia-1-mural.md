# Contrato — Fatia 1: postar ideia e ver no mural

Primeira fatia ponta a ponta. Objetivo único: provar que uma ideia sai da tela, passa pela API Go, grava no Postgres e volta para o mural.

Este contrato é o acordo entre front e back. Os dois lados codam contra ele em paralelo. Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

Login com Discord (ticket seguinte), habilidades, filtros, busca, paginação, comentários, "quero participar", troca de status.

Enquanto não houver login, toda ideia é criada por um **usuário fixo de teste**, inserido por migration/seed.

## Banco

Tabela `users` (mínima, só o necessário para ter autor):

| coluna | tipo | regra |
|---|---|---|
| id | bigserial | PK |
| discord_id | text | único, nulo permitido por enquanto |
| name | text | obrigatório |
| avatar_url | text | opcional |
| course | text | opcional |
| created_at | timestamptz | default now() |

Tabela `ideas`:

| coluna | tipo | regra |
|---|---|---|
| id | bigserial | PK |
| author_id | bigint | FK → users.id, obrigatório |
| title | text | obrigatório, 3–120 caracteres |
| description | text | obrigatório, até 5000 caracteres |
| course | text | obrigatório, valor da lista de cursos |
| category | text | obrigatório, valor da lista de categorias |
| slots | int | obrigatório, 1–20 |
| status | text | default `open`; valores: `open`, `in_progress`, `done` |
| created_at | timestamptz | default now() |

Índice em `ideas (created_at desc)` para a listagem.

## Listas fixas

> ⚠️ **Decisão pendente do Davi:** lista fixa (recomendado) ou texto livre. Se lista fixa, confirmar os valores abaixo.

- Cursos: `cc` (Ciência da Computação), `eng` (Engenharia), `ads` (ADS).
- Categorias: _a definir_ (ex.: `web`, `mobile`, `dados_ia`, `jogos`, `outro`).

A validação acontece na API. O front recebe as mesmas listas (constante compartilhada no MVP; endpoint próprio só se precisar).

## Endpoints

Base: `/api`. Corpo e resposta em JSON, chaves em snake_case.

### `POST /api/ideas` — cria uma ideia

Requisição:

```json
{
  "title": "App de carona para o campus",
  "description": "Quero montar um app simples para juntar quem mora perto...",
  "course": "cc",
  "category": "mobile",
  "slots": 3
}
```

Resposta `201 Created`:

```json
{
  "id": 1,
  "title": "App de carona para o campus",
  "description": "Quero montar um app simples para juntar quem mora perto...",
  "course": "cc",
  "category": "mobile",
  "slots": 3,
  "status": "open",
  "author": { "id": 1, "name": "Usuário de Teste" },
  "created_at": "2026-10-08T23:40:00Z"
}
```

Erros:

- `400 Bad Request` — campo faltando ou inválido. Formato:

```json
{
  "error": "validation_failed",
  "fields": { "slots": "deve ser entre 1 e 20" }
}
```

- `500 Internal Server Error` — `{ "error": "internal" }`, sem vazar detalhe interno.

### `GET /api/ideas` — lista o mural

Resposta `200 OK`, mais recentes primeiro:

```json
{
  "items": [
    {
      "id": 1,
      "title": "App de carona para o campus",
      "description": "Quero montar um app simples...",
      "course": "cc",
      "category": "mobile",
      "slots": 3,
      "status": "open",
      "author": { "id": 1, "name": "Usuário de Teste" },
      "created_at": "2026-10-08T23:40:00Z"
    }
  ]
}
```

Lista vazia retorna `{ "items": [] }`, nunca `null`. O objeto `items` já deixa espaço para paginação futura sem quebrar o contrato.

## Front

- Página do mural: lista os cards (título, curso, categoria, vagas, autor, data).
- Formulário "postar ideia": campos do POST, selects para curso e categoria, mostra os erros por campo vindos do `400`.
- Depois de postar com sucesso, a ideia aparece no topo do mural.

## Pronto quando

- [ ] `goose up` cria as tabelas e o usuário de teste.
- [ ] `POST` válido retorna 201 e grava; inválido retorna 400 com o campo.
- [ ] `GET` retorna as ideias na ordem certa, e `[]` quando vazio.
- [ ] Postar pela tela faz a ideia aparecer no mural.
- [ ] Testes dos handlers cobrindo sucesso e validação.
