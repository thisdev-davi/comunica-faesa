package ideas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Bot é o bot do Discord que cria a call do grupo (docs/contratos/fatia-19-call-no-discord.md). Token vazio desliga.
type Bot struct {
	Token      string
	GuildID    string
	CategoryID string // categoria do servidor onde os canais nascem
	API        string // base da REST do Discord; os testes apontam para um servidor falso
}

// Permissões do Discord são bits de um inteiro, enviados como texto (docs.discord.com/developers/topics/permissions).
const (
	permView    = 1 << 10
	permSend    = 1 << 11
	permHistory = 1 << 16
	permConnect = 1 << 20
	permSpeak   = 1 << 21
	maxName     = 100 // nome de cargo e de canal no Discord; o título da ideia vai até 120
)

// Sem timeout, um Discord que não responde seguraria o PUT /status para sempre.
var botClient = &http.Client{Timeout: 10 * time.Second}

// openCall garante a call do grupo: cria o cargo e o canal que faltam e dá o cargo ao autor e a quem está na lista.
// Rodar de novo só completa o que faltou, então é também a nova tentativa depois de uma falha.
// ponytail: um PUT por pessoa, em sequência; um 429 (limite do Discord) para no meio e o próximo disparo completa.
func (h *Handler) openCall(ctx context.Context, id int64) error {
	c, err := h.q.GetIdeaCall(ctx, id)
	if err != nil {
		return err
	}
	name := c.Title
	if r := []rune(name); len(r) > maxName {
		name = string(r[:maxName])
	}
	guild := "/guilds/" + h.Bot.GuildID
	if c.DiscordRoleID == nil {
		var role struct{ ID string }
		// permissions "0": o cargo não dá poder nenhum no servidor, só abre o canal.
		body := map[string]any{"name": name, "mentionable": true, "permissions": "0"}
		if _, err := h.Bot.call(ctx, http.MethodPost, guild+"/roles", body, &role); err != nil {
			return err
		}
		// Grava antes de seguir: se o canal falhar, o próximo disparo não cria outro cargo.
		if err := h.q.SetIdeaRole(ctx, SetIdeaRoleParams{ID: id, DiscordRoleID: &role.ID}); err != nil {
			return err
		}
		c.DiscordRoleID = &role.ID
	}
	if c.DiscordChannelID == nil {
		var me, channel struct{ ID string }
		if _, err := h.Bot.call(ctx, http.MethodGet, "/users/@me", nil, &me); err != nil {
			return err
		}
		body := map[string]any{
			"name": name, "type": 2, "parent_id": h.Bot.CategoryID, // 2 = canal de voz, que já tem chat de texto
			"permission_overwrites": []map[string]any{
				// O id do @everyone é o id do servidor. type 0 = cargo, 1 = pessoa.
				{"id": h.Bot.GuildID, "type": 0, "deny": strconv.Itoa(permView)},
				{"id": *c.DiscordRoleID, "type": 0, "allow": strconv.Itoa(permView | permSend | permHistory | permConnect | permSpeak)},
				// Sem ver o canal, o bot não consegue apagá-lo depois.
				{"id": me.ID, "type": 1, "allow": strconv.Itoa(permView)},
			},
		}
		if _, err := h.Bot.call(ctx, http.MethodPost, guild+"/channels", body, &channel); err != nil {
			return err
		}
		if err := h.q.SetIdeaChannel(ctx, SetIdeaChannelParams{ID: id, DiscordChannelID: &channel.ID}); err != nil {
			return err
		}
	}
	for _, member := range c.Members {
		status, err := h.Bot.call(ctx, http.MethodPut, guild+"/members/"+member+"/roles/"+*c.DiscordRoleID, nil, nil)
		if status == http.StatusNotFound { // saiu do servidor
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// closeCall apaga o canal e o cargo de uma ideia que já foi apagada. Falha só vai para o log: a ideia já não existe.
func (h *Handler) closeCall(ctx context.Context, c GetIdeaCallRow) {
	if c.DiscordChannelID != nil {
		if _, err := h.Bot.call(ctx, http.MethodDelete, "/channels/"+*c.DiscordChannelID, nil, nil); err != nil {
			slog.Error("apagar canal da call", "err", err)
		}
	}
	if c.DiscordRoleID != nil {
		if _, err := h.Bot.call(ctx, http.MethodDelete, "/guilds/"+h.Bot.GuildID+"/roles/"+*c.DiscordRoleID, nil, nil); err != nil {
			slog.Error("apagar cargo da call", "err", err)
		}
	}
}

// call faz uma chamada à REST do Discord como o bot, mandando body (se não for nil) em JSON e decodificando a
// resposta em out (se não for nil). Devolve o status para quem precisa distinguir o 404.
func (b Bot) call(ctx context.Context, method, path string, body, out any) (int, error) {
	var payload io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		payload = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.API+path, payload)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bot "+b.Token)
	req.Header.Set("User-Agent", "DiscordBot (https://github.com/thisdev-davi/comunica-faesa, 1)") // o Discord pede esse formato
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := botClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return res.StatusCode, fmt.Errorf("%s %s: status %d", method, path, res.StatusCode)
	}
	if out == nil {
		return res.StatusCode, nil
	}
	return res.StatusCode, json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
}
