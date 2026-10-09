package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thisdev-davi/comunica-faesa/backend/internal/ideas"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api parou", "err", err)
		os.Exit(1)
	}
}

func run() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL não definida")
	}
	port := cmp.Or(os.Getenv("PORT"), "8090")

	var authorID int64
	if v := os.Getenv("DEV_AUTHOR_ID"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id < 1 {
			return fmt.Errorf("DEV_AUTHOR_ID inválido: %q", v)
		}
		authorID = id
		slog.Warn("DEV_AUTHOR_ID definido: POST /api/ideas grava sem login, use só localmente")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("configurar banco: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("conectar no banco: %w", err)
	}

	h := ideas.NewHandler(pool, authorID)
	mux := http.NewServeMux()
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
