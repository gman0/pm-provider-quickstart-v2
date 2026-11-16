package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gman0/provider-quickstart-v2/pkg/portal/config"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/db"
	"github.com/gman0/provider-quickstart-v2/pkg/portal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	store, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	srv := server.New(cfg, store)
	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}
