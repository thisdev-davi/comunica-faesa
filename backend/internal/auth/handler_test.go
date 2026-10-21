package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

// fakeDiscord imita as três chamadas que o login faz ao Discord. name, memberStatus e roles mudam o que ele responde.
type fakeDiscord struct {
	name         string
	memberStatus int
	roles        []string // cargos do membro no servidor
}

func (f *fakeDiscord) start(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("code") != "code-ok" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"access_token":"tok","token_type":"Bearer"}`)
	})
	mux.HandleFunc("GET /users/@me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "9001", "username": "ana", "global_name": f.name, "avatar": "abc"})
	})
	mux.HandleFunc("GET /users/@me/guilds/guild-1/member", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(f.memberStatus)
		json.NewEncoder(w).Encode(map[string]any{"roles": f.roles})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTestHandler aponta o OAuth e a API do Discord para o servidor falso.
func newTestHandler(db DBTX, discord *httptest.Server) *Handler {
	h := NewHandler(db, "client", "secret", "http://localhost:3000/api/auth/discord/callback", "guild-1", []string{"role-mod", "role-admin"})
	if discord != nil {
		h.oauth.Endpoint = oauth2.Endpoint{AuthURL: discord.URL + "/authorize", TokenURL: discord.URL + "/token"}
		h.discordAPI = discord.URL
	}
	return h
}

// callback simula a volta do Discord com o cookie de state que o Login gravou.
func callback(h *Handler, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/discord/callback?"+query, nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "state-ok"})
	rec := httptest.NewRecorder()
	h.Callback(rec, req)
	return rec
}

func cookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// hostCookie confere o que o prefixo __Host- exige (Secure, Path=/, sem Domain) e o HttpOnly/SameSite de sempre.
func hostCookie(c *http.Cookie, name string) bool {
	return c != nil && c.Name == name && c.Secure && c.Path == "/" && c.Domain == "" && c.HttpOnly && c.SameSite == http.SameSiteLaxMode
}

func TestLoginRedirectsWithState(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(nil, nil).Login(rec, httptest.NewRequest(http.MethodGet, "/api/auth/discord", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, quer 302", rec.Code)
	}
	c := cookie(rec, stateCookie)
	if !hostCookie(c, "__Host-oauth_state") || c.Value == "" {
		t.Fatalf("cookie de state inválido: %+v", c)
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	if loc.Host != "discord.com" || loc.Query().Get("state") != c.Value || loc.Query().Get("scope") != "identify guilds.members.read" {
		t.Errorf("redirect inesperado: %s", loc)
	}
}

// Esses casos param antes do banco, então o handler roda sem conexão (nil).
func TestCallbackFailures(t *testing.T) {
	discord := (&fakeDiscord{name: "Ana", memberStatus: http.StatusNotFound}).start(t)
	cases := []struct {
		name, query, want string
	}{
		{"cancelou no discord", "error=access_denied&state=state-ok", "/login?erro=cancelado"},
		{"sem state", "code=code-ok", "/login?erro=falhou"},
		{"state diferente", "code=code-ok&state=outro", "/login?erro=falhou"},
		{"code recusado", "code=ruim&state=state-ok", "/login?erro=falhou"},
		{"fora do servidor", "code=code-ok&state=state-ok", "/login?erro=fora_do_servidor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := callback(newTestHandler(nil, discord), tc.query)
			if rec.Code != http.StatusFound || rec.Header().Get("Location") != tc.want {
				t.Fatalf("got %d %s, quer 302 %s", rec.Code, rec.Header().Get("Location"), tc.want)
			}
			if cookie(rec, sessionCookie) != nil {
				t.Error("não devia criar sessão")
			}
			if c := cookie(rec, stateCookie); !hostCookie(c, "__Host-oauth_state") || c.MaxAge >= 0 {
				t.Errorf("state devia ser apagado: %+v", c)
			}
		})
	}
}

// Sem cookie o logout não toca no banco, então roda sem conexão (nil); ainda assim apaga o cookie.
func TestLogoutWithoutSessionClearsCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(nil, nil).Logout(rec, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
	if c := cookie(rec, "__Host-session"); rec.Code != http.StatusNoContent || !hostCookie(c, "__Host-session") || c.MaxAge >= 0 {
		t.Fatalf("logout: %d, cookie %+v", rec.Code, c)
	}
}

func TestMeWithoutSessionIs401(t *testing.T) {
	h := newTestHandler(nil, nil)
	for method, hf := range map[string]http.HandlerFunc{"GET": h.Me, "PATCH": h.UpdateMe} {
		rec := httptest.NewRecorder()
		hf(rec, httptest.NewRequest(method, "/api/me", strings.NewReader(`{"course":"cc"}`)))
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"unauthenticated"`) {
			t.Errorf("%s: got %d %s, quer 401 unauthenticated", method, rec.Code, rec.Body)
		}
	}
}

// Caminho feliz contra Postgres real: primeiro login, cadastro do curso, segundo login e logout.
// Precisa de TEST_DATABASE_URL apontando para um banco já migrado.
func TestLoginMeLogout(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	lockDB(t, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}

	fake := &fakeDiscord{name: "Ana", memberStatus: http.StatusOK}
	h := newTestHandler(pool, fake.start(t))

	login := func(wantLocation string) *http.Cookie {
		t.Helper()
		rec := callback(h, "code=code-ok&state=state-ok")
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != wantLocation {
			t.Fatalf("callback: %d %s, quer 302 %s", rec.Code, rec.Header().Get("Location"), wantLocation)
		}
		c := cookie(rec, sessionCookie)
		if !hostCookie(c, "__Host-session") || c.Value == "" {
			t.Fatalf("cookie de sessão inválido: %+v", c)
		}
		return c
	}
	me := func(c *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.AddCookie(c)
		rec := httptest.NewRecorder()
		h.Me(rec, req)
		return rec
	}
	setCourse := func(c *http.Cookie, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(body))
		req.AddCookie(c)
		rec := httptest.NewRecorder()
		h.UpdateMe(rec, req)
		return rec
	}

	// Primeiro login: ainda sem curso, vai para o cadastro.
	session := login("/cadastro")
	rec := me(session)
	want := `{"id":1,"name":"Ana","avatar_url":"https://cdn.discordapp.com/avatars/9001/abc.png","course":null,"skills":[],"is_moderator":false}`
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("me: %d %s, quer %s", rec.Code, rec.Body, want)
	}

	bad := []struct{ name, body, want string }{
		{"curso fora da lista", `{"course":"med"}`, `{"error":"validation_failed","fields":{"course":"valor inválido"}}`},
		{"curso só espaços", `{"course":"   "}`, `{"error":"validation_failed","fields":{"course":"obrigatório"}}`},
		{"sem curso", `{}`, `{"error":"validation_failed","fields":{"course":"obrigatório"}}`},
		{"json malformado", `{`, `{"error":"invalid_json"}`},
		{"tipo errado", `{"course":1}`, `{"error":"invalid_json"}`},
	}
	for _, tc := range bad {
		if rec := setCourse(session, tc.body); rec.Code != 400 || strings.TrimSpace(rec.Body.String()) != tc.want {
			t.Errorf("%s: %d %s, quer 400 %s", tc.name, rec.Code, rec.Body, tc.want)
		}
	}
	if rec := setCourse(session, `{"course":"`+strings.Repeat("a", maxBodyBytes)+`"}`); rec.Code != 413 {
		t.Errorf("corpo grande: %d %s, quer 413", rec.Code, rec.Body)
	}
	if body := me(session).Body.String(); !strings.Contains(body, `"course":null`) {
		t.Fatalf("pedido inválido gravou o curso: %s", body)
	}

	// O PATCH devolve o mesmo corpo do GET, ainda sem moderação.
	rec = setCourse(session, `{"course":" cc "}`)
	want = strings.Replace(want, `"course":null`, `"course":"cc"`, 1)
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("patch: %d %s, quer %s", rec.Code, rec.Body, want)
	}
	if body := strings.TrimSpace(me(session).Body.String()); body != want {
		t.Fatalf("curso não gravou: %s", body)
	}

	// Habilidades (fatia 11): gravadas sem repetição e na ordem do catálogo; sem o campo, não mudam; [] limpa.
	skills := []struct{ name, body, want string }{
		{"limpa e ordena", `{"course":"ads","skills":["data","backend","data"]}`, `"course":"ads","skills":["backend","data"],"is_moderator":false}`},
		{"sem o campo mantém", `{"course":"cc"}`, `"course":"cc","skills":["backend","data"],"is_moderator":false}`},
		{"vazio limpa", `{"course":"cc","skills":[]}`, `"course":"cc","skills":[],"is_moderator":false}`},
	}
	for _, tc := range skills {
		rec := setCourse(session, tc.body)
		if rec.Code != 200 || !strings.HasSuffix(strings.TrimSpace(rec.Body.String()), tc.want) {
			t.Errorf("%s: %d %s, quer 200 terminando em %s", tc.name, rec.Code, rec.Body, tc.want)
		}
		if body := strings.TrimSpace(me(session).Body.String()); !strings.HasSuffix(body, tc.want) {
			t.Errorf("%s: GET /api/me = %s, quer terminando em %s", tc.name, body, tc.want)
		}
	}
	badSkills := []struct{ name, body, want string }{
		{"fora do catálogo", `{"course":"cc","skills":["culinaria"]}`, `{"error":"validation_failed","fields":{"skills":"valor inválido"}}`},
		{"mais de 5", `{"course":"cc","skills":["frontend","backend","mobile","design","data","ai"]}`, `{"error":"validation_failed","fields":{"skills":"escolha até 5"}}`},
		{"tipo errado", `{"course":"cc","skills":"backend"}`, `{"error":"invalid_json"}`},
	}
	for _, tc := range badSkills {
		if rec := setCourse(session, tc.body); rec.Code != 400 || strings.TrimSpace(rec.Body.String()) != tc.want {
			t.Errorf("%s: %d %s, quer 400 %s", tc.name, rec.Code, rec.Body, tc.want)
		}
	}

	// Segundo login, já com curso e com outro nome no Discord: vai para o mural, mesmo usuário, nome atualizado.
	fake.name = "Ana Maria"
	session = login("/")
	if body := me(session).Body.String(); !strings.Contains(body, `"id":1,"name":"Ana Maria"`) {
		t.Errorf("nome não atualizou: %s", body)
	}

	// Cargo de moderação (fatia 15): relido a cada login. Um cargo qualquer não conta; um dos configurados liga; sem ele, desliga.
	// O Viewer, que ideias e eventos recebem, lê o mesmo cargo da sessão.
	for _, tc := range []struct {
		roles []string
		want  string
		mod   bool
	}{
		{[]string{"role-outro"}, `"is_moderator":false`, false},
		{[]string{"role-outro", "role-admin"}, `"is_moderator":true`, true},
		{nil, `"is_moderator":false`, false},
	} {
		fake.roles = tc.roles
		c := login("/")
		if body := me(c).Body.String(); !strings.Contains(body, tc.want) {
			t.Errorf("cargos %v: %s, quer %s", tc.roles, body, tc.want)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/ideas", nil)
		req.AddCookie(c)
		if id, mod, err := h.Viewer(req); id != 1 || mod != tc.mod || err != nil {
			t.Errorf("cargos %v: Viewer = %d, %v, %v, quer 1, %v, nil", tc.roles, id, mod, err, tc.mod)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	h.Logout(rec, req)
	if c := cookie(rec, sessionCookie); rec.Code != http.StatusNoContent || !hostCookie(c, "__Host-session") || c.MaxAge >= 0 {
		t.Fatalf("logout: %d, cookie %+v", rec.Code, c)
	}
	if rec = me(session); rec.Code != http.StatusUnauthorized {
		t.Errorf("sessão continua valendo depois do logout: %d", rec.Code)
	}
}

// lockDB segura um advisory lock do Postgres até o fim do teste. O go test roda os pacotes em paralelo e
// os testes de banco de pacotes diferentes limpam as mesmas tabelas; o lock faz um esperar o outro.
func lockDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	conn, err := pool.Acquire(ctx) // o lock pertence à conexão, então ela fica reservada até o fim
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(1)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, "SELECT pg_advisory_unlock(1)")
		conn.Release()
	})
}
