# Contrato — Fatia 19: call do grupo no Discord

Primeira fatia da [fase do Discord](../PRD.md#fase-do-discord). Objetivo único: quando o autor diz que o grupo fechou (a ideia vai para **Em andamento**), um bot cria no servidor da comunidade um canal de voz privado para o grupo e um cargo com o nome da ideia, e o grupo acha o canal pela página da ideia.

Mudou o contrato? Atualiza este arquivo **antes** do código.

## Fora desta fatia (de propósito)

Canal de texto separado, evento agendado do Discord, escolher quem entra, tirar o cargo de quem desiste, renomear canal e cargo quando a ideia é editada, apagar canal e cargo de ideias concluídas, avisar o autor quando o Discord falha, botão de tentar de novo, bot escutando o Discord (gateway).

## Regras

- **O disparo** é o `PUT /api/ideas/{id}/status` com `in_progress`. Os outros status não disparam.
- **O que nasce:** um cargo com o título da ideia (cortado em 100 caracteres, o limite do Discord), que pode ser mencionado e não ganha cor nem aparece separado na lista de membros; e um canal de voz com o mesmo nome, dentro da categoria `DISCORD_CALLS_CATEGORY_ID`. Canal de voz no Discord já tem chat de texto junto.
- **Quem entra:** o canal é invisível para `@everyone` e aberto (ver, conectar, falar, escrever, ler o histórico) para o cargo. O cargo vai para o autor e para todos da lista de interessados no momento do disparo. Quem está na lista e saiu do servidor é pulado.
- **Uma call por ideia:** a ideia guarda o id do cargo e o do canal. Cada disparo cria só o que falta e dá o cargo a todos da lista de novo. Por isso:
  - reabrir e voltar para Em andamento não cria outra call, só dá o cargo a quem entrou na lista enquanto ela estava aberta;
  - se o Discord falhar no meio, o próximo disparo completa o que faltou.
- **O status não depende do Discord:** ele é salvo antes, e a resposta é `204` mesmo se o Discord falhar. A falha só vai para o log.
- **Concluir** mantém o canal e o cargo. **Apagar a ideia** (autor ou moderador) apaga o canal e o cargo depois de apagar a ideia, e uma falha aí também só vai para o log.
- **Bot desligado:** sem `DISCORD_BOT_TOKEN`, a API sobe e não cria call (como o `DISCORD_MOD_ROLE_IDS` vazio desliga a moderação). Token sem categoria impede a API de subir.
- **Limite conhecido:** o servidor aceita até 250 cargos, e só apagar a ideia libera um. Limpar os cargos de concluídas antigas entra quando chegar perto.

## Banco

Migration `00012`: duas colunas novas em `ideas`, nulas até a call ser criada.

| coluna | tipo | o que é |
|---|---|---|
| discord_role_id | text | id do cargo do grupo no Discord |
| discord_channel_id | text | id do canal de voz do grupo |

## Endpoints

### Mudança na fatia 7: `PUT /api/ideas/{id}/status`

Com `in_progress` e o bot ligado, cria a call (ver Regras) antes de responder. A resposta não muda: `204`, mesmo se o Discord falhar.

### Mudança na fatia 4: `GET /api/ideas/{id}`

Ganha `call_url`, o link do canal no Discord, só para o autor e para quem está na lista, e só quando o canal existe. Nos outros casos o campo não vem.

```json
{ "id": 7, "status": "in_progress", "call_url": "https://discord.com/channels/<servidor>/<canal>" }
```

### Mudança na fatia 6: `DELETE /api/ideas/{id}`

Apaga também o canal e o cargo no Discord, se existirem. A resposta não muda.

## Chamadas ao Discord

REST com `Authorization: Bot <token>`, feitas pela própria API Go.

| quando | chamada |
|---|---|
| falta o cargo | `POST /guilds/{servidor}/roles` |
| falta o canal | `GET /users/@me` (o id do bot, que precisa enxergar o canal para apagá-lo depois) e `POST /guilds/{servidor}/channels` (tipo 2, voz) |
| cada pessoa | `PUT /guilds/{servidor}/members/{discord_id}/roles/{cargo}`; `404` = saiu do servidor, pula |
| apagar a ideia | `DELETE /channels/{canal}` e `DELETE /guilds/{servidor}/roles/{cargo}` |

## Front

- Página da ideia: quando a resposta traz `call_url`, aparece logo abaixo do card o link "Entrar na call do grupo" (abre o Discord em nova aba), igual para o autor e para a lista. O interessado passa a ler "Você está no grupo. A call fica no Discord, só para quem está na lista."

## Configuração

| variável | quem lê | o que é |
|---|---|---|
| `DISCORD_BOT_TOKEN` | API | token do bot (Developer Portal → Bot). Vazio desliga a call |
| `DISCORD_CALLS_CATEGORY_ID` | API | id da categoria do servidor onde os canais nascem |

O bot entra no servidor com as permissões Gerenciar Cargos, Gerenciar Canais, Ver Canais, Conectar, Falar, Enviar Mensagens e Ler Histórico (passo a passo no README). O cargo novo nasce embaixo de todos, então fica abaixo do cargo do bot sem ninguém mexer.

## Pronto quando

- [ ] `goose up` cria as duas colunas.
- [ ] Ideia que vai para Em andamento cria o cargo e o canal e dá o cargo ao autor e à lista; reabrir e fechar de novo não cria outro, só dá o cargo a quem entrou.
- [ ] Discord fora do ar: o status muda (`204`), e o disparo seguinte completa.
- [ ] `call_url` só para o autor e a lista.
- [ ] Apagar a ideia apaga o canal e o cargo.
- [ ] Testes contra um Discord falso cobrindo os itens acima.
- [ ] Link testado no navegador, nos dois temas e na tela estreita.
