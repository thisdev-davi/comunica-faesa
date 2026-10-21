package events

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/thisdev-davi/comunica-faesa/backend/internal/ideas"
)

const maxBodyBytes = 64 << 10 // 64 KB

type Handler struct {
	q *Queries
	// viewer diz quem está logado (0 = visitante) e se é moderador, os dois lidos da mesma sessão.
	// Recebe uma função, não o pacote auth: os testes trocam por um stub.
	viewer func(*http.Request) (userID int64, moderator bool, err error)
}

func NewHandler(db DBTX, viewer func(*http.Request) (userID int64, moderator bool, err error)) *Handler {
	return &Handler{q: New(db), viewer: viewer}
}

// Corpo do POST e do PUT: criar e editar trocam os mesmos campos.
// starts_at chega como texto para o erro de formato sair por campo (com time.Time no struct,
// o Decode falharia inteiro com invalid_json); startsAt guarda o valor já convertido pelo validate.
type eventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAt    string `json:"starts_at"`
	Location    string `json:"location"`
	startsAt    time.Time
}

type authorResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

type eventResponse struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	StartsAt    time.Time      `json:"starts_at"`
	Location    string         `json:"location"`
	Author      authorResponse `json:"author"`
	CreatedAt   time.Time      `json:"created_at"`
	// Quantos confirmaram presença e se quem pede é um deles.
	AttendeeCount int64 `json:"attendee_count"`
	Going         bool  `json:"going"`
	// Só quem está logado recebe a lista. omitzero tira só o nil: para o visitante o campo some,
	// e o logado num evento sem confirmados recebe [].
	Attendees []attendeeResponse `json:"attendees,omitzero"`
}

// Mesmos campos, com os mesmos nomes, do ListAttendeesRow gerado (por isso AvatarUrl): a conversão é direta.
type attendeeResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	AvatarUrl *string `json:"avatar_url"`
}

type errorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// Create cria um evento; qualquer aluno logado, sem aprovação.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	authorID, _, ok := h.loggedIn(w, r)
	if !ok {
		return
	}
	req, ok := readEvent(w, r)
	if !ok {
		return
	}
	id, err := h.q.CreateEvent(r.Context(), CreateEventParams{
		AuthorID:    authorID,
		Title:       req.Title,
		Description: req.Description,
		StartsAt:    req.startsAt,
		Location:    req.Location,
	})
	if errors.Is(err, pgx.ErrNoRows) { // o insert só grava para quem já escolheu o curso
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "course_required"})
		return
	}
	if err != nil {
		internalError(w, "create event", err)
		return
	}
	h.writeEvent(w, r, http.StatusCreated, id, authorID)
}

// List é a agenda: os eventos de hoje em diante, do mais próximo. Para todo mundo, visitante incluído.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	viewer, _, err := h.viewer(r)
	if err != nil {
		internalError(w, "session", err)
		return
	}
	rows, err := h.q.ListEvents(r.Context(), viewer)
	if err != nil {
		internalError(w, "list events", err)
		return
	}
	items := make([]eventResponse, 0, len(rows)) // não-nil: lista vazia vira [] no JSON, não null
	for _, row := range rows {
		items = append(items, toResponse(row))
	}
	writeJSON(w, http.StatusOK, map[string][]eventResponse{"items": items})
}

// Update troca os campos do evento (os mesmos do criar). O criador ou um moderador.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := h.organizerOf(w, r)
	if !ok {
		return
	}
	req, ok := readEvent(w, r)
	if !ok {
		return
	}
	err := h.q.UpdateEvent(r.Context(), UpdateEventParams{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		StartsAt:    req.startsAt,
		Location:    req.Location,
	})
	if err != nil {
		internalError(w, "update event", err)
		return
	}
	h.writeEvent(w, r, http.StatusOK, id, userID)
}

// writeEvent responde com o evento que o POST ou o PUT acabou de gravar, no mesmo formato do GET (sem a lista de presença).
// going é do ponto de vista de quem gravou: a moderadora que edita evento alheio vê se ela vai.
func (h *Handler) writeEvent(w http.ResponseWriter, r *http.Request, status int, id, viewer int64) {
	row, err := h.q.GetEvent(r.Context(), GetEventParams{ID: id, Viewer: viewer})
	if errors.Is(err, pgx.ErrNoRows) { // apagado entre a gravação e aqui (o UPDATE de um evento que sumiu não acusa erro)
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	if err != nil {
		internalError(w, "get event", err)
		return
	}
	writeJSON(w, status, toResponse(ListEventsRow(row)))
}

// Delete apaga o evento de vez, e a lista de presença vai junto (ON DELETE CASCADE). O criador ou um moderador.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	_, id, ok := h.organizerOf(w, r)
	if !ok {
		return
	}
	if err := h.q.DeleteEvent(r.Context(), id); err != nil {
		internalError(w, "delete event", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// organizerOf devolve quem chama e o id do evento da rota quando quem chama é o criador dele ou um moderador.
// Senão já responde 401 (sem sessão), 404 (evento não existe) ou 403 (nem criador nem moderador) e devolve ok = false.
func (h *Handler) organizerOf(w http.ResponseWriter, r *http.Request) (userID, id int64, ok bool) {
	userID, moderator, ok := h.loggedIn(w, r)
	if !ok {
		return 0, 0, false
	}
	id, ok = pathID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return 0, 0, false
	}
	authorID, err := h.q.GetEventAuthor(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return 0, 0, false
	}
	if err != nil {
		internalError(w, "get event author", err)
		return 0, 0, false
	}
	if authorID != userID && !moderator {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "not_author"})
		return 0, 0, false
	}
	return userID, id, true
}

// Get devolve um evento, para todo mundo; também os que já passaram. Quem está logado recebe a lista de quem vai.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	viewer, _, err := h.viewer(r)
	if err != nil {
		internalError(w, "session", err)
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	row, err := h.q.GetEvent(r.Context(), GetEventParams{ID: id, Viewer: viewer})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	if err != nil {
		internalError(w, "get event", err)
		return
	}
	resp := toResponse(ListEventsRow(row))
	if viewer != 0 {
		people, err := h.q.ListAttendees(r.Context(), id)
		if err != nil {
			internalError(w, "list attendees", err)
			return
		}
		resp.Attendees = make([]attendeeResponse, 0, len(people)) // não-nil: sem confirmados, o logado recebe []
		for _, p := range people {
			resp.Attendees = append(resp.Attendees, attendeeResponse(p))
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// PutAttendance é o "vou": sem aprovação nem limite. Repetir não muda nada. Só evento da agenda (de hoje em diante).
func (h *Handler) PutAttendance(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := h.loggedIn(w, r)
	if !ok {
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	// A regra mora no próprio insert: conferir antes e gravar depois deixaria o evento sair da agenda no meio.
	added, err := h.q.AddAttendee(r.Context(), AddAttendeeParams{EventID: id, UserID: userID})
	if err != nil {
		internalError(w, "add attendee", err)
		return
	}
	if added == 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Não gravou: a consulta abaixo só escolhe a resposta.
	upcoming, err := h.q.IsUpcoming(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	if err != nil {
		internalError(w, "is upcoming", err)
		return
	}
	if !upcoming {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "past"})
		return
	}
	w.WriteHeader(http.StatusNoContent) // na agenda: a pessoa já estava na lista
}

// DeleteAttendance é o "desistir", até em evento passado. Responde 204 mesmo sem presença ou sem evento:
// o resultado é o mesmo, a pessoa não está na lista.
func (h *Handler) DeleteAttendance(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := h.loggedIn(w, r)
	if !ok {
		return
	}
	if id, ok := pathID(r); ok {
		if err := h.q.RemoveAttendee(r.Context(), RemoveAttendeeParams{EventID: id, UserID: userID}); err != nil {
			internalError(w, "remove attendee", err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// loggedIn devolve quem está logado e se é moderador; sem sessão já responde 401 (ou 500 se a sessão falhou) e devolve ok = false.
func (h *Handler) loggedIn(w http.ResponseWriter, r *http.Request) (userID int64, moderator, ok bool) {
	userID, moderator, err := h.viewer(r)
	if err != nil {
		internalError(w, "session", err)
		return 0, false, false
	}
	if userID == 0 {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthenticated"})
		return 0, false, false
	}
	return userID, moderator, true
}

// pathID lê o {id} da rota. Id que não é inteiro positivo é só um evento que não existe: quem chama responde 404 sem ir ao banco.
func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

// readEvent lê e valida o corpo. Com erro, já responde 400 ou 413 e devolve ok = false.
func readEvent(w http.ResponseWriter, r *http.Request) (req eventRequest, ok bool) {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "body_too_large"})
			return req, false
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_json"})
		return req, false
	}
	if fields := req.validate(time.Now()); len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "validation_failed", Fields: fields})
		return req, false
	}
	return req, true
}

// validate limpa os textos (trim), converte starts_at e devolve os erros por campo; vazio = válido.
// now vem de fora para o teste do "no futuro" não depender do relógio.
func (req *eventRequest) validate(now time.Time) map[string]string {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Location = strings.TrimSpace(req.Location)
	req.StartsAt = strings.TrimSpace(req.StartsAt)

	fields := map[string]string{}
	ideas.CheckText(fields, "title", req.Title, 3, 120)
	ideas.CheckText(fields, "description", req.Description, 1, 5000)
	ideas.CheckText(fields, "location", req.Location, 3, 120)
	t, err := time.Parse(time.RFC3339, req.StartsAt) // RFC3339 exige o fuso: "2026-10-20T19:00:00" sem ele é recusado
	switch {
	case req.StartsAt == "":
		fields["starts_at"] = "obrigatório"
	case err != nil:
		fields["starts_at"] = "data inválida"
	case !t.After(now):
		fields["starts_at"] = "deve ser no futuro"
	}
	req.startsAt = t
	return fields
}

func toResponse(r ListEventsRow) eventResponse {
	return eventResponse{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		StartsAt:    r.StartsAt.UTC(),
		Location:    r.Location,
		Author:      authorResponse{ID: r.AuthorID, Name: r.AuthorName, AvatarURL: r.AuthorAvatarUrl},
		CreatedAt:   r.CreatedAt.UTC(),

		AttendeeCount: r.AttendeeCount,
		Going:         r.Going,
	}
}

// writeJSON e internalError (como loggedIn e pathID) repetem os do pacote ideas de propósito, como no auth: um pacote "comum" iria contra a regra do projeto.
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
