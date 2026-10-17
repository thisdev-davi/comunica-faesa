package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thisdev-davi/comunica-faesa/backend/internal/auth"
	"github.com/thisdev-davi/comunica-faesa/backend/internal/events"
	"github.com/thisdev-davi/comunica-faesa/backend/internal/ideas"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api parou", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var missing []string
	env := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}
	dbURL := env("DATABASE_URL")
	clientID, clientSecret := env("DISCORD_CLIENT_ID"), env("DISCORD_CLIENT_SECRET")
	redirectURL, guildID := env("DISCORD_REDIRECT_URL"), env("DISCORD_GUILD_ID")
	if len(missing) > 0 {
		return fmt.Errorf("variáveis não definidas: %s", strings.Join(missing, ", "))
	}
	port := cmp.Or(os.Getenv("PORT"), "8090")
	// Opcional: sem a variável, ninguém é moderador. Fields também descarta espaços e vírgulas sobrando ("1, 2,").
	modRoles := strings.Fields(strings.ReplaceAll(os.Getenv("DISCORD_MOD_ROLE_IDS"), ",", " "))

	// ctx é cancelado no Ctrl+C ou no SIGTERM do docker stop
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("configurar banco: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("conectar no banco: %w", err)
	}

	a := auth.NewHandler(pool, clientID, clientSecret, redirectURL, guildID, modRoles)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           routes(a, ideas.NewHandler(pool, a.UserID), events.NewHandler(pool, a.UserID)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("api no ar", "addr", srv.Addr, "cargos_de_moderacao", len(modRoles))
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	// Para de aceitar conexões e espera as requisições em andamento (até 5s; o docker stop mata em 10s).
	slog.Info("api desligando")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// routes monta a API atrás da proteção contra requisição de outra origem: um POST/PUT/PATCH/DELETE
// que o navegador marca como vindo de outro site (Sec-Fetch-Site) leva 403 antes de chegar no handler.
// O SameSite=Lax do cookie não basta aqui: o sslip.io não está na Public Suffix List, então para o
// navegador qualquer *.sslip.io é "o mesmo site" e mandaria o cookie junto.
func routes(a *auth.Handler, h *ideas.Handler, ev *events.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/discord", a.Login)
	mux.HandleFunc("GET /api/auth/discord/callback", a.Callback)
	mux.HandleFunc("POST /api/auth/logout", a.Logout)
	mux.HandleFunc("GET /api/me", a.Me)
	mux.HandleFunc("PATCH /api/me", a.UpdateMe)
	mux.HandleFunc("GET /api/me/ideas", h.Mine)
	mux.HandleFunc("GET /api/users/{id}", h.Profile)
	mux.HandleFunc("POST /api/ideas", h.Create)
	mux.HandleFunc("GET /api/ideas", h.List)
	mux.HandleFunc("GET /api/ideas/{id}", h.Get)
	mux.HandleFunc("PUT /api/ideas/{id}", h.Update)
	mux.HandleFunc("DELETE /api/ideas/{id}", h.Delete)
	mux.HandleFunc("PUT /api/ideas/{id}/status", h.SetStatus)
	mux.HandleFunc("PUT /api/ideas/{id}/interest", h.PutInterest)
	mux.HandleFunc("DELETE /api/ideas/{id}/interest", h.DeleteInterest)
	mux.HandleFunc("POST /api/events", ev.Create)
	mux.HandleFunc("GET /api/events", ev.List)
	mux.HandleFunc("GET /api/events/{id}", ev.Get)
	mux.HandleFunc("PUT /api/events/{id}", ev.Update)
	mux.HandleFunc("DELETE /api/events/{id}", ev.Delete)
	mux.HandleFunc("PUT /api/events/{id}/attendance", ev.PutAttendance)
	mux.HandleFunc("DELETE /api/events/{id}/attendance", ev.DeleteAttendance)
	p := http.NewCrossOriginProtection()
	p.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"error":"cross_origin"}`+"\n")
	}))
	return p.Handler(mux)
}
