# Contrato — Fatia 13: agenda de eventos

Primeira fatia da fase de eventos (decisões na seção [Eventos do PRD](../PRD.md#fase-de-eventos)). Objetivo único: o aluno logado cria um evento (título, descrição, data e hora, local) e todo mundo, inclusive o visitante, vê a agenda dos próximos eventos.

Mudou o contrato? Atualiza este arquivo **antes** do código.

> **Atualizado pela [fatia 14](fatia-14-presenca-em-eventos.md):** todo evento ganha `attendee_count` e `going`; o card leva à página do evento (`/eventos/<id>`), e criar evento vai para a página dele.
>
> **Atualizado pela [fatia 16](fatia-16-editar-e-apagar-evento.md):** o criador e o moderador editam (`PUT /api/events/{id}`) e apagam (`DELETE /api/events/{id}`) o evento.

## Fora desta fatia (de propósito)

Página do evento e confirmação de presença "vou" (fatia 14), cargo de moderação (fatia 15), editar e apagar evento (fatia 16). Também ficam fora: horário de fim, tipo do evento, limite de vagas, ligação com uma ideia, seção de eventos passados, eventos no perfil, paginação.

## Regras

- Qualquer aluno logado cria evento, sem aprovação. Na tela, o logado sem curso vai para `/cadastro` antes, como no mural.
- O evento é independente de ideia.
- A agenda mostra os eventos **de hoje em diante** (no horário de Brasília), do mais próximo para o mais distante. O evento de hoje fica na lista até o fim do dia, mesmo depois de começar.
- O visitante vê a agenda inteira (sem prévia de 3): evento é convite.
- O horário é sempre o de Brasília. O Brasil não tem horário de verão desde 2019, então o front envia com `-03:00` fixo.

## Banco

Tabela `events` (nova):

| coluna | tipo | regra |
|---|---|---|
| id | bigserial | PK |
| author_id | bigint | FK → users.id, obrigatório |
| title | text | obrigatório, 3–120 caracteres |
| description | text | obrigatório, até 5000 caracteres |
| starts_at | timestamptz | obrigatório, no futuro ao criar |
| location | text | obrigatório, 3–120 caracteres |
| created_at | timestamptz | default now() |

Índice em `events (starts_at)` para a agenda.

## Regras de validação

Iguais às da [fatia 1](fatia-1-mural.md#regras-de-validação): trim antes de validar, limites em caracteres, caractere nulo é inválido, corpo de até 64 KB, campos desconhecidos ignorados.

- `starts_at`: texto RFC3339 **com fuso** (`"2026-10-20T19:00:00-03:00"`). Vazio é `"obrigatório"`; fora do formato é `"data inválida"`; agora ou antes é `"deve ser no futuro"`.

## Endpoints

### `POST /api/events` — cria um evento

Requisição:

```json
{
  "title": "Noite de trabalho do app de carona",
  "description": "Vamos fechar as telas de login. Traga o notebook.",
  "starts_at": "2026-10-20T19:00:00-03:00",
  "location": "Bloco B, sala 204"
}
```

Resposta `201 Created`: o evento criado, no formato de um item da agenda.

Erros:

- `400` — `{ "error": "invalid_json" }` ou `{ "error": "validation_failed", "fields": { "starts_at": "deve ser no futuro" } }`.
- `401` — sem sessão válida: `{ "error": "unauthenticated" }`.
- `413` — corpo acima de 64 KB: `{ "error": "body_too_large" }`.
- `500` — `{ "error": "internal" }`.

### `GET /api/events` — a agenda

Para todo mundo, visitante incluído. Eventos de hoje em diante, `starts_at` crescente (empate pelo `id`). Lista vazia vem como `[]`.

```json
{
  "items": [
    {
      "id": 1,
      "title": "Noite de trabalho do app de carona",
      "description": "Vamos fechar as telas de login. Traga o notebook.",
      "starts_at": "2026-10-20T22:00:00Z",
      "location": "Bloco B, sala 204",
      "author": { "id": 1, "name": "Usuário de Teste", "avatar_url": null },
      "created_at": "2026-10-16T12:00:00Z"
    }
  ]
}
```

Datas saem em UTC; o front formata no horário de Brasília. `author` é quem criou e aparece como "organizado por".

## Front

- **Navegação:** "Eventos" entre "Mural" e "Concluídos", para todo mundo.
- **`/eventos`:** título "Eventos", texto de apoio, "Criar evento" à direita (só logado), e a agenda em lista de `EventCard`. Visitante lê o convite para entrar. Vazio: "Nenhum evento marcado". Erro: o `StatePanel` de erro.
- **`EventCard`:** o bloco de data (dia e mês) em `primary` com o losango âmbar, o título, a hora e o local, a descrição em até 3 linhas e o organizador (foto e nome, link para o perfil). Nesta fatia, o card não leva a lugar nenhum além do perfil: a página do evento é da fatia 14.
- **`/eventos/novo`:** só logado com curso. Título, descrição, data e hora (`<input type="datetime-local">`, sem `min`: os erros são os da API) e local. Depois de criar, volta para `/eventos`.

## Checklist

- [ ] Migration cria `events`.
- [ ] `POST` válido grava e responde `201`; inválido responde `400` com os campos; sem sessão, `401`.
- [ ] `starts_at` no passado, sem fuso ou vazio é recusado.
- [ ] `GET` devolve só os de hoje em diante, do mais próximo; visitante recebe a lista inteira.
- [ ] `/eventos` e `/eventos/novo` testados no navegador nos dois temas e na tela estreita.
