package ideas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const validBody = `{"title":"App de carona","description":"Quero montar um app","course":"cc","category":"mobile","slots":3}`

// loggedAs troca a sessão de verdade por um usuário fixo (0 = visitante), sem moderação.
func loggedAs(id int64) func(*http.Request) (int64, bool, error) {
	return func(*http.Request) (int64, bool, error) { return id, false, nil }
}

// moderatorAs é o loggedAs de quem tem o cargo de moderação.
func moderatorAs(id int64) func(*http.Request) (int64, bool, error) {
	return func(*http.Request) (int64, bool, error) { return id, true, nil }
}

func do(t *testing.T, hf http.HandlerFunc, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	hf(rec, httptest.NewRequest(method, "/api/ideas", strings.NewReader(body)))
	return rec
}

// doID chama um handler de /api/ideas/{id} com o {id} preenchido, como o roteador faria.
func doID(t *testing.T, hf http.HandlerFunc, method, id string) *httptest.ResponseRecorder {
	t.Helper()
	return doIDBody(t, hf, method, id, "")
}

// doIDBody é o doID com corpo, para o PUT da ideia.
func doIDBody(t *testing.T, hf http.HandlerFunc, method, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/ideas/"+id, strings.NewReader(body))
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
		{"sem sessão", 0, validBody, 401, "unauthenticated", nil},
		{"json malformado", 1, `{`, 400, "invalid_json", nil},
		{"tipo errado", 1, `{"slots":"3"}`, 400, "invalid_json", nil},
		{"corpo grande", 1, `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, 413, "body_too_large", nil},
		{"tudo vazio", 1, `{}`, 400, "validation_failed", []string{"title", "description", "course", "category", "slots"}},
		{"título só espaços", 1, `{"title":"   ","description":"x","course":"cc","category":"web","slots":1}`, 400, "validation_failed", []string{"title"}},
		{"título com 2 caracteres e 4 bytes", 1, `{"title":"çã","description":"x","course":"cc","category":"web","slots":1}`, 400, "validation_failed", []string{"title"}},
		{"caractere nulo", 1, `{"title":"abc\u0000","description":"x","course":"cc","category":"web","slots":1}`, 400, "validation_failed", []string{"title"}},
		{"curso e categoria fora da lista", 1, `{"title":"abc","description":"x","course":"med","category":"culinaria","slots":1}`, 400, "validation_failed", []string{"course", "category"}},
		{"vagas acima de 20", 1, `{"title":"abc","description":"x","course":"cc","category":"web","slots":21}`, 400, "validation_failed", []string{"slots"}},
		{"habilidade fora do catálogo", 1, `{"title":"abc","description":"x","course":"cc","category":"web","slots":1,"skills":["culinaria"]}`, 400, "validation_failed", []string{"skills"}},
		{"mais de 5 habilidades", 1, `{"title":"abc","description":"x","course":"cc","category":"web","slots":1,"skills":["frontend","backend","mobile","design","data","ai"]}`, 400, "validation_failed", []string{"skills"}},
		{"habilidades com tipo errado", 1, `{"title":"abc","description":"x","course":"cc","category":"web","slots":1,"skills":"web"}`, 400, "invalid_json", nil},
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

func TestSessionErrorIs500(t *testing.T) {
	failing := func(*http.Request) (int64, bool, error) { return 0, false, errors.New("banco fora") }
	h := NewHandler(nil, failing)
	for method, hf := range map[string]http.HandlerFunc{"POST": h.Create, "GET": h.List} {
		rec := do(t, hf, method, validBody)
		if rec.Code != 500 || decode[errorResponse](t, rec).Error != "internal" {
			t.Errorf("%s: status = %d (%s), quer 500 internal", method, rec.Code, rec.Body)
		}
	}
	for name, hf := range map[string]http.HandlerFunc{
		"GET uma": h.Get, "PUT ideia": h.Update, "DELETE ideia": h.Delete, "PUT status": h.SetStatus,
		"PUT interesse": h.PutInterest, "DELETE interesse": h.DeleteInterest, "GET minhas": h.Mine, "GET perfil": h.Profile,
	} {
		if rec := doID(t, hf, http.MethodGet, "1"); rec.Code != 500 || decode[errorResponse](t, rec).Error != "internal" {
			t.Errorf("%s: status = %d (%s), quer 500 internal", name, rec.Code, rec.Body)
		}
	}
}

// Esses casos param antes do banco, então o handler roda sem conexão (nil).
func TestGetRejectsVisitorAndBadID(t *testing.T) {
	cases := []struct {
		name       string
		userID     int64
		id         string
		wantStatus int
		wantError  string
	}{
		{"visitante", 0, "1", 401, "unauthenticated"},
		{"id não numérico", 1, "abc", 404, "not_found"},
		{"id zero", 1, "0", 404, "not_found"},
		{"id negativo", 1, "-1", 404, "not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doID(t, NewHandler(nil, loggedAs(tc.userID)).Get, http.MethodGet, tc.id)
			if rec.Code != tc.wantStatus || decode[errorResponse](t, rec).Error != tc.wantError {
				t.Errorf("status = %d (%s), quer %d %s", rec.Code, rec.Body, tc.wantStatus, tc.wantError)
			}
		})
	}
}

// Esses casos param antes do banco, então o handler roda sem conexão (nil).
func TestEditRejectsVisitorAndBadID(t *testing.T) {
	visitor, logged := NewHandler(nil, loggedAs(0)), NewHandler(nil, loggedAs(1))
	cases := []struct {
		name       string
		hf         http.HandlerFunc
		id         string
		wantStatus int
		wantError  string
	}{
		{"PUT visitante", visitor.Update, "1", 401, "unauthenticated"},
		{"DELETE visitante", visitor.Delete, "1", 401, "unauthenticated"},
		{"PUT id inválido", logged.Update, "abc", 404, "not_found"},
		{"DELETE id zero", logged.Delete, "0", 404, "not_found"},
		{"PUT status visitante", visitor.SetStatus, "1", 401, "unauthenticated"},
		{"PUT status id inválido", logged.SetStatus, "abc", 404, "not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doIDBody(t, tc.hf, http.MethodPut, tc.id, validBody)
			if rec.Code != tc.wantStatus || decode[errorResponse](t, rec).Error != tc.wantError {
				t.Errorf("status = %d (%s), quer %d %s", rec.Code, rec.Body, tc.wantStatus, tc.wantError)
			}
		})
	}
}

func TestValidateCountsCharactersNotBytes(t *testing.T) {
	req := ideaRequest{
		Title:       strings.Repeat("ã", 120), // 120 caracteres, 240 bytes: len() reprovaria
		Description: "x",
		Course:      "cc",
		Category:    "web",
		Slots:       1,
	}
	if fields := req.validate(); len(fields) > 0 {
		t.Fatalf("esperava válido, veio %v", fields)
	}
}

// Esses casos param antes do banco, então o handler roda sem conexão (nil).
func TestInterestRejectsVisitorAndBadID(t *testing.T) {
	visitor, logged := NewHandler(nil, loggedAs(0)), NewHandler(nil, loggedAs(1))
	cases := []struct {
		name       string
		hf         http.HandlerFunc
		id         string
		wantStatus int
	}{
		{"PUT visitante", visitor.PutInterest, "1", 401},
		{"DELETE visitante", visitor.DeleteInterest, "1", 401},
		{"PUT id inválido", logged.PutInterest, "abc", 404},
		{"DELETE id inválido", logged.DeleteInterest, "abc", 204}, // nada a tirar: mesmo resultado de desistir
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := doID(t, tc.hf, http.MethodPut, tc.id); rec.Code != tc.wantStatus {
				t.Errorf("status = %d (%s), quer %d", rec.Code, rec.Body, tc.wantStatus)
			}
		})
	}
}

// Caminho feliz contra Postgres real. Precisa de TEST_DATABASE_URL apontando para um banco já migrado.
func TestCreateListAndGet(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (name, course, avatar_url) VALUES ('Autor de Teste', 'cc', 'https://cdn.discordapp.com/avatars/1/a.png') RETURNING id").Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(pool, loggedAs(authorID))

	rec := do(t, h.List, http.MethodGet, "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Fatalf("lista vazia: %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h.Create, http.MethodPost, `{"title":"  App de carona  ","description":"Quero montar um app","course":"cc","category":"mobile","slots":3}`)
	if rec.Code != 201 {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body)
	}
	first := decode[ideaResponse](t, rec)
	if first.Title != "App de carona" || first.Status != "open" || first.Author.Name != "Autor de Teste" || first.CreatedAt.IsZero() {
		t.Errorf("ideia criada inesperada: %+v", first)
	}
	if a := first.Author.AvatarURL; a == nil || *a != "https://cdn.discordapp.com/avatars/1/a.png" {
		t.Errorf("avatar do autor = %v", a)
	}

	if rec = do(t, h.Create, http.MethodPost, validBody); rec.Code != 201 {
		t.Fatalf("segundo POST: %d (%s)", rec.Code, rec.Body)
	}
	second := decode[ideaResponse](t, rec)

	rec = do(t, h.List, http.MethodGet, "")
	list := decode[struct{ Items []ideaResponse }](t, rec)
	if len(list.Items) != 2 || list.Items[0].ID != second.ID || list.Items[1].ID != first.ID {
		t.Fatalf("ordem errada, quer [%d %d]: %+v", second.ID, first.ID, list.Items)
	}

	// Com 4 ideias, o visitante vê só as 3 mais recentes e o logado vê todas.
	for range 2 {
		if rec = do(t, h.Create, http.MethodPost, validBody); rec.Code != 201 {
			t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
		}
	}
	all := decode[struct{ Items []ideaResponse }](t, do(t, h.List, http.MethodGet, "")).Items
	preview := decode[struct{ Items []ideaResponse }](t, do(t, NewHandler(pool, loggedAs(0)).List, http.MethodGet, "")).Items
	if len(all) != 4 || len(preview) != previewSize {
		t.Fatalf("logado viu %d (quer 4), visitante viu %d (quer %d)", len(all), len(preview), previewSize)
	}
	for i := range preview {
		if preview[i].ID != all[i].ID {
			t.Errorf("prévia não são as mais recentes: %+v", preview)
		}
	}

	// A página da ideia devolve o mesmo objeto do mural, mais a lista de interessados (vazia), porque quem pede é o autor.
	// Id que não existe é 404.
	want := first
	want.Interests = []interestResponse{}
	rec = doID(t, h.Get, http.MethodGet, strconv.FormatInt(first.ID, 10))
	if got := decode[ideaResponse](t, rec); rec.Code != 200 || !reflect.DeepEqual(got, want) {
		t.Errorf("GET da primeira: %d %+v, quer 200 %+v", rec.Code, got, want)
	}
	if rec = doID(t, h.Get, http.MethodGet, "999999"); rec.Code != 404 || decode[errorResponse](t, rec).Error != "not_found" {
		t.Errorf("GET inexistente: %d (%s), quer 404 not_found", rec.Code, rec.Body)
	}
}

// Só quem já escolheu o curso posta ideia: sem curso é 403 course_required; depois de escolher, 201.
func TestCreateRequiresCourse(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (name) VALUES ('Aluno Sem Curso') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(pool, loggedAs(userID))
	if rec := do(t, h.Create, http.MethodPost, validBody); rec.Code != 403 || decode[errorResponse](t, rec).Error != "course_required" {
		t.Errorf("sem curso: %d (%s), quer 403 course_required", rec.Code, rec.Body)
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET course = 'ads' WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	if rec := do(t, h.Create, http.MethodPost, validBody); rec.Code != 201 {
		t.Errorf("com curso: %d (%s), quer 201", rec.Code, rec.Body)
	}
}

// Fluxo do "quero participar" contra Postgres real: marcar, contar, ver a lista como autor e desistir.
func TestInterests(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID, fanID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID)
	if err == nil {
		// discord_id fictício: só existe no banco de teste.
		err = pool.QueryRow(ctx, `INSERT INTO users (name, discord_id) VALUES ('Ana Teste', 'discord-fake-1') RETURNING id`).Scan(&fanID)
	}
	if err != nil {
		t.Fatal(err)
	}
	author, fan := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(fanID))
	rec := do(t, author.Create, http.MethodPost, validBody)
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	id := strconv.FormatInt(decode[ideaResponse](t, rec).ID, 10)

	for _, step := range []struct {
		name       string
		hf         http.HandlerFunc
		id         string
		wantStatus int
	}{
		{"marca", fan.PutInterest, id, 204},
		{"marca de novo (idempotente)", fan.PutInterest, id, 204},
		{"autor na própria ideia", author.PutInterest, id, 409},
		{"ideia que não existe", fan.PutInterest, "999999", 404},
	} {
		if rec := doID(t, step.hf, http.MethodPut, step.id); rec.Code != step.wantStatus {
			t.Errorf("%s: status = %d (%s), quer %d", step.name, rec.Code, rec.Body, step.wantStatus)
		}
	}
	if rec := doID(t, author.PutInterest, http.MethodPut, id); decode[errorResponse](t, rec).Error != "own_idea" {
		t.Errorf("autor na própria ideia: %s, quer own_idea", rec.Body)
	}

	// Mural: os dois veem 1 interessado; só quem marcou aparece como interessado.
	for name, tc := range map[string]struct {
		h              *Handler
		wantInterested bool
	}{"autora": {author, false}, "interessada": {fan, true}} {
		items := decode[struct{ Items []ideaResponse }](t, do(t, tc.h.List, http.MethodGet, "")).Items
		if len(items) != 1 || items[0].InterestCount != 1 || items[0].Interested != tc.wantInterested {
			t.Errorf("mural da %s: %+v", name, items)
		}
	}

	// Rostos: quem está logado vê a foto e o nome, sem o discord_id; o visitante, só o número.
	wantFaces := fmt.Sprintf(`"interest_faces":[{"id":%d,"name":"Ana Teste","avatar_url":null}]`, fanID)
	if rec := do(t, author.List, http.MethodGet, ""); !strings.Contains(rec.Body.String(), wantFaces) {
		t.Errorf("rostos no mural: %s, quer %s", rec.Body, wantFaces)
	}
	if rec := doID(t, author.Get, http.MethodGet, id); !strings.Contains(rec.Body.String(), wantFaces) {
		t.Errorf("rostos na página: %s, quer %s", rec.Body, wantFaces)
	}
	if rec := do(t, NewHandler(pool, loggedAs(0)).List, http.MethodGet, ""); !strings.Contains(rec.Body.String(), `"interest_count":1,"interested":false,"interest_faces":[]`) {
		t.Errorf("visitante recebeu rostos: %s", rec.Body)
	}

	// Página: só a autora recebe a lista, com o discord_id; para os outros o campo nem vem no JSON.
	rec = doID(t, fan.Get, http.MethodGet, id)
	if strings.Contains(rec.Body.String(), `"interests"`) {
		t.Errorf("interessada recebeu a lista: %s", rec.Body)
	}
	got := decode[ideaResponse](t, doID(t, author.Get, http.MethodGet, id))
	if len(got.Interests) != 1 || got.Interests[0].Name != "Ana Teste" || got.Interests[0].DiscordID == nil || *got.Interests[0].DiscordID != "discord-fake-1" {
		t.Errorf("lista da autora: %+v", got.Interests)
	}

	// Desistir tira da lista; repetir continua 204. A autora passa a receber [] (não null, não ausente).
	for range 2 {
		if rec := doID(t, fan.DeleteInterest, http.MethodDelete, id); rec.Code != 204 {
			t.Fatalf("DELETE: %d (%s)", rec.Code, rec.Body)
		}
	}
	rec = doID(t, author.Get, http.MethodGet, id)
	if !strings.Contains(rec.Body.String(), `"interest_count":0,"interested":false,"interest_faces":[],"interests":[]`) {
		t.Errorf("depois de desistir: %s", rec.Body)
	}
}

// Editar e apagar contra Postgres real: só o autor mexe, e apagar leva os interessados junto.
func TestUpdateAndDelete(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID, otherID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID)
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO users (name) VALUES ('Outro Aluno') RETURNING id`).Scan(&otherID)
	}
	if err != nil {
		t.Fatal(err)
	}
	author, other := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(otherID))
	rec := do(t, author.Create, http.MethodPost, validBody)
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	created := decode[ideaResponse](t, rec)
	id := strconv.FormatInt(created.ID, 10)
	if rec := doID(t, other.PutInterest, http.MethodPut, id); rec.Code != 204 {
		t.Fatalf("PUT interesse: %d (%s)", rec.Code, rec.Body)
	}

	const edited = `{"title":"  Carona solidária  ","description":"Agora com mapa","course":"ads","category":"web","slots":1}`
	for _, step := range []struct {
		name       string
		hf         http.HandlerFunc
		id, body   string
		wantStatus int
		wantError  string
	}{
		{"outro aluno edita", other.Update, id, edited, 403, "not_author"},
		{"outro aluno com corpo inválido continua 403", other.Update, id, `{`, 403, "not_author"},
		{"outro aluno apaga", other.Delete, id, "", 403, "not_author"},
		{"autora com campo inválido", author.Update, id, `{"title":"ab","description":"x","course":"cc","category":"web","slots":1}`, 400, "validation_failed"},
		{"autora edita ideia que não existe", author.Update, "999999", edited, 404, "not_found"},
		{"autora apaga ideia que não existe", author.Delete, "999999", "", 404, "not_found"},
	} {
		rec := doIDBody(t, step.hf, http.MethodPut, step.id, step.body)
		if rec.Code != step.wantStatus || decode[errorResponse](t, rec).Error != step.wantError {
			t.Errorf("%s: %d (%s), quer %d %s", step.name, rec.Code, rec.Body, step.wantStatus, step.wantError)
		}
	}

	// Editar troca os campos (com trim) e mantém id, status, autor, data e interessados.
	rec = doIDBody(t, author.Update, http.MethodPut, id, edited)
	if rec.Code != 200 {
		t.Fatalf("PUT: %d (%s)", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), `"interests"`) {
		t.Errorf("PUT não devolve a lista: %s", rec.Body)
	}
	want := created
	want.Title, want.Description, want.Course, want.Category, want.Slots = "Carona solidária", "Agora com mapa", "ads", "web", 1
	want.InterestCount = 1
	want.InterestFaces = json.RawMessage(fmt.Sprintf(`[{"id":%d,"name":"Outro Aluno","avatar_url":null}]`, otherID))
	if got := decode[ideaResponse](t, rec); !reflect.DeepEqual(got, want) {
		t.Errorf("PUT devolveu %+v, quer %+v", got, want)
	}
	if got := decode[ideaResponse](t, doID(t, other.Get, http.MethodGet, id)); got.Title != "Carona solidária" || got.Slots != 1 {
		t.Errorf("GET depois do PUT: %+v", got)
	}

	// Apagar responde 204; a ideia e os interessados somem.
	if rec := doID(t, author.Delete, http.MethodDelete, id); rec.Code != 204 {
		t.Fatalf("DELETE: %d (%s)", rec.Code, rec.Body)
	}
	if rec := doID(t, author.Get, http.MethodGet, id); rec.Code != 404 {
		t.Errorf("GET depois do DELETE: %d, quer 404", rec.Code)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM interests`).Scan(&left); err != nil || left != 0 {
		t.Errorf("interessados depois do DELETE: %d (%v), quer 0", left, err)
	}
}

// Status contra Postgres real: só o autor troca, só ideia aberta recebe interessados e done_at acompanha o done.
func TestStatus(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID, fanID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID)
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO users (name) VALUES ('Ana Teste') RETURNING id`).Scan(&fanID)
	}
	if err != nil {
		t.Fatal(err)
	}
	author, fan := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(fanID))
	rec := do(t, author.Create, http.MethodPost, validBody)
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	created := decode[ideaResponse](t, rec)
	if created.DoneAt != nil || !strings.Contains(rec.Body.String(), `"done_at":null`) {
		t.Errorf("ideia nova com done_at: %s", rec.Body)
	}
	id := strconv.FormatInt(created.ID, 10)

	for _, step := range []struct {
		name       string
		hf         http.HandlerFunc
		id, body   string
		wantStatus int
		wantError  string
	}{
		{"outra pessoa troca", fan.SetStatus, id, `{"status":"done"}`, 403, "not_author"},
		{"status fora da lista", author.SetStatus, id, `{"status":"closed"}`, 400, "validation_failed"},
		{"status vazio", author.SetStatus, id, `{}`, 400, "validation_failed"},
		{"json malformado", author.SetStatus, id, `{`, 400, "invalid_json"},
		{"ideia que não existe", author.SetStatus, "999999", `{"status":"done"}`, 404, "not_found"},
	} {
		rec := doIDBody(t, step.hf, http.MethodPut, step.id, step.body)
		if rec.Code != step.wantStatus || decode[errorResponse](t, rec).Error != step.wantError {
			t.Errorf("%s: %d (%s), quer %d %s", step.name, rec.Code, rec.Body, step.wantStatus, step.wantError)
		}
	}

	// setStatus troca e devolve a ideia como o GET a vê.
	setStatus := func(status string) ideaResponse {
		t.Helper()
		if rec := doIDBody(t, author.SetStatus, http.MethodPut, id, `{"status":" `+status+` "}`); rec.Code != 204 {
			t.Fatalf("PUT status %s: %d (%s)", status, rec.Code, rec.Body)
		}
		return decode[ideaResponse](t, doID(t, author.Get, http.MethodGet, id))
	}
	interest := func() int { return doID(t, fan.PutInterest, http.MethodPut, id).Code }

	// Em andamento trava a lista, mas não tira quem já estava.
	if code := interest(); code != 204 {
		t.Fatalf("interesse em ideia aberta: %d", code)
	}
	if got := setStatus("in_progress"); got.Status != "in_progress" || got.DoneAt != nil || got.InterestCount != 1 {
		t.Errorf("em andamento: %+v", got)
	}
	if rec := doID(t, fan.PutInterest, http.MethodPut, id); rec.Code != 409 || decode[errorResponse](t, rec).Error != "closed" {
		t.Errorf("interesse em ideia em andamento: %d (%s), quer 409 closed", rec.Code, rec.Body)
	}

	// Concluir preenche done_at; concluir de novo mantém a data.
	done := setStatus("done")
	if done.Status != "done" || done.DoneAt == nil {
		t.Fatalf("concluída: %+v", done)
	}
	if again := setStatus("done"); again.DoneAt == nil || !again.DoneAt.Equal(*done.DoneAt) {
		t.Errorf("concluir de novo mudou done_at: %v, era %v", again.DoneAt, done.DoneAt)
	}
	if code := interest(); code != 409 {
		t.Errorf("interesse em ideia concluída: %d, quer 409", code)
	}

	// Reabrir limpa done_at e destrava.
	if got := setStatus("open"); got.Status != "open" || got.DoneAt != nil {
		t.Errorf("reaberta: %+v", got)
	}
	if code := interest(); code != 204 {
		t.Errorf("interesse em ideia reaberta: %d, quer 204", code)
	}
}

// Mural e vitrine contra Postgres real: concluída sai do mural, a vitrine ordena pela conclusão e o visitante vê 3.
func TestMuralAndShowcase(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	author, visitor := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(0))
	ids := make([]int64, 6) // ids[0] é a mais antiga
	for i := range ids {
		rec := do(t, author.Create, http.MethodPost, validBody)
		if rec.Code != 201 {
			t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
		}
		ids[i] = decode[ideaResponse](t, rec).ID
	}
	setStatus := func(id int64, status string) {
		t.Helper()
		if rec := doIDBody(t, author.SetStatus, http.MethodPut, strconv.FormatInt(id, 10), `{"status":"`+status+`"}`); rec.Code != 204 {
			t.Fatalf("PUT status: %d (%s)", rec.Code, rec.Body)
		}
	}
	// Conclui fora da ordem de postagem: a vitrine tem de seguir a conclusão, não a postagem.
	for _, i := range []int{1, 0, 2, 3} {
		setStatus(ids[i], "done")
	}
	setStatus(ids[4], "in_progress")

	list := func(h *Handler, query string) []int64 {
		t.Helper()
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequest(http.MethodGet, "/api/ideas?"+query, nil))
		if rec.Code != 200 {
			t.Fatalf("GET ?%s: %d (%s)", query, rec.Code, rec.Body)
		}
		var got []int64
		for _, it := range decode[struct{ Items []ideaResponse }](t, rec).Items {
			got = append(got, it.ID)
		}
		return got
	}
	for _, tc := range []struct {
		name string
		h    *Handler
		q    string
		want []int64
	}{
		{"mural sem as concluídas", author, "", []int64{ids[5], ids[4]}},
		{"vitrine pela conclusão", author, "status=done", []int64{ids[3], ids[2], ids[0], ids[1]}},
		{"vitrine do visitante: só 3", visitor, "status=done", []int64{ids[3], ids[2], ids[0]}},
	} {
		if got := list(tc.h, tc.q); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, quer %v", tc.name, got, tc.want)
		}
	}
}

// Filtro com valor fora da lista para antes do banco, inclusive para o visitante.
func TestListRejectsBadFilter(t *testing.T) {
	for _, q := range []string{"status=closed", "course=med", "category=culinaria", "skill=culinaria"} {
		for _, id := range []int64{0, 1} {
			rec := httptest.NewRecorder()
			NewHandler(nil, loggedAs(id)).List(rec, httptest.NewRequest(http.MethodGet, "/api/ideas?"+q, nil))
			name, _, _ := strings.Cut(q, "=")
			if got := decode[errorResponse](t, rec); rec.Code != 400 || got.Error != "validation_failed" || got.Fields[name] != "valor inválido" {
				t.Errorf("?%s (usuário %d): %d (%s), quer 400 em fields.%s", q, id, rec.Code, rec.Body, name)
			}
		}
	}
}

// Filtros do mural contra Postgres real: cada um, a soma, o vazio e o visitante, que só vê a prévia.
func TestListFilters(t *testing.T) {
	pool := testDB(t)
	var authorID int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	author, visitor := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(0))
	post := func(course, category, skills string) int64 {
		t.Helper()
		body := `{"title":"Ideia","description":"x","course":"` + course + `","category":"` + category + `","slots":2,"skills":[` + skills + `]}`
		rec := do(t, author.Create, http.MethodPost, body)
		if rec.Code != 201 {
			t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
		}
		return decode[ideaResponse](t, rec).ID
	}
	a := post("cc", "web", `"frontend","backend"`)
	b := post("cc", "mobile", `"backend"`)
	c := post("ads", "web", ``)
	d := post("eng", "web", `"backend"`)
	e := post("cc", "web", `"backend"`)
	for id, status := range map[int64]string{c: "in_progress", e: "done"} {
		if rec := doIDBody(t, author.SetStatus, http.MethodPut, strconv.FormatInt(id, 10), `{"status":"`+status+`"}`); rec.Code != 204 {
			t.Fatalf("PUT status: %d (%s)", rec.Code, rec.Body)
		}
	}

	list := func(h *Handler, query string) []int64 {
		t.Helper()
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequest(http.MethodGet, "/api/ideas?"+query, nil))
		if rec.Code != 200 {
			t.Fatalf("GET ?%s: %d (%s)", query, rec.Code, rec.Body)
		}
		got := []int64{}
		for _, it := range decode[struct{ Items []ideaResponse }](t, rec).Items {
			got = append(got, it.ID)
		}
		return got
	}
	for _, tc := range []struct {
		name string
		h    *Handler
		q    string
		want []int64
	}{
		{"vazio é sem filtro", author, "status=&course=&category=&skill=", []int64{d, c, b, a}},
		{"status", author, "status=in_progress", []int64{c}},
		{"curso", author, "course=cc", []int64{b, a}},
		{"categoria", author, "category=web", []int64{d, c, a}},
		{"habilidade", author, "skill=backend", []int64{d, b, a}},
		{"a soma", author, "course=cc&category=web&skill=backend", []int64{a}},
		{"vitrine com filtro", author, "status=done&course=cc", []int64{e}},
		{"nada bate", author, "skill=design", []int64{}},
		{"visitante ignora os filtros", visitor, "course=eng&status=in_progress", []int64{d, c, b}},
		{"visitante na vitrine", visitor, "status=done&course=ads", []int64{e}},
	} {
		if got := list(tc.h, tc.q); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, quer %v", tc.name, got, tc.want)
		}
	}
}

// Busca inválida para antes do banco, inclusive para o visitante. 100 caracteres com acento ainda passam (conta caractere, não byte).
func TestListRejectsBadSearch(t *testing.T) {
	for _, tc := range []struct{ q, want string }{
		{strings.Repeat("a", 101), "deve ter até 100 caracteres"},
		{"%00", "contém caractere inválido"},
		{"%FF", "contém caractere inválido"},
	} {
		for _, id := range []int64{0, 1} {
			rec := httptest.NewRecorder()
			NewHandler(nil, loggedAs(id)).List(rec, httptest.NewRequest(http.MethodGet, "/api/ideas?q="+tc.q, nil))
			if got := decode[errorResponse](t, rec); rec.Code != 400 || got.Error != "validation_failed" || got.Fields["q"] != tc.want {
				t.Errorf("?q=%.10s (usuário %d): %d (%s), quer 400 %q", tc.q, id, rec.Code, rec.Body, tc.want)
			}
		}
	}
	fields := map[string]string{}
	q := strings.Repeat("ã", 100)
	if got := search(httptest.NewRequest(http.MethodGet, "/api/ideas?q=%20"+q+"%20", nil), fields); len(fields) > 0 || got == nil || *got != "%"+q+"%" {
		t.Errorf("100 caracteres com acento: %v %v", fields, got)
	}
}

// Busca contra Postgres real: título, descrição, sem acento, soma com filtro, curinga literal e visitante.
func TestListSearch(t *testing.T) {
	pool := testDB(t)
	var authorID int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	author, visitor := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(0))
	post := func(title, description, course string) int64 {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"title": title, "description": description, "course": course, "category": "web", "slots": 2})
		rec := do(t, author.Create, http.MethodPost, string(body))
		if rec.Code != 201 {
			t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
		}
		return decode[ideaResponse](t, rec).ID
	}
	a := post("Maratona de Programação", "Treino semanal", "cc")
	b := post("App de carona", "Front em React, back em Go", "ads")
	c := post("Bolsa de 50% no curso", "Quem topa?", "cc")
	d := post("Site do DA", "Front em react_native", "eng")

	list := func(h *Handler, query string) []int64 {
		t.Helper()
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequest(http.MethodGet, "/api/ideas?"+query, nil))
		if rec.Code != 200 {
			t.Fatalf("GET ?%s: %d (%s)", query, rec.Code, rec.Body)
		}
		got := []int64{}
		for _, it := range decode[struct{ Items []ideaResponse }](t, rec).Items {
			got = append(got, it.ID)
		}
		return got
	}
	for _, tc := range []struct {
		name string
		h    *Handler
		q    string
		want []int64
	}{
		{"pelo título", author, "q=carona", []int64{b}},
		{"pela descrição", author, "q=semanal", []int64{a}},
		{"sem acento nem maiúscula", author, "q=PROGRAMACAO", []int64{a}},
		{"com acento acha sem acento", author, "q=s%C3%ADte", []int64{d}}, // "síte"
		{"soma com filtro", author, "q=react&course=ads", []int64{b}},
		{"% é literal", author, "q=50%25", []int64{c}},
		{"% sozinho não é curinga", author, "q=%25", []int64{c}},
		{"_ é literal", author, "q=react_", []int64{d}}, // como curinga, acharia também o "React," de b
		{`\ é literal`, author, "q=%5C", []int64{}},     // sem escape, viraria o padrão '%\%' e acharia o 50% de c
		{"espaços das pontas saem", author, "q=%20%20carona%20", []int64{b}},
		{"vazio é sem busca", author, "q=%20", []int64{d, c, b, a}},
		{"nada bate", author, "q=kotlin", []int64{}},
		{"visitante ignora a busca", visitor, "q=carona", []int64{d, c, b}},
	} {
		if got := list(tc.h, tc.q); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, quer %v", tc.name, got, tc.want)
		}
	}
}

// Minhas ideias contra Postgres real: só as de quem pede, em todos os status, e 401 para visitante.
func TestMine(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID, otherID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID)
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Outro Aluno', 'ads') RETURNING id`).Scan(&otherID)
	}
	if err != nil {
		t.Fatal(err)
	}
	author, other := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, loggedAs(otherID))

	if rec := do(t, author.Mine, http.MethodGet, ""); rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Fatalf("sem ideias: %d %s", rec.Code, rec.Body)
	}
	var mine []int64
	for range 2 {
		rec := do(t, author.Create, http.MethodPost, validBody)
		if rec.Code != 201 {
			t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
		}
		mine = append(mine, decode[ideaResponse](t, rec).ID)
	}
	if rec := do(t, other.Create, http.MethodPost, validBody); rec.Code != 201 {
		t.Fatalf("POST do outro: %d (%s)", rec.Code, rec.Body)
	}
	// A primeira concluída continua na lista (o mural já não a mostra).
	if rec := doIDBody(t, author.SetStatus, http.MethodPut, strconv.FormatInt(mine[0], 10), `{"status":"done"}`); rec.Code != 204 {
		t.Fatalf("PUT status: %d (%s)", rec.Code, rec.Body)
	}

	items := decode[struct{ Items []ideaResponse }](t, do(t, author.Mine, http.MethodGet, "")).Items
	if len(items) != 2 || items[0].ID != mine[1] || items[1].ID != mine[0] || items[1].Status != "done" {
		t.Errorf("minhas ideias: %+v, quer [%d %d] com a segunda concluída", items, mine[1], mine[0])
	}
	if rec := do(t, NewHandler(pool, loggedAs(0)).Mine, http.MethodGet, ""); rec.Code != 401 || decode[errorResponse](t, rec).Error != "unauthenticated" {
		t.Errorf("visitante: %d (%s), quer 401", rec.Code, rec.Body)
	}
}

// Esses casos param antes do banco, então o handler roda sem conexão (nil).
func TestProfileRejectsVisitorAndBadID(t *testing.T) {
	cases := []struct {
		name       string
		userID     int64
		id         string
		wantStatus int
		wantError  string
	}{
		{"visitante", 0, "1", 401, "unauthenticated"},
		{"id não numérico", 1, "abc", 404, "not_found"},
		{"id zero", 1, "0", 404, "not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doID(t, NewHandler(nil, loggedAs(tc.userID)).Profile, http.MethodGet, tc.id)
			if rec.Code != tc.wantStatus || decode[errorResponse](t, rec).Error != tc.wantError {
				t.Errorf("status = %d (%s), quer %d %s", rec.Code, rec.Body, tc.wantStatus, tc.wantError)
			}
		})
	}
}

// Habilidades procuradas contra Postgres real: o POST limpa e ordena, sem o campo fica [], o PUT troca,
// e a lista chega no GET, no mural e em "Minhas ideias".
func TestIdeaSkills(t *testing.T) {
	pool := testDB(t)
	var authorID int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(pool, loggedAs(authorID))

	rec := do(t, h.Create, http.MethodPost, `{"title":"App de carona","description":"x","course":"cc","category":"web","slots":2,"skills":["design","frontend","design"]}`)
	if got := decode[ideaResponse](t, rec); rec.Code != 201 || !reflect.DeepEqual(got.Skills, []string{"frontend", "design"}) {
		t.Fatalf("POST com habilidades: %d %s, quer [frontend design]", rec.Code, rec.Body)
	}
	if rec := do(t, h.Create, http.MethodPost, validBody); rec.Code != 201 || !strings.Contains(rec.Body.String(), `"skills":[]`) {
		t.Errorf("POST sem habilidades: %d %s, quer skills []", rec.Code, rec.Body)
	}

	id := strconv.FormatInt(decode[struct{ Items []ideaResponse }](t, do(t, h.List, http.MethodGet, "")).Items[1].ID, 10) // a primeira postada
	rec = doIDBody(t, h.Update, http.MethodPut, id, `{"title":"App de carona","description":"x","course":"cc","category":"web","slots":2,"skills":["data"]}`)
	if got := decode[ideaResponse](t, rec); rec.Code != 200 || !reflect.DeepEqual(got.Skills, []string{"data"}) {
		t.Fatalf("PUT: %d %s, quer [data]", rec.Code, rec.Body)
	}
	if got := decode[ideaResponse](t, doID(t, h.Get, http.MethodGet, id)); !reflect.DeepEqual(got.Skills, []string{"data"}) {
		t.Errorf("GET depois do PUT: %v, quer [data]", got.Skills)
	}
	for name, hf := range map[string]http.HandlerFunc{"mural": h.List, "minhas": h.Mine} {
		items := decode[struct{ Items []ideaResponse }](t, do(t, hf, http.MethodGet, "")).Items
		if len(items) != 2 || !reflect.DeepEqual(items[1].Skills, []string{"data"}) || items[0].Skills == nil {
			t.Errorf("%s: %+v, quer a segunda com [data] e a primeira com []", name, items)
		}
	}
}

func TestCleanSkills(t *testing.T) {
	cases := []struct {
		name      string
		in        []string
		want      []string
		wantError string
	}{
		{"vazio", []string{}, []string{}, ""},
		{"ordem do catálogo, sem repetir", []string{"data", "frontend", "data"}, []string{"frontend", "data"}, ""},
		{"5 é o limite", []string{"frontend", "backend", "mobile", "design", "data"}, []string{"frontend", "backend", "mobile", "design", "data"}, ""},
		{"6 passa do limite", []string{"frontend", "backend", "mobile", "design", "data", "ai"}, nil, "escolha até 5"},
		{"repetidas não contam no limite", []string{"ai", "ai", "ai", "ai", "ai", "ai"}, []string{"ai"}, ""},
		{"fora do catálogo", []string{"backend", "culinaria"}, nil, "valor inválido"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]string{}
			got := CleanSkills(fields, "skills", tc.in)
			if fields["skills"] != tc.wantError {
				t.Fatalf("erro = %q, quer %q", fields["skills"], tc.wantError)
			}
			if tc.wantError == "" && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("= %v, quer %v", got, tc.want)
			}
		})
	}
}

// Perfil contra Postgres real: dados do aluno, ideias postadas (todas, das mais novas para as mais antigas),
// as que participa (da marcação mais recente para a mais antiga) e o interested de quem está vendo.
func TestProfile(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var anaID, betoID, caioID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, course, discord_id, skills) VALUES ('Ana de Teste', 'cc', '100', '{backend,data}') RETURNING id`).Scan(&anaID)
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Beto de Teste', 'ads') RETURNING id`).Scan(&betoID)
	}
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO users (name) VALUES ('Caio de Teste') RETURNING id`).Scan(&caioID)
	}
	if err != nil {
		t.Fatal(err)
	}
	create := func(authorID int64) int64 {
		t.Helper()
		rec := do(t, NewHandler(pool, loggedAs(authorID)).Create, http.MethodPost, validBody)
		if rec.Code != 201 {
			t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
		}
		return decode[ideaResponse](t, rec).ID
	}
	ana1, ana2, beto1, beto2 := create(anaID), create(anaID), create(betoID), create(betoID)
	// Ana marca a beto2 antes da beto1: a lista segue a marcação, não a postagem. As datas são fixas para não empatar.
	if _, err := pool.Exec(ctx, `INSERT INTO interests (idea_id, user_id, created_at) VALUES ($1, $3, now() - interval '1 hour'), ($2, $3, now())`,
		beto2, beto1, anaID); err != nil {
		t.Fatal(err)
	}
	caio := NewHandler(pool, loggedAs(caioID))
	if rec := doID(t, caio.PutInterest, http.MethodPut, strconv.FormatInt(ana1, 10)); rec.Code != 204 {
		t.Fatalf("PUT interesse: %d (%s)", rec.Code, rec.Body)
	}

	rec := doID(t, caio.Profile, http.MethodGet, strconv.FormatInt(anaID, 10))
	if rec.Code != 200 {
		t.Fatalf("perfil da Ana: %d (%s)", rec.Code, rec.Body)
	}
	got := decode[profileResponse](t, rec)
	if got.ID != anaID || got.Name != "Ana de Teste" || got.Course == nil || *got.Course != "cc" || got.DiscordID == nil || *got.DiscordID != "100" ||
		!reflect.DeepEqual(got.Skills, []string{"backend", "data"}) {
		t.Errorf("dados da Ana: %+v", got)
	}
	ids := func(items []ideaResponse) []int64 {
		out := []int64{}
		for _, i := range items {
			out = append(out, i.ID)
		}
		return out
	}
	if want := []int64{ana2, ana1}; !reflect.DeepEqual(ids(got.Ideas), want) {
		t.Errorf("ideias postadas = %v, quer %v", ids(got.Ideas), want)
	}
	if want := []int64{beto1, beto2}; !reflect.DeepEqual(ids(got.Participating), want) {
		t.Errorf("participa de = %v, quer %v", ids(got.Participating), want)
	}
	// interested é de quem vê (Caio): ele está só na ana1.
	if len(got.Ideas) == 2 && (!got.Ideas[1].Interested || got.Ideas[0].Interested || got.Participating[0].Interested) {
		t.Errorf("interested deveria ser só da ana1 (Caio participa): %+v", got)
	}

	// Sem ideias nem curso: listas vazias ([], não null) e course null.
	rec = doID(t, caio.Profile, http.MethodGet, strconv.FormatInt(caioID, 10))
	if body := rec.Body.String(); rec.Code != 200 || !strings.Contains(body, `"course":null`) || !strings.Contains(body, `"skills":[]`) || !strings.Contains(body, `"ideas":[]`) {
		t.Errorf("perfil vazio do Caio: %d %s", rec.Code, body)
	}
	if got := decode[profileResponse](t, rec); len(got.Participating) != 1 || got.Participating[0].ID != ana1 {
		t.Errorf("Caio participa de %v, quer [%d]", ids(got.Participating), ana1)
	}

	// O autor (Beto) vê as habilidades de quem se interessou (Ana).
	rec = doID(t, NewHandler(pool, loggedAs(betoID)).Get, http.MethodGet, strconv.FormatInt(beto1, 10))
	if people := decode[ideaResponse](t, rec).Interests; len(people) != 1 || !reflect.DeepEqual(people[0].Skills, []string{"backend", "data"}) {
		t.Errorf("interessados da beto1: %+v, quer a Ana com [backend data]", people)
	}

	if rec := doID(t, caio.Profile, http.MethodGet, "999999"); rec.Code != 404 || decode[errorResponse](t, rec).Error != "not_found" {
		t.Errorf("perfil inexistente: %d (%s), quer 404 not_found", rec.Code, rec.Body)
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
	// CASCADE limpa também o que aponta para elas (interests, sessions).
	if _, err := pool.Exec(ctx, "TRUNCATE ideas, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	return pool
}

// O moderador apaga ideia alheia, mas não edita nem troca o status (docs/contratos/fatia-15-cargo-de-moderacao.md).
func TestModeratorDeletes(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	var authorID, modID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, course) VALUES ('Autora de Teste', 'cc') RETURNING id`).Scan(&authorID)
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO users (name) VALUES ('Moderador de Teste') RETURNING id`).Scan(&modID)
	}
	if err != nil {
		t.Fatal(err)
	}
	author, mod := NewHandler(pool, loggedAs(authorID)), NewHandler(pool, moderatorAs(modID))
	rec := do(t, author.Create, http.MethodPost, validBody)
	if rec.Code != 201 {
		t.Fatalf("POST: %d (%s)", rec.Code, rec.Body)
	}
	id := strconv.FormatInt(decode[ideaResponse](t, rec).ID, 10)

	if rec = doIDBody(t, mod.Update, http.MethodPut, id, validBody); rec.Code != 403 {
		t.Errorf("moderador edita: %d (%s), quer 403", rec.Code, rec.Body)
	}
	if rec = doIDBody(t, mod.SetStatus, http.MethodPut, id, `{"status":"done"}`); rec.Code != 403 {
		t.Errorf("moderador troca o status: %d (%s), quer 403", rec.Code, rec.Body)
	}
	if rec = doID(t, mod.Delete, http.MethodDelete, id); rec.Code != 204 {
		t.Fatalf("moderador apaga: %d (%s), quer 204", rec.Code, rec.Body)
	}
	if rec = doID(t, author.Get, http.MethodGet, id); rec.Code != 404 {
		t.Errorf("GET depois do DELETE do moderador: %d, quer 404", rec.Code)
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
