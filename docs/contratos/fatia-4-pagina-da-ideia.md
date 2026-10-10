# Contrato — Fatia 4: página da ideia

Objetivo único: o aluno logado abre uma ideia do mural e vê todos os detalhes numa página própria. O card do mural passa a mostrar só um resumo.

Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

"Quero participar" e lista de interessados (fatia 5), comentários, foto do autor, habilidades, editar ou apagar a ideia, troca de status.

## Banco

Sem migration.

## Endpoints

### `GET /api/ideas/{id}` — uma ideia

Só para quem está logado: a página é a ideia inteira, e o visitante continua vendo só a prévia do mural ([fatia 3](fatia-3-cadastro.md)).

Resposta `200 OK`, o mesmo objeto de um item do `GET /api/ideas` ([fatia 1](fatia-1-mural.md)):

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

- `401` — sem sessão válida: `{ "error": "unauthenticated" }`.
- `404` — a ideia não existe, ou o `id` não é um inteiro positivo (`abc`, `0`, `-1`): `{ "error": "not_found" }`. Para quem chama, um id inválido é só uma ideia que não existe.
- `500` — `{ "error": "internal" }`.

## Front

- `/ideias/[id]`: visitante vai para `/login`; logado sem curso vai para `/cadastro`. Senão, mostra curso, categoria, status, título, a descrição inteira, autor, vagas, data e um link "Voltar ao mural".
- Ideia inexistente ou id inválido: página "Ideia não encontrada" com o link para o mural, com status HTTP 404.
- A página é renderizada no servidor do Next, que chama a API Go direto repassando o cookie do navegador (mesmo esquema do mural).
- Card do mural: o título leva para a página da ideia (o card inteiro é clicável), e a descrição corta em três linhas.

## Pronto quando

- [ ] Clicar num card do mural abre a página da ideia com a descrição inteira.
- [ ] Ideia inexistente ou id inválido mostra "Ideia não encontrada".
- [ ] Visitante que abre `/ideias/<id>` vai para `/login`; a API responde `401`.
- [ ] Testes do handler cobrindo sucesso, `401` e `404`.
