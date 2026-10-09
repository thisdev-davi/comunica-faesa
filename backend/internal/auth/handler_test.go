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

// fakeDiscord imita as três chamadas que o login faz ao Discord. name e memberStatus mudam o que ele responde.
type fakeDiscord struct {
	name         string
	memberStatus int
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
		io.WriteString(w, `{}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTestHandler aponta o OAuth e a API do Discord para o servidor falso.
func newTestHandler(db DBTX, discord *httptest.Server) *Handler {
	h := NewHandler(db, "client", "secret", "http://localhost:3000/api/auth/discord/callback", "guild-1")
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

func TestLoginRedirectsWithState(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(nil, nil).Login(rec, httptest.NewRequest(http.MethodGet, "/api/auth/discord", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, quer 302", rec.Code)
	}
	c := cookie(rec, stateCookie)
	if c == nil || c.Value == "" || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
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
			if c := cookie(rec, stateCookie); c == nil || c.MaxAge >= 0 {
				t.Errorf("state devia ser apagado: %+v", c)
			}
		})
	}
}

func TestMeWithoutSessionIs401(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(nil, nil).Me(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"unauthenticated"`) {
		t.Fatalf("got %d %s, quer 401 unauthenticated", rec.Code, rec.Body)
	}
}

// Caminho feliz contra Postgres real. Precisa de TEST_DATABASE_URL apontando para um banco já migrado.
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

	login := func() *http.Cookie {
		t.Helper()
		rec := callback(h, "code=code-ok&state=state-ok")
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
			t.Fatalf("callback: %d %s", rec.Code, rec.Header().Get("Location"))
		}
		c := cookie(rec, sessionCookie)
		if c == nil || c.Value == "" || !c.HttpOnly || c.Path != "/" {
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

	session := login()
	rec := me(session)
	want := `{"id":1,"name":"Ana","avatar_url":"https://cdn.discordapp.com/avatars/9001/abc.png","course":null}`
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("me: %d %s, quer %s", rec.Code, rec.Body, want)
	}

	// Segundo login com outro nome no Discord: mesmo usuário, nome atualizado.
	fake.name = "Ana Maria"
	session = login()
	if body := me(session).Body.String(); !strings.Contains(body, `"id":1,"name":"Ana Maria"`) {
		t.Errorf("nome não atualizou: %s", body)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	h.Logout(rec, req)
	if c := cookie(rec, sessionCookie); rec.Code != http.StatusNoContent || c == nil || c.MaxAge >= 0 {
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
