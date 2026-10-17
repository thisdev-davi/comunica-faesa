# Contrato — Fatia 17: filtros do mural

Objetivo único: quem está logado estreita o mural por status, curso, categoria e habilidade procurada, para achar a ideia em que pode entrar sem rolar a lista inteira.

Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

Busca por texto (fatia 18), filtros na vitrine (a API já aceita, o front não mostra), mais de um valor por filtro, atalho "combinam com minhas habilidades", paginação (o mural devolve tudo; entra quando passar de umas centenas de ideias).

## Regras

- Quatro filtros, um valor cada, e eles se somam (curso **e** categoria **e** habilidade **e** status).
- Habilidade: a ideia aparece se procura aquela habilidade (uma das `skills` dela).
- Status no mural: "Aberta" ou "Em andamento". A concluída continua só na vitrine.
- Só quem está logado filtra. O visitante não vê a barra, e a API ignora os filtros dele: continua recebendo a prévia de 3 (senão, filtrando, ele veria mais que a prévia).
- O filtro mora na URL (`/?course=cc&skill=backend`): o link filtrado vai para o Discord, e o voltar do navegador funciona. As chaves são as mesmas da API, e o front repassa.

## Endpoints

### Mudança na fatia 1 e na fatia 8: `GET /api/ideas`

Parâmetros novos de query, todos opcionais. Vazio (`?course=`) é o mesmo que não mandar.

| parâmetro | valores | sem ele |
|---|---|---|
| `status` | `open`, `in_progress`, `done` | abertas e em andamento (o mural) |
| `course` | os cursos da fatia 1 | todos |
| `category` | as categorias da fatia 1 | todas |
| `skill` | o catálogo da fatia 11 | todas |

`?status=done` continua sendo a vitrine, agora também com os outros filtros. Antes desta fatia, `open` e `in_progress` respondiam `400`; agora filtram o mural.

A ordem e o formato da resposta não mudam. Para o visitante, só `status=done` vale (a vitrine); os outros filtros e os outros status são ignorados.

Erro (`400 validation_failed`): valor fora da lista, no campo do parâmetro, com `"valor inválido"`. Vale também para o visitante.

```json
{ "error": "validation_failed", "fields": { "skill": "valor inválido" } }
```

## Front

- Mural, só para quem está logado: entre o topo e a grade, um `<form method="get" role="search">` com quatro `Select` nativos ("Status", "Curso", "Categoria", "Habilidade"), cada um com a primeira opção "Todos" ou "Todas", e o botão "Filtrar". Funciona sem JavaScript. Mudar um select não recarrega a página sozinho: o aluno escolhe e aperta "Filtrar".
- Com algum filtro ativo, aparece "Limpar filtros" ao lado de "Filtrar", que volta para `/`.
- Os selects abrem com os valores da URL. Valor da URL fora da lista é descartado no front, antes de ir para a API.
- Vazio com filtro: `StatePanel` "Nenhuma ideia com esses filtros", com "Tire um filtro ou" e o link "limpe todos". Sem filtro, o vazio continua o de hoje.

## Pronto quando

- [ ] `GET /api/ideas` filtra por cada parâmetro e pela soma deles; vazio é sem filtro.
- [ ] Valor fora da lista responde `400` no campo.
- [ ] Visitante com filtros recebe a mesma prévia de sem filtros; `status=done` continua sendo a vitrine dele.
- [ ] Logado vê a barra, filtra, limpa; o link filtrado abre filtrado.
- [ ] Testes do handler: cada filtro, a soma, o vazio, o `400` e o visitante.
