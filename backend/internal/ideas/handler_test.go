package ideas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const validBody = `{"title":"App de carona","description":"Quero montar um app","course":"cc","category":"mobile","slots":3}`

func do(t *testing.T, hf http.HandlerFunc, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	hf(rec, httptest.NewRequest(method, "/api/ideas", strings.NewReader(body)))
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
		{"sem autor", 0, validBody, 401, "unauthenticated", nil},
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
			rec := do(t, NewHandler(nil, tc.authorID).Create, http.MethodPost, tc.body)
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
func TestCreateAndList(t *testing.T) {
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

	if _, err := pool.Exec(ctx, "TRUNCATE ideas, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	var authorID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (name) VALUES ('Autor de Teste') RETURNING id").Scan(&authorID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(pool, authorID)

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
}
