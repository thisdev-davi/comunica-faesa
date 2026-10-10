package ideas

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const (
	maxBodyBytes = 64 << 10 // 64 KB
	previewSize  = 3        // quantas ideias o visitante vê (docs/contratos/fatia-3-cadastro.md)
)

// Listas fixas do contrato (docs/contratos/fatia-1-mural.md). O front mantém a mesma lista com os rótulos.
// Courses é exportada porque o cadastro (pacote auth) valida o curso do aluno contra a mesma lista.
var (
	Courses    = map[string]bool{"cc": true, "eng": true, "ads": true}
	categories = map[string]bool{
		"web": true, "mobile": true, "ai": true, "data": true, "games": true,
		"competitive_programming": true, "security": true, "iot": true, "other": true,
	}
)

type Handler struct {
	q *Queries
	// userID diz quem está logado (0 = visitante). Recebe uma função, não o pacote auth: os testes trocam por um stub.
	userID func(*http.Request) (int64, error)
}

func NewHandler(db DBTX, userID func(*http.Request) (int64, error)) *Handler {
	return &Handler{q: New(db), userID: userID}
}

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Course      string `json:"course"`
	Category    string `json:"category"`
	Slots       int    `json:"slots"`
}

type authorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ideaResponse struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Course      string         `json:"course"`
	Category    string         `json:"category"`
	Slots       int32          `json:"slots"`
	Status      string         `json:"status"`
	Author      authorResponse `json:"author"`
	CreatedAt   time.Time      `json:"created_at"`
}

type errorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	authorID, err := h.userID(r)
	if err != nil {
		internalError(w, "session", err)
		return
	}
	if authorID == 0 {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthenticated"})
		return
	}

	var req createRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "body_too_large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_json"})
		return
	}

	if fields := req.validate(); len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "validation_failed", Fields: fields})
		return
	}

	row, err := h.q.CreateIdea(r.Context(), CreateIdeaParams{
		AuthorID:    authorID,
		Title:       req.Title,
		Description: req.Description,
		Course:      req.Course,
		Category:    req.Category,
		Slots:       int32(req.Slots), // seguro: validate garante 1–20
	})
	if err != nil {
		internalError(w, "create idea", err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(ListIdeasRow(row)))
}

// List devolve o mural: tudo para quem está logado, só as mais recentes para visitante.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userID(r)
	if err != nil {
		internalError(w, "session", err)
		return
	}
	var max *int32 // nil vira NULL no SQL: sem limite
	if userID == 0 {
		max = new(int32(previewSize)) // Go 1.26+: new aceita um valor e devolve o ponteiro para uma cópia dele
	}
	rows, err := h.q.ListIdeas(r.Context(), max)
	if err != nil {
		internalError(w, "list ideas", err)
		return
	}
	items := make([]ideaResponse, 0, len(rows)) // não-nil: lista vazia vira [] no JSON, não null
	for _, row := range rows {
		items = append(items, toResponse(row))
	}
	writeJSON(w, http.StatusOK, map[string][]ideaResponse{"items": items})
}

// Get devolve uma ideia, só para quem está logado: o visitante fica na prévia do mural.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userID(r)
	if err != nil {
		internalError(w, "session", err)
		return
	}
	if userID == 0 {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthenticated"})
		return
	}

	// Id que não é inteiro positivo é só uma ideia que não existe: 404, sem ir ao banco.
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	row, err := h.q.GetIdea(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	if err != nil {
		internalError(w, "get idea", err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(ListIdeasRow(row)))
}

// validate limpa os campos de texto (trim) e devolve os erros por campo; vazio = válido.
func (req *createRequest) validate() map[string]string {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Course = strings.TrimSpace(req.Course)
	req.Category = strings.TrimSpace(req.Category)

	fields := map[string]string{}
	checkText(fields, "title", req.Title, 3, 120)
	checkText(fields, "description", req.Description, 1, 5000)
	CheckOption(fields, "course", req.Course, Courses)
	CheckOption(fields, "category", req.Category, categories)
	if req.Slots < 1 || req.Slots > 20 {
		fields["slots"] = "deve ser entre 1 e 20"
	}
	return fields
}

func checkText(fields map[string]string, name, v string, minLen, maxLen int) {
	n := utf8.RuneCountInString(v) // caracteres, não bytes: "ação" tem 4
	switch {
	case n == 0:
		fields[name] = "obrigatório"
	case strings.ContainsRune(v, 0):
		fields[name] = "contém caractere inválido"
	case n < minLen || n > maxLen:
		fields[name] = fmt.Sprintf("deve ter entre %d e %d caracteres", minLen, maxLen)
	}
}

// CheckOption grava em fields o erro de um valor que precisa estar na lista; exportada para o cadastro usar as mesmas mensagens.
func CheckOption(fields map[string]string, name, v string, allowed map[string]bool) {
	switch {
	case v == "":
		fields[name] = "obrigatório"
	case !allowed[v]:
		fields[name] = "valor inválido"
	}
}

func toResponse(r ListIdeasRow) ideaResponse {
	return ideaResponse{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Course:      r.Course,
		Category:    r.Category,
		Slots:       r.Slots,
		Status:      r.Status,
		Author:      authorResponse{ID: r.AuthorID, Name: r.AuthorName},
		CreatedAt:   r.CreatedAt.UTC(),
	}
}

func internalError(w http.ResponseWriter, op string, err error) {
	slog.Error(op, "err", err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json", "err", err)
	}
}
