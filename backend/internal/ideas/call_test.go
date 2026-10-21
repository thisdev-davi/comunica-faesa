package ideas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeDiscord imita as chamadas do bot. Guarda cada uma como "MÉTODO caminho"; failChannel faz o próximo
// POST de canal responder 500, e "saiu" é um membro que já não está no servidor.
type fakeDiscord struct {
	mu          sync.Mutex
	calls       []string
	failChannel bool
	overwrites  []map[string]any // as permissões do último canal criado
}

func (f *fakeDiscord) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bot tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		switch r.Method + " " + r.URL.Path {
		case "GET /users/@me":
			json.NewEncoder(w).Encode(map[string]string{"id": "bot-1"})
		case "POST /guilds/guild-1/roles":
			json.NewEncoder(w).Encode(map[string]string{"id": "role-1"})
		case "POST /guilds/guild-1/channels":
			if f.failChannel {
				f.failChannel = false
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var body struct {
				ParentID   string           `json:"parent_id"`
				Overwrites []map[string]any `json:"permission_overwrites"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			if body.ParentID != "cat-1" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			f.overwrites = body.Overwrites
			json.NewEncoder(w).Encode(map[string]string{"id": "chan-1"})
		case "PUT /guilds/guild-1/members/saiu/roles/role-1":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNoContent) // os outros PUT de cargo e os DELETE
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// took devolve as chamadas feitas desde a última vez e zera a lista.
func (f *fakeDiscord) took() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := f.calls
	f.calls = nil
	return calls
}

// A call do grupo contra Postgres real e um Discord falso (docs/contratos/fatia-19-call-no-discord.md).
func TestCall(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	ids := make([]int64, 4)
	for i, u := range []string{
		`INSERT INTO users (name, course, discord_id) VALUES ('Autora de Teste', 'cc', 'd-autora') RETURNING id`,
		`INSERT INTO users (name, discord_id) VALUES ('Ana Teste', 'd-ana') RETURNING id`,
		`INSERT INTO users (name, discord_id) VALUES ('Saiu do Servidor', 'saiu') RETURNING id`,
		`INSERT INTO users (name, discord_id) VALUES ('Caio Teste', 'd-caio') RETURNING id`,
	} {
		if err := pool.QueryRow(ctx, u).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	discord := &fakeDiscord{failChannel: true}
	srv := discord.start(t)
	as := func(id int64) *Handler {
		h := NewHandler(pool, loggedAs(id))
		h.Bot = Bot{Token: "tok", GuildID: "guild-1", CategoryID: "cat-1", API: srv.URL}
		return h
	}
	author, ana, gone, caio := as(ids[0]), as(ids[1]), as(ids[2]), as(ids[3])

	rec := do(t, author.Create, http.MethodPost, validBody)
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	id := strconv.FormatInt(decode[ideaResponse](t, rec).ID, 10)
	for _, h := range []*Handler{ana, gone} {
		if code := doID(t, h.PutInterest, http.MethodPut, id).Code; code != 204 {
			t.Fatalf("interesse: %d", code)
		}
	}
	setStatus := func(status string) {
		t.Helper()
		if rec := doIDBody(t, author.SetStatus, http.MethodPut, id, `{"status":"`+status+`"}`); rec.Code != 204 {
			t.Fatalf("PUT status %s: %d (%s)", status, rec.Code, rec.Body)
		}
	}
	callURL := func(h *Handler) *string {
		t.Helper()
		return decode[ideaResponse](t, doID(t, h.Get, http.MethodGet, id)).CallURL
	}

	// Concluir não dispara.
	setStatus("done")
	if got := discord.took(); len(got) != 0 {
		t.Errorf("concluir chamou o Discord: %v", got)
	}

	// O Discord falha no canal: o status muda mesmo assim, e o cargo fica guardado.
	setStatus("in_progress")
	if got := discord.took(); !slices.Equal(got, []string{"POST /guilds/guild-1/roles", "GET /users/@me", "POST /guilds/guild-1/channels"}) {
		t.Errorf("primeiro disparo: %v", got)
	}
	if got := decode[ideaResponse](t, doID(t, author.Get, http.MethodGet, id)); got.Status != "in_progress" || got.CallURL != nil {
		t.Errorf("depois da falha: status %s, call_url %v", got.Status, got.CallURL)
	}

	// O disparo seguinte completa: não cria outro cargo, cria o canal e dá o cargo ao autor e à lista (quem saiu é pulado).
	setStatus("in_progress")
	want := []string{
		"GET /users/@me", "POST /guilds/guild-1/channels",
		"PUT /guilds/guild-1/members/d-autora/roles/role-1",
		"PUT /guilds/guild-1/members/d-ana/roles/role-1",
		"PUT /guilds/guild-1/members/saiu/roles/role-1",
	}
	if got := discord.took(); !slices.Equal(got, want) {
		t.Errorf("segundo disparo: %v, quer %v", got, want)
	}
	// @everyone (o id do servidor) não vê; o cargo e o bot veem.
	var hidden, role, bot bool
	for _, o := range discord.overwrites {
		switch o["id"] {
		case "guild-1":
			hidden = o["deny"] == strconv.Itoa(permView)
		case "role-1":
			role = o["allow"] == strconv.Itoa(permView|permSend|permHistory|permConnect|permSpeak)
		case "bot-1":
			bot = o["allow"] == strconv.Itoa(permView)
		}
	}
	if !hidden || !role || !bot {
		t.Errorf("permissões do canal: %v", discord.overwrites)
	}

	// O link só para o autor e a lista.
	wantURL := "https://discord.com/channels/guild-1/chan-1"
	for name, h := range map[string]*Handler{"autora": author, "ana": ana} {
		if got := callURL(h); got == nil || *got != wantURL {
			t.Errorf("call_url para %s: %v", name, got)
		}
	}
	if rec := doID(t, caio.Get, http.MethodGet, id); strings.Contains(rec.Body.String(), "call_url") {
		t.Errorf("call_url para quem não está na lista: %s", rec.Body)
	}

	// Reabrir e fechar de novo: só dá o cargo, agora também a quem entrou.
	setStatus("open")
	if code := doID(t, caio.PutInterest, http.MethodPut, id).Code; code != 204 {
		t.Fatalf("interesse depois de reabrir: %d", code)
	}
	setStatus("in_progress")
	if got := discord.took(); len(got) != 4 || !slices.Contains(got, "PUT /guilds/guild-1/members/d-caio/roles/role-1") {
		t.Errorf("fechar de novo: %v", got)
	}

	// Apagar a ideia apaga o canal e o cargo.
	if rec := doID(t, author.Delete, http.MethodDelete, id); rec.Code != 204 {
		t.Fatalf("DELETE: %d (%s)", rec.Code, rec.Body)
	}
	if got := discord.took(); !slices.Equal(got, []string{"DELETE /channels/chan-1", "DELETE /guilds/guild-1/roles/role-1"}) {
		t.Errorf("apagar: %v", got)
	}
}
