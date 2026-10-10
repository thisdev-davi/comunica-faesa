package ideas

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const validBody = `{"title":"App de carona","description":"Quero montar um app","course":"cc","category":"mobile","slots":3}`

// loggedAs troca a sessão de verdade por um usuário fixo (0 = visitante).
func loggedAs(id int64) func(*http.Request) (int64, error) {
	return func(*http.Request) (int64, error) { return id, nil }
}

func do(t *testing.T, hf http.HandlerFunc, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	hf(rec, httptest.NewRequest(method, "/api/ideas", strings.NewReader(body)))
	return rec
}

// getOne chama o Get com o {id} preenchido, como o roteador faria.
func getOne(t *testing.T, h *Handler, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/ideas/"+id, nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	h.Get(rec, req)
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
	failing := func(*http.Request) (int64, error) { return 0, errors.New("banco fora") }
	h := NewHandler(nil, failing)
	for method, hf := range map[string]http.HandlerFunc{"POST": h.Create, "GET": h.List} {
		rec := do(t, hf, method, validBody)
		if rec.Code != 500 || decode[errorResponse](t, rec).Error != "internal" {
			t.Errorf("%s: status = %d (%s), quer 500 internal", method, rec.Code, rec.Body)
		}
	}
	if rec := getOne(t, h, "1"); rec.Code != 500 || decode[errorResponse](t, rec).Error != "internal" {
		t.Errorf("GET uma: status = %d (%s), quer 500 internal", rec.Code, rec.Body)
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
			rec := getOne(t, NewHandler(nil, loggedAs(tc.userID)), tc.id)
			if rec.Code != tc.wantStatus || decode[errorResponse](t, rec).Error != tc.wantError {
				t.Errorf("status = %d (%s), quer %d %s", rec.Code, rec.Body, tc.wantStatus, tc.wantError)
			}
		})
	}
}

func TestValidateCountsCharactersNotBytes(t *testing.T) {
	req := createRequest{
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

// Caminho feliz contra Postgres real. Precisa de TEST_DATABASE_URL apontando para um banco já migrado.
func TestCreateListAndGet(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE ideas, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	var authorID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('Autor de Teste') RETURNING id").Scan(&authorID); err != nil {
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

	// A página da ideia devolve o mesmo objeto do mural; id que não existe é 404.
	rec = getOne(t, h, strconv.FormatInt(first.ID, 10))
	if got := decode[ideaResponse](t, rec); rec.Code != 200 || got != first {
		t.Errorf("GET da primeira: %d %+v, quer 200 %+v", rec.Code, got, first)
	}
	if rec = getOne(t, h, "999999"); rec.Code != 404 || decode[errorResponse](t, rec).Error != "not_found" {
		t.Errorf("GET inexistente: %d (%s), quer 404 not_found", rec.Code, rec.Body)
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
