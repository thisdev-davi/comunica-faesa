package events

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// loggedAs troca a sessão de verdade por um usuário fixo (0 = visitante).
func loggedAs(id int64) func(*http.Request) (int64, error) {
	return func(*http.Request) (int64, error) { return id, nil }
}

func do(t *testing.T, hf http.HandlerFunc, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	hf(rec, httptest.NewRequest(method, "/api/events", strings.NewReader(body)))
	return rec
}

// doID chama um handler de /api/events/{id} com o {id} preenchido, como o roteador faria.
func doID(t *testing.T, hf http.HandlerFunc, method, id string) *httptest.ResponseRecorder {
	t.Helper()
	return doIDBody(t, hf, method, id, "")
}

// doIDBody é o doID com corpo, para o PUT do evento.
func doIDBody(t *testing.T, hf http.HandlerFunc, method, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/events/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	hf(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("resposta não é JSON: %v (%s)", err, rec.Body)
	}
	return v
}

// body monta um corpo válido com starts_at trocável.
func body(startsAt string) string {
	return `{"title":"Noite de trabalho","description":"Traga o notebook","starts_at":"` + startsAt + `","location":"Bloco B, sala 204"}`
}

func future(d time.Duration) string { return time.Now().Add(d).Format(time.RFC3339) }

// Esses casos param antes do banco, então o handler roda sem conexão (nil).
func TestCreateRejectsBadInput(t *testing.T) {
	cases := []struct {
		name       string
		authorID   int64
		body       string
		wantStatus int
		wantError  string
		wantFields []string
	}{
		{"sem sessão", 0, body(future(time.Hour)), 401, "unauthenticated", nil},
		{"json malformado", 1, `{`, 400, "invalid_json", nil},
		{"corpo grande", 1, `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, 413, "body_too_large", nil},
		{"tudo vazio", 1, `{}`, 400, "validation_failed", []string{"title", "description", "starts_at", "location"}},
		{"local só espaços", 1, `{"title":"abc","description":"x","starts_at":"` + future(time.Hour) + `","location":"   "}`, 400, "validation_failed", []string{"location"}},
		{"data sem fuso", 1, body("2099-10-20T19:00:00"), 400, "validation_failed", []string{"starts_at"}},
		{"data com tipo errado", 1, `{"starts_at":20}`, 400, "invalid_json", nil},
		{"data no passado", 1, body(future(-time.Minute)), 400, "validation_failed", []string{"starts_at"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, NewHandler(nil, loggedAs(tc.authorID)).Create, http.MethodPost, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, quer %d (%s)", rec.Code, tc.wantStatus, rec.Body)
			}
			got := decode[errorResponse](t, rec)
			if got.Error != tc.wantError {
				t.Errorf("error = %q, quer %q", got.Error, tc.wantError)
			}
			if len(got.Fields) != len(tc.wantFields) {
				t.Errorf("fields = %v, quer só %v", got.Fields, tc.wantFields)
			}
			for _, f := range tc.wantFields {
				if got.Fields[f] == "" {
					t.Errorf("faltou erro no campo %q: %v", f, got.Fields)
				}
			}
		})
	}
}

// "No futuro" é estrito: o mesmo instante de agora já é recusado.
func TestValidateFuture(t *testing.T) {
	now := time.Date(2026, 10, 20, 22, 0, 0, 0, time.UTC)
	for startsAt, want := range map[string]string{
		"2026-10-20T19:00:00-03:00": "deve ser no futuro", // é o próprio now, escrito no horário de Brasília
		"2026-10-20T19:00:01-03:00": "",
	} {
		req := eventRequest{Title: "abc", Description: "x", Location: "abc", StartsAt: startsAt}
		if got := req.validate(now)["starts_at"]; got != want {
			t.Errorf("%s: erro = %q, quer %q", startsAt, got, want)
		}
	}
}

func TestSessionErrorIs500(t *testing.T) {
	failing := func(*http.Request) (int64, error) { return 0, errors.New("banco fora") }
	h := NewHandler(nil, failing)
	for name, hf := range map[string]http.HandlerFunc{
		"POST": h.Create, "GET agenda": h.List, "GET um": h.Get, "PUT presença": h.PutAttendance, "DELETE presença": h.DeleteAttendance,
		"PUT evento": h.Update, "DELETE evento": h.Delete,
	} {
		if rec := doID(t, hf, http.MethodGet, "1"); rec.Code != 500 || decode[errorResponse](t, rec).Error != "internal" {
			t.Errorf("%s: status = %d (%s), quer 500 internal", name, rec.Code, rec.Body)
		}
	}
}

// Sem sessão é 401 antes do id; id que não é inteiro positivo é 404 sem ir ao banco.
func TestAttendanceRejectsVisitorAndBadID(t *testing.T) {
	visitor, user := NewHandler(nil, loggedAs(0)), NewHandler(nil, loggedAs(1))
	cases := []struct {
		name       string
		hf         http.HandlerFunc
		id         string
		wantStatus int
		wantError  string
	}{
		{"vou sem sessão", visitor.PutAttendance, "1", 401, "unauthenticated"},
		{"desistir sem sessão", visitor.DeleteAttendance, "1", 401, "unauthenticated"},
		{"vou com id inválido", user.PutAttendance, "abc", 404, "not_found"},
		{"evento com id zero", visitor.Get, "0", 404, "not_found"},
	}
	for _, tc := range cases {
		rec := doID(t, tc.hf, http.MethodPut, tc.id)
		if rec.Code != tc.wantStatus || decode[errorResponse](t, rec).Error != tc.wantError {
			t.Errorf("%s: %d (%s), quer %d %s", tc.name, rec.Code, rec.Body, tc.wantStatus, tc.wantError)
		}
	}
	// Desistir com id inválido não tem o que apagar: 204, como desistir de quem não estava na lista.
	if rec := doID(t, user.DeleteAttendance, http.MethodDelete, "abc"); rec.Code != 204 {
		t.Errorf("desistir com id inválido: %d, quer 204", rec.Code)
	}
}

// Caminho feliz contra Postgres real. Precisa de TEST_DATABASE_URL apontando para um banco já migrado.
func TestCreateAndList(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('Organizador de Teste') RETURNING id").Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(pool, loggedAs(authorID))

	rec := do(t, h.List, http.MethodGet, "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Fatalf("lista vazia: %d %s", rec.Code, rec.Body)
	}

	// Criados fora de ordem: a agenda devolve do mais próximo para o mais distante.
	later := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	rec = do(t, h.Create, http.MethodPost, `{"title":"  Hackathon  ","description":"Fim de semana","starts_at":"`+later.Format(time.RFC3339)+`","location":"Bloco H"}`)
	if rec.Code != 201 {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body)
	}
	far := decode[eventResponse](t, rec)
	if far.Title != "Hackathon" || !far.StartsAt.Equal(later) || far.Author.Name != "Organizador de Teste" || far.CreatedAt.IsZero() {
		t.Errorf("evento criado inesperado: %+v", far)
	}
	if rec = do(t, h.Create, http.MethodPost, body(future(time.Hour))); rec.Code != 201 {
		t.Fatalf("segundo POST: %d (%s)", rec.Code, rec.Body)
	}
	near := decode[eventResponse](t, rec)

	// Um evento de ontem (só dá para gravar direto no banco) não aparece na agenda.
	if _, err := pool.Exec(ctx, "INSERT INTO events (author_id, title, description, starts_at, location) VALUES ($1, 'Passado', 'x', now() - interval '1 day 1 hour', 'abc')", authorID); err != nil {
		t.Fatal(err)
	}

	// O visitante vê a mesma agenda que o logado.
	for name, viewer := range map[string]int64{"logado": authorID, "visitante": 0} {
		list := decode[struct{ Items []eventResponse }](t, do(t, NewHandler(pool, loggedAs(viewer)).List, http.MethodGet, "")).Items
		if len(list) != 2 || list[0].ID != near.ID || list[1].ID != far.ID {
			t.Errorf("%s: quer [%d %d], veio %+v", name, near.ID, far.ID, list)
		}
	}
}

// Fluxo do "vou" contra Postgres real: confirmar, contar, ver a lista, desistir e o evento passado.
func TestAttendance(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID, guestID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('Organizador de Teste'), ('Aluna de Teste') RETURNING id").Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	guestID = authorID + 1 // RESTART IDENTITY no testDB: os ids saem em sequência
	author, guest, visitor := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(guestID)), NewHandler(pool, loggedAs(0))

	rec := do(t, author.Create, http.MethodPost, body(future(time.Hour)))
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	created := decode[eventResponse](t, rec)
	if created.AttendeeCount != 0 || created.Going {
		t.Errorf("evento novo já tem confirmados: %+v", created)
	}
	id := strconv.FormatInt(created.ID, 10)

	// Confirmar duas vezes não duplica.
	for range 2 {
		if rec = doID(t, guest.PutAttendance, http.MethodPut, id); rec.Code != 204 {
			t.Fatalf("PUT: %d (%s)", rec.Code, rec.Body)
		}
	}
	got := decode[eventResponse](t, doID(t, guest.Get, http.MethodGet, id))
	want := []attendeeResponse{{ID: guestID, Name: "Aluna de Teste"}}
	if got.AttendeeCount != 1 || !got.Going || !reflect.DeepEqual(got.Attendees, want) {
		t.Errorf("visto por quem vai: %+v", got)
	}
	// O criador não entra sozinho na lista; o visitante vê o número e não recebe a lista.
	if got = decode[eventResponse](t, doID(t, author.Get, http.MethodGet, id)); got.Going || got.AttendeeCount != 1 {
		t.Errorf("visto pelo criador: %+v", got)
	}
	rec = doID(t, visitor.Get, http.MethodGet, id)
	if got = decode[eventResponse](t, rec); rec.Code != 200 || got.AttendeeCount != 1 || got.Going || strings.Contains(rec.Body.String(), `"attendees"`) {
		t.Errorf("visto pelo visitante: %d %s", rec.Code, rec.Body)
	}
	list := decode[struct{ Items []eventResponse }](t, do(t, guest.List, http.MethodGet, "")).Items
	if len(list) != 1 || !list[0].Going || list[0].AttendeeCount != 1 {
		t.Errorf("agenda de quem vai: %+v", list)
	}

	// Desistir tira da lista; o logado num evento vazio recebe [].
	if rec = doID(t, guest.DeleteAttendance, http.MethodDelete, id); rec.Code != 204 {
		t.Fatalf("DELETE: %d (%s)", rec.Code, rec.Body)
	}
	rec = doID(t, guest.Get, http.MethodGet, id)
	if got = decode[eventResponse](t, rec); got.AttendeeCount != 0 || got.Going || !strings.Contains(rec.Body.String(), `"attendees":[]`) {
		t.Errorf("depois de desistir: %s", rec.Body)
	}

	// Evento de ontem: a página abre, confirmar é 409 past, desistir continua 204.
	var past int64
	if err := pool.QueryRow(ctx, "INSERT INTO events (author_id, title, description, starts_at, location) VALUES ($1, 'Passado', 'x', now() - interval '1 day 1 hour', 'abc') RETURNING id", authorID).Scan(&past); err != nil {
		t.Fatal(err)
	}
	pastID := strconv.FormatInt(past, 10)
	if rec = doID(t, guest.Get, http.MethodGet, pastID); rec.Code != 200 {
		t.Errorf("GET passado: %d (%s)", rec.Code, rec.Body)
	}
	if rec = doID(t, guest.PutAttendance, http.MethodPut, pastID); rec.Code != 409 || decode[errorResponse](t, rec).Error != "past" {
		t.Errorf("PUT passado: %d (%s), quer 409 past", rec.Code, rec.Body)
	}
	if rec = doID(t, guest.DeleteAttendance, http.MethodDelete, pastID); rec.Code != 204 {
		t.Errorf("DELETE passado: %d, quer 204", rec.Code)
	}

	// Evento que não existe: 404 no GET e no PUT.
	for name, hf := range map[string]http.HandlerFunc{"GET": guest.Get, "PUT": guest.PutAttendance} {
		if rec = doID(t, hf, http.MethodGet, "999999"); rec.Code != 404 || decode[errorResponse](t, rec).Error != "not_found" {
			t.Errorf("%s inexistente: %d (%s), quer 404 not_found", name, rec.Code, rec.Body)
		}
	}
}

// Editar e apagar contra Postgres real: o criador e o moderador podem, os outros não (fatia 16).
func TestUpdateAndDelete(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('Organizador de Teste'), ('Outro Aluno'), ('Moderadora de Teste') RETURNING id").Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	otherID, modID := authorID+1, authorID+2 // RESTART IDENTITY no testDB: os ids saem em sequência
	if _, err := pool.Exec(ctx, "UPDATE users SET is_moderator = true WHERE id = $1", modID); err != nil {
		t.Fatal(err)
	}
	author, other, mod, visitor := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(otherID)), NewHandler(pool, loggedAs(modID)), NewHandler(pool, loggedAs(0))

	rec := do(t, author.Create, http.MethodPost, body(future(time.Hour)))
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	created := decode[eventResponse](t, rec)
	id := strconv.FormatInt(created.ID, 10)
	if rec = doID(t, other.PutAttendance, http.MethodPut, id); rec.Code != 204 {
		t.Fatalf("PUT presença: %d (%s)", rec.Code, rec.Body)
	}

	later := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	edited := `{"title":"  Sala nova  ","description":"Mudou de sala","starts_at":"` + later.Format(time.RFC3339) + `","location":"Bloco C"}`
	for _, step := range []struct {
		name       string
		hf         http.HandlerFunc
		id, body   string
		wantStatus int
		wantError  string
	}{
		{"visitante edita", visitor.Update, id, edited, 401, "unauthenticated"},
		{"outro aluno edita", other.Update, id, edited, 403, "not_author"},
		{"outro aluno com corpo inválido continua 403", other.Update, id, `{`, 403, "not_author"},
		{"outro aluno apaga", other.Delete, id, "", 403, "not_author"},
		{"criador com data no passado", author.Update, id, body(future(-time.Hour)), 400, "validation_failed"},
		{"criador edita evento que não existe", author.Update, "999999", edited, 404, "not_found"},
		{"moderadora apaga evento que não existe", mod.Delete, "999999", "", 404, "not_found"},
	} {
		rec := doIDBody(t, step.hf, http.MethodPut, step.id, step.body)
		if rec.Code != step.wantStatus || decode[errorResponse](t, rec).Error != step.wantError {
			t.Errorf("%s: %d (%s), quer %d %s", step.name, rec.Code, rec.Body, step.wantStatus, step.wantError)
		}
	}

	// O criador edita: os campos trocam (com trim), e autor, criação e presença ficam.
	rec = doIDBody(t, author.Update, http.MethodPut, id, edited)
	if rec.Code != 200 {
		t.Fatalf("PUT do criador: %d (%s)", rec.Code, rec.Body)
	}
	want := created
	want.Title, want.Description, want.StartsAt, want.Location, want.AttendeeCount = "Sala nova", "Mudou de sala", later.UTC(), "Bloco C", 1
	if got := decode[eventResponse](t, rec); !reflect.DeepEqual(got, want) || strings.Contains(rec.Body.String(), `"attendees"`) {
		t.Errorf("PUT devolveu %s, quer %+v", rec.Body, want)
	}
	// A moderadora edita evento alheio, e a resposta diz going do ponto de vista dela.
	if rec = doIDBody(t, mod.Update, http.MethodPut, id, edited); rec.Code != 200 || decode[eventResponse](t, rec).Going {
		t.Errorf("PUT da moderadora: %d (%s)", rec.Code, rec.Body)
	}

	// A moderadora apaga: o evento e a presença somem.
	if rec = doID(t, mod.Delete, http.MethodDelete, id); rec.Code != 204 {
		t.Fatalf("DELETE da moderadora: %d (%s)", rec.Code, rec.Body)
	}
	if rec = doID(t, author.Get, http.MethodGet, id); rec.Code != 404 {
		t.Errorf("GET depois do DELETE: %d, quer 404", rec.Code)
	}
	var left int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM event_attendees").Scan(&left); err != nil || left != 0 {
		t.Errorf("presença depois do DELETE: %d (%v), quer 0", left, err)
	}

	// O criador apaga o próprio evento, mesmo depois de ele passar.
	var past int64
	if err := pool.QueryRow(ctx, "INSERT INTO events (author_id, title, description, starts_at, location) VALUES ($1, 'Passado', 'x', now() - interval '3 days', 'abc') RETURNING id", authorID).Scan(&past); err != nil {
		t.Fatal(err)
	}
	if rec = doID(t, author.Delete, http.MethodDelete, strconv.FormatInt(past, 10)); rec.Code != 204 {
		t.Errorf("DELETE do passado: %d (%s), quer 204", rec.Code, rec.Body)
	}
}

// testDB conecta no banco de teste com as tabelas vazias; pula o teste sem TEST_DATABASE_URL.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	lockDB(t, pool)
	// CASCADE limpa também o que aponta para users (ideias, sessões).
	if _, err := pool.Exec(ctx, "TRUNCATE events, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	return pool
}

// lockDB segura o mesmo advisory lock dos testes de ideas: os pacotes rodam em paralelo e limpam as mesmas tabelas.
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
