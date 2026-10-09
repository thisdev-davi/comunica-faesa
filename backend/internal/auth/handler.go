package auth

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/thisdev-davi/comunica-faesa/backend/internal/ideas"
)

const (
	sessionCookie = "session"
	stateCookie   = "oauth_state"
	sessionTTL    = 30 * 24 * time.Hour
	stateTTL      = 10 * time.Minute
	maxBodyBytes  = 64 << 10 // 64 KB
)

var errNotMember = errors.New("não é membro do servidor")

// ponytail: sessões expiradas ficam na tabela (só são ignoradas na leitura); limpar com um DELETE periódico se ela crescer.
type Handler struct {
	q          *Queries
	oauth      *oauth2.Config
	discordAPI string // base da API REST do Discord; os testes apontam para um servidor falso
	guildID    string
	secure     bool // cookies só por HTTPS: ligado quando a Redirect URL é https (produção)
}

func NewHandler(db DBTX, clientID, clientSecret, redirectURL, guildID string) *Handler {
	return &Handler{
		q: New(db),
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     endpoints.Discord,
			Scopes:       []string{"identify", "guilds.members.read"},
		},
		discordAPI: "https://discord.com/api/v10",
		guildID:    guildID,
		secure:     strings.HasPrefix(redirectURL, "https://"),
	}
}

// Login manda o navegador para o Discord. O state aleatório volta no callback e prova que o login começou aqui (anti-CSRF).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	state := rand.Text()
	h.setCookie(w, stateCookie, state, "/api/auth", int(stateTTL.Seconds()))
	http.Redirect(w, r, h.oauth.AuthCodeURL(state), http.StatusFound)
}

// Callback recebe a volta do Discord e sempre responde com redirect: para / (ou /cadastro, se falta o curso) se deu certo, para /login?erro=... se não.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	h.setCookie(w, stateCookie, "", "/api/auth", -1) // o state vale uma vez só
	q := r.URL.Query()
	if q.Get("error") != "" {
		toLogin(w, r, "cancelado")
		return
	}
	c, err := r.Cookie(stateCookie)
	// Comparação em tempo constante: o tempo da resposta não revela quantos caracteres batem.
	if err != nil || q.Get("state") == "" || subtle.ConstantTimeCompare([]byte(c.Value), []byte(q.Get("state"))) != 1 {
		toLogin(w, r, "falhou")
		return
	}

	user, err := h.discordUser(r.Context(), q.Get("code"))
	if errors.Is(err, errNotMember) {
		toLogin(w, r, "fora_do_servidor")
		return
	}
	if err == nil {
		err = h.startSession(r.Context(), w, user.ID)
	}
	if err != nil {
		slog.Error("login discord", "err", err)
		toLogin(w, r, "falhou")
		return
	}
	dest := "/"
	if user.Course == nil {
		dest = "/cadastro" // primeiro login: falta escolher o curso
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

// Me devolve o usuário da sessão, ou 401.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.session(r)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}
	if err != nil {
		internalError(w, "me", err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// SetCourse grava o curso do aluno logado (cadastro) e devolve o mesmo corpo do Me.
func (h *Handler) SetCourse(w http.ResponseWriter, r *http.Request) {
	u, err := h.session(r)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}
	if err != nil {
		internalError(w, "session", err)
		return
	}

	var req struct {
		Course string `json:"course"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "body_too_large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	course := strings.TrimSpace(req.Course)
	fields := map[string]string{}
	ideas.CheckOption(fields, "course", course, ideas.Courses)
	if len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation_failed", "fields": fields})
		return
	}

	if err := h.q.SetUserCourse(r.Context(), SetUserCourseParams{ID: u.ID, Course: &course}); err != nil {
		internalError(w, "set course", err)
		return
	}
	u.Course = &course
	writeJSON(w, http.StatusOK, u)
}

// Logout apaga a sessão no banco e o cookie. Responde 204 mesmo sem sessão: o resultado é o mesmo.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := h.q.DeleteSession(r.Context(), hashToken(c.Value)); err != nil {
			internalError(w, "logout", err)
			return
		}
	}
	h.setCookie(w, sessionCookie, "", "/", -1)
	w.WriteHeader(http.StatusNoContent)
}

// UserID devolve o id do usuário logado, ou 0 para visitante. Outros domínios recebem esta função para saber quem é o autor.
func (h *Handler) UserID(r *http.Request) (int64, error) {
	u, err := h.session(r)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return u.ID, err
}

// session busca o usuário do cookie; pgx.ErrNoRows quando não há sessão válida (sem cookie, desconhecida ou expirada).
func (h *Handler) session(r *http.Request) (GetSessionUserRow, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return GetSessionUserRow{}, pgx.ErrNoRows
	}
	return h.q.GetSessionUser(r.Context(), hashToken(c.Value))
}

// discordUser troca o code pelo token, confere se a pessoa está no servidor e grava (ou atualiza) o usuário.
func (h *Handler) discordUser(ctx context.Context, code string) (UpsertDiscordUserRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tok, err := h.oauth.Exchange(ctx, code)
	if err != nil {
		return UpsertDiscordUserRow{}, fmt.Errorf("trocar code: %w", err)
	}
	client := h.oauth.Client(ctx, tok) // põe o "Authorization: Bearer" em cada chamada

	var u struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"` // null no JSON vira ""
		Avatar     string `json:"avatar"`
	}
	if _, err := get(ctx, client, h.discordAPI+"/users/@me", &u); err != nil {
		return UpsertDiscordUserRow{}, err
	}
	if u.ID == "" {
		return UpsertDiscordUserRow{}, errors.New("discord não devolveu o id do usuário")
	}
	status, err := get(ctx, client, h.discordAPI+"/users/@me/guilds/"+h.guildID+"/member", nil)
	if status == http.StatusNotFound {
		return UpsertDiscordUserRow{}, errNotMember
	}
	if err != nil {
		return UpsertDiscordUserRow{}, err
	}

	var avatar *string
	if u.Avatar != "" {
		url := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", u.ID, u.Avatar)
		avatar = &url
	}
	return h.q.UpsertDiscordUser(ctx, UpsertDiscordUserParams{
		DiscordID: &u.ID,
		Name:      cmp.Or(u.GlobalName, u.Username),
		AvatarUrl: avatar,
	})
}

// startSession cria um token aleatório: o navegador guarda o token, o banco guarda só o hash.
func (h *Handler) startSession(ctx context.Context, w http.ResponseWriter, userID int64) error {
	token := rand.Text()
	err := h.q.CreateSession(ctx, CreateSessionParams{
		TokenHash: hashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionTTL),
	})
	if err != nil {
		return fmt.Errorf("criar sessão: %w", err)
	}
	h.setCookie(w, sessionCookie, token, "/", int(sessionTTL.Seconds()))
	return nil
}

// maxAge em segundos; -1 apaga o cookie.
func (h *Handler) setCookie(w http.ResponseWriter, name, value, path string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true, // JavaScript da página não lê: um XSS não rouba a sessão
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode, // outro site não consegue fazer POST levando o cookie
	})
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// get faz um GET na API do Discord e decodifica a resposta em v (se não for nil). Devolve o status para quem precisa distinguir o 404.
func get(ctx context.Context, client *http.Client, url string, v any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return res.StatusCode, fmt.Errorf("GET %s: status %d", url, res.StatusCode)
	}
	if v == nil {
		return res.StatusCode, nil
	}
	return res.StatusCode, json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(v)
}

func toLogin(w http.ResponseWriter, r *http.Request, reason string) {
	http.Redirect(w, r, "/login?erro="+reason, http.StatusFound)
}

// writeJSON e internalError repetem os do pacote ideas de propósito: são 10 linhas, e um pacote "comum" iria contra a regra do projeto.
func internalError(w http.ResponseWriter, op string, err error) {
	slog.Error(op, "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json", "err", err)
	}
}
