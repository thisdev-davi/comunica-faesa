package main

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thisdev-davi/comunica-faesa/backend/internal/auth"
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

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("configurar banco: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("conectar no banco: %w", err)
	}

	a := auth.NewHandler(pool, clientID, clientSecret, redirectURL, guildID)
	h := ideas.NewHandler(pool, a.UserID)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/discord", a.Login)
	mux.HandleFunc("GET /api/auth/discord/callback", a.Callback)
	mux.HandleFunc("POST /api/auth/logout", a.Logout)
	mux.HandleFunc("GET /api/me", a.Me)
	mux.HandleFunc("POST /api/ideas", h.Create)
	mux.HandleFunc("GET /api/ideas", h.List)

	// ponytail: sem graceful shutdown; adicionar quando houver deploy.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("api no ar", "addr", srv.Addr)
	return srv.ListenAndServe()
}
