package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/edubarr/airtrak/backend/gen/airtrak/auth/v1/authv1connect"
	"github.com/edubarr/airtrak/backend/gen/airtrak/health/v1/healthv1connect"
	"github.com/edubarr/airtrak/backend/internal/api"
	"github.com/edubarr/airtrak/backend/internal/auth"
	"github.com/edubarr/airtrak/backend/internal/config"
	"github.com/edubarr/airtrak/backend/internal/db"
	"github.com/edubarr/airtrak/backend/internal/store"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	queries := store.New(pool)

	authService, err := auth.NewService(pool, queries, auth.Config{
		JWTSecret:      cfg.JWTSecret,
		AccessTokenTTL: cfg.AccessTokenTTL,
		DefaultLocale:  cfg.DefaultLocale,
	})
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	interceptors := connect.WithInterceptors(api.AuthInterceptor(authService))

	authPath, authHandler := authv1connect.NewAuthServiceHandler(api.NewAuthHandler(authService), interceptors)
	mux.Handle(authPath, authHandler)

	healthPath, healthHandler := healthv1connect.NewHealthServiceHandler(api.NewHealthHandler(pool))
	mux.Handle(healthPath, healthHandler)

	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("starting API server", "addr", cfg.APIAddr)

	return server.ListenAndServe()
}
