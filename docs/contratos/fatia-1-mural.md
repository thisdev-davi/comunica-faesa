# Contrato — Fatia 1: postar ideia e ver no mural

Primeira fatia ponta a ponta. Objetivo único: provar que uma ideia sai da tela, passa pela API Go, grava no Postgres e volta para o mural.

Este contrato é o acordo entre front e back. Os dois lados codam contra ele em paralelo. Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

Login com Discord ([fatia 2](fatia-2-login.md)), habilidades, filtros, busca, paginação, comentários, "quero participar", troca de status.

> **Atualizado pela [fatia 2](fatia-2-login.md):** o autor da ideia é o usuário da sessão. O usuário fixo (`DEV_AUTHOR_ID`) usado nesta fatia foi removido.
>
> **Atualizado pela [fatia 3](fatia-3-cadastro.md):** `GET /api/ideas` devolve no máximo as 3 ideias mais recentes para visitante; logado vê todas.
>
> **Atualizado pela [fatia 4](fatia-4-pagina-da-ideia.md):** cada ideia tem uma página própria (`GET /api/ideas/{id}`, só para quem está logado); o card do mural leva até ela e corta a descrição em três linhas.
>
> **Atualizado pela [fatia 5](fatia-5-quero-participar.md):** cada item do `GET /api/ideas` ganha `interest_count` e `interested`.

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

Decidido: listas fixas. Valores em inglês; o front mostra os rótulos em português.

| curso | rótulo |
|---|---|
| `cc` | Ciência da Computação |
| `eng` | Engenharia |
| `ads` | ADS |

| categoria | rótulo |
|---|---|
| `web` | Web |
| `mobile` | Mobile |
| `ai` | IA |
| `data` | Dados |
| `games` | Jogos |
| `competitive_programming` | Maratona de Programação |
| `security` | Segurança |
| `iot` | IoT / Hardware |
| `other` | Outro |

A validação acontece na API. O front mantém a mesma lista como constante própria (com os rótulos); endpoint de listas só se precisar.

## Regras de validação

- `title`, `description`, `course` e `category` passam por trim antes de validar; o valor gravado é o limpo. Só espaços conta como vazio.
- Limites contados em **caracteres**, não bytes (`"ação"` tem 4).
- Texto com caractere nulo (`\u0000`) é inválido.
- Corpo da requisição limitado a 64 KB. Campos desconhecidos no JSON são ignorados.

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

- `400 Bad Request` — JSON malformado ou com tipo errado (ex.: `"slots": "3"`): `{ "error": "invalid_json" }`.
- `401 Unauthorized` — sem sessão válida: `{ "error": "unauthenticated" }`.
- `413 Content Too Large` — corpo acima de 64 KB: `{ "error": "body_too_large" }`.
- `500 Internal Server Error` — `{ "error": "internal" }`, sem vazar detalhe interno.

### `GET /api/ideas` — lista o mural

Resposta `200 OK`, mais recentes primeiro. Visitante (sem sessão) recebe no máximo 3 itens; logado recebe todos ([fatia 3](fatia-3-cadastro.md)).

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

- [ ] `goose up` cria as tabelas; o seed local cria o usuário de teste.
- [ ] `POST` válido retorna 201 e grava; inválido retorna 400 com o campo.
- [ ] `GET` retorna as ideias na ordem certa, e `{ "items": [] }` quando vazio.
- [ ] Postar pela tela faz a ideia aparecer no mural.
- [ ] Testes dos handlers cobrindo sucesso e validação.
