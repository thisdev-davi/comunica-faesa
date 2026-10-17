# Contrato — Fatia 16: editar e apagar o evento

Última fatia da [fase de eventos](../PRD.md#fase-de-eventos). Objetivo único: o criador corrige o próprio evento (mudou a sala, a hora) ou o apaga, e um moderador faz o mesmo em qualquer evento. Molde: a [fatia 6](fatia-6-editar-e-apagar.md), com o moderador da [fatia 15](fatia-15-cargo-de-moderacao.md).

Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

Status "cancelado" (apagar resolve), avisar quem confirmou presença (fica com o Discord), histórico de edições, arquivar em vez de apagar.

## Regras

- Só o criador ou quem tem `is_moderator` edita ou apaga. Os outros recebem `403`.
- Editar troca os mesmos campos do criar, com a mesma validação da [fatia 13](fatia-13-agenda-de-eventos.md#regras-de-validação), inclusive `starts_at` no futuro. Autor, data de criação e a lista de presença não mudam por aqui.
- Apagar é de vez: o evento some e a lista de presença vai junto (`event_attendees` tem `ON DELETE CASCADE`). Funciona também no evento que já passou.
- A sessão e o autor são conferidos antes do corpo: quem não pode editar recebe `403` mesmo com corpo inválido.

## Banco

Sem migration.

## Endpoints

Erros comuns aos dois:

- `401` — sem sessão válida: `{ "error": "unauthenticated" }`.
- `403` — nem criador nem moderador: `{ "error": "not_author" }`.
- `404` — o evento não existe, ou o `id` não é inteiro positivo: `{ "error": "not_found" }`.
- `500` — `{ "error": "internal" }`.

### `PUT /api/events/{id}` — edita o evento

Requisição: o mesmo corpo do `POST /api/events`, com todos os campos.

Resposta `200 OK`: o evento atualizado, no formato de um item da agenda (com `attendee_count` e `going`), **sem** `attendees`.

Erros, além dos comuns: `400 validation_failed` com `fields`, `400 invalid_json` e `413 body_too_large`, iguais aos do `POST`.

### `DELETE /api/events/{id}` — apaga o evento

Sem corpo. Responde `204 No Content`.

## Front

- **Página do evento:** o criador vê o painel "Seu evento"; o moderador que não é o criador vê o painel "Moderação", com a frase "Você pode editar e apagar este evento porque tem o cargo de moderação no Discord.". Os dois têm "Editar evento" (`secondary`) e "Apagar evento" (`danger`). Apagar confirma na caixa nativa ("Apagar este evento? Quem confirmou presença sai da lista e não dá para desfazer.") e volta para `/eventos`.
- **`/eventos/<id>/editar`:** o formulário do criar, já preenchido (data e hora no horário de Brasília), com "Editar evento" e "Salvar alterações". Quem não é criador nem moderador vai para a página do evento. Depois de salvar, volta para a página do evento ("Voltar ao evento" no topo).

## Checklist

- [ ] `PUT`: criador e moderador 200; outro 403 (mesmo com corpo inválido); inválido 400; inexistente 404.
- [ ] `DELETE`: criador e moderador 204, e a presença some junto; outro 403; inexistente 404.
- [ ] Painéis, editar e apagar testados no navegador, nos dois temas e na tela estreita.
